package lomipc

import (
    "encoding/json"
    . "lib/lomcommon"
)


/*
 * Lib Wrapper for calling the client APIs via JSON string
 *
 * Non go clients, can send the request in the form of JSON string
 * The JSON string should map to appropriate LomRequest
 *
 * The APIs here, try to unmarshal into appropriate struct types 
 * and construct expected LomRequest per request type and call
 * sendToServer. 
 *
 * It marshals the LomResponse as JSON string and send it back to 
 * caller.
 *
 * Reference on RPC: https://pkg.go.dev/net/rpc
 * <copy/paste>
 *      Only methods that satisfy these criteria will be made available for remote access; other methods will be ignored:
 *
 *      the method's type is exported.
 *      the method is exported.
 *      the method has two arguments, both exported (or builtin) types.
 *      the method's second argument is a pointer.
 *      the method has return type error.
 *      In effect, the method must look schematically like
 *
 *      func (t *T) MethodName(argType T1, replyType *T2) error
 *
 *      where T1 and T2 can be marshaled by encoding/gob. These requirements apply even if a 
 *      different codec is used. (In the future, these requirements may soften for custom codecs.)
 *
 *      The method's first argument represents the arguments provided by the caller;
 *      the second argument represents the result parameters to be returned to the caller.
 *      The method's return value, if non-nil, is passed back as a string that the client sees
 *      as if created by errors.New. If an error is returned, the reply parameter will not be
 *      sent back to the client.
 *
 */

/*
 * LoMRPCRequest
 *
 * Simplify / Wrap all lib APIs under this one RPC request.
 * Uses strings only to enable RPC call from any language.
 *
 * Input:
 *  reqJson - LoMRequest encoded as JSON string.
 *
 * Output:
 *  resJson - LoMResponse encoded as JSON string
 *
 * Return
 *  error - Non nil implies failure in RPC execution. No result is expected.
 */
func (tr *LoMTransport) LoMRPCRequest(reqJson *string, resJson *string) error {
    req := &LoMRequest{}

    if err := json.Unmarshal([]byte(*reqJson), &req); err != nil {
        return LogError("Failed to unmarshal to (%T) (%v)", req, err, req)
    }


    if bData, err := json.Marshal(req.ReqData); err != nil {
        return LogError("Failed to re-marshal (%d): req.ReqData (%v)", req.ReqType, req.ReqData)
    } else {
        switch (req.ReqType) {
        case TypeRegClient:
            req.ReqData = MsgRegClient{}

        case TypeDeregClient:
            req.ReqData = MsgDeregClient{}

        case TypeRegAction:
            d := &MsgRegAction{}
            if err := json.Unmarshal(bData, d); err == nil {
                req.ReqData = *d
            }
        case TypeDeregAction:
            d := &MsgDeregAction{}
            if err := json.Unmarshal(bData, d); err == nil {
                req.ReqData = *d
            }

        case TypeRecvServerRequest:
            req.ReqData = MsgRecvServerRequest{}

        case TypeSendServerResponse:
            d := &MsgSendServerResponse{}
            if err := json.Unmarshal(bData, d); err == nil {
                if d.ReqType == TypeServerRequestAction {
                    if rData, err := json.Marshal(d.ResData); err == nil {
                        rd := &ActionResponseData{}
                        if err := json.Unmarshal(rData, rd); err == nil {
                            d.ResData = *rd
                        }
                    }
                }
                req.ReqData = *d
            }

        case TypeNotifyActionHeartbeat:
            d := &MsgNotifyHeartbeat{}
            if err := json.Unmarshal(bData, d); err == nil {
                req.ReqData = *d
            }

        default:
            return LogError("Failed: Unknown ReqType(%d)", req.ReqType)
        }
    }

    res := &LoMResponse{}
    if err := tr.SendToServer(req, res); err != nil {
        return LogError("Failed to process (%s) err(%v)", ReqTypeToStr[req.ReqType], err)
    }

    if bData, err := json.Marshal(res); err != nil {
        return LogError("Failed to marshal for (%s) (%v) (%v)", ReqTypeToStr[req.ReqType],
                    err, res)
    } else {
        *resJson = string(bData)
    }
    return nil
}


