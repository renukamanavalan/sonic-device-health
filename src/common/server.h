#ifndef _SERVER_H_
#define _SERVER_H_

#include <string>
#include <memory>
#include <vector>
#include <unordered_set>
#include "consts.h"

typedef std::unordered_set<std::string> keys_set_t;
typedef keys_set_t::const_iterator keys_set_itc;

/*
 * Abstracted message written & read by server/engine
 *
 * Explicit derived classes written per request type.
 *
 */
typedef enum {
    REQ_TYPE_REGISTER_CLIENT = 0,
    REQ_TYPE_DEREGISTER_CLIENT,
    REQ_TYPE_REGISTER_ACTION,
    REQ_TYPE_HEARTBEAT,
    REQ_TYPE_ACTION_REQUEST,
    REQ_TYPE_SHUTDOWN,
    REQ_TYPE_ACTION_RESPONSE,
    REQ_TYPE_COUNT
} RequestType_t;

extern const char *REQ_TYPE_TO_STR[];


class ServerMsg {
    public:
        ServerMsg(RequestType_t type) : m_type(type) {};
        virtual ~ServerMsg() {};

        RequestType_t get_type() const { return m_type; }
        const char * get_type_str() const {
            return m_type < REQ_TYPE_COUNT ?
                    REQ_TYPE_TO_STR[m_type] : "UNKNOWN"; }

        virtual bool is_shutdown() const { return false; }

        virtual bool validate() const;

        virtual std::string get(const std::string key) const {
            map_str_str_t::const_iterator itc = m_data.find(key);
            return (itc == m_data.end()) ? "" : itc->second;
        }

        virtual int set(const std::string key, const std::string val);
        virtual int set(const map<std::string, std::string} &lst);

        virtual std::string to_str() const { return convert_to_json(get_type_str(m_type), m_data); };

        bool operator==(const ServerMsg &msg) const;

    protected:

        RequestType_t m_type;
        map_str_str_t m_data;

        keys_set_t m_reqd_keys;
        keys_set_t m_opt_keys;
};

typedef std::shared_ptr<ServerMsg> ServerMsg_ptr_t;

class RegisterClient : public ServerMsg {
    public:
        RegisterClient(): ServerMsg(REQ_TYPE_REGISTER_CLIENT) {
            m_reqd_keys = { REQ_CLIENT_NAME };
        };
};


class DeregisterClient : public ServerMsg {
    public:
        DeregisterClient(): ServerMsg(REQ_TYPE_DEREGISTER_CLIENT) {
            m_reqd_keys = { REQ_CLIENT_NAME };
        };
};


class RegisterAction : public ServerMsg {
    public:
        RegisterAction(): ServerMsg(REQ_TYPE_REGISTER_ACTION) {
            m_reqd_keys = { REQ_CLIENT_NAME, REQ_ACTION_NAME };
        };
};


class HeartbeatClient : public ServerMsg {
    public:
        HeartbeatClient(): ServerMsg(REQ_TYPE_HEARTBEAT) {
            m_reqd_keys = { REQ_CLIENT_NAME, REQ_ACTION_NAME, REQ_INSTANCE_ID };
        };
};


class ActionRequest : public ServerMsg {
    public:
        ActionRequest(): ServerMsg(REQ_TYPE_ACTION_REQUEST) {
            m_reqd_keys = {
                REQ_CLIENT_NAME, REQ_ACTION_NAME, REQ_ACTION_TYPE, REQ_INSTANCE_ID,
                REQ_ANOMALY_INSTANCE_ID };

            m_opt_keys = {
                REQ_ANOMALY_KEY, REQ_CONTEXT, REQ_TIMEOUT, REQ_HEARTBEAT_INTERVAL};
        };
};


class ShutdownRequest : public ServerMsg {
    public:
        ShutdownRequest(): ServerMsg(REQ_TYPE_SHUTDOWN) {
            m_reqd_keys = { REQ_CLIENT_NAME, REQ_ACTION_TYPE };
        };
        virtual bool is_shutdown() const { return true; }
};


class ActionResponse : public ServerMsg {
    public:
        ActionResponse(): ServerMsg(REQ_TYPE_ACTION_RESPONSE) {
            m_reqd_keys = {
                REQ_CLIENT_NAME, REQ_ACTION_NAME, REQ_ACTION_TYPE,
                REQ_INSTANCE_ID, REQ_ANOMALY_INSTANCE_ID, 
                REQ_ACTION_DATA, REQ_RESULT_CODE };
            m_opt_keys = { REQ_ANOMALY_KEY, REQ_RESULT_STR };
        };
};


ServerMsg_ptr_t create_server_msg(const std::string msg);

/* APIs for use by server/engine */

/* Required as the first call before using any other APIs */
int server_init(const std::vector<std::string> &clients);


/* Helps release all resources before exit */
/* declared in server_c.h */
// void server_deinit();

/*
 * Writes a message to client
 *
 * Input:
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
int write_server_message(const ServerMsg_ptr_t msg);


/*
 * Reads a message from client.
 *
 * Input:
 *  timeout - 
 *      0 - No wait. process a message if available and return.
 *    > 0 - Wait for these many seconds at most for a message, before timeout.
 *    < 0 - Wait forever until a message is available to read & process.
 *
 * Output:
 *  None
 *
 * Return:
 *  0 - Success.
 * <0 - Failure
 */

ServerMsg_ptr_t read_server_message(int timeout=-1);

/* Get formatted time stamp as needed for publishing */
const std::string get_timestamp();

#endif  // _SERVER_H_
