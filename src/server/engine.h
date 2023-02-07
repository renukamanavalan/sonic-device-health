#include <stdio.h>
#include <ctime>
#include <chrono>       // std::chrono::system_clock
#include <nlohmann/json.hpp>
#include <map>
#include <set>
#include <vector>
#include "client.h"
#include "common.h"
#include "consts.h"
#include "server.h"
#include "transport.h"
#include <uuid/uuid.h>


/* This is internal non-shared header files. Hence has using */
using namespace std;
using json = nlohmann::json;

class ActionHandler;
clase AnomalyActionHandler;

typedef shared_ptr<ActionHandler> ActionHandlerPtr_t;

typedef string action_name_t;
typedef string anomaly_action_name_t;
typedef string client_name_t;
typedef string instance_id_t;
typedef string anomaly_key_t;

typedef uint64_t epoch_secs_t;

typedef map<action_name_t, ActionHandlerPtr_t> action_handlers_t;
typedef action_handlers_t::const_iterator action_handlers_itc;
typedef action_handlers_t::iterator action_handlers_it;

typedef set<action_name_t> anomaly_action_name_set_t;
typedef anomaly_action_name_set_t::const_iterator anomaly_action_names_itc;
typedef anomaly_action_name_set_t::iterator anomaly_action_names_it;

typedef vector<action_name_t> anomaly_action_lst_t;
typedef anomaly_action_lst_t::const_iterator anomaly_action_lst_itc;
typedef anomaly_action_lst_t::iterator anomaly_action_lst_it;

typedef set<action_name_t> action_name_set_t;
typedef action_name_set_t::const_iterator action_name_set_itc;
typedef action_name_set_t::iterator action_name_set_it;

/* Holder for anomaly key vs last occurrence time stamp */
typedef map<anomaly_key_t, epoch_secs_t> key_epoch_instance_t;

/* Holder for anomaly key instances per action */
typedef map<anomaly_action_name_t, key_epoch_instance_t> action_key_intances_t;


/*
 * A lock object or instance. An object == active lock.
 * Assigned to an action - Currently only Anomaly handler takes it to
 * ensure only one mitigation sequence is active anytime.
 *
 * It has its expiry time set. Expired timer will make the lock
 * invalid. Yet, no one can take a lock until it is explicitly
 * released by caller. Else you risk a new mitigation sequence
 * initiated while one is still hanging.
 */
class lockInst
{
    public:
        /* Action name holding lock with timeout in milliseconds */
        lockInst(const string action_name, int timeout_ms=-1) :
            m_action_name(action_name), m_expired(false)
        {
            if (timeout_ms != -1) {
                m_lock_exp = get_epoch_millisecs_now() + timeout_ms;
            } else {
                m_lock_exp = 0;
            }
        }

        const string action_name() { return m_action_name; }

        bool is_expired() {
            if (!m_expired && (m_lock_exp != 0)) {
                m_expired = get_epoch_millisecs_now() < m_lock_exp ? true : false;
            }
            return m_expired;
        }
    private:
        string m_action_name;
        uint64_t m_lock_exp;
        bool m_expired;
};

typedef shared_ptr<lockInst> lockInstPtr_t;

class lockMgr
{
    public:
        /*
         * Acquire lock
         * NOTE: Expect engine to run in single thread and hence not handling
         *      multithread scenario.
         *
         * Multiple anomalies can/do run concurrently. But only one mitigation
         * sequence can be active anytime. So a detected anomaly acquires lock
         * before starting mitigation. The acquiring will fail if another action
         * has it, but added to pending list.
         *
         * A lock is held by an action. This has to be an anomaly action only as
         * lock is required for mitigation only.
         *
         * A duplicate lock request by action that is currently holding the lock
         * is no-op.
         *
         * Main loop calls inform_pending as needed.
         *
         * Input:
         *  action - Name of anomaly action.
         *  timeout -
         *      Lock expires after this count of milliseconds. A value
         *      of -1 implies no timeout.
         *
         * Output: None
         *
         * Return:
         *  true - lock acquired
         *  false - if lock is currently held by someone. But added to pending.
         */
        bool acquire_lock(const string action, int timeout_ms=-1) {
            string ct_action = get_ct_lock_action();
            if (!ct_action.empty()) {
                if (ct_action != action) {
                    m_pending.push_back(action);
                } else {
                    /* This action is already holding the lock */
                    log_error("%s: Duplicate lock request", action.c_str());
                }
            } else {
                m_ct_lock = lockInstPtr_t(new lockInst(action, timeout));
            }
            return validate_lock();
        };

        void release_lock(anomaly_action_name_t action) {
            if (action == get_ct_lock_action()) {
                m_ct_lock.reset();
            }
            anomaly_action_lst_it it = m_pending.find(action);
            if (it != m_pending.end()) {
                m_pending.erase(it);
            }
        }

