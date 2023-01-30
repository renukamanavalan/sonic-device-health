/*
 * common APIs used by events code.
 */
#include <iostream>
#include <limits.h>
#include <stdio.h>
#include <chrono>
#include <fstream>
#include <sstream>
#include <errno.h>
#include <fcntl.h>
#include <map>
#include <string.h>
#include <iomanip>      // std::put_time
#include <ctime>        // std::time_t, struct std::tm, std::localtime
#include <chrono>       // std::chrono::system_clock
#include <nlohmann/json.hpp>
#include <poll.h>
#include <unistd.h>
#include <unordered_map>
#include <sys/types.h>
#include <sys/stat.h>

#include "common.h"
#include "consts.h"
#include "transport.h"
#include "client.h"

using namespace std;
using namespace chrono;

#define FIFO_E2C_PATH "/tmp/lom_fifo_engine_to_%s"
#define FIFO_C2E_PATH "/tmp/lom_fifo_%s_to_engine"
    
#define TO_MS(n) ((n) * 1000)
  
#if 0
/*
 * Config that can be read from init_cfg
 */
#define INIT_CFG_PATH "/etc/LoM/init_cfg.json"

/* configurable entities' keys */
#define CFG_SUB_END_KEY "sub_path"       
#define CFG_PUB_END_KEY "pub_path"

#define CFG_STATS_UPD_SECS "stats_upd_secs"

#define CFG_TRANSPORT_KEY  "LOM_TRANSPORT"

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
#endif

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


typedef string client_t;
typedef int fd_t;

int poll_for_data(const fd_t *lst_fds, int cnt,
                fd_t *ready_fds, int *ready_fds_cnt,
                fd_t *err_fds, int *err_fds_cnt, int timeout)
{
    int rc = 0;
    struct pollfd fds[cnt], *p;
    int ready_cnt = 0, err_cnt = 0;

    p = fds;
    for(int i=0; i<cnt; ++i, ++p) {
        p->fd = *lst_fds++;
        p->events = POLLIN;
    }

    if (is_test_mode()) {
        stringstream ss;
        ss << "{ ";
        p = fds;
        for(int i=0; i<cnt; ++i, ++p) {
            ss << p->fd << " ";
        }
        ss << "}";
        DROP_TEST("lom_lib: poll_for_data START timeout(%d) %s",
                timeout, ss.str().c_str());
    }

    rc = poll (fds, cnt, TO_MS(timeout));
    DROP_TEST("lom_lib: poll_for_data rc=%d", rc);
    RET_ON_ERR(rc >= 0, "Poll failed");

    if (rc != 0) {
        p = fds;
        for(int i=0; (i < cnt); ++i, ++p) {
            if (p->revents & POLLIN) {
                ready_fds[ready_cnt++] = p->fd;
                DROP_TEST("lom_lib: poll_for_data ready fd=%d", p->fd);
            } else if (p->revents & (POLLNVAL|POLLERR|POLLHUP)) {
                err_fds[err_cnt++] = p->fd;
                DROP_TEST("lom_lib: poll_for_data ERR fd=%d", p->fd);
            }
        }
    }
    *ready_fds_cnt = ready_cnt;
    *err_fds_cnt = err_cnt;

out:
    DROP_TEST("lom_lib: poll_for_data EXIT rc=%d", rc);
    return rc;
}

class reader_writer
{
    public:
        reader_writer(int fd) {
            m_lst.push_back(fd);
            _reset();
        };

        reader_writer(const vector<fd_t> &fds) : m_lst(fds) {
            _reset();
        };

        ~reader_writer() {};

        int read_data(int timeout, string &data, int &fd) {
            int rc = 0, tcnt;

            fd = -1;
            if (m_ready_cnt == 0) {
                _reset();
                /* poll for fd with data */

                rc = poll_for_data(&m_lst[0], m_lst.size(), &m_ready[0], &m_ready_cnt,
                        &m_err[0], &tcnt, timeout);
                RET_ON_ERR(rc != -1, "Poll failed in reader");
                m_ready_index = 0;
            }
            if (rc == 0) {
                LOM_LOG_DEBUG("Read timed out list sz=%d", m_lst.size());
                fd = -1;
            } else {
                fd = m_ready[m_ready_index++];
                m_ready_cnt--;

                RET_ON_ERR(fd >= 0, "Internal error. ready fd is not valid index(%d) cnt(%d)",
                        m_ready_index, m_ready_cnt);
                RET_ON_ERR((rc = _read(fd, data)) == 0, "Failed to read fd(%d)", fd);
            }
        out:
            return rc;
        }


