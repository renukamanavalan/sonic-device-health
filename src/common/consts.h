#ifndef _CONSTS_H_
#define _CONSTS_H_

#ifdef __cplusplus
extern "C" {
#endif

/********************************************************
 * Messages between client and server                   *
 *                                                      *
 * Transport as JSON string of an encoded JSON object.  *
 * The following provides the key values for the JSON   *
 * object.                                              *
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
extern const char * REQ_REGISTER_CLIENT;
extern const char * REQ_DEREGISTER_CLIENT;
extern const char * REQ_REGISTER_ACTION;
extern const char * REQ_HEARTBEAT;
extern const char * REQ_ACTION_REQUEST;
extern const char * REQ_ACTION_RESPONSE;


/*
 * Expected attribute names from CDLL for Action req/resp
 * These can be refreshed from loaded DLL
 * e.g. _get_str_globals("REQ_ACTION_TYPE")
 */
extern const char * REQ_ACTION_TYPE;
extern const char * REQ_ACTION_TYPE_ACTION;
extern const char * REQ_ACTION_TYPE_SHUTDOWN;

extern const char * REQ_CLIENT_NAME;
extern const char * REQ_ACTION_NAME;
extern const char * REQ_INSTANCE_ID;
extern const char * REQ_ANOMALY_INSTANCE_ID;
extern const char * REQ_ANOMALY_KEY;
extern const char * REQ_CONTEXT;
extern const char * REQ_TIMEOUT;
extern const char * REQ_HEARTBEAT_INTERVAL;
extern const char * REQ_PAUSE;

extern const char * REQ_ACTION_DATA;
extern const char * REQ_RESULT_CODE;
extern const char * REQ_RESULT_STR;

extern const char * REQ_MITIGATION_STATE;
extern const char * REQ_MITIGATION_STATE_PENDING;
extern const char * REQ_MITIGATION_STATE_PROG;
extern const char * REQ_MITIGATION_STATE_TIMEOUT;
extern const char * REQ_MITIGATION_STATE_DONE;

extern const char * SUB_END_PATH;
extern const char * PUB_END_PATH;

extern const char * REQ_TIMESTAMP;
extern const char * REQ_ACTIONS;


/* Actions.conf */
extern const char * ACTION_CONF_TIMEOUT;
extern const char * ACTION_CONF_DISABLE;
extern const char * ACTION_CONF_MIMIC;
extern const char * ACTION_CONF_MANDATORY;
extern const char * ACTION_CONF_MIN_RECUR;
extern const char * ACTION_CONF_HB_INTERVAL;
extern const char * ACTION_CONF_MITIGATION_TIMEOUT;

#ifdef __cplusplus
}
#endif

#endif // _CONSTS_H_
