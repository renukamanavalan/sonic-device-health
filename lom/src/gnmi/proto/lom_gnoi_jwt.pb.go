// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
//     protoc-gen-go v1.26.0
//     protoc        v3.12.4
// source: lom_gnoi_jwt.proto

package gnmi_lom

import (
    protoreflect "google.golang.org/protobuf/reflect/protoreflect"
    protoimpl "google.golang.org/protobuf/runtime/protoimpl"
    reflect "reflect"
    sync "sync"
)

const (
    // Verify that this generated code is sufficiently up-to-date.
    _ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
    // Verify that runtime/protoimpl is sufficiently up-to-date.
    _ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type JwtToken struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    AccessToken string `protobuf:"bytes,1,opt,name=access_token,json=accessToken,proto3" json:"access_token,omitempty"`
    Type        string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
    ExpiresIn   int64  `protobuf:"varint,3,opt,name=expires_in,json=expiresIn,proto3" json:"expires_in,omitempty"`
}

func (x *JwtToken) Reset() {
    *x = JwtToken{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lom_gnoi_jwt_proto_msgTypes[0]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *JwtToken) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*JwtToken) ProtoMessage() {}

func (x *JwtToken) ProtoReflect() protoreflect.Message {
    mi := &file_lom_gnoi_jwt_proto_msgTypes[0]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use JwtToken.ProtoReflect.Descriptor instead.
func (*JwtToken) Descriptor() ([]byte, []int) {
    return file_lom_gnoi_jwt_proto_rawDescGZIP(), []int{0}
}

func (x *JwtToken) GetAccessToken() string {
    if x != nil {
        return x.AccessToken
    }
    return ""
}

func (x *JwtToken) GetType() string {
    if x != nil {
        return x.Type
    }
    return ""
}

func (x *JwtToken) GetExpiresIn() int64 {
    if x != nil {
        return x.ExpiresIn
    }
    return 0
}

type AuthenticateRequest struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    Username string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
    Password string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *AuthenticateRequest) Reset() {
    *x = AuthenticateRequest{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lom_gnoi_jwt_proto_msgTypes[1]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *AuthenticateRequest) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*AuthenticateRequest) ProtoMessage() {}

func (x *AuthenticateRequest) ProtoReflect() protoreflect.Message {
    mi := &file_lom_gnoi_jwt_proto_msgTypes[1]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use AuthenticateRequest.ProtoReflect.Descriptor instead.
func (*AuthenticateRequest) Descriptor() ([]byte, []int) {
    return file_lom_gnoi_jwt_proto_rawDescGZIP(), []int{1}
}

func (x *AuthenticateRequest) GetUsername() string {
    if x != nil {
        return x.Username
    }
    return ""
}

func (x *AuthenticateRequest) GetPassword() string {
    if x != nil {
        return x.Password
    }
    return ""
}

type AuthenticateResponse struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    Token *JwtToken `protobuf:"bytes,1,opt,name=Token,proto3" json:"Token,omitempty"`
}

func (x *AuthenticateResponse) Reset() {
    *x = AuthenticateResponse{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lom_gnoi_jwt_proto_msgTypes[2]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *AuthenticateResponse) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*AuthenticateResponse) ProtoMessage() {}

func (x *AuthenticateResponse) ProtoReflect() protoreflect.Message {
    mi := &file_lom_gnoi_jwt_proto_msgTypes[2]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use AuthenticateResponse.ProtoReflect.Descriptor instead.
func (*AuthenticateResponse) Descriptor() ([]byte, []int) {
    return file_lom_gnoi_jwt_proto_rawDescGZIP(), []int{2}
}

func (x *AuthenticateResponse) GetToken() *JwtToken {
    if x != nil {
        return x.Token
    }
    return nil
}

type RefreshRequest struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields
}

func (x *RefreshRequest) Reset() {
    *x = RefreshRequest{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lom_gnoi_jwt_proto_msgTypes[3]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *RefreshRequest) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*RefreshRequest) ProtoMessage() {}