        /* Return action that is holding the lock currently. Empty string if no lock */
        const string get_ct_lock_action() { 
            return validate_lock() ? m_ct_lock->action_name() : string();
        }

        bool validate_lock() {
            bool ret = m_ct_lock != NULL ? true : false;
#if 0
            Maybe not a good idea to release while someone
            think they have it
            if (m_ct_lock != NULL) {
                if (m_ct_lock->is_expired()) {
                    m_ct_lock.reset();
                }
                else {
                    ret = true;
                }
            }
#endif
            return ret;
        }

        void inform_pending() {
            /* loop until one takes it */
            while ((m_ct_lock == NULL) && !m_pending.empty()) {

                /*
                 * read & erase right away to ensure, we erase the one we picked
                 * There is a possibility that resume may add back.
                 */
                string name(*m_pending.cbegin());
                m_pending.erase(m_pending.begin());

                ActionHandlerPtr_t handler = mgr->get_handler(name);
                if (handler != NULL) {
                    handler->resume_on_lock();
                }
            }
        }
    private:

        lockInstPtr_t m_ct_lock;
        anomaly_action_lst_t m_pending;
};

typedef shared_ptr<lockMgr> lockMgrPtr_t;

lockMgrPtr_t get_lock_mgr();


/*
 *  Every action is owned and handled here.
 * 
 *  An action is associated with one client. More than one client is a bug.
 *  An action is provided with its config, which can be used to tweak
 *  its default behavior. BTW, a single client is likely to register
 *  multiple actions.
 *
 *  An action can be anomaly or any.
 *
 *  An anomaly action is a specialized action that handles mitigation.
 *  Anomaly action initiates request to the corresponding plugin at the start
 *  and at the end of a mitigation run.
 *  Upon detection, it raises request to the ActionHandler for the first action
 *  in the mitigation sequence. Upon response from this action-handler, it
 *  move to next and on until the last. Upon response from last one, it
 *  marks the sequence as complete and publish completion of the
 *  anomaly. The sequence could get aborted by timeout or if any action
 *  in sequence fails.
 *
 *  A non anomaly action is a simple one that receives request on demand from
 *  an anomaly action. Upon response, it process & publish it and then call the 
 *  anomaly action which raised the request with the received response. This
 *  helps anomaly action resume with its mitigation sequence.
 *
 *  Note: A safety check or a mitigation plugin could get called from multiple
 *  Anomaly actions. In other words a single action could be part of binding
 *  sequence of multiple anomalies.
 *
 */

/*
 * A series of states an Action handler coudld be at any time.
 * LOCK_PENDING & MITIGATING are anomaly handler specific.
 * Any handler timesout, but wait for response before going idle
 * as we can't issue another request until we get the response.
 */
typedef enum {
    REQ_NONE = 0,       /* No request issued. Action is idle */
    REQ_ACTIVE,         /* Request active - waiting for resp/timeout */
    REQ_TIMEDOUT,       /* Timeout is sent. Still waiting for resp */
    REQ_LOCK_PENDING,   /* Anomaly waiting on lock to start mitigation */
    REQ_MITIGATING,     /* Req under active mitigation */
    REQ_STATE_COUNT
} req_state_t;

const char *req_state_str(req_state_t t);


/*
 * Generic Action Handler used by all actions including anomalies.
 * For anomaly actions, it has its own class AnomalyActionHandler which
 * derives from this. The AnomalyActionHandler overrides as needed.
 *
 * ActionHandler is fed with its name, its client name & its conf.
 * It keeps some running data as needed.
 */
class ActionHandler
{
    public:
        ActionHandler(const string client_name, const string action_name,
                const json &action_conf);

        /* AnomalyActionHandler overrides. Hence false always. */
        virtual bool is_anomaly() { return false; };

        /*
         *  anomaly_action_name -
         *      Anomaly action which is initiating this request.
         *  anomaly_instance_id -
         *      Instance ID of the corresponding detected anomaly that
         *      is triggering this reqiest
         *  anomaly_key - Key of the detected anomaly.
         *  context -
         *      Context from all preceding actions for this anomaly.
         *  last_result_code -
         *      Failure code from preceding actions
         *      Commonly binding is aborted if an action would fail.
         *      But exception can be cleanup actions, which may like
         *      to get called in any case. Info only.
         *          
         */
        virtual int raise_request(const anomaly_action_name_t name,
                const instance_id_t id,
                const anomaly_key_t key,
                const string context,
                int last_result_code);

        /*
         * Message read from client
         * Expect heartbeat or response 
         */
        virtual int process_response(ServerMsg_ptr_t msg);


