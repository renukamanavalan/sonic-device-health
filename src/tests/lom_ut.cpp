#include <fstream>      // std::ifstream
#include <sstream>      // std::ifstream
#include <stdio.h>
#include <vector>
#include <string>
#include <semaphore.h>
#include <thread>
#include <nlohmann/json.hpp>
#include <unistd.h>
#include "consts.h"
#include "common.h"
#include "client.h"
#include "server.h"
#include "transport.h"

using namespace std;
using json = nlohmann::json;

#define TEST_ERR_PREFIX "TEST_ERROR:"

#define TEST_CASE_FILE "/usr/share/tests/test_data_ut.json"

static bool
_is_commented(const string key)
{
    return (key.compare(0, 1, "_") == 0) ? true : false;
}

static string
args_to_str(const string cmd, const vector<string> args)
{
    stringstream ss;

    ss << "cmd:(" << cmd << ") args:(";
    for(vector<string>::const_iterator itc = args.begin();
            itc != args.end(); ++itc) {
        ss << *itc << " ";
    }
    ss << ")";
    return ss.str();
}

class mock_peer
{
    public:
        mock_peer(const vector<string> &clients) : m_clients(clients) {
            m_se_tx = init_server_transport(m_clients);
        }
        mock_peer(string client) : m_client(client)
        {
            m_cl_tx = init_client_transport(m_client);
        }

        bool is_valid() { return (m_cl_tx != NULL) || (m_se_tx != NULL) ? true : false; };
        bool is_client() { return !m_client.empty(); };

        int write(const vector<string> &args)
        {
            int rc = 0;

            if (is_client()) {
                rc = m_cl_tx->write(args[0]);
            } else {
                rc = m_se_tx->write(args[0], args[1]);
            }
            RET_ON_ERR(rc == 0, "mock_peer: Failed to write");
        out:
            return rc;
        }

        int read(vector<string> &args, int timeout=-1)
        {
            int rc = 0;

            vector<string>().swap(args);
            string s1, s2;

            if (is_client()) {
                rc = m_cl_tx->read(s1, timeout);
            } else {
                rc = m_se_tx->read(s1, s2, timeout);
            }
            RET_ON_ERR(rc == 0, "mock_peer: Failed to read");
            args.push_back(s1);
            args.push_back(s2);
        out:
            return rc;
        }


    private:
        string m_client;
        vector<string> m_clients;
        client_transport_ptr_t m_cl_tx;
        server_transport_ptr_t m_se_tx;
};

typedef shared_ptr<mock_peer> mock_peer_ptr_t;

static int
test_client(const string cmd, const vector<string> args)
{
    int rc = 0;
    
    if (cmd == REQ_REGISTER_CLIENT) {
        rc = register_client(args[0].c_str());
    } else if (cmd == REQ_DEREGISTER_CLIENT) {
        rc = deregister_client();
    } else if (cmd == REQ_REGISTER_ACTION) {
        rc = register_action(args[0].c_str());
    } else if (cmd == REQ_HEARTBEAT) {
        rc = touch_heartbeat(args[0].c_str(), args[1].c_str());
    } else if (cmd == REQ_ACTION_RESPONSE) {
        rc = write_action_response(args[0].c_str());
    } else if (cmd == REQ_ACTION_REQUEST) {
        ServerMsg_ptr_t read_msg;
        string str_test;

        string str_read(read_action_request(2));
        RET_ON_ERR(!str_read.empty(), "Empty request string received");

        read_msg = create_server_msg(str_read);
        RET_ON_ERR(read_msg != NULL, "Failed to create msg from (%s)", str_read.c_str());
        RET_ON_ERR(read_msg->validate(), "Failed to validate (%s)", str_read.c_str());

        str_test = string(args.empty() ? "" : args[0]);
        if (!str_test.empty()) {
            ServerMsg_ptr_t test_msg =  create_server_msg(str_test);
            RET_ON_ERR(test_msg->validate(), "%s Invalid msg (%s)",
                    TEST_ERR_PREFIX, str_test.c_str());

            RET_ON_ERR((*read_msg) == (*test_msg), "Failed to match exp(%s) != read(%s)",
                    test_msg->to_str().c_str(), str_read.c_str());
        } else {
            LOM_LOG_INFO("**** Received msg: (%s)", str_read.c_str());
        }
    } else {
        RET_ON_ERR(false, "Invalid command (%s) provided", cmd); 
    }
out:
    LOM_LOG_DEBUG("test_client: rc=%d input:%s\n", rc, args_to_str(cmd, args).c_str());
    return rc;
}


