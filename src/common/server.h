#ifndef _SERVER_H_
#define _SERVER_H_

#include "consts.h"

/* APIs for use by server/engine */

/* Required as the first call before using any other APIs */
int server_init();


/* Helps release all resources before exit */
void server_deinit();

typedef struct RegisterRequest {
    RequestType_t                       req;
    std::string                         client_name;
    /* action_name empty if not REGISTER_ACTION */
    std::string                         action_name;
};

typedef struct HeartbeatTouch {
    std::string                         client_name;
    std::string                         action_name;
    std::string                         instance_id;
    std::string                         anomaly_instance_id;
};

typedef struct ActionRequest {
    RequestType_t                       req;
    std::string                         client_name;
    std::string                         action_name;
    std::string                         instance_id;
    std::string                         anomaly_instance_id;
    std::string                         anomaly_key;
    std::map<std::string, std::string>  context;
    std::int                            timeout;
} ActionRequest_t;

typedef struct ActionResponse {
    RequestType_t                       req;
    std::string                         client_name;
    std::string                         action_name;
    std::string                         instance_id;
    std::string                         anomaly_instance_id;
    std::string                         anomaly_key;
    std::string                         action_data;
    std::int                            result_code;
    std::string                         result_str;
} ActionResponse_t;

/*
 * Base class for request handler.
 *
 * Server instantiates a derived class per request type and register the same.
 * process_msg invokes the right handler with right set of args per request type.
 * 
 * The derived class for a type may only implement operator of expected signature
 * for that request type.
 */
class RequestHandler {
    public:
        RequestHandler(RequestType_t type): m_req(type) {};
        virtual ~RequestHandler() {};

        virtual int operator()(const RegisterRequest_t &) { printf("E_NOTIMPL\n"); return -1; }
        virtual int operator()(const HeartbeatTouch_t &) { printf("E_NOTIMPL\n"); return -1; }
        virtual int operator()(const ActionResponse_t &) { printf("E_NOTIMPL\n"); return -1; }


    protected:
        RequestType_t m_req;

};

typedef std::shared_ptr<RequestHandler> RequestHandler_ptr;

typedef std::map<RequestType_t, RequestHandler_ptr> request_handlers;

/*
 * Regiser handler for each request type.
 *
 * Registered handler operator is called with appropriate args per request type
 */
int register_handler(RequestType_t type, RequestHandler_ptr);   



/*
 * Writes a message to client
 *
 * Input:
 *  client_id -
 *      The client-ID of intended recipient
 *
 *  message - JSON string of encoded JSON object
 *
 * Output:
 *  None
 *
 * Return:
 *      0 - Implies success
 *   != 0 - Implies failure
 *
 */
int write_message(const ActionRequest_t &);


/*
 * Reads and process atmost one message from client
 *
 * Input:
 *  timeout - 
 *      0 - No wait. process a message if available and return.
 *    > 0 - Wait for these many seconds at most for a message, before timeout.
 *    < 0 - Wait forever until a message is available to read & process.
 *
 */
int process_a_message(int timeout=-1);

#endif  // _SERVER_H_
