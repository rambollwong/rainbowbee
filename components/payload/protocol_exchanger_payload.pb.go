// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.24.3
// source: protocol_exchanger_payload.proto

package payload

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

type ProtocolExchangerPayload_ProtocolExchangerPayloadType int32

const (
	ProtocolExchangerPayload_PUSH     ProtocolExchangerPayload_ProtocolExchangerPayloadType = 0
	ProtocolExchangerPayload_PUSH_OK  ProtocolExchangerPayload_ProtocolExchangerPayloadType = 1
	ProtocolExchangerPayload_REQUEST  ProtocolExchangerPayload_ProtocolExchangerPayloadType = 2
	ProtocolExchangerPayload_RESPONSE ProtocolExchangerPayload_ProtocolExchangerPayloadType = 3
)

// Enum value maps for ProtocolExchangerPayload_ProtocolExchangerPayloadType.
var (
	ProtocolExchangerPayload_ProtocolExchangerPayloadType_name = map[int32]string{
		0: "PUSH",
		1: "PUSH_OK",
		2: "REQUEST",
		3: "RESPONSE",
	}
	ProtocolExchangerPayload_ProtocolExchangerPayloadType_value = map[string]int32{
		"PUSH":     0,
		"PUSH_OK":  1,
		"REQUEST":  2,
		"RESPONSE": 3,
	}
)

func (x ProtocolExchangerPayload_ProtocolExchangerPayloadType) Enum() *ProtocolExchangerPayload_ProtocolExchangerPayloadType {
	p := new(ProtocolExchangerPayload_ProtocolExchangerPayloadType)
	*p = x
	return p
}

func (x ProtocolExchangerPayload_ProtocolExchangerPayloadType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ProtocolExchangerPayload_ProtocolExchangerPayloadType) Descriptor() protoreflect.EnumDescriptor {
	return file_protocol_exchanger_payload_proto_enumTypes[0].Descriptor()
}

func (ProtocolExchangerPayload_ProtocolExchangerPayloadType) Type() protoreflect.EnumType {
	return &file_protocol_exchanger_payload_proto_enumTypes[0]
}

func (x ProtocolExchangerPayload_ProtocolExchangerPayloadType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ProtocolExchangerPayload_ProtocolExchangerPayloadType.Descriptor instead.
func (ProtocolExchangerPayload_ProtocolExchangerPayloadType) EnumDescriptor() ([]byte, []int) {
	return file_protocol_exchanger_payload_proto_rawDescGZIP(), []int{0, 0}
}

type ProtocolExchangerPayload struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pid         string                                                `protobuf:"bytes,1,opt,name=pid,proto3" json:"pid,omitempty"`
	Protocols   []string                                              `protobuf:"bytes,2,rep,name=protocols,proto3" json:"protocols,omitempty"`
	PayloadType ProtocolExchangerPayload_ProtocolExchangerPayloadType `protobuf:"varint,3,opt,name=payload_type,json=payloadType,proto3,enum=payload.ProtocolExchangerPayload_ProtocolExchangerPayloadType" json:"payload_type,omitempty"`
}

func (x *ProtocolExchangerPayload) Reset() {
	*x = ProtocolExchangerPayload{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protocol_exchanger_payload_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtocolExchangerPayload) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtocolExchangerPayload) ProtoMessage() {}

