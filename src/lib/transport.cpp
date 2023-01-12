/*
 * common APIs used by events code.
 */
#include <stdio.h>
#include <chrono>
#include <fstream>
#include <errno.h>
#include <map>
#include "string.h"
#include "json.hpp"
#include "zmq.h"
#include <unordered_map>

#include "common.h"

using namespace std;
using namespace chrono;
  
/*
 * Config that can be read from init_cfg
 */
#define INIT_CFG_PATH "/etc/LoM/init_cfg.json"

/* configurable entities' keys */
/* zmq proxy's sub & pub end points */
#define SUB_END_KEY "sub_path"       
#define PUB_END_KEY "pub_path"

#define STATS_UPD_SECS "stats_upd_secs"


/*
 * ZMQ socket will not close, if it has messages to send.
 * Set max time for wait.
 */
static const int LINGER_TIMEOUT = 100;  /* Linger timeout in millisec ll config entries */

typedef map<string, string> map_str_str_t;
#define CFG_VAL map_str_str_t::value_type

static const map_str_str_t s_cfg_default = {
    CFG_VAL(SUB_END_KEY, "tcp://127.0.0.1:5578"),
    CFG_VAL(PUB_END_KEY, "tcp://127.0.0.1:5579"),
    CFG_VAL(STATS_UPD_SECS, "5")
};

static void
_read_init_config(const char *init_cfg_file)
{
    /* Set default and override from file */
    cfg_data = s_cfg_default;

    if (init_cfg_file == NULL) {
        return;
    }

    ifstream fs (init_cfg_file);

    if (!fs.is_open()) 
        return;

    stringstream buffer;
    buffer << fs.rdbuf();

    const auto &data = nlohmann::json::parse(buffer.str());

    const auto it = data.find(CFG_EVENTS_KEY);
    if (it == data.end())
        return;

    const auto edata = *it;
    for (map_str_str_t::iterator itJ = cfg_data.begin();
            itJ != cfg_data.end(); ++itJ) {
        auto itE = edata.find(itJ->first);
        if (itE != edata.end()) {
            itJ->second = *itE;
        }
    }

    return;
}

static string
_get_config(const string key)
{
    if (cfg_data.empty()) {
        read_init_config(INIT_CFG_PATH);
    }   
    /* Intentionally crash for non-existing key, as this
     * is internal code bug
     */
    return cfg_data[key];
}

static const string
_get_timestamp()
{
    stringstream ss, sfrac;

    auto timepoint = system_clock::now();
    time_t tt = system_clock::to_time_t (timepoint);
    struct tm * ptm = localtime(&tt);

    uint64_t ms = duration_cast<microseconds>(timepoint.time_since_epoch()).count();
    uint64_t sec = duration_cast<seconds>(timepoint.time_since_epoch()).count();
    uint64_t mfrac = ms - (sec * 1000 * 1000);

    sfrac << mfrac;

    ss << put_time(ptm, "%FT%H:%M:%S.") << sfrac.str().substr(0, 6) << "Z";
    return ss.str();
}

/*
 * events are published as two part zmq message.
 * First part only has the event source, so receivers could
 * filter by source.
 *
 * Second part contains JSON String of the data being sent.
 */

static int 
_zmq_read_part(void *sock, int flag, int &more, string &data)
{
    zmq_msg_t msg;

    more = 0;
    zmq_msg_init(&msg);
    int rc = zmq_msg_recv(&msg, sock, flag);
    if (rc != -1) {
        size_t more_size = sizeof (more);

        zmq_getsockopt (sock, ZMQ_RCVMORE, &more, &more_size);

        data = string((const char *)zmq_msg_data(&msg), zmq_msg_size(&msg));
    }
    else {
        /* override with zmq err */
        rc = zmq_errno();
        RET_ON_ERR(rc == 11, "Failure to read part rc=%d", rc);
    }
    rc = 0;
out:
    zmq_msg_close(&msg);

    return rc;
}

   
static int
_zmq_send_part(void *sock, int flag, const string &data)
{
    zmq_msg_t msg;

    int rc = zmq_msg_init_size(&msg, data.size());
    RET_ON_ERR(rc == 0, "Failed to init msg size=%d", data.size());

    strncpy((char *)zmq_msg_data(&msg), data.c_str(), data.size());

    rc = zmq_msg_send (&msg, sock, flag);
    if (rc == -1) {
        /* override with zmq err */
        rc = zmq_errno();
        RET_ON_ERR(false, "Failed to send part %d", rc);
    }
    /* zmq_msg_send returns count of bytes sent */
    rc = 0;
out:
    zmq_msg_close(&msg);
    return rc;
}