func (x *RefreshRequest) ProtoReflect() protoreflect.Message {
    mi := &file_lom_gnoi_jwt_proto_msgTypes[3]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use RefreshRequest.ProtoReflect.Descriptor instead.
func (*RefreshRequest) Descriptor() ([]byte, []int) {
    return file_lom_gnoi_jwt_proto_rawDescGZIP(), []int{3}
}

type RefreshResponse struct {
    state         protoimpl.MessageState
    sizeCache     protoimpl.SizeCache
    unknownFields protoimpl.UnknownFields

    Token *JwtToken `protobuf:"bytes,1,opt,name=Token,proto3" json:"Token,omitempty"`
}

func (x *RefreshResponse) Reset() {
    *x = RefreshResponse{}
    if protoimpl.UnsafeEnabled {
        mi := &file_lom_gnoi_jwt_proto_msgTypes[4]
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        ms.StoreMessageInfo(mi)
    }
}

func (x *RefreshResponse) String() string {
    return protoimpl.X.MessageStringOf(x)
}

func (*RefreshResponse) ProtoMessage() {}

func (x *RefreshResponse) ProtoReflect() protoreflect.Message {
    mi := &file_lom_gnoi_jwt_proto_msgTypes[4]
    if protoimpl.UnsafeEnabled && x != nil {
        ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
        if ms.LoadMessageInfo() == nil {
            ms.StoreMessageInfo(mi)
        }
        return ms
    }
    return mi.MessageOf(x)
}

// Deprecated: Use RefreshResponse.ProtoReflect.Descriptor instead.
func (*RefreshResponse) Descriptor() ([]byte, []int) {
    return file_lom_gnoi_jwt_proto_rawDescGZIP(), []int{4}
}

func (x *RefreshResponse) GetToken() *JwtToken {
    if x != nil {
        return x.Token
    }
    return nil
}

var File_lom_gnoi_jwt_proto protoreflect.FileDescriptor

var file_lom_gnoi_jwt_proto_rawDesc = []byte{
    0x0a, 0x12, 0x6c, 0x6f, 0x6d, 0x5f, 0x67, 0x6e, 0x6f, 0x69, 0x5f, 0x6a, 0x77, 0x74, 0x2e, 0x70,
    0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x67, 0x6e, 0x6d, 0x69, 0x2e, 0x6c, 0x6f, 0x6d, 0x22, 0x60,
    0x0a, 0x08, 0x4a, 0x77, 0x74, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x21, 0x0a, 0x0c, 0x61, 0x63,
    0x63, 0x65, 0x73, 0x73, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
    0x52, 0x0b, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x12, 0x0a,
    0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70,
    0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x73, 0x5f, 0x69, 0x6e, 0x18,
    0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x73, 0x49, 0x6e,
    0x22, 0x4d, 0x0a, 0x13, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65,
    0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e,
    0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e,
    0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18,
    0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22,
    0x40, 0x0a, 0x14, 0x41, 0x75, 0x74, 0x68, 0x65, 0x6e, 0x74, 0x69, 0x63, 0x61, 0x74, 0x65, 0x52,
    0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x28, 0x0a, 0x05, 0x54, 0x6f, 0x6b, 0x65, 0x6e,
    0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x67, 0x6e, 0x6d, 0x69, 0x2e, 0x6c, 0x6f,
    0x6d, 0x2e, 0x4a, 0x77, 0x74, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x05, 0x54, 0x6f, 0x6b, 0x65,
    0x6e, 0x22, 0x10, 0x0a, 0x0e, 0x52, 0x65, 0x66, 0x72, 0x65, 0x73, 0x68, 0x52, 0x65, 0x71, 0x75,
    0x65, 0x73, 0x74, 0x22, 0x3b, 0x0a, 0x0f, 0x52, 0x65, 0x66, 0x72, 0x65, 0x73, 0x68, 0x52, 0x65,
    0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x28, 0x0a, 0x05, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x18,
    0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x67, 0x6e, 0x6d, 0x69, 0x2e, 0x6c, 0x6f, 0x6d,
    0x2e, 0x4a, 0x77, 0x74, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x05, 0x54, 0x6f, 0x6b, 0x65, 0x6e,
    0x42, 0x0d, 0x5a, 0x0b, 0x2e, 0x2f, 0x3b, 0x67, 0x6e, 0x6d, 0x69, 0x5f, 0x6c, 0x6f, 0x6d, 0x62,
    0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
    file_lom_gnoi_jwt_proto_rawDescOnce sync.Once
    file_lom_gnoi_jwt_proto_rawDescData = file_lom_gnoi_jwt_proto_rawDesc
)

func file_lom_gnoi_jwt_proto_rawDescGZIP() []byte {
    file_lom_gnoi_jwt_proto_rawDescOnce.Do(func() {
        file_lom_gnoi_jwt_proto_rawDescData = protoimpl.X.CompressGZIP(file_lom_gnoi_jwt_proto_rawDescData)
    })
    return file_lom_gnoi_jwt_proto_rawDescData
}

var file_lom_gnoi_jwt_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_lom_gnoi_jwt_proto_goTypes = []interface{}{
    (*JwtToken)(nil),             // 0: gnmi.lom.JwtToken
    (*AuthenticateRequest)(nil),  // 1: gnmi.lom.AuthenticateRequest
    (*AuthenticateResponse)(nil), // 2: gnmi.lom.AuthenticateResponse
    (*RefreshRequest)(nil),       // 3: gnmi.lom.RefreshRequest
    (*RefreshResponse)(nil),      // 4: gnmi.lom.RefreshResponse
}
var file_lom_gnoi_jwt_proto_depIdxs = []int32{
    0, // 0: gnmi.lom.AuthenticateResponse.Token:type_name -> gnmi.lom.JwtToken
    0, // 1: gnmi.lom.RefreshResponse.Token:type_name -> gnmi.lom.JwtToken
    2, // [2:2] is the sub-list for method output_type
    2, // [2:2] is the sub-list for method input_type
    2, // [2:2] is the sub-list for extension type_name
    2, // [2:2] is the sub-list for extension extendee
    0, // [0:2] is the sub-list for field type_name
}

func init() { file_lom_gnoi_jwt_proto_init() }
func file_lom_gnoi_jwt_proto_init() {
    if File_lom_gnoi_jwt_proto != nil {
        return
    }
    if !protoimpl.UnsafeEnabled {
        file_lom_gnoi_jwt_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*JwtToken); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
        file_lom_gnoi_jwt_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*AuthenticateRequest); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
        file_lom_gnoi_jwt_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*AuthenticateResponse); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
        file_lom_gnoi_jwt_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*RefreshRequest); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
        file_lom_gnoi_jwt_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
            switch v := v.(*RefreshResponse); i {
            case 0:
                return &v.state
            case 1:
                return &v.sizeCache
            case 2:
                return &v.unknownFields
            default:
                return nil
            }
        }
    }
    type x struct{}
    out := protoimpl.TypeBuilder{
        File: protoimpl.DescBuilder{
            GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
            RawDescriptor: file_lom_gnoi_jwt_proto_rawDesc,
            NumEnums:      0,
            NumMessages:   5,
            NumExtensions: 0,
            NumServices:   0,
        },
        GoTypes:           file_lom_gnoi_jwt_proto_goTypes,
        DependencyIndexes: file_lom_gnoi_jwt_proto_depIdxs,
        MessageInfos:      file_lom_gnoi_jwt_proto_msgTypes,
    }.Build()
    File_lom_gnoi_jwt_proto = out.File
    file_lom_gnoi_jwt_proto_rawDesc = nil
    file_lom_gnoi_jwt_proto_goTypes = nil
    file_lom_gnoi_jwt_proto_depIdxs = nil
}