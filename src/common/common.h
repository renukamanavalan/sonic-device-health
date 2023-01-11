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
        int _ze = zmq_errno();                                                  \
        string _s;                                                              \
        if ((_e != 0) || (_ze != 0)) {                                          \
            _s = string(msg) + " err=" + stoi(_e) + " zmq_err=" + stoi(_ze);    \
        }                                                                       \
        log_error(__FUNCTION__, _s.empty() ? msg : _s.c_str()), ##__VA_ARGS__); \
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

typedef enum errcodes {
    LOM_LIB_SUCCESS = 0,
    LOM_LIB_UNKNOWN,
    LOM_LIB_COUNT
};

void set_last_error(int err, const string errmsg);

#endif // _COMMON_H_