        int write_data(int fd, const string data) {
            int rc = 0, sz=data.size();

            rc = write(fd, &sz, sizeof(sz));
            RET_ON_ERR(rc >= 0, "Failed to write rc=%d sz=%d fd=%d", rc, sizeof(sz), fd);

            rc = write(fd, data.c_str(), sz);
            RET_ON_ERR(rc >= 0, "Failed to write rc=%d sz=%d fd=%d", rc, sz, fd);
            rc = 0;
        out:
            return rc;
        }

    private:
        int _read(int fd, string &data)
        {
            int rc = 0;
            int sz, ret;

            ret = read(fd, &sz, sizeof(sz));
            RET_ON_ERR(ret >= 0, "read command failed on fd=%d", fd);
            RET_ON_ERR(sz >= 0, "read incorrect size sz=%d fd=%d", sz, fd);
            RET_ON_ERR(sz <= 2048, "read size looks too large sz=%d fd=%d", sz, fd);

            {
                char buf[sz+1];
                ret = read(fd, buf, sz);
                RET_ON_ERR(ret >= 0, "read command failed on fd=%d", fd);
                buf[ret] = 0;
                data = string(buf);
            }
        out:
            return rc;
        }

        void _reset() {
            vector<int>(m_lst.size(), -1).swap(m_ready);
            vector<int>(m_lst.size(), -1).swap(m_err);
            
            m_ready_index = 0;
            m_ready_cnt = 0;
        }


        vector<fd_t> m_lst, m_ready, m_err;
        int m_ready_index, m_ready_cnt;
};

typedef shared_ptr<reader_writer> reader_writer_ptr_t;

class transportImpl : public server_transport, public client_transport
{
    public:
        transportImpl() : m_valid(false) {};

        virtual ~transportImpl() {
        }

        virtual bool is_valid() { return m_valid; }

        virtual bool is_client() { return !m_client.empty(); }

        virtual int set_client_mode(const string client)
        {
            int rc = 0;

            RET_ON_ERR(!client.empty(), "Expect non empty client name");

            RET_ON_ERR(m_rfds.empty(), "Duplicate set mode");

            m_client = client;

            /* create read end only; write end is created on first demand */
            RET_ON_ERR((rc = create_a_fd(false, client)) == 0,
                    "Failed to create client read fd (%s)", client.c_str());
            RET_ON_ERR(m_rfds.size() == 1, "Expect only one fd");
            m_reader_writer.reset(new reader_writer(m_rfds.begin()->second));
        out:
            m_valid = (rc == 0) ? true : false;
            return rc;

        }

        virtual int set_server_mode(const vector<string> clients)
        {
            int rc = 0;
            vector<int> lst;
            RET_ON_ERR(!clients.empty(), "Expect non empty client list");

            RET_ON_ERR(m_rfds.empty(), "Duplicate set mode");

            /* create read end only; write end is created on first demand */
            for(vector<string>::const_iterator itc = clients.begin();
                    itc != clients.end(); ++itc) {
                RET_ON_ERR((rc = create_a_fd(true, *itc)) == 0,
                        "Failed to create server read fd for (%s)", (*itc).c_str());
            }
            for (map<fd_t, client_t>::const_iterator itc = m_rclients.begin();
                    itc != m_rclients.end(); ++itc) {
                lst.push_back(itc->first);
            }
            m_reader_writer.reset(new reader_writer(lst));
        out:
            m_valid = (rc == 0) ? true : false;
            return rc;
        }

        virtual int write(const string client, const string msg)
        {
            int rc = 0;
            map<client_t, fd_t>::const_iterator itc = m_wfds.find(client);
            
            if (itc == m_wfds.end()) {
                /*
                 * Create on first demand, as wr can only be created after peer
                 * created the read end.
                 * c2e == is_client
                 */
                RET_ON_ERR((rc = create_a_fd(is_client(), client)) == 0,
                        "Failed to create write fd for client (%s) is_client(%d)",
                        client.c_str(), is_client());
            }

            itc = m_wfds.find(client);
            RET_ON_ERR(itc != m_wfds.end(), "Missing entry for (%s) is_client(%d)", 
                    client.c_str(), is_client());

            rc = m_reader_writer->write_data(itc->second, msg);
            RET_ON_ERR(rc == 0, "Failed to write size(%d)", msg.size());
        out:
            return rc;
        }

