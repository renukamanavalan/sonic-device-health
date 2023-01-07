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
 * As JSON object keys are strings, a conversion method *
 * is provided as enum to str and vice versa            *
 *                                                      *
 * The list of attrs indeed vary by request type.       *
 * The map below lists attrs expected per request type. *
 *                                                      *
 ********************************************************/
typedef enum RequestType {
    REGISTER_CLIENT = 0,
    REGISTER_ACTION,
    DEREGISTER_CLIENT,
    ACTION_HEARTBEAT,
    ACTION_REQUEST,
    ACTION_RESPONSE,
    ACTION_SHUTDOWN,
    CLIENT_REQ_COUNT
} RequestType_t;


typedef enum AttributesName {
    REQUEST_TYPE = 0,
    CLIENT_NAME,
    ACTION_NAME,
    INSTANCE_ID,
    ANOMALY_INSTANCE_ID,
    ANOMALY_KEY,
    CONTEXT,
    TIMEOUT,
    ACTION_DATA,
    RESULT_CODE,
    RESULT_STR,
    ATTR_CNT
} AttributesName_t;

/* Attr name enum to string and vice versa*/
typedef const char *AttributesNameStr_t;
AttributesNameStr_t AttributesNameStr(AttributesName_t);

AttributesName_t AttributesName(AttributesNameStr_t);


typedef std::vector<AttributesName_t> attr_lst_t;
typedef std:map<RequestType_t, const attr_lst_t> req_attrs_lst_t;

const req_attrs_lst_t req_attrs_lst = {
    { REGISTER_CLIENT, attr_lst_t { REQUEST_TYPE, CLIENT_NAME }},
    { REGISTER_ACTION, attr_lst_t { REQUEST_TYPE, CLIENT_NAME, ACTION_NAME }},
    { DEREGISTER_CLIENT, attr_lst_t { REQUEST_TYPE, CLIENT_NAME }},
    { ACTION_HEARTBEAT, attr_lst_t { REQUEST_TYPE, CLIENT_NAME,
                                       ACTION_NAME, INSTANCE_ID }},
    { ACTION_REQUEST,
        attr_lst_t { REQUEST_TYPE, CLIENT_NAME, ACTION_NAME, INSTANCE_ID,
            ANOMALY_INSTANCE_ID, ANOMALY_KEY, CONTEXT, TIMEOUT }},
    { ACTION_RESPONSE,
        attr_lst_t { REQUEST_TYPE, CLIENT_NAME, ACTION_NAME, INSTANCE_ID,
            ANOMALY_INSTANCE_ID, ANOMALY_KEY, ACTION_DATA, RESULT_CODE, RESULT_STR }},
    { ACTION_SHUTDOWN, attr_lst_t { REQUEST_TYPE }}
};

/*
 * JSON string of message object, where attrs vary per request as
 * listed above.
 */
typedef std::string message_t;

#endif // _CONSTS_H_
