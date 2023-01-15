#include <stdio.h>
#include <stdarg.h>
#include <syslog.h>
#include <sstream>
#include <fstream>
#include <nlohmann/json.hpp>
#include "common.h"

using json = nlohmann::json;

#define DEFAULT_IDENTITY "LoM"
#define LOG_FACILITY LOG_LOCAL0

static int s_log_level = LOG_ERR;

static bool s_log_initialized = false;

static bool s_test_mode = false;

using namespace std;


void set_log_level(int lvl)
{
    s_log_level = lvl;
}

int get_log_level() { return s_log_level; }

void set_test_mode() { s_test_mode = true; set_log_level(LOG_DEBUG); }
bool is_test_mode() { return s_test_mode; }

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



void
log_write(int lvl, const char *caller, const char *msg, ...)
{
    if (lvl <= s_log_level) {
        stringstream ss;
        ss << "LOM: " << caller << ": " << msg;

        va_list ap;
        va_start(ap, msg);
        vsyslog(lvl, ss.str().c_str(), ap);
        if (lvl == LOG_DEBUG) {
            /* Print to stdout, debug messages and all if in test mode */
            vprintf(ss.str().c_str(), ap);
        }
        va_end(ap);
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


void set_last_error(const char *caller, int e, int ze, int rc,
        const char *msg, ...)
{
    stringstream ss;
    char buf[1024];

    ss << caller << ":";
    if (e != 0) {
        ss << "err:" << e << " ";
    }

    if (ze != 0) {
        ss << "zerr:" << ze << " ";
    }
    ss << "rc:" << rc << " ";
    ss << msg;
  
    va_list ap;
    va_start(ap, msg);

    vsnprintf(buf, sizeof(buf), ss.str().c_str(), ap);
    va_end(ap);

    syslog(LOG_ERR, buf);
    s_errorMgr.set_error(rc, buf);
}


void log_close()
{
    closelog();
    s_log_initialized = false;
}

int get_last_error() { return s_errorMgr.get_error(); }

const char *get_last_error_msg() { return s_errorMgr.get_error_msg().c_str(); }


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
        RET_ON_ERR((*itp).is_string(), "Invalid json str(%s). Expect params value as string",
                    slice.c_str());
        params[itp.key()] = itp.value();
    }
out:
    return rc;
}

int
convert_from_json(const string json_str, string &key, map_str_str_t &params)
{
    int rc = 0;
    const auto &data = json::parse(json_str);

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

