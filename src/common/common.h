#ifndef _COMMON_H_
#define  _COMMON_H_

#include <string>
#include <errno.h>
#include <stdarg.h>
#include <syslog.h>
#include <map>

void set_test_mode();
bool is_test_mode();

extern int errno;

#define LOM_LOG_ERROR(msg, ...) log_write(LOG_ERR, __FUNCTION__, msg, ##__VA_ARGS__)
#define LOM_LOG_INFO(msg, ...) log_write(LOG_INFO, __FUNCTION__, msg, ##__VA_ARGS__)
#define LOM_LOG_DEBUG(msg, ...) log_write(LOG_DEBUG, __FUNCTION__, msg, ##__VA_ARGS__)

#define RET_ON_ERR(res, msg, ...)\
    if (!(res)) {                                                               \
        int _e = errno;                                                         \
        set_last_error(__FILE__, __LINE__, __FUNCTION__, _e, rc, msg, ##__VA_ARGS__);               \
        if (rc == 0) {                                                          \
            rc = -1;                                                            \
        }                                                                       \
        goto out; }

#define DROP_TEST(msg, ...) {       \
    printf("%s::%d------------- DROP: ", __FILE__, __LINE__);   \
    printf( msg, ##__VA_ARGS__);    \
    printf("\n"); }

#define ARRAYSIZE(d) (sizeof(d)/sizeof((d)[0]))

/*********************
 * Log helpers       *
 *********************/

void set_log_level(int lvl);
int get_log_level();

void log_init(const char *identifier=NULL, int facility=0);
void log_close();

void log_write(int loglvl, const char *caller, const char *msg, ...);

/*********************
 * Error set         *
 *********************/

typedef enum {
    LOM_ERR_OK = 0,
    LOM_TIMEOUT,
    LOM_ERR_UNKNOWN = -1
} error_codes_t;


void set_last_error(const char *fl, int ln, const char *caller,
        int e, int rc, const char *msg, ...);


/* JSON conversion helpers */
typedef std::map<std::string, std::string> map_str_str_t;

std::string convert_to_json(const std::string key, const map_str_str_t &params);
int convert_from_json(const std::string json_str, std::string &key, map_str_str_t &params);

#endif // _COMMON_H_

