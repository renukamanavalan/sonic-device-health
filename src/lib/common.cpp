#include <stdio.h>
#include <stdarg.h>
#include <syslog.h>
#include <sstream>
#include <fstream>
#include "common.h"
#include "client.h"

using namespace std;
using json = nlohmann::json;
using namespace chrono;

#define DEFAULT_IDENTITY "LoM"
#define LOG_FACILITY LOG_LOCAL0

static int s_log_level = LOG_ERR;

static bool s_log_initialized = false;

static bool s_test_mode = false;

void set_log_level(int lvl)
{
    s_log_level = lvl;
}

int get_log_level() { return s_log_level; }

void set_test_mode() { s_test_mode = true; }
bool is_test_mode() { return s_test_mode; }

thread_local string thr_name;

void
set_thread_name(const char *name)
{
    thr_name= string(name);
}

string
get_thread_name()
{
    return thr_name;
}


const char *
_syslog_lvl_to_str(int lvl)
{
    static const char *str_levels[8] = {
        "LOG_EMERG ",
        "LOG_ALERT ",
        "LOG_CRIT ",
        "LOG_ERR ",
        "LOG_WARNING ",
        "LOG_NOTICE ",
        "LOG_INFO ",
        "LOG_DEBUG " };


    return (lvl < (int)ARRAYSIZE(str_levels)) ? str_levels[lvl] : "LOG_UKNOWN ";
}


void
log_init(const char *ident,  int facility)
{

    if (!s_log_initialized) {
        int fac = (((facility >= LOG_LOCAL0) && (facility <= LOG_LOCAL7)) ? 
                facility : LOG_FACILITY);

        openlog(ident == NULL ? DEFAULT_IDENTITY : ident, LOG_PID, fac);
        s_log_initialized = true;
    }
}

void log_write(int lvl, const char *caller, const char *msg)
{
    clog_write(lvl, caller, msg);
}


void
clog_write(int lvl, const char *caller, const char *msg, ...)
{
    if (lvl <= s_log_level) {
        char buf[1024];
        stringstream ss;

        ss << get_thread_name() << ":LOM: "
            << _syslog_lvl_to_str(lvl) << caller << ": " << msg;

        {
        va_list ap;
        va_start(ap, msg);
        vsnprintf(buf, sizeof(buf), ss.str().c_str(), ap);
        va_end(ap);
        }

        buf[sizeof(buf) - 1] = 0;

        syslog(lvl, "%s", buf);
        if (is_test_mode() || (get_log_level() == LOG_DEBUG)) {
            printf("%s\n", buf);
        }
    }
}

class errorMgr {
    public:
        errorMgr() : m_code(0) {};

        void set_error(int code, const string msg) {
            m_code = code;
            m_msg = msg;
        }
        int get_error() { return m_code; }
        string get_error_msg() { return m_msg; }

    private:
        int m_code;
        string m_msg;
};

static errorMgr s_errorMgr;


void
set_last_error(const char *fl, int ln, const char *caller,
        int e, int rc, const char *msg, ...)
{
    char buf[1024];
    string fmt_caller;

    {
        stringstream ss;
        ss << fl << ":" << ln << " " << caller << ":";
        fmt_caller = ss.str();
    }

    {
        va_list ap;
        stringstream ss;

        if (e != 0) {
            ss << "err:" << e << " (" << strerror(e) << ") ";
        }
        ss << "rc:" << rc << " " << msg;

        va_start(ap, msg);
        vsnprintf(buf, sizeof(buf), ss.str().c_str(), ap);
        va_end(ap);

        buf[sizeof(buf) - 1] = 0;
    }

    log_write(LOG_ERR, fmt_caller.c_str(), buf);
    s_errorMgr.set_error(rc, buf);
}


void log_close()
{
    closelog();
    s_log_initialized = false;
}

int lom_get_last_error() { return s_errorMgr.get_error(); }

// const char *lom_get_last_error_msg() { return s_errorMgr.get_error_msg().c_str(); }
// const char *lom_get_last_error_msg() { return "Hello World!"; }
const char *lom_get_last_error_msg()
{
    static string s;
    
    s = s_errorMgr.get_error_msg();
    return s.c_str();
}


/* Map to JSON string and vice versa */
string
convert_to_json(const string key, const map_str_str_t &params)
{
    json msg = json::object();
    json params_data = json::object();

    for (map_str_str_t::const_iterator itc = params.begin();
                itc != params.end(); ++itc) {
        params_data[itc->first] = itc->second;
    }
    msg[key] = params_data;
    return msg.dump();
}

