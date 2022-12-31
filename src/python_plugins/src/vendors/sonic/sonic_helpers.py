#! /usr/bin/env python3

import json
import syslog
# from swsscommon.swsscommon import events_init_publisher, event_publish, FieldValueMap
# import common

publisher_handle = None

def publish_init(src:str = "LoM"):
    global publisher_handle

    if not publisher_handle:
        # publisher_handle = events_init_publisher(src)
        publisher_handle = "Initialized"


def publish_event(tag:str, data:{}):
    if not publisher_handle:
        log_error("publisher_handle not availanble. Call publish_init")
        return

    """
    param_dict = FieldValueMap()

    for k, v in data.items():
        if type(v) == dict:
            param_dict[k] = json.dumps(v)
        else:
            param_dict[k] = str(v)

    event_publish(publisher_handle, tag, param_dict)
    """
    log_str = "LoM_PUBLISH:{}:{}".format(tag, json.dumps(data))

    # common.log_info(log_str)
    syslog.syslog(syslog.LOG_INFO, log_str)

    print("********** DROP: HELPERS: {}".format(log_str))

    
def main():
    import time

    publish_init("test-publish")

    for i  in range(10):
        publish_event("hello_"+str(i), {"foo": "bar", "run": i})
        time.sleep(2)

    return


if __name__ == "__main__":
    main()

        