static int
_zmq_message_send(void *sock, const string &pt1, const string &pt2)
{
    int rc = -1;
    RET_ON_ERR(!pt1.empty() && !pt2.empty(),
            "Expect non-empty pt1=%d pt2=%d", pt1.size(), pt2.size());

    rc = zmq_send_part(sock, ZMQ_SNDMORE, pt1);

    /* send second part, only if first is sent successfully */
    if (rc == 0) {
        rc = zmq_send_part(sock, 0, pt2);
    }
out:
    return rc;
}

   
static int
_zmq_message_read(void *sock, int flag, string &pt1, string &pt2)
{
    int more = 0, rc, rc2 = 0;

    rc = zmq_read_part(sock, flag, more, pt1);

    RET_ON_ERR (more, "Expect two part message PT1=%s", pt1.c_str());

    /*
     * read second part if more is set, irrespective
     * of any failure. More is set, only if sock is valid.
     */
    rc2 = zmq_read_part(sock, 0, more, pt2);

    RET_ON_ERR((rc == 0) || (rc == 11), "Failure to read part1 rc=%d", rc);
    if (rc2 != 0) {
        rc = rc2;
        RET_ON_ERR(false, "Failed to read part2 rc=%d", rc);
    }
    if (more) {
        rc = -1;
        RET_ON_ERR(false, "Don't expect more than 2 parts, rc=%d", rc);
    }
out:
    return rc;
}

class transport
{
    public:
        running_mode(int rd_timeout_ms = -1):
            m_zmq_ctx(NULL), m_is_client_mode(false),
            m_wr_sock(NULL), m_rd_sock(NULL), m_rd_timeout_ms(rd_timeout_ms)
        {};

        virtual ~running_mode() {
            zmq_close(m_wr_sock);
            zmq_close(m_rd_sock);
            zmq_ctx_term(m_zmq_ctx);
        }

        bool is_valid() {
            return (m_wr_sock != NULL);
        }

        int set_mode(const string client_name = string())
        {
            int rc = 0;

            void *zmq_ctx = zmq_ctx_new();
            void *wr_sock = NULL;
            void *rd_sock = NULL;

            wr_sock = zmq_socket (zmq_ctx, ZMQ_PUB);
            RET_ON_ERR(wr_sock != NULL, "Failed to ZMQ_PUB socket");

            rc = zmq_setsockopt (wr_sock, ZMQ_LINGER, &LINGER_TIMEOUT, sizeof (LINGER_TIMEOUT));
            RET_ON_ERR(rc == 0, "Failed to ZMQ_LINGER to %d", LINGER_TIMEOUT);

            rc = zmq_connect (wr_sock, get_config(SUB_END_KEY).c_str());
            RET_ON_ERR(rc == 0, "client fails to connect %s",
                    get_config(SUB_END_KEY).c_str());

            rd_sock = zmq_socket (zmq_ctx, ZMQ_SUB);
            RET_ON_ERR(rd_sock != NULL, "Failed to ZMQ_PUB socket");

            rc = zmq_connect (rd_sock, get_config(PUB_END_KEY).c_str());
            RET_ON_ERR(rc == 0, "client fails to connect %s",
                    get_config(PUB_END_KEY).c_str());

            /* client_name empty in server mode. Hence subscribe to any */
            rc = zmq_setsockopt(rd_sock, ZMQ_SUBSCRIBE, client_name.c_str(),
                    client_name.size());
            RET_ON_ERR(rc == 0, "Fails to set option rc=%d", rc);

            if (m_rd_timeout_ms != -1) {
                rc = zmq_setsockopt (rd_sock, ZMQ_RCVTIMEO, &m_rd_timeout_ms,
                        sizeof (m_rd_timeout_ms));
                RET_ON_ERR(rc == 0, "Failed to ZMQ_RCVTIMEO to %d", m_rd_timeout_ms);
            }

            m_is_client_mode = !client_name.empty();
            m_client_name = client_name;
            m_wr_sock = wr_sock;
            m_rd_sock = rd_sock;
            m_zmq_ctx = zmq_ctx;
            zmq_ctx = NULL;

            {
                stringstream ss;
                ss << "is_client: " << m_is_client_mode << " client:" << m_client_name;
                m_self_str = ss.str();
            }
        out:
            if (zmq_ctx != NULL) {
                zmq_close(wr_sock);
                zmq_close(rd_sock);
            }
            return rc;
        }

