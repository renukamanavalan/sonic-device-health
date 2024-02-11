1. ZMQ sockets are not thread safe and hence stays within a single go routine.

2. A single zmq context used across. 
   Terminated only upon system shutdown
3. Socket details saved per channelMode_t as publisher/subscriber/requester/...

4. Sockets created are saved against {mode, chType, requester}. More for tracking to close during termination.

5. zmq_channel:managePublish - creates sock. Run until give i/p data channel closes or system shutdown.
    No duplicate check. In cases of "bind", it could fail on duplicate. But no explicit check.
    Runs as go routine until close.

6. zmq_channel:manageSubscribe - Similar to publish. Get writable data channel and readable ctl channel, in additon to channel for returning init error. When caller closes ctrl chan, it closes underlying socket and closes the sibscription. The go routine exits.

7. zmq_channel:openPubChannel called { chtype, topic & i/p data chan }. Multiple writers can share this data channel.
    Close of this channel will close the underlying ZMQ channel.
    Duplicate openPubChannel is blocked.


8. zmq_channel:openSubChannel is copy of pub in everyway. The caller provides ctl-chan, closingof which closes the subscription.

9. zmq_channel:runPubSubProxyInt & doRunPubSubProxy helps run only single proxy per chType.


client:

1. GetPubChannel == zmq_channel:openPubChannel 
2. GetSubChannel == zmq_channel:openSubChannel 
3. RunPubSubProxy == zmq_channel:doRunPubSubProxy  returns chClose chan<- int to help shutdown proxy.
4. TelemetryServiceInit -- Kick off proxies for events & counters. Call RunPubSubProxy for events & counters.
5. TelemetryServiceShut -- shut all that initialized via TelemetryServiceInit

6. PublishInit & PublishTerminate -- Track all auto created publishers
7. PublishAny -- Auto create as needed & publish.
    PublishEvent & PublishCounters are specific wrappers for PublishAny



