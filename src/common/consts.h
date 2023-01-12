#ifndef _CONSTS_H_
#define _CONSTS_H_

/********************************************************
 * Messages between client and server                   *
 *                                                      *
 * Transport as JSON string of an encoded JSON object.  *
 * The following provides the key values for the JSON   *
 * object.                                              *
 * RequestType_t - Enumerates all request types         *
 * AttributesName_t - Enumerates all attribute types    *
 *                                                      *
 * The list of attrs indeed vary by request type.       *
 * The map below lists attrs expected per request type. *
 *                                                      *
 ********************************************************/

/*
 * requests
 * These are between clib client & server, hence mocked here.
 */
#define REQ_REGISTER_CLIENT "register_client"
#define REQ_DEREGISTER_CLIENT "deregister_client"
#define REQ_REGISTER_ACTION "register_action"
#define REQ_HEARTBEAT "heartbeat"
#define REQ_ACTION_REQUEST "action_request"
#define REQ_ACTION_RESPONSE "action_response"


/*
 * Expected attribute names from CDLL for Action req/resp
 * These can be refreshed from loaded DLL
 * e.g. _get_str_globals("REQ_TYPE")
 */
#define REQ_TYPE "request_type"
#define REQ_TYPE_ACTION "action"
#define REQ_TYPE_SHUTDOWN "shutdown"

#define REQ_CLIENT_NAME "client_name"
#define REQ_ACTION_NAME "action_name"
#define REQ_INSTANCE_ID "instance_id"
#define REQ_ANOMALY_INSTANCE_ID "anomaly_instance_id"
#define REQ_ANOMALY_KEY "anomaly_key"
#define REQ_CONTEXT "context"
#define REQ_TIMEOUT "timeout"
#define REQ_HEARTBEAT_INTERVAL "heartbeat_interval"
#define REQ_PAUSE "action_pause"

#define REQ_ACTION_DATA "action_data"
#define REQ_RESULT_CODE "result_code"
#define REQ_RESULT_STR "result_str"

#define REQ_MITIGATION_STATE "state"
#define REQ_MITIGATION_STATE_INIT "init"
#define REQ_MITIGATION_STATE_PROG "in-progress"
#define REQ_MITIGATION_STATE_TIMEOUT "timeout"
#define REQ_MITIGATION_STATE_DONE "complete"

/*
 * JSON string of message object, where attrs vary per request as
 * listed above.
 */
typedef std::string message_t;

#endif // _CONSTS_H_
