// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: torrent_state.proto

package torrent_state

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

type Control_Action int32

const (
	Control_START Control_Action = 0
	Control_STOP  Control_Action = 1
)

// Enum value maps for Control_Action.
var (
	Control_Action_name = map[int32]string{
		0: "START",
		1: "STOP",
	}
	Control_Action_value = map[string]int32{
		"START": 0,
		"STOP":  1,
	}
)

func (x Control_Action) Enum() *Control_Action {
	p := new(Control_Action)
	*p = x
	return p
}

func (x Control_Action) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Control_Action) Descriptor() protoreflect.EnumDescriptor {
	return file_torrent_state_proto_enumTypes[0].Descriptor()
}

func (Control_Action) Type() protoreflect.EnumType {
	return &file_torrent_state_proto_enumTypes[0]
}

func (x Control_Action) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Control_Action.Descriptor instead.
func (Control_Action) EnumDescriptor() ([]byte, []int) {
	return file_torrent_state_proto_rawDescGZIP(), []int{1, 0}
}

type Subscription struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Infohash string `protobuf:"bytes,1,opt,name=infohash,proto3" json:"infohash,omitempty"`
}

func (x *Subscription) Reset() {
	*x = Subscription{}
	if protoimpl.UnsafeEnabled {
		mi := &file_torrent_state_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Subscription) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Subscription) ProtoMessage() {}

func (x *Subscription) ProtoReflect() protoreflect.Message {
	mi := &file_torrent_state_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Subscription.ProtoReflect.Descriptor instead.
func (*Subscription) Descriptor() ([]byte, []int) {
	return file_torrent_state_proto_rawDescGZIP(), []int{0}
}

func (x *Subscription) GetInfohash() string {
	if x != nil {
		return x.Infohash
	}
	return ""
}

type Control struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Action   Control_Action `protobuf:"varint,1,opt,name=action,proto3,enum=main.Control_Action" json:"action,omitempty"`
	Infohash string         `protobuf:"bytes,2,opt,name=infohash,proto3" json:"infohash,omitempty"`
}

func (x *Control) Reset() {
	*x = Control{}
	if protoimpl.UnsafeEnabled {
		mi := &file_torrent_state_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Control) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Control) ProtoMessage() {}

func (x *Control) ProtoReflect() protoreflect.Message {
	mi := &file_torrent_state_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Control.ProtoReflect.Descriptor instead.
func (*Control) Descriptor() ([]byte, []int) {
	return file_torrent_state_proto_rawDescGZIP(), []int{1}
}

func (x *Control) GetAction() Control_Action {
	if x != nil {
		return x.Action
	}
	return Control_START
}

func (x *Control) GetInfohash() string {
	if x != nil {
		return x.Infohash
	}
	return ""
}

type TorrentInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Path     string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	Infohash string `protobuf:"bytes,2,opt,name=infohash,proto3" json:"infohash,omitempty"`
}

func (x *TorrentInfo) Reset() {
	*x = TorrentInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_torrent_state_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TorrentInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TorrentInfo) ProtoMessage() {}

func (x *TorrentInfo) ProtoReflect() protoreflect.Message {
	mi := &file_torrent_state_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TorrentInfo.ProtoReflect.Descriptor instead.
func (*TorrentInfo) Descriptor() ([]byte, []int) {
	return file_torrent_state_proto_rawDescGZIP(), []int{2}
}

func (x *TorrentInfo) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *TorrentInfo) GetInfohash() string {
	if x != nil {
		return x.Infohash
	}
	return ""
}

type SingleTorrentState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SingleTorrentState) Reset() {
	*x = SingleTorrentState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_torrent_state_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SingleTorrentState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SingleTorrentState) ProtoMessage() {}

