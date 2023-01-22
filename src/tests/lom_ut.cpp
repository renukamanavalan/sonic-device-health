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
#include "server_c.h"
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

        int write(const string s1, const string s2)
        {
            int rc = 0;

            RET_ON_ERR (!is_client(), "Expect only one str for write");
            rc = m_se_tx->write(s1, s2);
            RET_ON_ERR(rc == 0, "mock_peer: Failed to write");
        out:
            return rc;
        }

        int write(const string s)
        {
            int rc = 0;

            RET_ON_ERR (is_client(), "Expect two str for write");
            rc = m_cl_tx->write(s);
            RET_ON_ERR(rc == 0, "mock_peer: Failed to write");
        out:
            return rc;
        }

        int read(string &s1, string &s2, int timeout=-1)
        {
            int rc = 0;

            RET_ON_ERR (!is_client(), "Expect only one str for read");
            rc = m_se_tx->read(s1, s2, timeout);
            RET_ON_ERR(rc == 0, "mock_peer: Failed to read");
        out:
            return rc;
        }

        int read(string &s, int timeout=-1)
        {
            int rc = 0;

            RET_ON_ERR (is_client(), "Expect two str for read");
            rc = m_cl_tx->read(s, timeout);
            RET_ON_ERR(rc == 0, "mock_peer: Failed to read");
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
    int rc = 0, fd;
    
    if (cmd == REQ_REGISTER_CLIENT) {
        rc = register_client(args[0].c_str(), &fd);
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
            RET_ON_ERR((rc = peer->write(write_data[0], write_data[1])) == 0, "Failed to write via mock server");
        } else {
            RET_ON_ERR(write_data.empty(), "TEST ERROR: check write data %s", 
                    tcid.c_str());
        }
        {
            string cmd = tc_entry.value("cmd", "");
            vector<string> args = tc_entry.value("args", vector<string>());

            if (cmd == "mock_peer") {
                mock_peer_ptr_t p(new mock_peer(args));
                RET_ON_ERR(p->is_valid(), "Mock peer fails to create");
                peer = p;
            } else {
                rc = test_client(cmd, args);
                RET_ON_ERR(rc == 0, "Failed to run client cmd(%s)", cmd.c_str());
            }
        }
        if (read_data.size() == 2) {
            string id, str_msg;

            RET_ON_ERR(peer != NULL, "Missing peer to read");
            RET_ON_ERR((rc = peer->read(id, str_msg)) == 0, "Failed to read via mock server");
            ServerMsg_ptr_t msg_read = create_server_msg(str_msg);
            ServerMsg_ptr_t msg_exp = create_server_msg(read_data[1]);

            RET_ON_ERR(id == read_data[0], "Test compare fail on client read(%s) != exp(%s)",
                    id.c_str(), read_data[0].c_str());

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

typedef string str_client_t;

int
run_a_server_test_case(const string tcid, const json &tcdata)
{
    int rc = 0;

    LOM_LOG_INFO("Running test case %s", tcid.c_str());

    /* Create mock client end */
    map<str_client_t, mock_peer_ptr_t> mock_peers;


    for (auto itc = tcdata.cbegin(); itc != tcdata.cend(); ++itc) {
        string key = itc.key();

        if (_is_commented(key)) {
            continue;
        }

        json tc_entry = (*itc);
        string cmd = tc_entry.value("cmd", "");
        string data = tc_entry.value("data", "");
        string client = tc_entry.value("client", "");
        ServerMsg_ptr_t test_msg, peer_msg;
        mock_peer_ptr_t peer;

        LOM_LOG_INFO("Running test entry %s:%s", tcid.c_str(), key.c_str());

        if (!data.empty()) {
            test_msg = create_server_msg(data);
            RET_ON_ERR(test_msg != NULL, "TEST error: Failed to create msg (%s)", data.c_str());
        }


        if (!client.empty()) {
            map<str_client_t, mock_peer_ptr_t>::const_iterator itc = mock_peers.find(client);
            RET_ON_ERR(itc != mock_peers.end(), "TEST error: Failed to find mock peer for (%s)",
                    client.c_str());
            peer = itc->second;
            RET_ON_ERR(peer != NULL, "TEST error mock peer for (%s) is null", client.c_str());
        }

        if (cmd == "mock_peer") {
            vector<string> clients = tc_entry.value("clients", vector<string>());
        
            RET_ON_ERR(!clients.empty(), "TEST error: Expect clients");

            RET_ON_ERR((rc = server_init(clients)) == 0, "Failed to call server_init");

            for(vector<string>::const_iterator itc = clients.begin();
                        itc != clients.end(); ++itc) {
                mock_peer_ptr_t p(new mock_peer(*itc));
                RET_ON_ERR(p->is_valid(), "Mock peer fails to create (%s)", (*itc).c_str());
                mock_peers[*itc] = p;
            }
        } else if (cmd == "read") {
            RET_ON_ERR((rc = peer->write(data)) == 0, "Failed to write via mock client");

            string read_str(read_server_message_c(2));
            RET_ON_ERR(!read_str.empty(), "read_server_message_c failed; returned empty string");
            peer_msg = create_server_msg(read_str);
            RET_ON_ERR(peer_msg != NULL, "read_server_message failed to parse (%s)",
                    read_str.c_str());

        } else if (cmd == "write") {
            string read_msg;

            RET_ON_ERR((rc = write_server_message_c(data.c_str())) == 0,
                    "Failed to write_server_message");
            RET_ON_ERR((rc = peer->read(read_msg)) == 0, "Failed to read via mock client");
            peer_msg = create_server_msg(read_msg);
            RET_ON_ERR(peer_msg != NULL, "msg read from mock failed (%s)",
                    read_msg.c_str());
        } else {
            RET_ON_ERR(false, "Unknown server test cmd (%s)", cmd.c_str());
        }
        if (peer_msg != NULL) {
            RET_ON_ERR((*test_msg) == (*peer_msg), "test_msg(%s) != peer_msg(%s)",
                    test_msg->to_str().c_str(), peer_msg->to_str().c_str());
        }
    }
out:
    server_deinit();

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

int 
run_server_testcases(const json &tccases)
{
    int rc = 0;

    LOM_LOG_INFO("Running all server test_cases");

    for (auto itc = tccases.cbegin(); itc != tccases.cend(); ++itc) {

        string key = itc.key();
        if (_is_commented(key)) {
            LOM_LOG_INFO("Skip commented testcase %s", key.c_str());
            continue;
        }
        rc = run_a_server_test_case(key, itc.value());
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
    RET_ON_ERR(rc == 0, "run_client_testcases failed rc=%d", rc);

    rc = run_server_testcases(data.value("server_test_cases", json()));
    RET_ON_ERR(rc == 0, "run_server_testcases failed rc=%d", rc);

    LOM_LOG_INFO("SUCCEEDED in running test cases");
out:
    return rc;
}

