/*
 *  The engine code.
 *
 *  This manages the entire system of plugins per actions & bindings.conf
 *  
 *  The server waits for client & action registrations.
 *
 *  Raises ActionHandler / AnomalyActionHandler based on the action is anomaly
 *  detector or not. 
 *  Ths distinction of anomaly detector or not is based on bindings.conf
 *  All keys in this conf are anomaly actions and the value is array of
 *  action names to be invoked as sequence upon anomaly detection.
 *
 *  Anomaly detectors self raise requests and request other non-anomaly
 *  handlers to raise requests on demand.
 *
 *  Each handler takes care of publishing data from the plugin.
 *  Anomaly handler handles additionally the binding sequence and in-progress
 *  and completion state publishing for this anomaly.
 *
 *  Heartbeats received are recorded by each handler. The engine collectively
 *  send periodic heartbeats with list of action names that sent heartbeat
 *  since last heartbeat published by engine.
 *
 *  The engine core
 *      Creates action handlers on registration
 *          Creates as Anomaly handler, if it is key in bindings.conf
 *      Remove handlers on client de-registration.
 *      Watches for responses from clients and pass the same to appropriate 
 *      handler. 
 *  
 *  Anomaly handler acquires lock before starting mitigation sequence.
 *  Pending handlers wait for resume
 *  Engine checks on each pending periodically as they may timeout
 *  Enging uses pending list from LockMgr
 */
#include <csignal>
#include "engine.h"


#define ENV_CONFIG_FILE_PATH "LOM_CONFIG_PATH"
#define DEFAULT_CONFIG_PATH "/usr/shared/LoM/config"

#define GLOBALS_CONF_FILENAME "lom.rc.json"
#define ACTIONS_CONF_FILENAME "actions.conf.json"
#define BINDINGS_CONF_FILENAME "bindings.conf.json"
#define PROCS_CONF_FILENAME "procs.conf.json"

#define DEFAULT_MITIGATION_TIMEOUT_SECS 120  /* 2 mins */
#define DEFAULT_ACTION_TIMEOUT_SECS 60  /* 1 mins */
#define MAX_ACTION_TIMEOUT_SECS 300  /* 5 mins */

static json s_actions_conf;

map<action_name_t, anomaly_action_lst_t> bindings_t;
static bindings_t s_bindings;

vector<client_name_t> clients_lst_t;
static clients_lst_t s_clients;

static json global_rc_data;


/* Any code may set, if it needs engine to exit */
bool fatal_failure = false;

volatile int signal_raised = 0;

/* Singletons */

static ActionManagerPtr_t s_action_manager;
static lockMgrPtr_t s_lock_manager;

ActionManagerPtr_t get_action_manager() { return s_action_manager; };

lockMgrPtr_t get_lock_mgr() { return s_lock_manager; };



void signal_handler(int signal)
{
    signal_raised = signal;
}

/*
 * Loads all config files.
 * Any failure here is fatal that engine exits.
 */
int
load_config()
{
    int rc = 0;
    string path(getenv(ENV_CONFIG_FILE_PATH));

    if (path.empty()) {
        path = DEFAULT_CONFIG_PATH;
    }

    if (path.back() != '/') {
        path += "/";
    }

    {
        /* Read actions conf */
        string fl(path + ACTIONS_CONF_FILENAME);
        s_actions_conf = parse_json_file(fl);
        /*
         * Expect an entry per each action. Even in the absence of any config
         * done by user, the actions.conf must exist with default values
         * as per YANG schema DB/DB-actions.yang - LOM_ACTIONS
         *
         * Any action registered w/o conf is rejected.
         */
        RET_ON_ERR(!s_actions_conf.empty(), "Failed to read actions.config (%s)", fl.c_str());
    }

    {
        /* Read bindings into map<anomaly_action, <ordered list of followups > */
        /*
         * sample:
         * {
         *      "action-0" { "0": "action-1", "1": "action-2", "4": "action-4" },
         *      "action-1" { "0": "action-6", "1": "action-2", "3": "action-5" }
         * }
         *
         * refer "./DB/DB-Actions.yang"
         */
        string fl = path + BINDINGS_CONF_FILENAME;
        json data = parse_json_file(fl);
        for (auto it_data = data.cbegin(); it_data != data.cend(); ++it_data) {
            string anomaly(it_data.key());
            json b = (*it_data);

            /* collect in map<key, action-value> */
            map<int, string> tmp;
            for (auto itb = b.cbegin(); itb != b.cend(); ++itb) {
                action = (*itb).get<string>();
                RET_ON_ERR(!action.empty(), "%s: Empty action for seq (%s)",
                        anomaly.c_str(), itb.key().c_str());

                tmp[stoi(itb.key())] = action;
            }

            /* Walking collected map will go sorted to get order right */
            vector<string> ordered_actions;
            for (map<int, string>::const_iterator itc_tmp = tmp.begin();
                    /* Map walks keys as sorted low to high */
                    itc_tmp != tmp.end(); ++itc_tmp) {
                ordered_actions.push_back(itc_tmp->second);
            }
            s_bindings[anomaly].swap(ordered_actions);
        }
        /*
         * Bindings drive the engine as start all keys as anomaly actions
         * and use their value for mitigation sequence.
         * If empty, no anomalies and hence nothing todo. Bail out.
         */
        RET_ON_ERR(!s_bindings.empty(), "Failed to read bindings.config (%s)", fl.c_str());
    }

    {
        /* Read only client names from procs.conf */
        string fl = path + PROCS_CONF_FILENAME;
        json data = parse_json_file(fl);
        for (auto it = data.cbegin(); it != data.cend(); ++it) {
            /* Maintain the order */
            s_clients.push_back(it.key());
        }

        /*
         * Proc keys are name of clients. We need that to setup transport.
         * Only clients known to serve can connect.
         */
        RET_ON_ERR(!s_clients.empty(), "Failed to read bindings.config (%s)", fl.c_str());
    }
    
    {
        /* Read global config. */
        string fl = path + GLOBALS_CONF_FILENAME;
        global_rc_data = parse_json_file(fl);

        RET_ON_ERR(!global_rc_data.empty(), "Failed to read global RC (%s)", fl.c_str());
    }
out:
    return rc;
}

