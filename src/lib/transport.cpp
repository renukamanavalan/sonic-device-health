/*
 * common APIs used by events code.
 */
#include <stdio.h>
#include <chrono>
#include <fstream>
#include <sstream>
#include <errno.h>
#include <map>
#include <string.h>
#include <iomanip>      // std::put_time
#include <ctime>        // std::time_t, struct std::tm, std::localtime
#include <chrono>       // std::chrono::system_clock
#include <nlohmann/json.hpp>
#include <unistd.h>
#include <unordered_map>
#include "zmq.h"

#include "common.h"
#include "consts.h"
#include "transport.h"

using namespace std;
using namespace chrono;

#define TO_MS(n) ((n) * 1000)
  
/*
 * Config that can be read from init_cfg
 */
#define INIT_CFG_PATH "/etc/LoM/init_cfg.json"

/* configurable entities' keys */
/* zmq proxy's sub & pub end points */
#define CFG_SUB_END_KEY "sub_path"       
#define CFG_PUB_END_KEY "pub_path"

#define CFG_STATS_UPD_SECS "stats_upd_secs"

#define CFG_TRANSPORT_KEY  "LOM_TRANSPORT"


/*
 * ZMQ socket will not close, if it has messages to send.
 * Set max time for wait.
 */
static const int LINGER_TIMEOUT = 100;  /* Linger timeout in millisec ll config entries */

typedef map<string, string> map_str_str_t;
#define CFG_VAL map_str_str_t::value_type

static const map_str_str_t s_cfg_default = {
    CFG_VAL(CFG_SUB_END_KEY, SUB_END_PATH),
    CFG_VAL(CFG_PUB_END_KEY, PUB_END_PATH),
    CFG_VAL(CFG_STATS_UPD_SECS, "5")
};

static map_str_str_t s_cfg_data;

