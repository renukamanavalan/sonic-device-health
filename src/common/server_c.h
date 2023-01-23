#ifndef _SERVER_C_H_
#define _SERVER_C_H_

/* C-APIs for server side APIs */

#ifdef __cplusplus
extern "C" {
#endif


/* APIs for use by server/engine */

/*
 * Required as the first call before using any other APIs
 * Array of client names as array of char pointers.
 */
int server_init_c(const char *clients[], int cnt);


/* Helps release all resources before exit */
void server_deinit();

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
int write_server_message_c(const char *msg);


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
 *  != NULL - Pointer to message as JSON string.
 *      pointer is valid until next read.
 *  == NULL - Failure
 */

const char *read_server_message_c(int timeout=-1);

#ifdef __cplusplus
}
#endif
#endif  // _SERVER_C_H_
