// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.28.2
// source: core/rawdb/db.proto

package rawdb

import (
	common "github.com/dominant-strategies/go-quai/common"
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

type ProtoNumber struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Number uint64 `protobuf:"varint,1,opt,name=number,proto3" json:"number,omitempty"`
}

func (x *ProtoNumber) Reset() {
	*x = ProtoNumber{}
	if protoimpl.UnsafeEnabled {
		mi := &file_core_rawdb_db_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtoNumber) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtoNumber) ProtoMessage() {}

func (x *ProtoNumber) ProtoReflect() protoreflect.Message {
	mi := &file_core_rawdb_db_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtoNumber.ProtoReflect.Descriptor instead.
func (*ProtoNumber) Descriptor() ([]byte, []int) {
	return file_core_rawdb_db_proto_rawDescGZIP(), []int{0}
}

func (x *ProtoNumber) GetNumber() uint64 {
	if x != nil {
		return x.Number
	}
	return 0
}

type ProtoLegacyTxLookupEntry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hash       *common.ProtoHash `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	BlockIndex uint64            `protobuf:"varint,2,opt,name=block_index,json=blockIndex,proto3" json:"block_index,omitempty"`
	Index      uint64            `protobuf:"varint,3,opt,name=index,proto3" json:"index,omitempty"`
}

func (x *ProtoLegacyTxLookupEntry) Reset() {
	*x = ProtoLegacyTxLookupEntry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_core_rawdb_db_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtoLegacyTxLookupEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtoLegacyTxLookupEntry) ProtoMessage() {}

func (x *ProtoLegacyTxLookupEntry) ProtoReflect() protoreflect.Message {
	mi := &file_core_rawdb_db_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtoLegacyTxLookupEntry.ProtoReflect.Descriptor instead.
func (*ProtoLegacyTxLookupEntry) Descriptor() ([]byte, []int) {
	return file_core_rawdb_db_proto_rawDescGZIP(), []int{1}
}

func (x *ProtoLegacyTxLookupEntry) GetHash() *common.ProtoHash {
	if x != nil {
		return x.Hash
	}
	return nil
}

func (x *ProtoLegacyTxLookupEntry) GetBlockIndex() uint64 {
	if x != nil {
		return x.BlockIndex
	}
	return 0
}

func (x *ProtoLegacyTxLookupEntry) GetIndex() uint64 {
	if x != nil {
		return x.Index
	}
	return 0
}

var File_core_rawdb_db_proto protoreflect.FileDescriptor

var file_core_rawdb_db_proto_rawDesc = []byte{
	0x0a, 0x13, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x72, 0x61, 0x77, 0x64, 0x62, 0x2f, 0x64, 0x62, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x64, 0x62, 0x1a, 0x19, 0x63, 0x6f, 0x6d, 0x6d, 0x6f,
	0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x25, 0x0a, 0x0b, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x4e, 0x75, 0x6d,
	0x62, 0x65, 0x72, 0x12, 0x16, 0x0a, 0x06, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x06, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x22, 0x78, 0x0a, 0x18, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x4c, 0x65, 0x67, 0x61, 0x63, 0x79, 0x54, 0x78, 0x4c, 0x6f, 0x6f, 0x6b,
	0x75, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x25, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x48, 0x61, 0x73, 0x68, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x1f,
	0x0a, 0x0b, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x04, 0x52, 0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12,
	0x14, 0x0a, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05,
	0x69, 0x6e, 0x64, 0x65, 0x78, 0x42, 0x33, 0x5a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6f, 0x6d, 0x69, 0x6e, 0x61, 0x6e, 0x74, 0x2d, 0x73, 0x74, 0x72,
	0x61, 0x74, 0x65, 0x67, 0x69, 0x65, 0x73, 0x2f, 0x67, 0x6f, 0x2d, 0x71, 0x75, 0x61, 0x69, 0x2f,
	0x63, 0x6f, 0x72, 0x65, 0x2f, 0x72, 0x61, 0x77, 0x64, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_core_rawdb_db_proto_rawDescOnce sync.Once
	file_core_rawdb_db_proto_rawDescData = file_core_rawdb_db_proto_rawDesc
)

func file_core_rawdb_db_proto_rawDescGZIP() []byte {
	file_core_rawdb_db_proto_rawDescOnce.Do(func() {
		file_core_rawdb_db_proto_rawDescData = protoimpl.X.CompressGZIP(file_core_rawdb_db_proto_rawDescData)
	})
	return file_core_rawdb_db_proto_rawDescData
}

var file_core_rawdb_db_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_core_rawdb_db_proto_goTypes = []any{
	(*ProtoNumber)(nil),              // 0: db.ProtoNumber
	(*ProtoLegacyTxLookupEntry)(nil), // 1: db.ProtoLegacyTxLookupEntry
	(*common.ProtoHash)(nil),         // 2: common.ProtoHash
}
var file_core_rawdb_db_proto_depIdxs = []int32{
	2, // 0: db.ProtoLegacyTxLookupEntry.hash:type_name -> common.ProtoHash
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_core_rawdb_db_proto_init() }
func file_core_rawdb_db_proto_init() {
	if File_core_rawdb_db_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_core_rawdb_db_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*ProtoNumber); i {
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
		file_core_rawdb_db_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*ProtoLegacyTxLookupEntry); i {
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
			RawDescriptor: file_core_rawdb_db_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_core_rawdb_db_proto_goTypes,
		DependencyIndexes: file_core_rawdb_db_proto_depIdxs,
		MessageInfos:      file_core_rawdb_db_proto_msgTypes,
	}.Build()
	File_core_rawdb_db_proto = out.File
	file_core_rawdb_db_proto_rawDesc = nil
	file_core_rawdb_db_proto_goTypes = nil
	file_core_rawdb_db_proto_depIdxs = nil
}