/* Helper for Emum to Str for logging purpose */
const char *req_state_str(req_state_t t)
{
    static const char *s_lst[REQ_STATE_COUNT] = {
        "REQ_NONE",
        "REQ_ACTIVE",
        "REQ_TIMEDOUT",
        "REQ_LOCK_PENDING",
        "REQ_MITIGATING" };
    return t < REQ_STATE_COUNT ? s_lst[t] : "REQ_STATE_UNKNOWN";
}   

/*
 * Generic Action Handler used by all actions including anomalies.
 * For anomaly actions, it has its own class AnomalyActionHandler which
 * derives from this. The AnomalyActionHandler overrides as needed.
 *
 * ActionHandler is fed with its name, its client name & its conf.
 * It keeps some running data as needed.
 */
ActionHandler::ActionHandler(const string client_name, const string action_name,
        json action_conf) :
    m_client_name(client_name), m_action_name(action_name), m_req_state(REQ_NONE),
    m_req_timeout(0), m_hb_touch(0)
{
    m_action_conf.update(action_conf);
    
    /* Pre-read & cache for runtime use */
    m_conf_timeout = m_action_conf.value(ACTION_CONF_TIMEOUT,
            DEFAULT_ACTION_TIMEOUT_SECS);
    if (m_conf_timeout > MAX_ACTION_TIMEOUT_SECS) {
        log_error("%s: Timeout (%d) defaulted to Max(%d)",
                m_conf_timeout, MAX_ACTION_TIMEOUT_SECS);
        m_conf_timeout = MAX_ACTION_TIMEOUT_SECS;
    }
    m_conf_mimic = m_action_conf.value(ACTION_CONF_MIMIC, false);
    m_conf_mandatory = m_action_conf.value(ACTION_CONF_MANDATORY, false);
    m_min_recur = m_action_conf.value(ACTION_CONF_MIN_RECUR, 0);
}


/*
 * Shared code by anomaly & non-anomaly actions
 * For anomalies, it is self raised on startup  & end of mitigation 
 * sequence. 
 * For non anomalies it is raised by ANomaly handler during mitigation
 * sequence.
 */
int
ActionHandler::raise_request(const anomaly_action_name_t anomaly_name,
        const instance_id_t anomaly_id, const anomaly_key_t key,
        const string context, int last_result_code)
{
    int rc = 0;
    
    ServerMsg_ptr_t msg(new ActionRequest);

    RET_ON_ERR(is_enabled(anomaly_name, key, last_result_code != 0),
            "%s:Action not enabled for %s/%s", m_action_name.c_str(),
            anomaly_name.c_str(), key.c_str());

    rc = msg->set({
            { REQ_CLIENT_NAME, client_name() },
            { REQ_ACTION_NAME, action_name() },
            { REQ_ACTION_TYPE, REQ_ACTION_TYPE_ACTION },
            { REQ_INSTANCE_ID, instance_id(true) },
            { REQ_ANOMALY_INSTANCE_ID, anomaly_id },
            { REQ_ANOMALY_KEY, key },
            { REQ_CONTEXT, context },
            { REQ_TIMEOUT, m_conf_timeout }});

    RET_ON_ERR(rc == 0, "%s: Failed to create request message for %s",
            m_action_name.c_str(), anomaly_name.c_str());

    RET_ON_ERR(msg->validate(), "%s: Failed to validate (%s)",
            m_action_name.c_str(), msg->to_str().c_str());

    rc = write_server_message(msg);
    RET_ON_ERR(rc == 0, "%s: Failed to write message for %s",
            m_action_name.c_str(), anomaly_name.c_str());
    
    /* Set state & save anomaly name along with expiry time. */
    m_req_state = REQ_ACTIVE;
    m_request = msg;
    m_anomaly_name = anomaly_name;
    if (m_conf_timeout > 0) {
        m_req_timeout = get_epoch_secs_now() + m_conf_timeout;
        get_action_manager()->register_call_after(m_action_name, m_req_timeout);
    }
    _add_instance(anomaly_name, key);
out:
    return rc;
}