#if 0
static int
test_server(bool is_write, const string data)
{
    int rc = 0;

    ServerMsg_ptr_t test_msg;
   
    if (!data.empty()) {
        test_msg = create_server_msg(data);
        RET_ON_ERR(test_msg->validate(), "%s Invalid msg (%s)",
                TEST_ERR_PREFIX, data.c_str());
    } else {
        RET_ON_ERR(!is_write, "%s write expects data", TEST_ERR_PREFIX);
    }

    if (is_write) {
        RET_ON_ERR(test_msg != NULL, "expect non null message to write");
        RET_ON_ERR(test_msg->get_type() == REQ_ACTION_REQUEST,
                "req is not action_Request but (%s)", test_msg->get_type().c_str());

        rc = write_server_message(test_msg);
        RET_ON_ERR(rc == 0, "Failed to write message (%s)", data.c_str());
    }
    else {
        ServerMsg_ptr_t read_msg = read_server_message(2);
        RET_ON_ERR(read_msg->validate(), "Failed to read message (%s)",
                read_msg->to_str().c_str());

        if (test_msg != NULL) {
            RET_ON_ERR(*read_msg == *test_msg, "Read message (%s) != expect (%s)",
                    read_msg->to_str().c_str(), data.c_str());
        }
        else {
            LOM_LOG_INFO("**** Received msg: (%s)", read_msg->to_str().c_str());
        }
    }
out:
    LOM_LOG_DEBUG("test_server rc=%d data=%s", rc, data.c_str());
    return rc;
}
#endif

/*
 * Server/client communicates to peer via zmq
 *
 * For testing, both can't run in same thread.
 * So run peer in another thread.
 *
 * Job of peer:
 *  1. Create transport for peer 
 *  2. Upon signal from client, execute next peer cmd from the given list.
 *  3. Peer commands are just read/write
 *  4. Upon completion of each signal main thread
 */

typedef enum {
    CMD_NONE = 0,
    CMD_INIT,
    CMD_WRITE,
    CMD_READ,
    CMD_QUIT
} mock_cmd_t;


int
run_a_client_test_case(const string tcid, const json &tcdata)
{
    int rc = 0;

    LOM_LOG_INFO("Running test case %s", tcid.c_str());

    /* Create mock server end */
    mock_peer_ptr_t peer;
    
    for (auto itc = tcdata.cbegin(); itc != tcdata.cend(); ++itc) {
        string key = itc.key();

        if (_is_commented(key)) {
            continue;
        }

        json tc_entry = (*itc);

        vector<string> write_data = tc_entry.value("write", vector<string>());
        vector<string> read_data = tc_entry.value("read", vector<string>());

        LOM_LOG_INFO("Running test entry %s:%s", tcid.c_str(), key.c_str());
        if (write_data.size() == 2) {
            RET_ON_ERR(peer != NULL, "Missing peer to write");
            RET_ON_ERR((rc = peer->write(write_data)) == 0, "Failed to write via mock server");
        } else {
            RET_ON_ERR(write_data.empty(), "TEST ERROR: check write data %s", 
                    tcid.c_str());
        }
        {
            string cmd = tc_entry.value("cmd", "");
            vector<string> args = tc_entry.value("args", vector<string>());

            if (cmd == "mock_server") {
                mock_peer_ptr_t p(new mock_peer(args));
                RET_ON_ERR(p->is_valid(), "Mock peer fails to create");
                peer = p;
            } else {
                rc = test_client(cmd, args);
                RET_ON_ERR(rc == 0, "Failed to run client cmd(%s)", cmd.c_str());
            }
        }
        if (read_data.size() == 2) {
            vector<string> peer_data;

            RET_ON_ERR(peer != NULL, "Missing peer to read");
            RET_ON_ERR((rc = peer->read(peer_data)) == 0, "Failed to read via mock server");
            ServerMsg_ptr_t msg_read = create_server_msg(peer_data[1]);
            ServerMsg_ptr_t msg_exp = create_server_msg(read_data[1]);

            RET_ON_ERR(peer_data[0] == read_data[0], "Test compare fail on client read(%s) != exp(%s)",
                    peer_data[0].c_str(), read_data[0].c_str());

            RET_ON_ERR((*msg_read) == (*msg_exp), "Test compare fail on msg read(%s) != exp(%s)",
                    msg_read->to_str().c_str(), msg_exp->to_str().c_str());
        } else {
            RET_ON_ERR(read_data.empty(), "TEST ERROR: check read data %s", 
                    tcid.c_str());
        }
    }
out:
    LOM_LOG_INFO("%s test case %s rc=%d", (rc == 0 ? "completed" : "aborted"),
            tcid.c_str(), rc);
    return rc;
}


int 
run_client_testcases(const json &tccases)
{
    int rc = 0;

    LOM_LOG_INFO("Running all client test_cases");

    for (auto itc = tccases.cbegin(); itc != tccases.cend(); ++itc) {

        string key = itc.key();
        if (_is_commented(key)) {
            LOM_LOG_INFO("Skip commented testcase %s", key.c_str());
            continue;
        }
        rc = run_a_client_test_case(key, itc.value());
        RET_ON_ERR(rc == 0, "Failed to run test case %s", key.c_str());
    }
out:
    return rc;
}

int main(int argc, const char **argv)
{
    int rc = 0;
    string tcfile(argc > 1 ? argv[1] : TEST_CASE_FILE);

    set_test_mode();

    ifstream f(tcfile.c_str());
    json data = json::parse(f, nullptr, false);

    RET_ON_ERR(!data.is_discarded(), "Failed to parse file %s", tcfile.c_str());

    rc = run_client_testcases(data.value("client_test_cases", json()));
    RET_ON_ERR(rc == 0, "run_testcases failed rc=%d", rc);

    LOM_LOG_INFO("SUCCEEDED in running test cases");
out:
    return rc;
}