func (x *SingleTorrentState) ProtoReflect() protoreflect.Message {
	mi := &file_torrent_state_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SingleTorrentState.ProtoReflect.Descriptor instead.
func (*SingleTorrentState) Descriptor() ([]byte, []int) {
	return file_torrent_state_proto_rawDescGZIP(), []int{3}
}

var File_torrent_state_proto protoreflect.FileDescriptor

var file_torrent_state_proto_rawDesc = []byte{
	0x0a, 0x13, 0x74, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x6d, 0x61, 0x69, 0x6e, 0x22, 0x2a, 0x0a, 0x0c, 0x53,
	0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x69,
	0x6e, 0x66, 0x6f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x69,
	0x6e, 0x66, 0x6f, 0x68, 0x61, 0x73, 0x68, 0x22, 0x72, 0x0a, 0x07, 0x43, 0x6f, 0x6e, 0x74, 0x72,
	0x6f, 0x6c, 0x12, 0x2c, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x14, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f,
	0x6c, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x1a, 0x0a, 0x08, 0x69, 0x6e, 0x66, 0x6f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x69, 0x6e, 0x66, 0x6f, 0x68, 0x61, 0x73, 0x68, 0x22, 0x1d, 0x0a, 0x06,
	0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x09, 0x0a, 0x05, 0x53, 0x54, 0x41, 0x52, 0x54, 0x10,
	0x00, 0x12, 0x08, 0x0a, 0x04, 0x53, 0x54, 0x4f, 0x50, 0x10, 0x01, 0x22, 0x3d, 0x0a, 0x0b, 0x54,
	0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x1a,
	0x0a, 0x08, 0x69, 0x6e, 0x66, 0x6f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x69, 0x6e, 0x66, 0x6f, 0x68, 0x61, 0x73, 0x68, 0x22, 0x14, 0x0a, 0x12, 0x53, 0x69,
	0x6e, 0x67, 0x6c, 0x65, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65,
	0x32, 0x81, 0x02, 0x0a, 0x0e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x54, 0x6f, 0x72, 0x72,
	0x65, 0x6e, 0x74, 0x12, 0x39, 0x0a, 0x0a, 0x41, 0x64, 0x64, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e,
	0x74, 0x12, 0x11, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74,
	0x49, 0x6e, 0x66, 0x6f, 0x1a, 0x18, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x53, 0x69, 0x6e, 0x67,
	0x6c, 0x65, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3c,
	0x0a, 0x0d, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x12,
	0x11, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x49, 0x6e,
	0x66, 0x6f, 0x1a, 0x18, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65,
	0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x39, 0x0a, 0x0e,
	0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x12, 0x0d,
	0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x1a, 0x18, 0x2e,
	0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65, 0x54, 0x6f, 0x72, 0x72, 0x65,
	0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3b, 0x0a, 0x09, 0x53, 0x75, 0x62, 0x73, 0x63,
	0x72, 0x69, 0x62, 0x65, 0x12, 0x12, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e, 0x53, 0x75, 0x62, 0x73,
	0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x1a, 0x18, 0x2e, 0x6d, 0x61, 0x69, 0x6e, 0x2e,
	0x53, 0x69, 0x6e, 0x67, 0x6c, 0x65, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61,
	0x74, 0x65, 0x30, 0x01, 0x42, 0x10, 0x5a, 0x0e, 0x74, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_torrent_state_proto_rawDescOnce sync.Once
	file_torrent_state_proto_rawDescData = file_torrent_state_proto_rawDesc
)

func file_torrent_state_proto_rawDescGZIP() []byte {
	file_torrent_state_proto_rawDescOnce.Do(func() {
		file_torrent_state_proto_rawDescData = protoimpl.X.CompressGZIP(file_torrent_state_proto_rawDescData)
	})
	return file_torrent_state_proto_rawDescData
}

var file_torrent_state_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_torrent_state_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_torrent_state_proto_goTypes = []interface{}{
	(Control_Action)(0),        // 0: main.Control.Action
	(*Subscription)(nil),       // 1: main.Subscription
	(*Control)(nil),            // 2: main.Control
	(*TorrentInfo)(nil),        // 3: main.TorrentInfo
	(*SingleTorrentState)(nil), // 4: main.SingleTorrentState
}
var file_torrent_state_proto_depIdxs = []int32{
	0, // 0: main.Control.action:type_name -> main.Control.Action
	3, // 1: main.ControlTorrent.AddTorrent:input_type -> main.TorrentInfo
	3, // 2: main.ControlTorrent.RemoveTorrent:input_type -> main.TorrentInfo
	2, // 3: main.ControlTorrent.ControlTorrent:input_type -> main.Control
	1, // 4: main.ControlTorrent.Subscribe:input_type -> main.Subscription
	4, // 5: main.ControlTorrent.AddTorrent:output_type -> main.SingleTorrentState
	4, // 6: main.ControlTorrent.RemoveTorrent:output_type -> main.SingleTorrentState
	4, // 7: main.ControlTorrent.ControlTorrent:output_type -> main.SingleTorrentState
	4, // 8: main.ControlTorrent.Subscribe:output_type -> main.SingleTorrentState
	5, // [5:9] is the sub-list for method output_type
	1, // [1:5] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_torrent_state_proto_init() }
func file_torrent_state_proto_init() {
	if File_torrent_state_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_torrent_state_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Subscription); i {
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
		file_torrent_state_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Control); i {
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
		file_torrent_state_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TorrentInfo); i {
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
		file_torrent_state_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SingleTorrentState); i {
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
			RawDescriptor: file_torrent_state_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_torrent_state_proto_goTypes,
		DependencyIndexes: file_torrent_state_proto_depIdxs,
		EnumInfos:         file_torrent_state_proto_enumTypes,
		MessageInfos:      file_torrent_state_proto_msgTypes,
	}.Build()
	File_torrent_state_proto = out.File
	file_torrent_state_proto_rawDesc = nil
	file_torrent_state_proto_goTypes = nil
	file_torrent_state_proto_depIdxs = nil
}