void
ActionHandler::_add_instance(const anomaly_action_name_t anomaly_name,
        const anomaly_key_t key)
{
    action_key_intances_t tmp;
    tmp.swap(m_instances);

    epoch_t t_min = get_epoch_secs_now() - m_min_recur;

    for(action_key_intances_t::iterator it_a = tmp.begin();
            it_a != m_instances.end(); ++it_a) {
        for (key_epoch_instance_t::iterator it_k = it_a->second.begin();
                it_k != it_a->second.end(); ++it_k) {
            if (it_k->second > t_min) {
                /* Take those that are still valid */
                m_instances[it_a->first][it_k->first] = itk->second;
            }
        }
    }
    m_instances[anomaly_name][key] = get_epoch_secs_now();
}

bool
ActionHandler::_validate_timeout(bool is_anomaly)
{
    int rc = 0;

    /* Expect non zero timeout. As non-anomaly handlers are resumed for timeout only.*/
    RET_ON_ERR(m_req_timeout > 0, "%s: Handler has no timeout.", m_action_name.c_str());

    /*
     * Called from member that is overridden.
     * To ensure expected class method is called, let the caller specify
     * the type and match it with current.
     */
    RET_ON_ERR(is_anomaly() == is_anomaly,
            "%s: Code for incorrect handler is_anomaly(%d)",
            m_action_name.c_str(), is_anomaly());

    /* Only possibility is timeout */
    RET_ON_ERR((m_req_state != REQ_NONE) && (m_req_state != REQ_TIMEDOUT),
            "%s:Action state (%d) is incorrect for timeout handler",
            m_action_name.c_str(), req_state_str(m_req_state));

    epoch_secs_t tnow = get_epoch_secs_now();
    if (tnow < m_req_timeout) {
        get_action_manager()->register_call_after(m_action_name, m_req_timeout);
        RET_ON_ERR(false, "%s:Not yet expired tnow-expiry=%d",
                (int)(m_req_timeout-tnow));
    }
    m_req_timeout = 0;
    m_req_state = REQ_TIMEDOUT;
out:
    ret rc == 0 ? true : false;
}


/*
 * Used only for timeout. Anomaly handler overrides and handle lock availability
 * Validate the call. 
 * Inform the corresponding anomaly handler that timed out.
 * Set req state as timedout, so when delayed response arrives, it need not
 * be reported to Anomaly handler.
 */
int
ActionHandler::check_timeout()
{
    int rc = 0;
    bool clear_state = false;
    ActionHandlerPtr_t handler = get_action_manager()->get_handler(m_anomaly_name);
    ServerMsg_ptr_t msg(new ActionResponse);

    RET_ON_ERR (_validate_timeout(false), "%s: Internal error: unexpected check_timeout",
            m_action_name.c_str());

    /* timeout is valid; Clear the state */
    clear_state = true;

    rc = msg->set({
            { REQ_CLIENT_NAME, client_name() },
            { REQ_ACTION_NAME, action_name() },
            { REQ_INSTANCE_ID, instance_id(false) },
            { REQ_ANOMALY_INSTANCE_ID, m_request->get(REQ_ANOMALY_INSTANCE_ID) },
            { REQ_ANOMALY_KEY, m_request->get(REQ_ANOMALY_KEY) },
            { REQ_ACTION_DATA, "{}" },
            { REQ_RESULT_CODE, to_string(ETIME) },
            { REQ_RESULT_STR, "Action timedout"},
            { REQ_TIMEOUT, m_conf_timeout }});

    RET_ON_ERR(rc == 0, "%s: Failed to create timeout response message (%s)",
            m_action_name.c_str(), msg->to_str().c_str());

    RET_ON_ERR(msg->validate(), "%s: Failed to validate timeout message (%s)",
            m_action_name.c_str(), msg->to_str().c_str());

    rc = do_publish(msg);
    if (rc != 0) {
        log_error("%s: Failed to publish (%s)", m_action_name.c_str(),
                msg->to_str().sub_str(0, 80).c_str());
    }

    /* Inform the anomaly handler */
    RET_ON_ERR (handler != NULL, "%s: Failed to find handler for (%s)",
                m_action_name.c_str(), m_anomaly_name.c_str());
            
    rc = handler->process_response(m_action_name, msg);
    RET_ON_ERR("%s: Anomaly handler (%s) failed to process timeout response",
            m_action_name.c_str(), m_request->action_name().c_str());
out:
    if (clear_state) {
        /*
         * Timeout could be because the client exited, in which case no
         * response can be expected. Hence clear the state.
         * If there would be any response later, it will be dropped as 
         * expected instance id is cleared.
         */
        process_sequence_complete();
    }
    return rc;
}