        virtual int write(const string msg)
        {
            /* Write from client */
            return write(m_client, msg);
        }

        virtual int read(string &client_id, string &msg, int timeout)
        {
            int fd;
            int rc = m_reader_writer->read_data(timeout, msg, fd);
            RET_ON_ERR(rc == 0, "read_data failed fd=%d", fd);
            DROP_TEST("fd=%d", fd);
            if (fd >= 0) {
                client_id = m_rclients[fd];
            } else {
                /* timeout occurred */
                client_id = string();
                msg = string();
                DROP_TEST("msg cleared");
            }
        out:
            return rc;
        }

        virtual int read(string &msg, int timeout)
        {
            string s;
            return read(s, msg, timeout);
        }

        virtual fd_t get_read_fd() { return m_rfds.empty() ? -1 : m_rfds.begin()->second; };

    private:
        int create_a_fd(bool c2e, const string client);

        bool m_valid;
        string m_client;

        map<client_t, fd_t> m_rfds, m_wfds;
        map<fd_t, client_t> m_rclients, m_wclients;

        reader_writer_ptr_t m_reader_writer;
};

static string
_get_path(bool c2e, const string client)
{
    int rc = 0;
    string ret;
    char buf[100];

    rc = snprintf(buf, sizeof(buf), c2e ? FIFO_C2E_PATH : FIFO_E2C_PATH, client.c_str());
    RET_ON_ERR(rc < (int)sizeof(buf), "Internal error. path(%s) name(%s) too long %d > %d",
            c2e ? FIFO_C2E_PATH : FIFO_E2C_PATH, client.c_str(),
            rc, (int)sizeof(buf));
    buf[rc] = 0;
    ret = string(buf);
out:
    return ret; 
}

int
transportImpl::create_a_fd(bool c2e, const string client)
{
    int rc = 0;
    string path(_get_path(c2e, client));
    /*
     * Truth table
     *
     * c2e | is_client |  mode
     *-------------------------
     *  T  |    T      |   WR
     *  T  |    F      |   RD
     *  F  |    T      |   RD
     *  F  |    F      |   WR
     *-------------------------
     * so c2e == is_client ? WR : RD
     */
    int is_read_end = (c2e != is_client()) ? true : false;
    int fd;

    if (is_read_end) {
        /* Creating this path for first time */
        unlink(path.c_str());

        rc = mknod(path.c_str(), S_IFIFO | 0666, 0);
        RET_ON_ERR(rc == 0, "Failed to create node for (%s)", path.c_str());

        /* Note: O_NONBLOCK is required as wr end is not open yet. */
        fd = open(path.c_str(), O_RDONLY | O_NONBLOCK);
        RET_ON_ERR(fd >= 0, "Failed to create read fd(%s)", path.c_str());

        m_rfds[client] = fd;
        m_rclients[fd] = client;
        DROP_TEST("client=%s is_client=%d rfd=%d", client.c_str(), is_client(), fd);

    } else {
        /*
         * Pause wr_fd creation until first write request.
         * This is because write creation will fail until read end is opened.
         */
        fd = open(path.c_str(), O_WRONLY | O_NONBLOCK);
        RET_ON_ERR(fd >= 0, "Failed to create write fd (%s)", path.c_str());

        m_wfds[client] = fd;
        m_wclients[fd] = client;
    }
out:
    return rc;
}

client_transport_ptr_t
init_client_transport(const string client)
{
    /*
     * Only one transport expected per process. This flag to help capture
     * design/implementation level misuse. Hence not thread protected.
     */
    client_transport_ptr_t tx;

    int rc = 0;
    transportImpl *p = new transportImpl();
    tx = client_transport_ptr_t(dynamic_cast<client_transport *>(p));

    p->set_client_mode(client);
    RET_ON_ERR(p->is_valid(), "Failed to init transport for client (%s)",
            client.c_str());
out:
    if (rc != 0) {
        tx.reset();
    }
    return tx;

}

server_transport_ptr_t
init_server_transport(const vector<string> &clients)
{
    /*
     * Only one transport expected per process. This flag to help capture
     * design/implementation level misuse. Hence not thread protected.
     */
    server_transport_ptr_t tx;

    int rc = 0;
    transportImpl *p = new transportImpl();
    tx = server_transport_ptr_t(dynamic_cast<server_transport *>(p));

    p->set_server_mode(clients);
    RET_ON_ERR(p->is_valid(), "Failed to init transport for server.");
out:
    if (rc != 0) {
        tx.reset();
    }
    return tx;

}

