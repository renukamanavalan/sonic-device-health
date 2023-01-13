#ifndef _SERVER_H_
#define _SERVER_H_

#include "consts.h"

typedef std::unordered_set<std::string> keys_set_t;
typedef keys_set_t::const_iterator keys_set_itc;

/*
 * Abstracted message written & read by server/engine
 *
 * Explicit derived classes written per request type.
 *
 */
class ServerMsg {
    public:
        ServerMsg(RequestType_t type) : m_type(type) { init(); }
        virtual ~ServerMsg() {};

        RequestType_t get_type() const { return m_type; }

        virtual bool validate() const;

        virtual std::string get(const std::string key) const {
            map_str_str_t::const_iterator itc = m_data.find(key);
            return (itc == m_data.end()) ? "" : itc->second;
        }

        virtual int set(const std::string key, const std::string val) 
        {
            keys_set_itc itc = m_reqd_keys.find(key);
            if (itc == m_reqd_keys.end()) {
                itc = m_opt_keys.find(key);
                RET_ON_ERR(itc != m_opt_keys.end(), "Unexpected key %s", key.c_str());
            }
            else {
                RET_ON_ERR(!val.empty(), "required key %s val is empty", key.c_str());
            }
            m_data[key] = val;
        out:
            return rc;
        }

        virtual std::string to_str() const { return convert_to_json(m_type, m_data); };

        bool operator==(const ServerMsg &msg) const
        {
            return ((m_type == msg.m_type) && (m_data == msg.m_data)) ? true : false;
        };

    protected:
        virtual void init() = 0;

        RequestType_t m_type;
        map_str_str_t m_data;

        keys_set_t m_reqd_keys;
        keys_set_t m_opt_keys;
};

typedef std::shared_ptr<ServerMsg> ServerMsg_ptr_t;

class RegisterClient : ServerMsg {
    public:
        RegisterClient(): ServerMsg(REQ_REGISTER_CLIENT) {};

        virtual void init() {
            m_reqd_keys = { REQ_CLIENT_NAME };
        }
};


class DeregisterClient : ServerMsg {
    public:
        DeregisterClient(): ServerMsg(REQ_DEREGISTER_CLIENT) {};

        virtual void init() {
            m_reqd_keys = { REQ_CLIENT_NAME };
        }
};


class RegisterAction : ServerMsg {
    public:
        RegisterAction(): ServerMsg(REQ_REGISTER_ACTION) {};

        virtual void init() {
            m_reqd_keys = { REQ_CLIENT_NAME, REQ_ACTION_NAME };
        }
};


class HeartbeatClient : ServerMsg {
    public:
        HeartbeatClient(): ServerMsg(REQ_HEARTBEAT) {};

        virtual void init() {
            m_reqd_keys = { REQ_CLIENT_NAME, REQ_ACTION_NAME, REQ_INSTANCE_ID };
        }
};


class ActionRequest : ServerMsg {
    public:
        ActionRequest(): ServerMsg(REQ_ACTION_REQUEST) {};

        virtual void init() {
            m_reqd_keys = {
                REQ_CLIENT_NAME, REQ_ACTION_NAME, REQ_INSTANCE_ID,
                REQ_ANOMALY_INSTANCE_ID };

            m_opt_keys = {
                REQ_ANOMALY_KEY, REQ_CONTEXT, REQ_TIMEOUT, REQ_HEARTBEAT_INTERVAL};
        }
};


class ActionResponse : ServerMsg {
    public:
        ActionResponse(): ServerMsg(REQ_ACTION_RESPONSE) {};

        virtual void init() {
            m_reqd_keys = {
                REQ_CLIENT_NAME, REQ_ACTION_NAME, REQ_INSTANCE_ID,
                REQ_ANOMALY_INSTANCE_ID, REQ_ANOMALY_KEY, 
                REQ_ACTION_DATA, REQ_RESULT_CODE, REQ_RESULT_STR };
        }
};


ServerMsg_ptr_t create_server_msg(const std::string msg);

/* APIs for use by server/engine */

/* Required as the first call before using any other APIs */
int server_init();


/* Helps release all resources before exit */
void server_deinit();

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
int write_message(const ActionRequest &);


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

ServerMsg_ptr_t read_message(int timeout=-1);

#endif  // _SERVER_H_