/*
 * Non-anomaly actions only. Called from main loop upon response.
 * Validate the response.
 * Publish it
 * Inform anomaly handler, if req not timed out.
 * Reset req state to None and  de-register any callback for timeout.
 */
int
ActionHandler::process_response(ServerMsg_ptr_t msg)
{
    int rc = 0;
    bool clear_state = false;
    ActionHandlerPtr_t handler = get_action_manager()->get_handler(m_anomaly_name);

    /* Publish Response even if timed out.*/
    rc = do_publish(msg);
    if (rc != 0) {
        log_error("%s: Failed to publish (%s)", m_action_name.c_str(),
                msg->to_str().sub_str(0, 80).c_str());
    }

    RET_ON_ERR(m_req_state == REQ_ACTIVE, "%s: Unexpected req state (%s)", 
             m_action_name.c_str(), req_state_str(m_req_state));
    
    RET_ON_ERR(m_request != NULL, "%s: Expected non null request", 
            m_action_name.c_str());

    RET_ON_ERR(!is_anomaly(), "%s: Code for non-anomaly only", m_action_name.c_str());

    RET_ON_ERR(msg->get(REQ_INSTANCE_ID) == m_instance_id, "%s instance %s != %s",
            m_action_name.c_str(), msg->get(REQ_INSTANCE_ID).c_str(),
            m_instance_id.c_str());

    /* req state & cached instance id match; Hence correct response. */
    clear_state = true;

    /* Inform the anomaly handler */
    RET_ON_ERR (handler != NULL, "%s: Failed to find handler for (%s)",
                m_action_name.c_str(), m_anomaly_name.c_str());

    rc = handler->process_response(m_action_name, msg);
    RET_ON_ERR("%s: Anomaly handler (%s) failed to process res",
        m_action_name.c_str(), m_anomaly_name.c_str());
out:
    if (clear_state) {
        process_sequence_complete();
    }
}


/*
 * Is this handler ready to take a request.
 * Used by all actions (implying anomaly & non-anomaly).
 */
bool
ActionHandler::is_enabled(const anomaly_action_name_t name, const anomaly_key_t key,
        bool is_failed)
{
    bool ret = false;

    RET_ON_ERR(m_req_state == REQ_NONE, "%s:Action state (%d) != NONE",
            m_action_name.c_str(), req_state_str(m_req_state));

    RET_ON_ERR(!is_failed || m_conf_mandatory, "In failed mode and action not mandatory",
            m_action_name.c_str());

    if (m_min_recur > 0) {
        action_key_intances_t::const_iterator itc_act = m_instances.find(name);
        if (itc_act != m_instances.end()) {
            key_epoch_instance_t::const_iterator itc_key = itc_act->second.find(key);
            if (itc_key != itc_act->second.end()) {
                epoch_t t_inst = itc_key->second;
                epoch_t t_now = get_epoch_secs_now();
                RET_ON_ERR((t_now - t_inst) >= m_min_recur,
                        "%s: Too soon anomaly(%s) key(%s) repeat(%d)", m_action_name.c_str(),
                        name.c_str(), key.c_str(), (int)(t_now - t_inst));
            }
        }
    }
    ret = true;
out:
    return ret;
}


/*
 * Get sequence of binding handlers for bound actions for given anomaly.
 * Even if one handler is not available, it returns empty sequence
 * as the mitigation needs to run either all or none.
 *
 * The first handler is the anomaly handler itself.
 */
static int
get_binding_handlers(const anomaly_action_name_t anomaly_name,
        const anomaly_key_t key, vector<ActionHandlerPtr_t> &ret)
{
    int rc = 0;
    vector<ActionHandlerPtr_t> tmp;
    ActionHandlerPtr_t handler;

    ActionManagerPtr_t mgr = get_action_manager();

    bindings_t::const_iterator itc_b = s_bindings.find(anomaly_name);
    RET_ON_ERR(itc_b != s_bindings.end(), "%s not found in bindings %d",
            anomaly_name.c_str(), (int)s_bindings.size());

    /* push anomaly itsef as first */
    handler = mgr->get_handler(anomaly_name);
    RET_ON_ERR(handler != NULL, "%s: Failed to get handler for (%s)",
            anomaly_name.c_str(), anomaly_name.c_str());

    tmp.push_back(handler);

    const anomaly_action_lst_t &lst_actions = itc_b->second;
    for (anomaly_action_lst_itc itc_a = lst_actions.begin();
            itc_a != lst_actions.end(); ++itc_a) {

        handler = mgr->get_handler(*itc_a);

        RET_ON_ERR(handler != NULL, "%s: Failed to get handler for (%s)",
                anomaly_name.c_str(), (*itc_a).c_str());

        RET_ON_ERR(handler->is_enabled(anomaly_name, key, false),
                "%s: Handler not enabled for (%s)",
                anomaly_name.c_str(), (*itc_a).c_str());

        tmp.push_back(handler); 
    }
    RET_ON_ERR(!tmp.empty(), "%s: Failed to get any handlers", anomaly_name.c_str());

    /* Implies one/more handlers for given action are available */
    ret.swap(tmp);

out:
    return rc;
}


