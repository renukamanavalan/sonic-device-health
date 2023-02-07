#ifndef _COMMON_H_
#define _COMMON_H_

#include <string>
#include <errno.h>
#include <stdarg.h>
#include <syslog.h>
#include <thread>
#include <map>
#include <nlohmann/json.hpp>

extern int errno;

#define LOM_LOG_ERROR(msg, ...) clog_write(LOG_ERR, __FUNCTION__, msg, ##__VA_ARGS__)
#define LOM_LOG_INFO(msg, ...) clog_write(LOG_INFO, __FUNCTION__, msg, ##__VA_ARGS__)
#define LOM_LOG_DEBUG(msg, ...) clog_write(LOG_DEBUG, __FUNCTION__, msg, ##__VA_ARGS__)

#define RET_ON_ERR(res, msg, ...)                                                       \
    if (!(res)) {                                                                       \
        int _e = errno;                                                                 \
        set_last_error(__FILE__, __LINE__, __FUNCTION__, _e, rc, msg, ##__VA_ARGS__);   \
        if (rc == 0) {                                                                  \
            rc = -1;                                                                    \
        }                                                                               \
        goto out; }

#define DROP_TEST(msg, ...)

std::string get_thread_name();

#define XDROP_TEST(msg, ...) {                                                              \
    stringstream _ss;                                                                       \
    _ss << std::this_thread::get_id();                                                      \
    printf("%s:%s::%d------------- DROP: ", get_thread_name().c_str(), __FILE__, __LINE__); \
    printf( msg, ##__VA_ARGS__);                                                            \
    printf("\n"); }

#define ARRAYSIZE(d) (sizeof(d)/sizeof((d)[0]))

/*********************
 * Log helpers       *
 *********************/

void log_init(const char *identifier=NULL, int facility=0);
void log_close();

void clog_write(int loglvl, const char *caller, const char *msg, ...);

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

/*
 * Return JSON object upon parsing file. The object will be empty
 * on failure to parse. Use get_last_error for details.
 */
nlohmann::json parse_json_file(const std::string fpath);

/*
 * Return JSON object upon parsing input string. The object will be empty
 * on failure to parse. Use get_last_error for details.
 */
nlohmann::json parse_json_str(const std::string json_str);


/* Helpers to get data from JSON object */
bool json_has_key(const nlohmann::json &data, const std::string key);

/* For Key value:
 * Explicit type conversion between the JSON value and a compatible value which
 * is CopyConstructible and DefaultConstructible. The value is converted by
 * calling the json_serializer<ValueType> from_json() method.
 * For more detail refer: /usr/include/nlohmann/json.hpp (@brief get a value (explicit))
 *
 * Return:
 *  True - If key exists & conversion succeeds.
 *  False - If either of the above is not true
 */
template <typename T>
bool json_get_val(const nlohmann::json &data, const std::string key, T &val);

/* Get any JSON value as string */
string json_get_as_string(const json &v);


uint64_t get_epoch_secs_now();
uint64_t get_epoch_millisecs_now();

#endif // _COMMON_H_