        /*
         * Response from another handler
         * Valid only for Anomaly action handler which overrides it.
         * Having it here to avoid casting to derived class.
         */
        virtual int process_response(const string action_name,
                ServerMsg_ptr_t msg) {
            int rc;

            RET_ON_ERR(false, "%s: Expected only for Anomaly handlers",
                    m_action_name.c_str());
        out:
            return rc;
        }


        virtual touch_heartbeat(const instance_id_t id)
        {
            int rc = 0;

            RET_ON_ERR(id == m_instance_id, "%s: heartbeat: Id mismatch (%s) / (%s)",
                    m_action_name.c_str(), id.c_str(), m_instance_id.c_str());
            
            m_hb_touch = get_epoch_secs_now();
        out:
            return rc;
        }

        /*
         * Try to resume if it were pending on lock acquire
         */
        virtual int resume_on_lock() {
            int rc = 0;
            RET_ON_ERR(false, "%s:expected only for anomaly handler",
                    m_action_name.c_str());
        out:
            return rc;
        }


        /* Verify timeout */
        virtual int check_timeout();

        void process_sequence_complete()
        {
            m_req_state = REQ_NONE;
            get_action_manager()->deregister_call_after(m_action_name);
            m_request.reset(NULL);
            m_anomaly_name.clear();
            m_instance_id.clear();
            m_req_timeout = 0;
        }

        /*
         * Event publish
         */
        virtual int do_publish(ServerMsg_ptr_t msg) {
            return lom_do_publish(msg->to_str().c_str());
        }

        /*
         * Get status as ready for invoking request or not.
         */
        virtual bool is_enabled(const anomaly_action_name_t name,
                const string anomaly_key, bool is_failed);

        /*
         * Gets
         */
        const string client_name() { return m_client_name; };
        const string action_name() { return m_action_name; };
        const string instance_id(bool is_new=false) {
            if (is_new) {
                char uuid_str[37];
                uuid_t uuid;

                uuid_generate(uuid);
                uuid_unparse_lower(uuid, uuid_str);
                                    
                m_instance_id = string(uuid_str);
            }
            return m_instance_id;
        }

        epoch_secs_t last_heartbeat() { return m_hb_touch; }

    protected:
        void _add_instance(const anomaly_action_name_t anomaly_name,
                const anomaly_key_t key);

        bool _validate_timeout(bool is_anomaly);

    protected:
        string m_client_name;
        string m_action_name;
        json m_action_conf;

        /* Entities that govern from .conf - pre-read */
        int m_conf_timeout;
        bool m_conf_mimic;
        bool m_conf_mandatory;
        int  m_min_recur;

        /*
         * Info related to current ongoing request.
         *
         * As this is not cleared until next request, this could
         * also be the info of last executed request.
         */
        req_state_t m_req_state;

        /*
         * The request raised.
         * This is maintained until request process is complete.
         * For non-anomaly, it is complete upon receiving response.
         * For anomaly, it is complete upon completing mitigation
         * sequence.
         *
         * A non-anomaly action could get invoked by any anomaly
         * action. For anomaly, it is self raised at start and at
         * the end of a mitigating sequence.
         *
         * These variables are cleared at the end of request.
         */
        ServerMsg_ptr_t m_request;
        string m_anomaly_name;  /* Name of anomaly requesting */

        string m_instance_id; /* Instance ID of current request. */
        epoch_secs_t m_req_timeout; /* req is timedout at this time point */

        /*
         * Save last run info to honor min recurrence
         */
        action_key_intances_t m_instances; 

        /* Last heartbeat */
        epoch_secs_t m_hb_touch;
};

