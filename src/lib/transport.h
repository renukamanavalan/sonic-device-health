
/* The internal code that caches runtime-IDs could retire upon de-init */
#ifndef _TRANSPORT_H_
#define _TRANSPORT_H_
/*
 * common APIs used for read & write between server & clients.
 */

/*
 * Initialize the transport for the client.
 *
 * All communications sent will be addressed with this client as source
 * Receive only communications meant for this client
 *
 * Input:
 *  client_name - Name of the client
 *
 * Output:
 *  None
 *
 * Return:
 *  0 - for success
 * !0 - Error code
 */

int init_client_transport(const std::string client_name);

/*
 * Initialize the transport for the server.
 *
 * Server writes to any client and read from all.
 *
 * Input:
 *  None
 *
 * Output:
 *  None
 *
 * Return:
 *  0 - for success
 * !0 - Error code
 */

int init_server_transport(void);


/*
 * Writes a string message on the initialized transport
 *
 * Input:
 *  message
 *
 * Output:
 *  None
 *
 * Return:
 *  0 - for success
 * !0 - Error code
 */
int write_transport(const std::string msg, const std::string dest = std::string());


/*
 * Reads a string message on the initialized transport
 *
 * Input:
 *  None
 *
 * Output:
 *  message
 *
 * Return:
 *  0 - for success
 * !0 - Error code
 */
int read_transport(std::string &client_id, std::string &message, bool dont_wait=false);


/*
 * deinit transport
 *
 */
void deinit_transport();


/*
 * Poll for data in zmq read fd & given fds
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
 *  -2 - Timeout
 *  -1 - Message from server/engine
 *  >= 0 -- Fd that has message
 *  <other values> -- undefined.
 */
int poll_for_data(int *lst_fds, int cnt, int timeout);

#endif /* !_TRANSPORT_H_ */ 