AnomalyActionHandler::AnomalyActionHandler(const string client_name,
        const string action_name, const json &action_conf,
        const vector<string> &binding) :
    ActionHandler(client_name, action_name, action_conf)
{
    int rc = 0;
    m_conf_timeout = 0;         /* Ensure no timeout for Anomaly request */
    m_lstHandlers_index = 0;
    m_lock_acquired = false;
    m_mitigation_failed = false;

    m_mitigation_timeout = m_action_conf.value(ACTION_CONF_MITIGATION_TIMEOUT,
            DEFAULT_MITIGATION_TIMEOUT_SECS);
    m_mitigate_exp = 0;
    m_bindings = binding;
    int rc = _raise_request();
    if (rc != 0) {
        log_error("%s: Failed to initiate request", action_name.c_str());
    }
}

/*
 * Specialized process response upon receiving message from the anomaly plugin
 * as provided by server.
 * 
 * Validates the message first. Any validation failure will drop the message on floor.
 * Validated is published and kicks off mitigation sequence.
 * Acquire lock; If acquired raises request to first action in binding sequence.
 *
 * Publish sets mitigation state to REQ_MITIGATION_STATE_PENDING if lock acquire
 * failed. State to REQ_MITIGATION_STATE_PROG, if request raised to first action
 * in binding sequence. Else set to REQ_MITIGATION_STATE_DONE with failuer code.
 *
 * Register for mitigation timeout callback.
 */
int
AnomalyActionHandler::process_response(ServerMsg_ptr_t msg)
{
    int rc = 0;
    char errMsg[200];
    vector<ActionHandlerPtr_t> handler;
    string anomaly_key;

    RET_ON_ERR(m_req_state == REQ_ACTIVE, "%s: Unexpected req state (%s)", 
             m_action_name.c_str(), req_state_str(m_req_state));
    
    RET_ON_ERR(is_anomaly(), "%s: Code for anomaly only", m_action_name.c_str());

    RET_ON_ERR(!m_lock_acquired, "%s: Internal error. Expect no lock",
            m_action_name.c_str());

    RET_ON_ERR(msg->get(REQ_INSTANCE_ID) == m_instance_id, "%s instance %s != %s",
            m_action_name.c_str(), msg->get(REQ_INSTANCE_ID).c_str(),
            m_instance_id.c_str());

    /* Response arrived. Move to next state depending on binding sequence */
    m_req_state = REQ_NONE;
    RET_ON_ERR(msg->get(REQ_RESULT_CODE) == 0, "%s: Detected anomaly has non zero result=%d",
            m_action_name.c_str(), msg->get(REQ_RESULT_CODE));

    /* Message is validated. Hence handled after here */

    m_context = json::object();
    m_context[m_action_name] = action_data;
    m_req_state = REQ_LOCK_PENDING;
    m_anomaly_resp = msg;
    rc = resume_on_lock();

out:
    return rc;
}


int
AnomalyActionHandler::resume_on_lock()
{
    int rc = 0;
    ServerMsg_ptr_t msg = m_anomaly_resp;
    char errMsg[200];
    bool anomaly_complete = false;

    RET_ON_ERR(msg != NULL, "%s: Internal ERROR: expect non NULL ptr",
            m_action_name.c_str());

    m_lock_acquired = get_lock_mgr()->acquire_lock(m_action_name);
    if (!m_lock_acquired) {
        log_info("%s: Failed to get lock", m_action_name.c_str());
        msg->set(REQ_MITIGATION_STATE, REQ_MITIGATION_STATE_PENDING);
        if (0 != do_publish(msg)) {
            log_error("%s: Failed to publish (%s)", m_action_name.c_str(),
                    msg->to_str().sub_str(0, 80).c_str());
        }
    } else {
        get_binding_handlers(m_action_name, msg->get(REQ_ANOMALY_KEY),
                m_lstHandlers);
        m_lstHandlers_index = 0;
        m_req_state = REQ_MITIGATING;

        AnomalyActionHandler::process_response(m_action_name, m_anomaly_resp);
    }
out:
    return rc;
}


