// lom_internal.proto describes the message format used internally by LoM.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.12.4
// source: lom_internal.proto

package gnmi_lom

import (
	gnmi "github.com/openconfig/gnmi/proto/gnmi"
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

// Value is the message that reprents a stream of updates for a given path, used internally.
type Value struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// prefix used with path
	Prefix *gnmi.Path `protobuf:"bytes,1,opt,name=Prefix,proto3" json:"Prefix,omitempty"`
	// The device specific, or path corresponding to a value.
	Path *gnmi.Path `protobuf:"bytes,2,opt,name=Path,proto3" json:"Path,omitempty"`
	// timestamp for the corresponding value, nanoseconds since epoch.
	// If timestamp is not set the default will assume to
	// be the current system time.
	Timestamp int64 `protobuf:"varint,3,opt,name=Timestamp,proto3" json:"Timestamp,omitempty"`
	// The value to be sent to client
	Val *gnmi.TypedValue `protobuf:"bytes,4,opt,name=Val,proto3" json:"Val,omitempty"`
	// Each message sent is sequentially indexed.
	// This is used to track dropped messages within the gNMI server code.
	// The ones sent successfully by server and not received by client is
	// unknown in subscribe mode as communication is one way. But as underlying
	// protocol is TCP, the probability of loss is very small.
	SendIndex int64 `protobuf:"varint,5,opt,name=SendIndex,proto3" json:"SendIndex,omitempty"`
}

func (x *Value) Reset() {
	*x = Value{}
	if protoimpl.UnsafeEnabled {
		mi := &file_lom_internal_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Value) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Value) ProtoMessage() {}

func (x *Value) ProtoReflect() protoreflect.Message {
	mi := &file_lom_internal_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Value.ProtoReflect.Descriptor instead.
func (*Value) Descriptor() ([]byte, []int) {
	return file_lom_internal_proto_rawDescGZIP(), []int{0}
}

func (x *Value) GetPrefix() *gnmi.Path {
	if x != nil {
		return x.Prefix
	}
	return nil
}

func (x *Value) GetPath() *gnmi.Path {
	if x != nil {
		return x.Path
	}
	return nil
}

func (x *Value) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *Value) GetVal() *gnmi.TypedValue {
	if x != nil {
		return x.Val
	}
	return nil
}

func (x *Value) GetSendIndex() int64 {
	if x != nil {
		return x.SendIndex
	}
	return 0
}

var File_lom_internal_proto protoreflect.FileDescriptor

var file_lom_internal_proto_rawDesc = []byte{
	0x0a, 0x12, 0x6c, 0x6f, 0x6d, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x67, 0x6e, 0x6d, 0x69, 0x2e, 0x6c, 0x6f, 0x6d, 0x1a, 0x30,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6f, 0x70, 0x65, 0x6e, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x67, 0x6e, 0x6d, 0x69, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x67, 0x6e, 0x6d, 0x69, 0x2f, 0x67, 0x6e, 0x6d, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xab, 0x01, 0x0a, 0x05, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x22, 0x0a, 0x06, 0x50, 0x72,
	0x65, 0x66, 0x69, 0x78, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x67, 0x6e, 0x6d,
	0x69, 0x2e, 0x50, 0x61, 0x74, 0x68, 0x52, 0x06, 0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x12, 0x1e,
	0x0a, 0x04, 0x50, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x67,
	0x6e, 0x6d, 0x69, 0x2e, 0x50, 0x61, 0x74, 0x68, 0x52, 0x04, 0x50, 0x61, 0x74, 0x68, 0x12, 0x1c,
	0x0a, 0x09, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x09, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x22, 0x0a, 0x03,
	0x56, 0x61, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x67, 0x6e, 0x6d, 0x69,
	0x2e, 0x54, 0x79, 0x70, 0x65, 0x64, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x03, 0x56, 0x61, 0x6c,
	0x12, 0x1c, 0x0a, 0x09, 0x53, 0x65, 0x6e, 0x64, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x09, 0x53, 0x65, 0x6e, 0x64, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x42, 0x0d,
	0x5a, 0x0b, 0x2e, 0x2f, 0x3b, 0x67, 0x6e, 0x6d, 0x69, 0x5f, 0x6c, 0x6f, 0x6d, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_lom_internal_proto_rawDescOnce sync.Once
	file_lom_internal_proto_rawDescData = file_lom_internal_proto_rawDesc
)

func file_lom_internal_proto_rawDescGZIP() []byte {
	file_lom_internal_proto_rawDescOnce.Do(func() {
		file_lom_internal_proto_rawDescData = protoimpl.X.CompressGZIP(file_lom_internal_proto_rawDescData)
	})
	return file_lom_internal_proto_rawDescData
}

var file_lom_internal_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_lom_internal_proto_goTypes = []interface{}{
	(*Value)(nil),           // 0: gnmi.lom.Value
	(*gnmi.Path)(nil),       // 1: gnmi.Path
	(*gnmi.TypedValue)(nil), // 2: gnmi.TypedValue
}
var file_lom_internal_proto_depIdxs = []int32{
	1, // 0: gnmi.lom.Value.Prefix:type_name -> gnmi.Path
	1, // 1: gnmi.lom.Value.Path:type_name -> gnmi.Path
	2, // 2: gnmi.lom.Value.Val:type_name -> gnmi.TypedValue
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_lom_internal_proto_init() }
func file_lom_internal_proto_init() {
	if File_lom_internal_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_lom_internal_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Value); i {
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
			RawDescriptor: file_lom_internal_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_lom_internal_proto_goTypes,
		DependencyIndexes: file_lom_internal_proto_depIdxs,
		MessageInfos:      file_lom_internal_proto_msgTypes,
	}.Build()
	File_lom_internal_proto = out.File
	file_lom_internal_proto_rawDesc = nil
	file_lom_internal_proto_goTypes = nil
	file_lom_internal_proto_depIdxs = nil
}
