#include <stdlib.h>
#include <swss/events.h>
#include <nlohmann/json.hpp>
#include "helper.h"

using namespace std;
using json = nlohmann::json;

#define RET_ON_ERR(res, msg, ...)   \
    if (!(res)) {                   \
        printf( msg, ##__VA_ARGS__);\
        rc = -1;                    \
        goto out; }

static event_handle_t s_pub_handle;

int
lom_init_publish(const char *csource)
{
    int rc = 0;

    RET_ON_ERR((csource != NULL) && (strlen(csource) != 0), "Expect non empty source")

    if (s_pub_handle == NULL) {
        string source(csource);
        s_pub_handle = events_init_publisher(source);
    }

    RET_ON_ERR(s_pub_handle != NULL, "Failed to create handle");
out:
    return rc;
}


int
lom_do_publish(const char *sdata)
{
    int rc = 0;
    event_params_t params;
    json data = json::parse(sdata);
    string tag=data.cbegin().key();
    json val = data[tag];

    RET_ON_ERR(s_pub_handle != NULL, "Require non null publish handle");

    RET_ON_ERR(val.is_object(), "data[%s] is not object type(%s)",
            tag.c_str(), val.type_name());

    for (auto it = val.cbegin(); it != val.cend(); ++it) {
        params[it.key()] = (*it).get<string>();
    }

    rc = event_publish(s_pub_handle, tag, &params);
    RET_ON_ERR(rc == 0, "Failed to publish (%s)", sdata);

    printf("Published for tag=%s\n", tag.c_str());

out:
    return rc;
}

void
lom_deinit_publish()
{
    if (s_pub_handle != NULL) {
        events_deinit_publisher(s_pub_handle);
        s_pub_handle = NULL;
    }
}

int sample_test(const char *src, const char *sdata)
{
    printf("sample_test(%s, %s) called\n", src, sdata);
    int rc = lom_init_publish(src);

    RET_ON_ERR(rc == 0, "Failed lom_init_publish");

    rc = lom_do_publish(sdata);

    RET_ON_ERR(rc == 0, "Failed publish");

    lom_deinit_publish();

    printf("All Good\n");
out:
    lom_deinit_publish();
    return rc;
}


int main(int argc, char **argv)
{
    const char *sample_data = "{ \"TEST_INFO\": { \"Foo\": \"Bar\", \"Hello\" : \"World\"} }";

    return sample_test(argc>1 ? argv[1] : "TEST", argc>2 ? argv[2] : sample_data);
}