template<typename T>
int
get_params(T& data, map_str_str_t &params, string slice)
{
    int rc = 0;

    for (auto itp = data.cbegin(); itp != data.cend(); ++itp) {
        RET_ON_ERR((*itp).is_string(), "key=(%s); Expect params value as string; type(%s).",
                    itp.key().c_str(), itp.value().type_name());
        params[itp.key()] = itp.value();
    }
out:
#if 0
    Debug code if needed
    if (rc != 0) {
        int i=0;
        for (auto itp = data.cbegin(); itp != data.cend(); ++itp) {
            stringstream ss;
            ss << "*itp=(" << (*itp) << ") type(" << (*itp).type_name() << ") key("
                << itp.key() << ") val (" << itp.value() << ")";
            DROP_TEST("%d: %s", i++, ss.str().c_str());
        }
    }
#endif
    return rc;
}

int
convert_from_json(const string json_str, string &key, map_str_str_t &params)
{
    int rc = 0;
    json data;

    try {
        data = json::parse(json_str);
    } catch (json::parse_error& ex) {
        stringstream ss;
        LOM_LOG_ERROR("Failed to parse (%s)", json_str.c_str());
        ss << ex.byte;
        RET_ON_ERR(false, "Parse failed ex:(%s)", ss.str().c_str())
    }

    if (data.size() == 1) {
        auto it = data.cbegin();
        key = it.key();
        RET_ON_ERR((*it).is_object(), "Invalid json str(%s). Expect object as val",
                    json_str.substr(0, 20).c_str());
        rc = get_params(*it, params, json_str.substr(0, 20));
    } else {
        key = "";
        rc = get_params(data, params, json_str.substr(0, 20));
    }
    RET_ON_ERR(rc == 0, "Failed to get params key=%s", key.c_str());
out:
    return rc;
}


/* JSON helpers */

json
parse_json_file(const std::string fpath)
{
    int rc = 0;

    ifstream ifs(fpath);
    RET_ON_ERR (ifs.is_open(), "Failed to open (%s)", fpath.c_str());

    try {
        return json::parse(ifs);
    } catch (json::parse_error& ex) {
        RET_ON_ERR(false, "Failed to parse %s: %s", fpath.c_str(), ex.what());
    }
out:
    return json();
}


/*
 * Return JSON object upon parsing input string. The object will be empty
 * on failure to parse. Use get_last_error for details.
 */
json
parse_json_str(const std::string json_str)
{
    int rc = 0;

    try {
        return json::parse(json_str);
    } catch (json::parse_error& ex) {
        RET_ON_ERR(false, "Failed to parse %s: %s",
                    json_str.substr(0, 20).c_str(), ex.what());
    }
out:
    return json();
}


/* Helpers to get data from JSON object */
bool
json_has_key(const json &data, const string key)
{
    return data.find(key) != data.end() ? true : false;
}


/* For Key value:
 * Explicit type conversion between the JSON value and a compatible value which
 * is CopyConstructible and DefaultConstructible. The value is converted by
 * calling the json_serializer<ValueType> from_json() method.
 * For more detail refer: /usr/include/nlohmann/json.hpp (@brief get a value (explicit))
 *
 * ref: https://json.nlohmann.me/api/basic_json/type/
 * Available types:
 *  Value type                  return value
 *  ----------------------------------------
 *   null                       value_t::null
 *   boolean                    value_t::boolean
 *   string                     value_t::string
 *   number (integer)           value_t::number_integer
 *   number (unsigned integer)  value_t::number_unsigned
 *   number (floating-point)    value_t::number_float
 *   object                     value_t::object
 *   array                      value_t::array  returne as vector<...>
 *    binary                     value_t::binary
 *
 * e.g.     result = d.value(key, vector<string>());
 *
 * Return:
 *  True - If key exists & conversion succeeds.
 *  False - If either of the above is not true
 */
template <typename T>
T json_get_val(const nlohmann::json &data, const std::string key,
        T&& defval)
{
    return data.value(key, defval);
}


string
json_get_as_string(const json &v)
{
    return v.is_string() ? v.get<string>() : v.dump();
}


uint64_t get_epoch_secs_now()
{
    return duration_cast<seconds>(system_clock::now().time_since_epoch()).count();
}

uint64_t get_epoch_millisecs_now()
{
    return duration_cast<milliseconds>(system_clock::now().time_since_epoch()).count();
}

