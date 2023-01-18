/*
 * common APIs used by events code.
 */
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

using namespace std;
using namespace chrono;

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

    rc = poll (fds, cnt, TO_MS(timeout));
    RET_ON_ERR(rc >= 0, "Poll failed");

    if (rc != 0) {
        p = fds;
        for(int i=0; (i < cnt); ++i, ++p) {
            if (p->revents & (POLLIN|POLLERR|POLLHUP)) {
                ready_fds[ready_cnt++] = p->fd;
            } else if (p->revents & (POLLNVAL|POLLERR|POLLHUP)) {
                err_fds[err_cnt++] = p->fd;
            }
        }
    }
    *ready_fds_cnt = ready_cnt;
    *err_fds_cnt = err_cnt;

out:
    return rc;
}

class reader_writer
{
    public:
        reader_writer(const vector<fd_t> &fds) : m_lst(fds) {
            _reset();
        };

        int read_data(int timeout, string &data, int &fd) {
            int rc = 0, tcnt;

            fd = -1;
            if (m_ready_cnt == 0) {
                _reset();
                /* poll for fd with data */

                rc = poll_for_data(&m_lst[0], m_lst.size(), &m_ready[0], &m_ready_cnt,
                        &m_err[0], &tcnt);
                RET_ON_ERR(rc == -1, "Poll failed in reader");
            }
            fd = m_ready[m_ready_index++];
            m_ready_cnt--;

            RET_ON_ERR(fd >= 0, "Internal error. ready fd is not valid index(%d) cnt(%d)",
                    m_ready_index, m_ready_cnt);
            RET_ON_ERR((rc = _read(fd, data)) == 0, "Failed to read fd(%d)", fd);
        out:
            return rc;
        }


        int write_data(int fd, const string data) {
            int rc = 0, sz=data.size();

            rc = write(fd, &sz, sizeof(sz));
            RET_ON_ERR(rc >= 0, "Failed to write rc=%d sz=%d fd=%d", rc, sizeof(sz), fd);

            rc = write(fd, data.c_str(), sz);
            RET_ON_ERR(rc >= 0, "Failed to write rc=%d sz=%d fd=%d", rc, sz, fd);
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

        virtual int set_client_mode(const string client_name)
        {
            int rc = 0;
            vector<string> lst{ client_name };

            RET_ON_ERR(!client_name.empty(), "Expect non empty client name");

            m_client = client_name;

            rc = set_server_mode(lst);

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

            for(vector<string>::const_iterator itc = clients.begin();
                    itc != clients.end(); ++itc) {
                RET_ON_ERR((rc = create_a_fd(*itc)) == 0,
                        "Failed to create fd for (%s) client-mode", (*itc).c_str());
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

            RET_ON_ERR(itc != m_wfds.end(), "Missing entry for (%s) is_client(%d)", 
                    get_client_name(client).c_str(), is_client());

            rc = m_reader_writer->write_data(itc->second, msg);
            RET_ON_ERR(rc == 0, "Failed to write size(%d)", msg.size());
        out:
            return rc;
        }

        virtual int write(const string msg)
        {
            /* Write from client */
            return write(string(), msg);
        }

        virtual int read(string &client_id, string &msg, int timeout)
        {
            int fd;
            int rc = m_reader_writer->read_data(timeout, msg, fd);
            RET_ON_ERR(rc == 0, "read_data failed fd=%d", fd);
            client_id = m_rclients[fd];
        out:
            return rc;
        }

        virtual int read(string &msg, int timeout)
        {
            string s;
            return read(s, msg, timeout);
        }

        virtual fd_t get_read_fd() { return m_rfds.empty() ? -1 : m_rfds[0]; };

    private:
        int create_a_fd(const string client_name);

        bool validate_client(const string client)
        {
            return ((m_rfds.find(client) != m_rfds.end()) && 
                    (m_wfds.find(client) != m_wfds.end())) ? true : false;  
        }

        const string get_client_key(const string client) { return is_client() ? "" : client; };
        const string get_client_name(const string client) { return is_client() ? m_client : client; };

        bool m_valid;
        string m_client;

        map<client_t, fd_t> m_rfds, m_wfds;
        map<fd_t, client_t> m_rclients, m_wclients;

        reader_writer_ptr_t m_reader_writer;
};

void
get_fifo_names(const string client_name, string &s_c_2_e, string &s_e_2_c)
{
    stringstream ss_c_2_e, ss_e_2_c;
    ss_c_2_e << "/tmp/lom_fifo_" << client_name << "_engine";
    ss_e_2_c << "/tmp/lom_fifo_engine_" << client_name ;

    s_c_2_e = ss_c_2_e.str();
    s_e_2_c = ss_e_2_c.str();

    /* Ensure removed */
    unlink(s_c_2_e.c_str());
    unlink(s_e_2_c.c_str());
}

int
transportImpl::create_a_fd(const string client_name)
{
    int rc = 0;
    string s_c_2_e, s_e_2_c;
    bool is_cl = is_client();
    string key_client = get_client_key(client_name);
    int wr_fd, rd_fd;

    get_fifo_names(client_name, s_c_2_e, s_e_2_c);

    const char *wr_end = is_cl ? s_c_2_e.c_str() : s_e_2_c.c_str();
    const char *rd_end = is_cl ? s_e_2_c.c_str() : s_c_2_e.c_str(); 

    RET_ON_ERR(!validate_client(key_client), "Duplicate ? for (%s) is_client=%d", 
            client_name.c_str(), is_client());

    wr_fd = open(wr_end, O_WRONLY);
    RET_ON_ERR(wr_fd >=0, "Failed to create wr(%s)", wr_end);

    rd_fd = open(rd_end, O_RDONLY);
    RET_ON_ERR(rd_fd >= 0, "Failed to create rd(%s)", rd_end);


    m_rfds[key_client] = rd_fd;
    m_wfds[key_client] = wr_fd;

    m_rclients[rd_fd] = key_client;
    m_wclients[wr_fd] = key_client;

out:
    return rc;
}

client_transport_ptr_t
init_client_transport(const string client_name)
{
    /*
     * Only one transport expected per process. This flag to help capture
     * design/implementation level misuse. Hence not thread protected.
     */
    client_transport_ptr_t tx;

    int rc = 0;
    transportImpl *p = new transportImpl();
    tx = client_transport_ptr_t(dynamic_cast<client_transport *>(p));

    p->set_client_mode(client_name);
    RET_ON_ERR(p->is_valid(), "Failed to init transport for client (%s)",
            client_name.c_str());
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