func (x *ProtocolExchangerPayload) ProtoReflect() protoreflect.Message {
	mi := &file_protocol_exchanger_payload_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtocolExchangerPayload.ProtoReflect.Descriptor instead.
func (*ProtocolExchangerPayload) Descriptor() ([]byte, []int) {
	return file_protocol_exchanger_payload_proto_rawDescGZIP(), []int{0}
}

func (x *ProtocolExchangerPayload) GetPid() string {
	if x != nil {
		return x.Pid
	}
	return ""
}

func (x *ProtocolExchangerPayload) GetProtocols() []string {
	if x != nil {
		return x.Protocols
	}
	return nil
}

func (x *ProtocolExchangerPayload) GetPayloadType() ProtocolExchangerPayload_ProtocolExchangerPayloadType {
	if x != nil {
		return x.PayloadType
	}
	return ProtocolExchangerPayload_PUSH
}

var File_protocol_exchanger_payload_proto protoreflect.FileDescriptor

var file_protocol_exchanger_payload_proto_rawDesc = []byte{
	0x0a, 0x20, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x5f, 0x65, 0x78, 0x63, 0x68, 0x61,
	0x6e, 0x67, 0x65, 0x72, 0x5f, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x22, 0xff, 0x01, 0x0a, 0x18,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x45, 0x78, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65,
	0x72, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x70, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x70, 0x69, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x73, 0x12, 0x61, 0x0a, 0x0c, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x3e,
	0x2e, 0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f,
	0x6c, 0x45, 0x78, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x72, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61,
	0x64, 0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x45, 0x78, 0x63, 0x68, 0x61, 0x6e,
	0x67, 0x65, 0x72, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x52, 0x0b,
	0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x22, 0x50, 0x0a, 0x1c, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x45, 0x78, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x72,
	0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x50,
	0x55, 0x53, 0x48, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x50, 0x55, 0x53, 0x48, 0x5f, 0x4f, 0x4b,
	0x10, 0x01, 0x12, 0x0b, 0x0a, 0x07, 0x52, 0x45, 0x51, 0x55, 0x45, 0x53, 0x54, 0x10, 0x02, 0x12,
	0x0c, 0x0a, 0x08, 0x52, 0x45, 0x53, 0x50, 0x4f, 0x4e, 0x53, 0x45, 0x10, 0x03, 0x42, 0x36, 0x5a,
	0x34, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61, 0x6d, 0x62,
	0x6f, 0x6c, 0x6c, 0x77, 0x6f, 0x6e, 0x67, 0x2f, 0x72, 0x61, 0x69, 0x6e, 0x62, 0x6f, 0x77, 0x62,
	0x65, 0x65, 0x2f, 0x63, 0x6f, 0x6d, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x73, 0x2f, 0x70, 0x61,
	0x79, 0x6c, 0x6f, 0x61, 0x64, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protocol_exchanger_payload_proto_rawDescOnce sync.Once
	file_protocol_exchanger_payload_proto_rawDescData = file_protocol_exchanger_payload_proto_rawDesc
)

func file_protocol_exchanger_payload_proto_rawDescGZIP() []byte {
	file_protocol_exchanger_payload_proto_rawDescOnce.Do(func() {
		file_protocol_exchanger_payload_proto_rawDescData = protoimpl.X.CompressGZIP(file_protocol_exchanger_payload_proto_rawDescData)
	})
	return file_protocol_exchanger_payload_proto_rawDescData
}

var file_protocol_exchanger_payload_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_protocol_exchanger_payload_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_protocol_exchanger_payload_proto_goTypes = []interface{}{
	(ProtocolExchangerPayload_ProtocolExchangerPayloadType)(0), // 0: payload.ProtocolExchangerPayload.ProtocolExchangerPayloadType
	(*ProtocolExchangerPayload)(nil),                           // 1: payload.ProtocolExchangerPayload
}
var file_protocol_exchanger_payload_proto_depIdxs = []int32{
	0, // 0: payload.ProtocolExchangerPayload.payload_type:type_name -> payload.ProtocolExchangerPayload.ProtocolExchangerPayloadType
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_protocol_exchanger_payload_proto_init() }
func file_protocol_exchanger_payload_proto_init() {
	if File_protocol_exchanger_payload_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protocol_exchanger_payload_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProtocolExchangerPayload); i {
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
			RawDescriptor: file_protocol_exchanger_payload_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_protocol_exchanger_payload_proto_goTypes,
		DependencyIndexes: file_protocol_exchanger_payload_proto_depIdxs,
		EnumInfos:         file_protocol_exchanger_payload_proto_enumTypes,
		MessageInfos:      file_protocol_exchanger_payload_proto_msgTypes,
	}.Build()
	File_protocol_exchanger_payload_proto = out.File
	file_protocol_exchanger_payload_proto_rawDesc = nil
	file_protocol_exchanger_payload_proto_goTypes = nil
	file_protocol_exchanger_payload_proto_depIdxs = nil
}