void
AnomalyActionHandler::check_timeout()
{
    int rc = 0;
    ServerMsg_ptr_t msg = m_anomaly_resp;
    char errMsg[200];

    if (_validate_timeout(true)) {
        RET_ON_ERR(msg != NULL, "%s: Internal ERROR: expect non NULL ptr",
                m_action_name.c_str());

    snprintf(errMsg, sizeof(errMsg), "%s:Timed out %d seconds",
            m_action_name.c_str(), m_mitigation_timeout);
    msg->set(REQ_MITIGATION_STATE, REQ_MITIGATION_STATE_DONE);
    msg->set(REQ_RESULT_STR, errMsg);
    m_mitigation_failed = true;

    if (0 != do_publish(msg)) {
        log_error("%s: Failed to publish (%s)", m_action_name.c_str(),
                msg->to_str().sub_str(0, 80).c_str());
    }

    /*
     * Mark it failed, but not complete.
     *
     * Don't complete yet. Wait for outstanding/currently active one
     * to respond. Make sure to call mandatory ones in sequence, if any.
     * Upon sequeunce completio, call it complete.
     * 
     * This is the clean approach and the handlers are now in right
     * state for next sequence.
     *
     * Not to worry about any client crash that means no response ever,
     * as the corresponding action handler does handle with timeout.
     * Hence a response can be expected reliably.
     */
}


/*
 * Response from Action invoked via binding sequence.
 */
int
AnomalyActionHandler::process_response(const action_name_t action_name, ServerMsg_ptr_t msg)
{
    int rc = 0, result_code = 0;
    string result_str;
    ActionHandlerPtr_t handler;
    anomaly_key_t key = msg->get(REQ_ANOMALY_KEY);

    RET_ON_ERR(m_lock_acquired, "%s: Internal error: Lock not present",
            m_action_name.c_str());
    /*
     * Validated response. Just ensure this is the current active one.
     */
    RET_ON_ERR(m_req_state == REQ_MITIGATING, "%s:Request state is not mitigating",
            m_action_name.c_str());

    RET_ON_ERR((m_lstHandlers_index == 0) || (m_lstHandlers_index < m_lstHandlers.size()),
            "%s: Internal Err: Invalid index %d/%d", m_lstHandlers_index,
            (int)m_lstHandlers.size());

    handler = m_lstHandlers[m_lstHandlers_index];

    RET_ON_ERR(action_name == handler->action_name(),
            "%s: Response from unexpected action(%s) != %s", m_action_name.c_str(),
            action_name.c_str(), handler->action_name().c_str());

    RET_ON_ERR(m_anomaly_resp->get(REQ_ANOMALY_KEY) == key,
            "%s: Unexpected anomaly key(%s) != %s", m_action_name.c_str(),
            key.c_str(), m_anomaly_resp->get(REQ_ANOMALY_KEY).c_str());

    {
        string sdata = msg->get(REQ_ACTION_DATA);
        json action_data = sdata.empty() ? json::object() :
            json::parse(msg->get(REQ_ACTION_DATA));
        m_context[action_name] = action_data;
    }

    result_code = stoi(msg->get(REQ_RESULT_CODE));

    if ((result_code != 0) && !m_mitigation_failed) {
        /* Record the first failure */
        char errMsg[200];

        snprintf(errMsg, sizeof(errMsg), "%s:Failed in %s: %s",
                m_action_name.c_str(), action_name.c_str(),
                msg->get(REQ_RESULT_STR).c_str());

        m_anomaly_resp->set(REQ_RESULT_CODE, to_string(result_code));
        m_anomaly_resp->set(REQ_RESULT_STR, errMsg);

        m_mitigation_failed = true;
    }

    /* Move the index */
    ++m_lstHandlers_index;

    if (m_mitigation_failed) {
        for (; m_lstHandlers_index < (int)m_lstHandlers.size(); ++m_lstHandlers_index) {
            if (m_lstHandlers[m_lstHandlers_index]->is_enabled(m_action_name,
                        key, true)) {
                break;
            }
        }
    }
    if (m_lstHandlers_index < (int)m_lstHandlers.size()) {
        int res = m_mitigation_failed ? stoi(m_anomaly_resp->get(REQ_RESULT_CODE)) : 0;
        rc = m_lstHandlers[m_lstHandlers_index]->raise_request(m_action_name,
                m_instance_id, key, m_contex.dump(), res);
        if (!m_mitigation_failed && (rc != 0)) {
            char errMsg[200];

            snprintf(errMsg, sizeof(errMsg), "%s:Failed to raise request to %s",
                    m_action_name.c_str(), 
                    m_lstHandlers[m_lstHandlers_index]->action_name().c_str());
            m_anomaly_resp->set(REQ_RESULT_CODE, to_string(rc));
            m_anomaly_resp->set(REQ_RESULT_STR, errMsg);

            m_mitigation_failed = true;
        }
        if (rc != 0) {
            /* Failure to write req is fatal; Can't proceed with remaining */
            log_error("%s: Failed to write request to %.Abort %d/%d",
                    m_action_name.c_str(),
                    m_lstHandlers[m_lstHandlers_index]->action_name().c_str(),
                    m_lstHandlers_index, (int)m_lstHandlers.size());
            m_lstHandlers_index = (int)m_lstHandlers.size();
        }
    }


    if (m_lstHandlers_index == (int)m_lstHandlers.size()) {
        msg->set(REQ_MITIGATION_STATE, REQ_MITIGATION_STATE_DONE);
        if (!m_mitigation_failed) {
            char errMsg[200];

            snprintf(errMsg, sizeof(errMsg), "%s: Succeeded in mitigation",
                    m_action_name.c_str());
            m_anomaly_resp->set(REQ_RESULT_CODE, "0");
            m_anomaly_resp->set(REQ_RESULT_STR, errMsg);
        }
        if (0 != do_publish(m_anomaly_resp)) {
            log_error("%s: Failed to publish (%s)", m_action_name.c_str(),
                    msg->to_str().sub_str(0, 80).c_str());
        }

        _process_sequence_complete();
    }
out:
    return rc;
}


ActionManager::ActionManager() : m_last_heartbeat(0)
{
    int default = 5;
    m_hb_interval_secs = json_get_val(global_rc_data, "HEARTBEAT_INTERVAL", default);
}


int
ActionManager::register_client(const string cl_name)
{
    int rc = 0;

    rc = deregister_client(cl_name);
    RET_ON_ERR(rc == 0, "Attempt to deregister for sanity failed");

    RET_ON_ERR(m_clients.find(cl_name) == m_clients.end(), 
            "Client pre-exist even after attempt to de-register (%s)",
            cl_name.c_str());

    m_clients[cl_name] = action_name_set_t();
out:
    return rc;
};


int
ActionManager::deregister_client(const string cl_name)
{
    int rc = 0;

    clients_lst_it it_cl = m_clients.find(cl_name);
    if (it_cl != m_clients.end()) {
        const action_name_set_t& lst = it_cl->second;

        for(action_name_set_itc itc_act = lst.begin(); itc_act != lst.end(); ++itc_act) {
            action_handlers_it it_h = m_handlers.find(*itc_act);
            if (it_h == m_handlers.end()) {
                log_error("Internal error: cl(%s) act(%s) missing handler",
                         cl_name.c_str(), (*itc_act).c_str());
            } else {
                /* Delete ActionHandler object */
                m_handlers.erase(it_h);
            }
            m_anomaly_actions.erase(*itc_act);
        }
        m_clients.erase(it_cl);
    }
out:
    return rc;
}


int
ActionManager::register_action(const string cl_name, const string action_name)
{
    int rc = 0;
    ActionHandlerPtr_t handler;
    bool is_anomaly = s_bindings.find(action_name) != s_bindings.end() ? true : false;

    RET_ON_ERR(s_actions_conf.contains(action_name), "%s: Missing actions conf",
            action_name.c_str());
                
    {
        /* verify client exist. Action not already registered for this client */
        clients_lst_itc it_cl = m_clients.find(cl_name);
        RET_ON_ERR (it_cl != m_clients.end(), "%s: Failed to register for missing client (%s)",
                action_name.c_str(), cl_name.c_str());
        action_name_set_t& lst = it_cl->second;

        action_name_set_itc itc_act = lst.find(action_name);
        RET_ON_ERR(itc_act == lst.end(), "%s: Duplicate register by client (%s)",
                action_name.c_str(), cl_name.c_str());
    }

    /* verify this action not registered by any other client */
    handler = get_handler(action_name);
    RET_ON_ERR(handler == NULL, "%s: client:%s already registered by client:%s",
            action_name.c_str(), cl_name.c_str(), itc_h->second->client_name().c_str());
    
    RET_ON_ERR(!s_actions_conf[action_name].value(ACTION_CONF_DISABLE, false),
            "%s: %s is disabled. Skip registering", cl_name.c_str(), action_name.c_str());

    if (is_anomaly) {
        handler.reset(new AnomalyActionHandler(cl_name, action_name, s_actions_conf[action_name],
                    s_bindings[action_name]));
    } else {
        handler.reset(new ActionHandler(cl_name, action_name, s_actions_conf[action_name]));
    }
    RET_ON_ERR(handler != NULL, "Failed to create handler %s/%s",
            cl_name.c_str(), action_name.c_str());

    /* Save the action & handler */
    m_handlers[action_name] = handler;
    m_clients[cl_name].insert(action_name));
    if (is_anomaly) {
        m_anomaly_actions.insert(action_name);
    }
    
out:
    return rc;
}


