#include <stdio.h>
#include <syslog.h>

#define DEFAULT_IDENTITY "LoM"
#define LOG_FACILITY LOG_LOCAL0

static bool s_log_initialized = false;

void log_init(const char *ident,  int facility = LOG_LOCAL0)
{

    if (!s_log_initialized) {
        int fac = (((facility >= LOG_LOCAL0) && (facility <= LOG_LOCAL7)) ? 
                facility : LOG_FACILITY);

        openlog(ident == NULL ? DEFAULT_IDENTITY : ident, fac);
        s_log_initialized = true;
    }
}



void log_write(int lvl, const char *caller, const char *msg, ...)
{
    stringstream ss;
    ss << "LOM: " << caller << ": " << msg;

    va_list ap;
    va_start(ap, msg);
    vsyslog(lvl, ss.str().c_str(), ap);
    va_end(ap);
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
        stirng m_msg;
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

    vsnprintf(buf, ss.str().c_str(), ap);
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
    nlohmann::json msg = nlohmann::json::object();
    nlohmann::json params_data = nlohmann::json::object();

    for (map_str_str_t::const_iterator itc = params.begin();
                itc != params.end(); ++itc) {
        params_data[itc->first] = itc->second;
    }
    msg[key] = params_data;
    return msg.dump();
}

template<typename T>
void
get_params(T& data, map_str_str_t &params, string slice)
{
    for (auto itp = data.cbegin(); itp != data.cend(); ++itp) {
        RET_ON_ERR((*itp).is_string(), "Invalid json str(%s). Expect params value as string",
                    slice.c_str());
        params[itp.key()] = itp.value();
    }
}

int
convert_from_json(const string json_str, string &key, map_str_str_t &params)
{
    int rc = 0;
    const auto &data = nlohmann::json::parse(json_str);

    if (data.size() == 1) {
        auto it = data.cbegin();
        key = it.key();
        RET_ON_ERR((*it).is_object(), "Invalid json str(%s). Expect object as val",
                    json_str.substr(0, 20).c_str());
        get_params(*it, params, json_str.substr(0, 20));
    } else {
        key = "";
        get_params(data, params, json_str.substr(0, 20));
    }
out:
    return rc;
}

