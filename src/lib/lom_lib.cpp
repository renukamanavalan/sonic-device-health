#include <stdio.h>
#include <stdarg.h>
#include <sstream>
#include <syslog.h>
#include <unistd.h>
#include "consts.h"
#include "common.h"
#include "client.h"
#include "server.h"
#include "transport.h"

using namespace std;

typedef struct registered {
    string client_name;
    unordered_set<string> actions;
} registered_t;

static registered_t s_registered;

/* Expect only either transport, except for test code */
static client_transport_ptr_t s_client_tx;
static server_transport_ptr_t s_server_tx;

ServerMsg_ptr_t
create_server_msg(const string msg)
{
    map_str_str_t data;
    int rc = 0;
    string type;
    ServerMsg_ptr_t req, ret;

    rc = convert_from_json(msg, type, data);
    RET_ON_ERR(rc == 0, "Failed to parse JSON (%s)", msg.c_str());

    if (type == REQ_REGISTER_CLIENT) {
        req.reset(new RegisterClient());
    } else if (type == REQ_DEREGISTER_CLIENT) {
        req.reset(new DeregisterClient());
    } else if (type == REQ_REGISTER_ACTION) {
        req.reset(new RegisterAction());
    } else if (type == REQ_HEARTBEAT) {
        req.reset(new HeartbeatClient());
    } else if (type == REQ_ACTION_RESPONSE) {
        req.reset(new ActionResponse());
    } else if (type == REQ_ACTION_REQUEST) {
        req.reset(new ActionRequest());
    }

    for(map_str_str_t::const_iterator itc = data.begin();
            itc != data.end(); ++itc) {
        rc = req->set(itc->first, itc->second);
        RET_ON_ERR("Failed to set. Type(%s) key(%s) val(%s)",
                type.c_str(), itc->first.c_str(), itc->second.c_str());
    }

    RET_ON_ERR(req->validate(), "req (%s) failed to validate", msg.c_str());

    ret = req;
out:
    return ret;
}


bool
ServerMsg::validate() const
{
    int rc = 0;

    RET_ON_ERR (!m_reqd_keys.empty(), "Expect non empty reqd keys");
    for (keys_set_itc itc = m_reqd_keys.begin(); itc != m_reqd_keys.end(); ++itc) {
        map_str_str_t::const_iterator itc_data = m_data.find(*itc);

        RET_ON_ERR((itc_data != m_data.end()) && (!itc_data->second.empty()),
                "Failed to find required key (%s)", (*itc).c_str());
    }
out:
    return rc == 0 ? true : false;
}

int
ServerMsg::set(const std::string key, const std::string val) 
{
    int rc = 0;

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

bool
ServerMsg::operator==(const ServerMsg &msg) const
{
    return ((m_type == msg.m_type) && (m_data == msg.m_data)) ? true : false;
}


/*
 * client side access APIs as per client.h
 */

int
register_client(const char *client_id)
{
    int rc = 0;
    client_transport_ptr_t tx;

    {
        stringstream ss;

        ss << "LoM:" << client_id;
        log_init(ss.str().c_str());
    }

    string str_id(client_id);
    ServerMsg_ptr_t msg(new RegisterClient());

    rc = msg->set(REQ_CLIENT_NAME, str_id);
    RET_ON_ERR(rc == 0, "Failed to set client name %s", client_id);

    RET_ON_ERR(s_registered.client_name.empty() || (s_client_tx == NULL),
            "Duplicate registration exist: %s new:%s",
            s_registered.client_name.c_str(), client_id);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg->to_str().c_str());

    tx = init_client_transport(str_id);
    RET_ON_ERR(tx != NULL, "Failed to init client");

    sleep(5);

    rc = tx->write(msg->to_str());
    RET_ON_ERR(rc == 0, "Failed to write register client");

    s_registered.client_name = str_id;
    unordered_set<string>().swap(s_registered.actions);
    s_client_tx = tx;
out:
    LOM_LOG_DEBUG("register_client returned rc=%d", rc);
    return rc;
}


int
deregister_client(void)
{
    int rc = 0;
    ServerMsg_ptr_t msg(new DeregisterClient());

    RET_ON_ERR(s_client_tx != NULL, "No transport to server");

    rc = msg->set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg->to_str().c_str());

    rc = s_client_tx->write(msg->to_str());
    RET_ON_ERR(rc == 0, "Failed to write deregister client");

out:
    string().swap(s_registered.client_name);
    unordered_set<string>().swap(s_registered.actions);

    s_client_tx.reset();
    return rc;
}