int
ActionManager::set_heartbeat(const action_name_t name, const instance_id_t id)
{
    int rc = 0;
    ActionHandlerPtr_t handler = get_handler(name);

    RET_ON_ERR(handler != NULL, "%s: Heartbeat: Failed to find handler", name.c_str());

    RET_ON_ERR((rc = handler->touch_heartbeat(id)) == 0, "%s: failed to touch heartbeat",
            name.c_str());
out:
    return rc;
}


int
ActionManager::process_response(const action_name_t name, ServerMsg_ptr_t msg)
{
    int rc = 0;
    ActionHandlerPtr_t handler = get_handler(name);
            
    RET_ON_ERR(handler != NULL, "%s: response: Failed to find handler", name.c_str());

    RET_ON_ERR((rc = handler->process_response(msg)) == 0,
            "%s: failed to process response", name.c_str());

out:
    check_lock_resume();
    return rc;
}

void
ActionManager::_time_check()
{
    epoch_secs_t tnow = get_epoch_secs_now();

    /* collect actions that are due for callback. */
    action_name_set_t lst;
    while (!m_secs_vs_action.empty()) {
        /* Get the earliest time in seconds */
        secs_vs_action_t::iterator it_sa = m_secs_vs_action.begin();
        if (it_sa->first <= tnow) {
            lst.insert(msecs.cbegin()->second.begin(), msecs.cbegin()->end());
            m_secs_vs_action.erase(it_sa);
        } else { 
            break;
        }
    }

    /*
     * Call back the collected.
     *
     * Use local var to callback as member variable
     * could get updated during callback and in a buggy
     * scenario, this could lead to tight loop
     */
    for (action_name_set_itc itc = lst.begin(); itc != lst.end(); ++itc) {
        /* Purge action vs due setting */
        m_action_vs_secs.erase(*itc);
    }

    for (action_name_set_itc itc = lst.begin(); itc != lst.end(); ++itc) {
        string name(*itc);
        ActionHandlerPtr_t handler = get_handler(name);

        if (handler == NULL) {
            log_error("%s: Failed to callback on timer due to missing handler",
                    name.c_str());
            handler->check_timeout();
        }
    }

    /* Send Heartbeat, if due */
    tnow = get_epoch_secs_now();
    if ((tnow - m_last_heartbeat) >= m_hb_interval_secs) {

        json data = json::object();
        json actions = json::array();

        /* Collect actions that have heartbeat since last */
        for (action_handlers_itc itc = m_handlers.begin(); itc != m_handlers.end(); ++itc) {
            if (itc->second > m_last_heartbeat) {
                actions.insert(itc->first);
            }
        }

        data[REQ_TIMESTAMP] = tnow;
        data[REQ_ACTIONS] = actions;    /* Could be empty */

        string s(data.dump());

        int rc = lom_do_publish(s.c_str());
        if (rc != 0) {
            log_error("LoM Engine:: Failed to publish heartbeat %s", s.c_str());
        }
    }
}

