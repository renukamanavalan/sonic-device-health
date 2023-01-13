#include <stdio.h>
#include "consts.h"
#include "common.h"
#include "client.h"
#include "server.h"
#include "transport.h"

typedef struct registered {
    string client_name;
    unordered_set<string> actions;
} registered_t;

static registered_t s_registered;

ServerMsg_ptr_t
create_server_message(const string msg)
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
    bool ret = false;

    for (keys_set_itc itc = m_reqd_keys.begin(); itc != m_reqd_keys.end(); ++itc) {
        map_str_str_t::const_iterator itc_data = m_data.find(*itc);

        RET_ON_ERR((itc_data != m_data.end()) && (!itc->second.empty()),
                "Failed to find required key (%s)", (*itc).c_str());
    }
    ret = true;
out:
    return ret;
}

bool
ServerMsg::operator==(const ServerMsg &msg) const
{
    return ((m_type == msg.m_type) && (m_data == msg.m_data)) ? true : false;



/*
 * client side access APIs as per client.h
 */

int
register_client(const char *client_id)
{
    int rc = -1;

    {
        stringstream ss;

        ss << "LoM:" << client_id;
        log_init(ss.str().c_str(), LOG_LOCAL0);
    }

    string str_id(client_id);
    ServerMsg_ptr_t msg(new RegisterClient());

    rc = msg.set(REQ_CLIENT_NAME, str_id);
    RET_ON_ERR(rc == 0, "Failed to set client name %s", client_id);

    RET_ON_ERR(s_registered.client_name.empty(),
            "Duplicate registration exist: %s new:%s",
            s_registered.client_name.c_str(), client_id);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg.to_str().c_str());

    rc = init_client_transport(str_id);
    RET_ON_ERR(rc == 0, "Failed to init client");

    rc = write_message(msg.to_str());
    RET_ON_ERR(rc == 0, "Failed to write register client");

    s_registered.client_name = str_id;
    unordered_set().swap(s_registered.actions);
out:
    return rc;
}


int
deregister_client(void)
{
    int rc = -1;
    ServerMsg_ptr_t msg(new DeregisterClient());

    RET_ON_ERR(!s_registered.client_name.empty(), "No registered client");

    rc = msg.set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg.to_str().c_str());

    rc = write_message(msg.to_str());
    RET_ON_ERR(rc == 0, "Failed to write deregister client");

    string().swap(s_registered.client_name);
    unordered_set<string>().swap(s_registered.actions);

out:
    return rc;
}


int
register_action(const char *action)
{
    int rc = -1;
    ServerMsg_ptr_t msg(new RegisterAction());
    string str_action(action);

    RET_ON_ERR(s_registered.actions.find(str_action) == 
            s_registered.actions.end(), "Duplicate action (%s) registration",
            action);

    rc = msg.set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());
    rc = msg.set(REQ_ACTION_NAME, str_action);
    RET_ON_ERR(rc == 0, "Failed to set action name %s", action);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg.to_str().c_str());

    rc = write_message(msg.to_str());
    RET_ON_ERR(rc == 0, "Failed to write register action");

    s_registered.actions.insert(str_action);
out:
    return rc;
}

void
touch_heartbeat(const char *action, const char *instance_id)
{
    int rc = -1;
    ServerMsg_ptr_t msg(new HeartbeatClient());
    string str_action(action), str_id(instance_id);

    RET_ON_ERR(s_registered.actions.find(str_action) != 
            s_registered.actions.end(), "Missing action (%s) registration",
            action);

    rc = msg.set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());

    rc = msg.set(REQ_ACTION_NAME, str_action);
    RET_ON_ERR(rc == 0, "Failed to set action name %s", action);

    rc = msg.set(REQ_INSTANCE_ID, str_id);
    RET_ON_ERR(rc == 0, "Failed to set instance id %s", instance_id);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", msg.to_str().c_str());

    rc = write_message(msg.to_str());
    RET_ON_ERR(rc == 0, "Failed to write heartbeat");
out:
    return rc;
}


const char *read_action_request(int timeout=-1)
{
    string id, msg;
    ServerMsg_ptr_t req;

    int rc = read_transport(id, msg);
    RET_ON_ERR(rc == 0, "Failed to read msg from engine");


    req = create_server_msg(msg);

    RET_ON_ERR(req->validate(), "req (%s) failed to validate", msg.c_str());

    id = req->get(REQ_CLIENT_NAME);
    RET_ON_ERR(id == s_registered.client_name, "Read id(%s) != client(%s)",
           id.c_str(), s_registered.client_name.c_str());

    return msg.c_str(); 
out:
    return "";
}


int
write_action_response(const char *res)
{
    int rc = -1;
    ServerMsg_ptr_t msg;
    string str_res(res);

    msg = create_server_msg(str_res);
    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", res);

    rc = msg.set(REQ_CLIENT_NAME, s_registered.client_name);
    RET_ON_ERR(rc == 0, "Failed to set client name %s",
            s_registered.client_name.c_str());

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", res);

    rc = write_transport(msg.to_str());
    RET_ON_ERR(rc == 0, "Failed to write action response");
out:
    return rc;
}
 

/*
 * Server side access APIs as per server.h
 */

int
server_init()
{
    log_init("LoM-Engine", LOG_LOCAL0);
    return init_server_transport();
}

void
server_deinit()
{
    deinit_transport();
}

int
write_message(const ActionRequest &req)
{
    int rc = -1;

    RET_ON_ERR(req->validate(), "req (%s) failed to validate",
            req.to_str().c_str());

    rc = write_transport(req.to_str(), req.get(REQ_CLIENT_NAME));
    RET_ON_ERR(rc == 0, "Failed to write to client");
out:
    return rc;
}



ServerMsg_ptr_t
read_message(int timeout=-1)
{
    int rc = 0;
    string client_id, str_msg;
    ServerMsg_ptr_t msg, ret;

    if (timeout != -1) {
        rc = poll_for_data(NULL, 0, timeout);
        if (rc != -1) {
            RET_ON_ERR(rc == -2, "Failed to poll for data from clients");
            rc = LOM_TIMEOUT;
            RET_ON_ERR(false, "Read message timeout.");
            /*  -2 is timeout. Nothing to read */
            goto out;
        }
    }

    rc = read_transport(client_id, str_msg);
    RET_ON_ERR(rc == 0, "Failed to read from transport");

    msg = create_server_msg(str_msg);

    RET_ON_ERR(msg->validate(), "req (%s) failed to validate", str_msg.c_str());

    ret = msg;
out:
    return ret;
}

