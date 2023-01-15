#include <fstream>      // std::ifstream
#include <sstream>      // std::ifstream
#include <stdio.h>
#include <vector>
#include <string>
#include <nlohmann/json.hpp>
#include "consts.h"
#include "common.h"
#include "client.h"
#include "server.h"

using namespace std;
using json = nlohmann::json;

#define TEST_ERR_PREFIX "TEST_ERROR:"

#define TEST_CASE_FILE "test_data_ut.json"

static bool
_is_commented(const string key)
{
    return (key.compare(0, 1, "_") == 0) ? true : false;
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

        rc = poll_for_data(NULL, 0, 2);
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
    return rc;
}


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

        rc = write_message(test_msg);
        RET_ON_ERR(rc == 0, "Failed to write message (%s)", data.c_str());
    }
    else {
        rc = poll_for_data(NULL, 0, 2);
        RET_ON_ERR(rc == -1, "Poll failed rc=%d", rc);

        ServerMsg_ptr_t read_msg = read_message();
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
    return rc;
}


int
run_a_test_case(const string tcid, const json &tcdata)
{
    int rc = 0;

    LOM_LOG_INFO("Running test case %s", tcid.c_str());

    for (auto itc = tcdata.cbegin(); itc != tcdata.cend(); ++itc) {

        string key = itc.key();
        if (_is_commented(key)) {
            LOM_LOG_INFO("Skip commented entry %s", key.c_str());
            continue;
        }

        json tc_entry = (*itc);

        LOM_LOG_INFO("Running test entry %s:%s", tcid.c_str(), key.c_str());

        bool is_client = tc_entry.value("is_client", false);
        if (is_client) {
            string cmd = tc_entry.value("cmd", "");
            vector<string> args = tc_entry.value("args", vector<string>());

            rc = test_client(cmd, args);
            RET_ON_ERR(rc == 0, "Failed to run client cmd(%s)", cmd.c_str());
        } else {
            rc = test_server(tc_entry.value("is_write", false),
                    tc_entry.value("data", string()));
            RET_ON_ERR(rc == 0, "Failed to run sever is_write=%d data(%s)",
                    tc_entry.value("is_write", false),
                    tc_entry.value("data", string()).c_str());
        }
    }
out:
    LOM_LOG_INFO("%s test case %s rc=%d", (rc == 0 ? "completed" : "aborted"),
            tcid.c_str(), rc);
    return rc;
}


int 
run_testcases(const json &tccases)
{
    int rc = 0;

    LOM_LOG_INFO("Running all test_cases\n");

    for (auto itc = tccases.cbegin(); itc != tccases.cend(); ++itc) {

        string key = itc.key();
        if (_is_commented(key)) {
            LOM_LOG_INFO("Skip commented testcase %s", key.c_str());
            continue;
        }

        rc = run_a_test_case(key, itc.value());
        RET_ON_ERR(rc == 0, "Failed to run test case %s", key.c_str());
    }
out:
    return rc;
}
             

int main(int argc, const char **argv)
{
    int rc = 0;

    set_test_mode();

    string tcfile(argc > 1 ? argv[0] : TEST_CASE_FILE);

    ifstream f(tcfile.c_str());

    json data = json::parse(f, nullptr, false);

    RET_ON_ERR(!data.is_discarded(), "Failed to parse file");

    rc = run_testcases(data.value("test_cases", json()));
    RET_ON_ERR(rc == 0, "run_testcases failed rc=%d", rc);

    LOM_LOG_INFO("SUCCEEDED in running test cases");

out:
    return rc;
}

