
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

int init_server_transport(const std::string client_name);


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
int write_message(const std::string message);


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
int read_message(std::string &message);


/*
 * deinit transport
 *
 /
void deinit_transport();


#endif /* !_TRANSPORT_H_ */ 
