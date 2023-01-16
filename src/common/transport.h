/*
 * common APIs used by events code.
 */
#include <stdio.h>
#include <string>
#include <memory>
#include <string.h>

class transport
{
    public:
        virtual ~transport() {};

        virtual bool is_valid() = 0;

        virtual int set_mode(const std::string client_name = std::string()) = 0;

        virtual int write(const std::string msg, const std::string dest = std::string()) = 0;

        virtual int read(std::string &client_id, std::string &msg, bool dont_wait = false) = 0;

        virtual int poll_for_data(int *lst_fds=NULL, int cnt=0, int timeout=-1) = 0;
};

typedef std::shared_ptr<transport> transport_ptr_t;

transport_ptr_t init_transport(const std::string client_name = std::string(),
        int timeout = -1);