        int write(const string msg, const string dest = string())
        {
            int rc = 0;
            RET_ON_ERR(m_is_client_mode == dest.empty(),
                    "Client specifies no dest; server specifies. self(%s) dest:(%s)",
                    m_self_str.c_str(), dest.c_str());

            /* Set sender name if from client. Server receives any. */
            rc = _zmq_message_send(m_wr_sock, dest.empty() ? m_client_name : dest, msg);
            RET_ON_ERR(rc == 0, "Failed to send self(%s) rc=%d", m_self_str.c_str(), rc);
        out:
            return rc;
        }

        int read(string &client_id, string &msg, bool dont_wait = false)
        {
            int rc = 0;

            rc = _zmq_message_read(m_rd_sock, dont_wait ? ZMQ_DONTWAIT : 0,
                    client_id, msg);
            RET_ON_ERR(rc == 0, "Failed to recv self(%s) rc=%d", m_self_str.c_str(), rc);
        out:
            return rc;
        }

        int read_sock() { return m_rd_sock; };


    private:
        void *m_zmq_ctx;
        bool m_is_client_mode;
        string m_client_name;
        void *m_wr_sock;
        void *m_rd_sock;
        int m_rd_timeout_ms;

        string m_self_str;

};

typedef shared_ptr<transport> transport_ptr_t;

/* ZMQ sockets are not thread safe. Protect from accidental use across threads */
thread_local transport_ptr_t t_transport;

int
init_client_transport(const string client_name)
{
    /*
     * Only one transport expected per process. This flag to help capture
     * design/implementation level misuse. Hence not thread protected.
     */
    static bool s_tx_initialized = false;

    int rc = 0;
    RET_ON_ERR(!s_tx_initialized, "Duplicate init/multi-init");
    RET_ON_ERR(!client_name.empty(), "Require non-empty client name");

    transport_ptr_t tx(new transport());

    tx->set_mode(client_name);
    RET_ON_ERR(tx->is_valid(), "Failed to init transport for client (%s)",
            client_name.c_str());
    t_transport = tx;
    s_tx_initialized = true;
out:
    return rc;

}


int
init_server_transport(void)
{
    /* Called only by engine */
    int rc = 0;
    RET_ON_ERR(t_transport.empty(), "Duplicate init");

    transport_ptr_t tx(new transport());

    tx->set_mode();
    RET_ON_ERR(tx->is_valid(), "Failed to init transport for server");
    t_transport = tx;
out:
    return rc;

}

int
close_transport()
{
    t_transport.reset(NULL);
}

int
write_transport(const string msg, const string dest = string())
{
    int rc = 0;

    RET_ON_ERR(!t_transport.empty(), "No transport available to write.");

    rc = t_transport->write(msg, dest);
out:
    return rc;
}

int read_transport(string &client_id, string &msg, bool dont_wait = false)
{
    int rc = 0;

    RET_ON_ERR(!t_transport.empty(), "No transport available to read.");

    rc = t_transport->read(client_id, msg, dont_wait);
out:
    return rc;
}


int poll_for_data(int *lst_fds, int cnt, int timeout)
{
    zmq_pollitem_t items[cnt+1];

    items[0].socket = t_transport->read_sock();
    items[0].events = ZMQ_POLLIN;

    zmq_pollitem_t *p = items + 1;
    for(int i=0; i<cnt; ++i, ++p) {
        p->fd = *lst_fds++;
        p->events = ZMQ_POLLIN;
    }

    int rc = zmq_poll (items, cnt+1, timeout);
    switch (rc) {
    case -1:
        return -3;
    
    case 0:
        /* timeout */
        return -2;

    default:
        break;
    }

    if (items[0].revents & ZMQ_POLLIN) {
        /* Data available from engine */
        return -1;
    }
    for(int i=1; i<= cnt; ++i) {
        if (items[i] & ZMQ_POLLIN) {
            return items[i].fd;
        }
    }
    /* Unexpected result */
    return -3;
}