clase AnomalyActionHandler : public ActionHandler
{
    public:
        AnomalyActionHandler(const string client_name, const string action_name,
                const json &action_conf, const vector<string> &binding);

        virtual bool is_anomaly() { return true; };

        /*
         * Anomaly handlers raise request by self. Not called from external
         * Block explicitly.
         */
        virtual int raise_request(const anomaly_action_name_t name,
                const instance_id_t id,
                const anomaly_key_t key,
                const string context,
                int last_result_code) {
            int rc = 0;

            RET_ON_ERR(false, "%s:Anomaly action is not called by others",
                    m_action_name.c_str());
        out:
            return rc;
        };

        /*
         * Message read from client
         * Expect response 
         */
        virtual int process_response(ServerMsg_ptr_t msg);

        /*
         * Response from action invoked by this anomaly per binding sequence
         * during mitigation
         */
        virtual int process_response(const string action_name,
                ServerMsg_ptr_t msg);

        /*
         * Try to resume if it were pending on lock acquire
         */
        virtual int resume_on_lock();

        /* Verify timeout */
        virtual int check_timeout();

        void process_sequence_complete()
        {
            ActionHandler::process_sequence_complete();

            get_lock_mgr()->release_lock(m_action_name);
            m_lock_acquired = false;

            m_context = json::object();
            m_mitigation_failed = false;
            m_anomaly_resp.reset(NULL);
            m_mitigate_exp = 0;

            vector<ActionHandlerPtr_t>().swap(m_lstHandlers);
            m_lstHandlers_index = 0;
            
            /* Raise anomaly request */
            _raise_request();
        }

    private:
        virtual int _raise_request()
        {
            /* Raising request for self */
            return ActionHandler::raise_request("", "", "", "{}", 0);
        }

        void _process_timeout();

        /* Member variables */
        int m_mitigation_timeout;   /* total time to mitigate from conf */

        /* Variables associated with current mitigation run */
        bool m_lock_acquired;           /* Is lock acquired */
        json m_context;                 /* Context collected from completed actions */
        bool m_mitigation_failed;       /* One/more actions in sequence failed */
        ServerMsg_ptr_t m_anomaly_resp; /* Copy of anomaly action response */
        epoch_secs_t m_mitigate_exp;    /* timepoint for timeout for this run */

        /*
         *  Handlers are built on each response, as the handlers' state
         *  could change.
         */
        vector<ActionHandlerPtr_t> m_lstHandlers;
        int m_lstHandlers_index;        /* Index of current action */

};


class ActionManager
{
    public:
        ActionManager();

        /*
         * Register new client.
         * If pre-existing, it implies that the client process
         * restarted w/o de-register. So de-register & re-register.
         */
        int register_client(const string cl_name);


        /*
         * Remove all action handlers and client entry.
         * No-op if client don't exist.
         */
        int deregister_client(const string cl_name);


        /*
         * Register action -- creates handler
         * Fails on unknown client or duplication action register.
         * Failure is not fatal to engine.
         */
        int register_action(const string cl_name,
                const string action_name);

        /* set heartbeat from client */
        int set_heartbeat(const action_name_t name, const instance_id_t id);


        /* Process received response */
        int process_response(const action_name_t name, ServerMsg_ptr_t msg);

        /* Assistance to Handlers */
        void deregister_call_after(const action_name_t name)
        {
            action_vs_secs_t::iterator it_as = m_action_vs_secs.find(name);
            if (it_as != m_action_vs_secs.end()) {
                secs_vs_action_t::iterator it_sa = m_secs_vs_action.find(it_as->second);
                if (it_sa != m_secs_vs_action.end()) {
                    it_sa->second.erase(name);
                    if (it_sa->second.empty()) {
                        m_secs_vs_action.erase(it_sa);
                    }
                }
                m_action_vs_secs.erase(it_as);
            }
        }


        /* Assistance to Handlers */
        void register_call_after(const action_name_t name, epoch_secs_t call_at_secs)
        {
            /* Any pre-existing reg can be removed */
            deregister_call_after(name);

            m_action_vs_secs[name] = call_at_secs;
            m_secs_vs_action[call_at_secs].insert(name);
        }


        int get_next_time_check()
        {
            int ret = m_hb_interval;

            if (!m_secs_vs_action.empty()) {
                epoch_secs_t tnow = get_epoch_secs_now();

                epoch_secs_t t_first = m_secs_vs_action.cbegin()->first;
                int diff = t_first > tnow ? (t_first - tnow) : 0;
                if (diff < ret) {
                    ret = diff;
                }
            }
            return ret;
        }


        void time_check()
        {
            if (!m_secs_vs_action.empty()) {
                _time_check();
            }
        }

            
        ActionHandlerPtr_t get_handler(const action_name_t name)
        {
            action_handlers_it it = m_handlers.find(name);
            return it != m_handlers.end() ? it->second : ActionHandlerPtr_t();
        }

    private:

        void _time_check();

        action_handlers_t m_handlers;
        anomaly_action_name_set_t m_anomaly_actions;

        typedef map<client_name_t, action_name_set_t> client_actions_lst_t;
        typedef client_actions_lst_t::const_iterator client_actions_lst_itc;
        typedef client_actions_lst_t::iterator client_actions_lst_it;
        client_actions_lst_t m_clients;


        /* callbacks on time */
        typedef map<epoch_secs_t, action_name_set_t> secs_vs_action_t;
        typedef map<action_name_t, epoch_secs_t> action_vs_secs_t;

        secs_vs_action_t m_secs_vs_action;
        action_vs_secs_t m_action_vs_secs;

        epoch_secs_t m_last_heartbeat;
        int m_hb_interval_secs;
};

typedef shared_ptr<ActionManager> ActionManagerPtr_t;
ActionManagerPtr_t get_action_manager();




