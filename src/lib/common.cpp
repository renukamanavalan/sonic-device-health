#include <stdio.h>
#include <syslog.h>

#define DEFAULT_IDENTITY "LoM"
#define LOG_FACILITY LOG_LOCAL0


void log_init(const char *ident,  int facility)
{
    int fac = (((facility >= LOG_LOCAL0) && (facility <= LOG_LOCAL7)) ? 
            facility : LOG_FACILITY);

    openlog(ident == NULL ? DEFAULT_IDENTITY : ident, fac);
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


void log_close()
{
    closelog();
}

#if 0
static struct errcode_val {
    int code;
    const char* msg;
} s_error_codes[] = {
    { LOM_LIB_SUCCESS, "ERR_OK" },
    { LOM_LIB_UNKNOWN, "ERR_UNKNOWN" }
};

class errorMgr {
    public:
        errorMgr() : m_code(0) {
            const struct errcode_val *p = s_error_codes;
            for(int i=0; i<ARRAYSIZE(s_error_codes); ++i, ++p) {
               m_default[p->code] = p->msg;
            }
        }

        void set_error(int code, const string msg) {
            m_code = code;
            if (!msg.empty()) {
                m_msg = msg;
            } else {
                m_msg = m_default[code];
            }
        }
        int get_error() { return m_code; }
        string get_error_msg() { return m_msg; }
        int get_all(string &msg) { msg=m_msg; return m_code; }

    private:
        int m_code;
        stirng m_msg;
        map<int, const string> m_default;
};

static errorMgr s_errorMgr;

void error_init()
{
    for(int i=0; i<

void set_last_error(int err, const string errmsg)
{
    s_errorMgr.set_error(err, errmsg);
}

int get_last_error() { return s_errorMgr.get_error(); }

const char *get_last_error_str() { return s_errorMgr.get_error_msg().c_str(); }
#endif

