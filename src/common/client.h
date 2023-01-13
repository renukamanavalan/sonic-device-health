#ifndef _CLIENT_H_
#define _CLIENT_H_

/*
 * APIs in use by clients
 *
 * We provide C-binding to facilitate direct use by Go, RUST & Python.
 * We don't need to use SWIG to provide any Go/RUST/Python binding.
 */

#ifdef __cplusplus
extern "C" {
#endif

/*
 * Requests used by clients/plugins to reach server.
 *
 */

/*
 * Get the last error encountered.
 *
 * Input:
 *  None
 *
 * Output:
 *  errcode -- Last returned error code.
 *
 * Return:
 *  last encountered error code
 */
int get_last_error();


/*
 * Get the last error encountered as string.
 *
 * Input:
 *  None
 *
 * Output:
 *  None
 *
 * Return:
 *  Human readable string matching error code.
 */
const char *get_last_error_msg();


/*
 * Register the client
 *
 * A plugin-mgr process acts as a client to Engine
 * The plugin-manager manages one or more plugins.
 * A plugin-mgr could register multiple actions.
 * A plugin-mgr restart is guaranteed to use the same client ID,
 * which can help engine clean old registrations and start new.
 *
 * Input:
 *  client_id -- Name of the client Identifier
 *      A plugin-mgr reuses this ID upon restart.
 *      Engine identifies actions against this ID
 *      to block any duplicate registrations from
 *      different processes.
 *
 * Output:
 *  None
 *
 * Return:
 *  0 for success
 *  !=0 implies error
 */
int register_client(const char *client_id);

/*
 * Register the actions 
 *
 * Expect/Require register_client preceded this call.
 *
 * Input:
 *  action -- Name of the action.
 *
 * Output:
 *  None
 *
 * Return:
 *  0 for success
 *  !=0 implies error
 */
int register_action(const char *action);


/*
 * Deregister the client
 *
 * Input:
 *  client_id - Id used during registration.
 *
 * Output:
 *  None
 *
 * Return:
 *  None.
 *
 */
void deregister_client(void);


/*
 * Heartbeat touch
 *
 * Calls heartbeat touch upon heartbeat touch from an running action.
 *
 * Input:
 *  action_name - Name of the action 
 *
 *  instance-id - ID given in corresponding request.
 *
 * Output:
 *  None
 *
 * Return:
 *  0 for success
 *  !=0 implies error
 *
 */
int touch_heartbeat(const char *action, const char *instance_id);


/*
 * Action request from server
 *
 * A JSON string with message attrs as above for applicable attrs
 * Refer: server.h: ActionRequest:: m_reqd_keys & m_opt_keys 
 *          for complete list of required & optional attrs.
 *
 * CONtEXT is collection of action-data from preceding actions
 * hence, it will be empty for first action in the sequence which
 * is an anomaly action.
 * 
 * CONTEXT
 * The context is formatted as
 * {
 *      <preceding action name> : <JSON string of the action data from that action>
 *      ...
 * } 
 *
 * ANOMALY_INSTANCE_ID == INSTANCE_ID given to anomaly/first action in the sequence.
 * ANOMALY_KEY == as returned by Anomaly action in the response.
 *
 * ANOMALY_KEY can be used to group all anomalies reported for a specific 
 * root cause. For e.g. i/f name is the anomaly key for i/f flap.
 *
 * ANOMALY_KEY + ANOMALY_INSTANCE_ID == can track all for a single instance
 *  
 * Input:
 *  timeout -
 *      0 - No wait, return immediately with or w/o request
 *    > 0 - Max count of seconds to wait for request.
 *    < 0 - Block until a request is read
 *
 * Output:
 *  None
 *
 * Return:
 *  Non-NULL - Request read as JSON string
 *  NULL/empty string - Timeout or internal error. Use get_last_error
 *                      to get the error code.
 *
 */
const char *read_action_request(int timeout=-1);


/*
 * Write Action response
 *
 * A JSON string with message attrs as above for applicable attrs
 * Refer server.h: ActionResponse: m_reqd_keys
 *
 * Action response is expected to have a set of attrs as AttributesNameStr_t
 * as key in the JSON object encoded as string.
 *
 * req_attrs_lst_t
 *
 *  ACTION_DATA - A JSON string. The encoded JSON object is per schema of this
 *                action as returned by the plugin.
 *  RESULT_CODE - The numerical return code, where 0 implies success and anything
 *                else implies failure
 *                action.
 *  RESULT_STR  - The human readable text translation of result-code.
 *
 *  NOTE: RESULT_CODE is expected for anomaly's response too, as that indicates
 *        if the detection code had any internal failure or not. Only for
 *        RESULT_CODE == 0, the action-data is taken/considered for a detected
 *        anomaly.
 *        
 * Input:
 *  res - response being returned.
 *
 * Output:
 *  None
 *
 * Return
 *  0 for sucess
 *  !=0 implies failure
 */

int write_action_response(const char *res);


/*
 *  Poll for request from server/engine and as well
 *  listen for data from any of the fds provided
 *
 * Input:
 *  lst_fds: list of fds to listen for data
 *  cnt: Count of fds in list.
 *  timeout: Count of seconds to wait before calling time out.
 *      0 - Check and return immediately
 *     -1 - Block until data arrives on any one/more.
 *     >0 - Count of seconds to wait.
 *
 * Output:
 *  None
 *
 * Return:
 *  -3 - Failure
 *  -2 - Timeout
 *  -1 - Message from server/engine
 *  >= 0 -- Fd that has message
 *  <other values> -- undefined.
 */
int poll_for_data(int *lst_fds, int cnt, int timeout);



#ifdef __cplusplus
}
#endif

#endif // _CLIENT_H_
