#ifndef _COMMON_H_
define  _COMMON_H_

#include <errno.h>
#include <stdarg.h>

extern int errno;
extern int last_errno;
extern int last_zmq_errno;

#define LOM_LOG_ERROR(msg, ...) log_write(LOG_ERR, __FUNCTION__, msg, ##__VA_ARGS__)
#define LOM_LOG_INFO(msg, ...) log_info(LOG_INFO, __FUNCTION__, msg, ##__VA_ARGS__)
#define LOM_LOG_DEBUG(msg, ...) log_debug(LOG_DEBUG, __FUNCTION__, msg, ##__VA_ARGS__)

#define RET_ON_ERR(res, msg, ...)\
    if (!(res)) {                                                               \
        int _e = errno;                                                         \
        if (rc == 0) {                                                          \
            rc = -1;                                                            \
        }                                                                       \
        set_last_error(__FUNCTION__, _e, zmq_errno(), rc, msg, ##__VA_ARGS__);  \
        goto out; }

#define ARRAYSIZE(d) (sizeof(d)/sizeof((d)[0]))

/*********************
 * Log helpers       *
 *********************/

void log_init(const char *identifier=NULL, int facility=0);
void log_close();

void log_write(int loglvl, const char *caller, const char *msg);

/*********************
 * Error set         *
 *********************/

void set_last_error(const char *caller, int e, int ze, int rc,
                const char *msg, ...);
int get_last_error();
const char *get_last_error_msg();


/* JSON conversion helpers */
typedef map<string, string> map_str_str_t;

std::string convert_to_json(const std::string key, const map_str_str_t &params);
int convert_from_json(const std::string json_str, std::string &key, map_str_str_t &params);

#endif // _COMMON_H_

