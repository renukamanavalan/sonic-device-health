/*
 * common APIs used by events code.
 */
#include <stdio.h>
#include <string>
#include <memory>
#include <string.h>
#include <vector>

class server_transport
{
    public:
        virtual ~server_transport() {};

        virtual int write(const std::string client, const std::string msg) = 0;

        virtual int read(std::string &client_id, std::string &msg, int timeout = -1) = 0;
};

class client_transport {
    public:
        virtual ~client_transport() {};

        virtual int write(const std::string msg) = 0;

        virtual int read(std::string &msg, int timeout = -1) = 0;

        virtual int get_read_fd() = 0;
};


typedef std::shared_ptr<client_transport> client_transport_ptr_t;
typedef std::shared_ptr<server_transport> server_transport_ptr_t;

client_transport_ptr_t init_client_transport(const std::string client_name);
server_transport_ptr_t init_server_transport(const std::vector<std::string> &clients);