int process_message(ServerMsg_ptr_t msg)
{
    int rc = 0;

    switch (msg->get_type()) {
        case REQ_TYPE_REGISTER_CLIENT:
            RET_ON_ERR(rc = s_action_manager->register_client(msg->get(REQ_CLIENT_NAME)) == 0,
                        "Failed to register client");
            break;

        case REQ_TYPE_DEREGISTER_CLIENT:
            RET_ON_ERR(rc = s_action_manager->deregister_client(msg->get(REQ_CLIENT_NAME)) == 0,
                        "Failed to deregister client");
            break;

        case REQ_TYPE_REGISTER_ACTION:
            RET_ON_ERR(rc = s_action_manager->register_action(
                        msg->get(REQ_CLIENT_NAME), msg->get(REQ_ACTION_NAME)) == 0,
                        "Failed to deregister client");

        case REQ_TYPE_HEARTBEAT:
            RET_ON_ERR(rc = s_action_manager->set_heartbeat(msg->get(REQ_ACTION_NAME),
                        msg->get(REQ_INSTANCE_ID)) == 0, "Failed to deregister client");
            break;

        case REQ_TYPE_ACTION_RESPONSE:
            RET_ON_ERR(rc = s_action_manager->process_response(msg->get(REQ_ACTION_NAME),
                        msg) == 0, "Failed to process response");
            break;

        default:
            RET_ON_ERR(false, "Unexpected request type=%d/(%s)",
                    msg->get_type(), msg->get_typ_str());
    }
out:
    return rc;
}


int main_loop()
{
    int rc = 0;

    s_action_manager.reset(new ActionManager());
    s_lock_manager.reset(new lockMgr());

    rc = server_init(s_clients);

    RET_ON_ERR(rc == 0, "Failed to init server Clients(%d)", clients.size());

    while((signal_raised != 0) && !fatal_failure) {
        int timeout = s_action_manager->get_next_time_check();
        ServerMsg_ptr_t msg = read_server_message(timeout);

        if (msg != NULL) {
            rc  = process_message(msg);
            if (rc != 0) {
                /* Don't fail the engine. Log error. */
                log_error("Failed to process msg (%s)",
                        msg->to_str().sub_str(0, 80).c_str());
            }
            s_lock_manager.inform_pending();
        } else {
            log_debug(msg != NULL, "Failed to read message");
        }
        /* Call those who has asked for callback */
        s_action_manager->time_check();
    }

out:
    server_deinit();
    return rc;
}

int main(int argc, char **argv)
{
    int rc = 0;
    signal(SIGHUP, signal_handler);

    RET_ON_ERR(lom_init_publish("LOM"));

    while(!fatal_failure) {
        rc = load_config();
        RET_ON_ERR(rc == 0, "Failed to load config");

        rc = main_loop();
        RET_ON_ERR(rc == 0, "Engine main loop failed. Exiting.");

        if (signal_raised != 0) {
            if (signal_raised != SIGHUP) {
                break;
            }
            log_info("SIGHUP raised; Restarting engine");
            signal_raised = 0;
        }
    }
    log_info("Engine exiting... signal_raised=%d fatal_failure=%d",
        signal_raised, fatal_failure);
out:
    lom_deinit_publish();
    return rc;
}
     




