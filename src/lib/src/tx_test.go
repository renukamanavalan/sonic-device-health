package main

import (
    "errors"
    "fmt"
    "transport"
    "client_transport"
)

type TestClientData struct {
    type    transport.MsgType
    args    []string
    result  client_transport.ServerResult
}

type TestServerData struct {
    transport.Msg
    transport.Reply
}

type TestData struct {
    TestClientData
    TestServerData
}

var testData = [1]TestData {
            {   TestClientData { transport.TypeRegClient, []string{"Foo"}, 
                        client_transport.ServerResult{0, ""}},
                TestServerData { transport.Msg { transport.TypeRegClient, "Foo", ""},
                        transport.Reply { 0, "" } }}}



func testClient(ch chan interface{}, chRes chan int) {
    txClient = &client_transport.ClientTx
    ret := 0

    for i := 0; i < len(testData); i++ {
        tdata := &testData[i]
        var res *client_transport.ServerResult
        var err errors

        switch tdata.type {
        case TypeRegClient:
            res, err = txClient.RegisterClient(tdata.args[0])
        default:
            log.fatal("TODO - Not yet implemented (%d)", tdata.type)
        }

        if (err != nil) {
            ret = -1
            log.Printf("%d: Type(%d) Failed err(%v)", i, tdata.type, err)
        } else if (*res != tdata.result) {
            log.Printf("%d: Type(%d) Failed to match res(%v) != exp(%v)",
                    i, tdata.type, res, tdata.result)
            ret = -2
        }
        if ret {
            ch <- "failed"
            break
        }
    }
    chRes <- ret
}


func main()
{
    ret := 0
    tx, err := transport.ServerInit()
    if err != nil {
        log.Fatal("Failed to init server")
    }
    ch := make(chan interface{})
    chResult := make(chan int)

    go testClient(ch, chResult)

    go func abort(tout int) {
        time.Sleep(tout * time.Second)
        ch <- "Abort"
    }(10)

    for i := 0; i < len(testData); i++ {
        tdata := &testData[i]

        p := tx.ReadFromClient(ch)
        if p == nil {
            log.Printf("ReadFromClient returned nil")
            ret = -1
            break
        }
        if (*p != tdata.TestServerData.Msg) {
            log.Printf("Server: %d: Type(%d) Failed to match msg(%v) != exp(%v)",
                                i, tdata.type, *p, tdata.TestServerData.Msg)
            ret = -1
            break
        }
        msg.ch <- tdata.transport.Reply
    }

    if ret != 0 {
        log.Printf("main/test-server failed")
    } else if ret = <- chResult; ret {
        log.Printf("testClient failed")
    }
    else {
        log.Printf("SUCCEEDED\n")
    }
}

