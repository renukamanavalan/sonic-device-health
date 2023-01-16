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

        rc = client_poll_for_data(NULL, 0, 2);
        RET_ON_ERR(rc == -1, "Poll failed rc=%d", rc);

        string str_read(read_action_request());
        RET_ON_ERR(!str_read.empty(), "Empty request string received");

        read_msg = create_server_msg(str_read);
        RET_ON_ERR(read_msg->validate(), "Failed to validate (%s)", str_read.c_str());

        str_test = string(args.empty() ? "" : args[0]);
        if (!str_test.empty()) {
            ServerMsg_ptr_t test_msg =  create_server_msg(str_test);
            RET_ON_ERR(test_msg->validate(), "%s Invalid msg (%s)",
                    TEST_ERR_PREFIX, str_test.c_str());

            RET_ON_ERR(read_msg == test_msg, "Failed to match exp(%s) != read(%s)",
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

class mock_peer {
    public:
        mock_peer(string client) : m_client(client),
                m_cmd(CMD_NONE), m_rc(0), m_timeout(-1)
        {
            sem_init(&m_signalMockReq, 0, 0);
            sem_init(&m_signalMockRes, 0, 0);
        };


        /* Called from main thread */
        int next_cmd(mock_cmd_t cmd, string &data1, string &data2, int timeout=-1) {
            m_cmd = cmd;
            m_data1 = data1;
            m_data2 = data2;
            m_timeout = timeout;

            sem_post(&m_signalMockReq);
            printf("DROP -- signalled for thread for req\n");
            sem_wait(&m_signalMockRes);
            printf("DROP -- received signal from thread for res\n");

            data1 = m_data1;
            data2 = m_data2;
            return m_rc;
        }

        void run() {
            while (m_cmd != CMD_QUIT) {
                sem_wait(&m_signalMockReq);
                printf("DROP -- received signal from main thread for req\n");

                m_rc = 0;

                switch(m_cmd) {
                case CMD_INIT:
                    m_tx = init_transport(m_client); 
                    m_rc = m_tx != NULL ? 0 : -1;
                    break;
                case CMD_WRITE:
                    m_rc = m_tx->write(m_data1, m_data2);
                    break;
                case CMD_READ:
                    {
                    int ret = -1;
                    if (m_timeout != -1) {
                        ret = m_tx->poll_for_data(NULL, 0, m_timeout);
                        if (ret != -1) {
                            LOM_LOG_ERROR("Failed to poll ret=%d", ret);
                        }
                    }
                    if (ret == -1) {
                        m_rc = m_tx->read(m_data1, m_data2);
                    } else {
                        m_rc = -1;
                    }
                    break;
                    }
                case CMD_QUIT:
                    LOM_LOG_INFO("Ending mock thread");
                    break;
                default:
                    LOM_LOG_ERROR("TEST ERROR: Unknown cmd (%d)", m_cmd);
                    break;
                }
                sem_post(&m_signalMockRes);
                printf("DROP -- signalled main thread for res\n");
            }
        }


    private:
        string m_client;
        transport_ptr_t m_tx;
        sem_t m_signalMockReq, m_signalMockRes;
        mock_cmd_t m_cmd;
        string m_data1, m_data2;
        int m_rc;
        int m_timeout;
};

typedef shared_ptr<mock_peer> mock_peer_ptr_t;


int
run_a_client_test_case(const string tcid, const json &tcdata)
{
    int rc = 0;
    string data1, data2;

    LOM_LOG_INFO("Running test case %s", tcid.c_str());

    /* Create mock server end */
    mock_peer_ptr_t peer(new mock_peer(""));
    thread thr(&mock_peer::run, peer.get());
    
    /* Init the peer first, as subscribe need to precede PUBLISH */
    rc = peer->next_cmd(CMD_INIT, data1, data2);
    RET_ON_ERR(rc == 0, "mock peer failed to run init");

    /* Just let async bind & subscribe complete */
    sleep(1);

    for (auto itc = tcdata.cbegin(); itc != tcdata.cend(); ++itc) {
        string key = itc.key();

        if (_is_commented(key)) {
            LOM_LOG_INFO("Skip commented entry %s", key.c_str());
            continue;
        }

        json tc_entry = (*itc);

        vector<string> write_data = tc_entry.value("write", vector<string>());
        vector<string> read_data = tc_entry.value("read", vector<string>());

        LOM_LOG_INFO("Running test entry %s:%s", tcid.c_str(), key.c_str());
        if (write_data.size() == 2) {
            data1 = write_data[0];
            data2 = write_data[1];
            rc = peer->next_cmd(CMD_WRITE, data1, data2);
            RET_ON_ERR(rc == 0, "mock peer failed to write (%s) (%s)",
                    data1.c_str(), data2.c_str());
        } else {
            RET_ON_ERR(write_data.empty(), "TEST ERROR: check write data %s", 
                    tcid.c_str());
        }
        {
            string cmd = tc_entry.value("cmd", "");
            vector<string> args = tc_entry.value("args", vector<string>());

            rc = test_client(cmd, args);
            RET_ON_ERR(rc == 0, "Failed to run client cmd(%s)", cmd.c_str());
        }
        if (read_data.size() == 2) {
            rc = peer->next_cmd(CMD_READ, data1, data2, 2);
            RET_ON_ERR(rc == 0, "mock peer failed to read");

            RET_ON_ERR(data1 == read_data[0], "Test compare fail read(%s) != exp(%s)",
                    data1.c_str(), read_data[0].c_str());
            RET_ON_ERR(data2 == read_data[1], "Test compare fail read(%s) != exp(%s)",
                    data2.c_str(), read_data[1].c_str());
        } else {
            RET_ON_ERR(read_data.empty(), "TEST ERROR: check read data %s", 
                    tcid.c_str());
        }
    }
out:
    LOM_LOG_INFO("%s test case %s rc=%d", (rc == 0 ? "completed" : "aborted"),
            tcid.c_str(), rc);
    peer->next_cmd(CMD_QUIT, data1, data2);
    thr.join();
    return rc;
}


int 
run_client_testcases(const json &tccases)
{
    int rc = 0;

    LOM_LOG_INFO("Running all client test_cases\n");

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

    set_test_mode();

    string tcfile(argc > 1 ? argv[1] : TEST_CASE_FILE);

    ifstream f(tcfile.c_str());
    json data = json::parse(f, nullptr, false);

    RET_ON_ERR(!data.is_discarded(), "Failed to parse file");
    LOM_LOG_DEBUG("%s: data.is_discarded = %d\n", tcfile.c_str(), data.is_discarded());

    rc = run_client_testcases(data.value("client_test_cases", json()));
    RET_ON_ERR(rc == 0, "run_testcases failed rc=%d", rc);

    LOM_LOG_INFO("SUCCEEDED in running test cases");

out:
    return rc;
}

