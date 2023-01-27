#include "consts.h"


/*
 * requests
 * These are between clib client & server, hence mocked here.
 */
const char * REQ_REGISTER_CLIENT = "register_client";
const char * REQ_DEREGISTER_CLIENT = "deregister_client";
const char * REQ_REGISTER_ACTION = "register_action";
const char * REQ_HEARTBEAT = "heartbeat";
const char * REQ_ACTION_REQUEST = "action_request";
const char * REQ_ACTION_RESPONSE = "action_response";


/*
 * Expected attribute names from CDLL for Action req/resp
 * These can be refreshed from loaded DLL
 * e.g. _get_str_globals("REQ_ACTION_TYPE")
 */
const char * REQ_ACTION_TYPE = "request_type";
const char * REQ_ACTION_TYPE_ACTION = "action";
const char * REQ_ACTION_TYPE_SHUTDOWN = "shutdown";

const char * REQ_CLIENT_NAME = "client_name";
const char * REQ_ACTION_NAME = "action_name";
const char * REQ_INSTANCE_ID = "instance_id";
const char * REQ_ANOMALY_INSTANCE_ID = "anomaly_instance_id";
const char * REQ_ANOMALY_KEY = "anomaly_key";
const char * REQ_CONTEXT = "context";
const char * REQ_TIMEOUT = "timeout";
const char * REQ_HEARTBEAT_INTERVAL = "heartbeat_interval";
const char * REQ_PAUSE = "action_pause";

const char * REQ_ACTION_DATA = "action_data";
const char * REQ_RESULT_CODE = "result_code";
const char * REQ_RESULT_STR = "result_str";

const char * REQ_MITIGATION_STATE = "state";
const char * REQ_MITIGATION_STATE_INIT = "init";
const char * REQ_MITIGATION_STATE_PROG = "in-progress";
const char * REQ_MITIGATION_STATE_TIMEOUT = "timeout";
const char * REQ_MITIGATION_STATE_DONE = "complete";

const char * SUB_END_PATH = "tcp://127.0.0.1:5572";
const char * PUB_END_PATH = "tcp://127.0.0.1:5570";