int
register_action(const char *action)
{
    int rc = 0;
    ServerMsg_ptr_t msg(new RegisterAction());
    string str_action(action);

    RET_ON_ERR(s_client_tx != NULL, "No transport to server");

    RET_ON_ERR(s_registered.actions.find(str_action) == 
            s_registered.actions.end(), "Duplicate action (%s) registration",
            action);

    rc = msg->set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());
    rc = msg->set(REQ_ACTION_NAME, str_action);
    RET_ON_ERR(rc == 0, "Failed to set action name %s", action);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg->to_str().c_str());

    rc = s_client_tx->write(msg->to_str());
    RET_ON_ERR(rc == 0, "Failed to write register action");

    s_registered.actions.insert(str_action);
out:
    return rc;
}

int
touch_heartbeat(const char *action, const char *instance_id)
{
    int rc = 0;
    ServerMsg_ptr_t msg(new HeartbeatClient());
    string str_action(action), str_id(instance_id);

    RET_ON_ERR(s_client_tx != NULL, "No transport to server");

    RET_ON_ERR(s_registered.actions.find(str_action) != 
            s_registered.actions.end(), "Missing action (%s) registration",
            action);

    rc = msg->set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());

    rc = msg->set(REQ_ACTION_NAME, str_action);
    RET_ON_ERR(rc == 0, "Failed to set action name %s", action);

    rc = msg->set(REQ_INSTANCE_ID, str_id);
    RET_ON_ERR(rc == 0, "Failed to set instance id %s", instance_id);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg->to_str().c_str());

    rc = s_client_tx->write(msg->to_str());
    RET_ON_ERR(rc == 0, "Failed to write heartbeat");
out:
    return rc;
}


const char *
read_action_request(int timeout)
{
    string id, msg, req_client;
    ServerMsg_ptr_t req;
    int rc = 0;

    RET_ON_ERR(s_client_tx != NULL, "No transport to server");

    rc = s_client_tx->read(msg);
    RET_ON_ERR(rc == 0, "Failed to read msg from engine");
    RET_ON_ERR(id == s_registered.client_name, "Read id(%s) != client(%s)",
           id.c_str(), s_registered.client_name.c_str());


    req = create_server_msg(msg);

    RET_ON_ERR(req->validate(), "req (%s) failed to validate", msg.c_str());

    req_client = req->get(REQ_CLIENT_NAME);
    RET_ON_ERR(id == req_client, "Read req_client(%s) != client(%s)",
           id.c_str(), req_client.c_str());
out:
    return (rc == 0) ? msg.c_str() : "";
}


int
write_action_response(const char *res)
{
    int rc = 0;
    ServerMsg_ptr_t msg;
    string str_res(res);

    RET_ON_ERR(s_client_tx != NULL, "No transport to server");

    msg = create_server_msg(str_res);
    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", res);

    rc = msg->set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", res);

    rc = s_client_tx->write(msg->to_str());
    RET_ON_ERR(rc == 0, "Failed to write action response");
out:
    return rc;
}
 
/*
 * Server side access APIs as per server.h
 */

int
server_init(const vector<string> clients)
{
    int rc = 0;
    server_transport_ptr_t tx;

    log_init("LoM-Engine", LOG_LOCAL0);

    RET_ON_ERR(s_server_tx == NULL, "Transport exists");
    tx = init_server_transport(clients);
    RET_ON_ERR(tx != NULL, "Failed to init server");
    s_server_tx = tx;
out:
    return rc;
}

void
server_deinit()
{
    s_server_tx.reset();
}

int
write_server_message(const ServerMsg_ptr_t req)
{
    int rc = 0;

    RET_ON_ERR(s_server_tx != NULL, "No transport for server");

    RET_ON_ERR(req != NULL, "Expect non null ptr");
    RET_ON_ERR(req->validate(), "req (%s) failed to validate",
            req->to_str().c_str());

    rc = s_server_tx->write(req->get(REQ_CLIENT_NAME), req->to_str());
    RET_ON_ERR(rc == 0, "Failed to write to client");
out:
    return rc;
}



ServerMsg_ptr_t
read_server_message(int timeout)
{
    int rc = 0;
    string client_id, str_msg;
    ServerMsg_ptr_t msg, ret;

    RET_ON_ERR(s_server_tx != NULL, "No transport for server");

    rc = s_server_tx->read(client_id, str_msg, timeout);
    RET_ON_ERR(rc == 0, "Failed to read from transport");

    msg = create_server_msg(str_msg);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", str_msg.c_str());

    ret = msg;
out:
    return ret;
}