static void
_read_init_config(const char *init_cfg_file)
{
    /* Set default and override from file */
    s_cfg_data = s_cfg_default;

    if (init_cfg_file == NULL) {
        return;
    }

    ifstream fs (init_cfg_file);

    if (!fs.is_open()) 
        return;

    stringstream buffer;
    buffer << fs.rdbuf();

    const auto &data = nlohmann::json::parse(buffer.str());

    const auto it = data.find(CFG_TRANSPORT_KEY);
    if (it == data.end())
        return;

    const auto edata = *it;
    for (map_str_str_t::iterator itJ = s_cfg_data.begin();
            itJ != s_cfg_data.end(); ++itJ) {
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
    if (s_cfg_data.empty()) {
        _read_init_config(INIT_CFG_PATH);
    }   
    /* Intentionally crash for non-existing key, as this
     * is internal code bug
     */
    return s_cfg_data[key];
}

const string
get_timestamp()
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

    rc = _zmq_send_part(sock, ZMQ_SNDMORE, pt1);

    /* send second part, only if first is sent successfully */
    if (rc == 0) {
        rc = _zmq_send_part(sock, 0, pt2);
    }
out:
    return rc;
}

   
static int
_zmq_message_read(void *sock, int flag, string &pt1, string &pt2)
{
    int more = 0, rc, rc2 = 0;

    rc = _zmq_read_part(sock, flag, more, pt1);

    RET_ON_ERR (more, "Expect two part message PT1=%s", pt1.c_str());

    /*
     * read second part if more is set, irrespective
     * of any failure. More is set, only if sock is valid.
     */
    rc2 = _zmq_read_part(sock, 0, more, pt2);

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

class zmq_shared {
    public:
        zmq_shared() : m_zmq_ctx(NULL) {};

        ~zmq_shared() { zmq_ctx_term(m_zmq_ctx); }

        void *ctx() {
            if (m_zmq_ctx == NULL) {
                m_zmq_ctx = zmq_ctx_new();
            }
            return m_zmq_ctx;
        }

    protected:
        void *m_zmq_ctx;
};

typedef shared_ptr<zmq_shared> zmq_shared_ptr_t;
static zmq_shared_ptr_t s_shared_ctx;


class transportImpl : public transport {
    public:
        transportImpl(int rd_timeout = -1):
            m_wr_sock(NULL), m_rd_sock(NULL),
            m_rd_timeout(rd_timeout)
        {};

        virtual ~transportImpl() {
            zmq_close(m_wr_sock);
            zmq_close(m_rd_sock);
        }

        virtual bool is_valid() {
            return (m_wr_sock != NULL);
        }

        virtual int set_mode(const string client_name)
        {
            int rc = 0;

            if (s_shared_ctx == NULL) {
                s_shared_ctx = zmq_shared_ptr_t(new zmq_shared());
            }
            void *z_ctx = s_shared_ctx->ctx();
            void *wr_sock = NULL;
            void *rd_sock = NULL;

            wr_sock = zmq_socket (z_ctx, ZMQ_PUB);
            RET_ON_ERR(wr_sock != NULL, "Failed to ZMQ_PUB socket");

            rc = zmq_setsockopt (wr_sock, ZMQ_LINGER, &LINGER_TIMEOUT, sizeof (LINGER_TIMEOUT));
            RET_ON_ERR(rc == 0, "Failed to ZMQ_LINGER to %d", LINGER_TIMEOUT);

            rd_sock = zmq_socket (z_ctx, ZMQ_SUB);
            RET_ON_ERR(rd_sock != NULL, "Failed to ZMQ_PUB socket");

            if (client_name.empty()) {
                bind_server(z_ctx, wr_sock, rd_sock);
            } else {
                connect_client(z_ctx, wr_sock, rd_sock);
            }
            /*
             * Connect/bind is async and takes time.
             * Any write before connection setup will be dropped on floor.
             * Either sleep for a second 
             * Or let server set a REP end and let client connect and
             * send/receive a message to shadow the async set up.
             * This might save some milliseconds, but this being called
             * at the initial process setup, sleeping a second is simpler
             * than adding extra code to save one time init cost.
             */
            sleep(1);
            printf("DROP SLept ......... client(%s) wr_sock=0x%p rd_sock=0x%p\n",
                    client_name.c_str(), wr_sock, rd_sock);

            /* client_name empty in server mode. Hence subscribe to any */
            rc = zmq_setsockopt(rd_sock, ZMQ_SUBSCRIBE, client_name.c_str(),
                    client_name.size());
            RET_ON_ERR(rc == 0, "Fails to set option rc=%d", rc);

            if (m_rd_timeout != -1) {
                int ms = TO_MS(m_rd_timeout);
                rc = zmq_setsockopt (rd_sock, ZMQ_RCVTIMEO, &ms, sizeof(ms));
                RET_ON_ERR(rc == 0, "Failed to ZMQ_RCVTIMEO to %d", m_rd_timeout);
            }

            m_client_name = client_name;
            m_wr_sock = wr_sock;
            m_rd_sock = rd_sock;
            wr_sock = NULL;
            rd_sock = NULL;

            {
                stringstream ss;
                ss << " client:" << m_client_name;
                m_self_str = ss.str();
            }
        out:
            zmq_close(wr_sock);
            zmq_close(rd_sock);
            LOM_LOG_DEBUG("transport: rc=%d client:%s", rc, m_client_name.c_str());
            return rc;
        }

        virtual int write(const string msg, const string dest)
        {
            int rc = 0;
            RET_ON_ERR(m_client_name.empty() != dest.empty(),
                    "Client specifies no dest; server specifies. self(%s) dest:(%s)",
                    m_self_str.c_str(), dest.c_str());

            /* Set sender name if from client. Server receives any. */
            rc = _zmq_message_send(m_wr_sock, dest.empty() ? m_client_name : dest, msg);
            RET_ON_ERR(rc == 0, "Failed to send self(%s) rc=%d", m_self_str.c_str(), rc);
        out:
            LOM_LOG_DEBUG("write: rc=%d self(%s)", rc, m_self_str.c_str());
            return rc;
        }

        virtual int read(string &client_id, string &msg, bool dont_wait)
        {
            int rc = 0;

            rc = _zmq_message_read(m_rd_sock, dont_wait ? ZMQ_DONTWAIT : 0,
                    client_id, msg);
            RET_ON_ERR(rc == 0, "Failed to recv self(%s) rc=%d", m_self_str.c_str(), rc);
        out:
            return rc;
        }

        virtual int poll_for_data(int *lst_fds, int cnt, int timeout)
        {
            int rc = -3, ret;
            zmq_pollitem_t items[cnt+1];
            zmq_pollitem_t *p;

            items[0].socket = m_rd_sock;
            items[0].events = ZMQ_POLLIN;

            p = items + 1;
            for(int i=0; i<cnt; ++i, ++p) {
                p->fd = *lst_fds++;
                p->events = ZMQ_POLLIN;
            }


            printf("DROP: items[0].socket=0x%p cnt=%d timeout=%d\n",
                    items[0].socket, cnt, timeout);
            printf("DROP: time=%d\n", (int)time(0));
            ret = zmq_poll (items, cnt+1, TO_MS(timeout));
            printf("DROP: rc=%d time=%d z=%d\n", rc, (int)time(0), zmq_errno());
            switch (ret) {
            case -1:
                rc = -3;
                RET_ON_ERR(false, "zmq_poll failed");
                break;
            
            case 0:
                /* timeout */
                rc = -2;

            default:
                if (items[0].revents & ZMQ_POLLIN) {
                    /* Data available from engine */
                    rc = -1;
                }
                rc = -3;
                for(int i=1; i<= cnt; ++i) {
                    if (items[i].revents & ZMQ_POLLIN) {
                        rc = items[i].fd;
                        break;
                    }
                }
                break;
            }
        out:
            return rc;
        }

    private:
        /*
         * Engine creates PUB & SUB points
         * 
         * Engine creates PUB socket and write out via PUB end point.
         * Clients creates SUB socket and connect to PUB point for read.
         *
         * Engine creates SUB socket and listen via via SUB end point.
         * Clients creates PUB socket and connect to SUB point for writes.
         */
        int connect_client(void *zmq_ctx, void *wr_sock, void *rd_sock)
        {
            int rc = zmq_connect (wr_sock, _get_config(CFG_SUB_END_KEY).c_str());
            RET_ON_ERR(rc == 0, "client fails to connect %s",
                    _get_config(CFG_SUB_END_KEY).c_str());

            rc = zmq_connect (rd_sock, _get_config(CFG_PUB_END_KEY).c_str());
            RET_ON_ERR(rc == 0, "client fails to connect %s",
                    _get_config(CFG_PUB_END_KEY).c_str());
        out:
            LOM_LOG_DEBUG("DROP: rc=%d write(%s) read(%s)", rc,
                    _get_config(CFG_SUB_END_KEY).c_str(),
                    _get_config(CFG_PUB_END_KEY).c_str());
            return rc;
        }

        int bind_server(void *zmq_ctx, void *wr_sock, void *rd_sock)
        {
            int rc = zmq_bind (wr_sock, _get_config(CFG_PUB_END_KEY).c_str());
            RET_ON_ERR(rc == 0, "server fails to bind %s",
                    _get_config(CFG_PUB_END_KEY).c_str());

            rc = zmq_bind (rd_sock, _get_config(CFG_SUB_END_KEY).c_str());
            RET_ON_ERR(rc == 0, "server fails to bind %s",
                    _get_config(CFG_SUB_END_KEY).c_str());
        out:
            LOM_LOG_DEBUG("DROP: rc=%d read(%s) write(%s)", rc,
                    _get_config(CFG_SUB_END_KEY).c_str(),
                    _get_config(CFG_PUB_END_KEY).c_str());
            return rc;
        }


        string m_client_name;
        void *m_wr_sock;
        void *m_rd_sock;
        int m_rd_timeout;

        string m_self_str;

};

typedef shared_ptr<transport> transport_ptr_t;

transport_ptr_t
init_transport(const string client_name, int timeout)
{
    /*
     * Only one transport expected per process. This flag to help capture
     * design/implementation level misuse. Hence not thread protected.
     */
    transport_ptr_t tx;

    int rc = 0;
    tx = transport_ptr_t(new transportImpl(timeout));

    tx->set_mode(client_name);
    RET_ON_ERR(tx->is_valid(), "Failed to init transport for client (%s)",
            client_name.c_str());
out:
    if (rc != 0) {
        tx.reset();
    }
    return tx;

}



