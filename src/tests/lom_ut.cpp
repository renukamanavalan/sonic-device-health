#include <stdio.h>
#include "consts.h"
#include "common.h"
#include "client.h"
#include "server.h"

using namespace std;


static int server_index = 0;
static int client_index = 0;

/* Failure code from server thread */
static int server_result = -1;

/* Abort server thread indicator from main thread */
static bool terminate_server = false;

#define TEST_ERR_PREFIX "TEST_ERROR:"

#define TEST_CASE_FILE "test_data_ut.json"

static int
test_client(const string cmd, const vector<string> args)
{
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
    } else if (cmd != REQ_ACTION_REQUEST) {
        RET_ON_ERR(false, "Invalid command (%s) provided", cmd); 
    }
    else {
        rc = poll_for_data(NULL, 0, 2);
        RET_ON_ERR(rc == -1, "Poll failed rc=%d", rc);

        string str_read(read_action_request());
        RET_ON_ERR(!str_read.empty(), "Empty request string received");

        ServerMsg_ptr_t read_msg = create_server_msg(str_read);
        RET_ON_ERR(read_msg.validate(), "Failed to validate (%s)", str_read.c_str());

        string str_test(args.empty() ? "" : args[0]);
        if (!str_test.empty()) {
            ServerMsg_ptr_t test_msg =  create_server_msg(str_test);
            RET_ON_ERR(test_msg.validate(), "%s Invalid msg (%s)",
                    TEST_ERR_PREFIX, str_test.c_str());

            RET_ON_ERR(read_msg == test_msg, "Failed to match exp(%s) != read(%s)",
                    test_msg.to_str().c_str(), str_read.c_str());
        } else {
            log_info("**** Received msg: (%s)", str_read.c_str());
        }
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
        RET_ON_ERR(test_msg.validate(), "%s Invalid msg (%s)",
                TEST_ERR_PREFIX, data.c_str());
    } else {
        RET_ON_ERR(!is_write, "%s write expects data", TEST_ERR_PREFIX);
    }

    if (is_write) {
        ActionRequest *p = dynamic_cast<ActionRequest>(test_msg.get());
        RET_ON_ERR(p != NULL, "Fail to cast (%s) to ActionRequest", data.c_str());

        rc = write_message(*p);
        RET_ON_ERR(rc == 0, "Failed to write message (%s)", data.c_str());
    }
    else {
        rc = poll_for_data(NULL, 0, 2);
        RET_ON_ERR(rc == -1, "Poll failed rc=%d", rc);

        ServerMsg_ptr_t read_msg = read_message();
        RET_ON_ERR(read_msg.validate(), "Failed to read message (%s)",
                read_msg.to_str().c_str());

        if (test_msg != NULL) {
            RET_ON_ERR(read_msg == *test_msg, "Read message (%s) != expect (%s)",
                    read_msg.to_str().c_str(), data.c_str());
        }
        else {
            log_info("**** Received msg: (%s)", read_msg.to_str().c_str());
        }
    }
out:
    return rc;
}


int
run_a_test_case(const string tcid, const json &tcdata)
{
    int rc = 0;

    log_info("Running test case %s", tcid.c_str());

    for (auto itc = tcdata.cbegin(); itc != tcdata.cend(); ++itc) {

        key = itc.key();
        if (!key.empty() && (key[0] == "_")) {
            log_info("Skip commented entry %s", key.c_str());
            continue;
        }

        json tc_entry = (*itc);

        log_info("Running test entry %s:%s", tcid.c_str(), key.c_str());

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
    log_info("%s test case %s rc=%d", (rc == 0 ? "completed" : "aborted"),
            tcid.c_str(), rc);
    return rc;
}


int 
run_testcases(const json &tccases)
{
    int rc = 0;

    log_info("Running all test_cases\n");

    for (auto itc = tccases.cbegin(); itc != tccases.cend(); ++itc) {

        key = itc.key();
        if (!key.empty() && (key[0] == "_")) {
            log_info("Skip commented testcase %s", key.c_str());
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

    ifstream f(tcfile);

    data = json::parse(f, nullptr, false);

    RET_ON_ERR(!data.is_discarded(), "Failed to parse file");

    rc = run_testcases(data.value("test_cases", json()));
    RET_ON_ERR(rc == 0, "run_testcases failed rc=%d", rc);

    LOG_INFO("SUCCEEDED in running test cases");

out:
    return rc;
}

