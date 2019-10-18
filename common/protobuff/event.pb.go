// Code generated by protoc-gen-go. DO NOT EDIT.
// source: event.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Event_EventType int32

const (
	Event_RESERVED      Event_EventType = 0
	Event_BLOCK_ADDED   Event_EventType = 1
	Event_EPOCH_STARTED Event_EventType = 2
	Event_VIEW_CHANGED  Event_EventType = 3
	Event_COMMITTED     Event_EventType = 4
	Event_ACCOUNT       Event_EventType = 5
	Event_BLOCK         Event_EventType = 6
)

var Event_EventType_name = map[int32]string{
	0: "RESERVED",
	1: "BLOCK_ADDED",
	2: "EPOCH_STARTED",
	3: "VIEW_CHANGED",
	4: "COMMITTED",
	5: "ACCOUNT",
	6: "BLOCK",
}

var Event_EventType_value = map[string]int32{
	"RESERVED":      0,
	"BLOCK_ADDED":   1,
	"EPOCH_STARTED": 2,
	"VIEW_CHANGED":  3,
	"COMMITTED":     4,
	"ACCOUNT":       5,
	"BLOCK":         6,
}

func (x Event_EventType) String() string {
	return proto.EnumName(Event_EventType_name, int32(x))
}

func (Event_EventType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{0, 0}
}

type Request_RequestType int32

const (
	Request_ACCOUNT Request_RequestType = 0
	Request_BLOCK   Request_RequestType = 1
)

var Request_RequestType_name = map[int32]string{
	0: "ACCOUNT",
	1: "BLOCK",
}

var Request_RequestType_value = map[string]int32{
	"ACCOUNT": 0,
	"BLOCK":   1,
}

func (x Request_RequestType) String() string {
	return proto.EnumName(Request_RequestType_name, int32(x))
}

func (Request_RequestType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{1, 0}
}

type Event struct {
	Type                 Event_EventType `protobuf:"varint,1,opt,name=type,proto3,enum=Event_EventType" json:"type,omitempty"`
	Id                   []byte          `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Payload              *any.Any        `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *Event) Reset()         { *m = Event{} }
func (m *Event) String() string { return proto.CompactTextString(m) }
func (*Event) ProtoMessage()    {}
func (*Event) Descriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{0}
}

func (m *Event) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Event.Unmarshal(m, b)
}
func (m *Event) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Event.Marshal(b, m, deterministic)
}
func (m *Event) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Event.Merge(m, src)
}
func (m *Event) XXX_Size() int {
	return xxx_messageInfo_Event.Size(m)
}
func (m *Event) XXX_DiscardUnknown() {
	xxx_messageInfo_Event.DiscardUnknown(m)
}

var xxx_messageInfo_Event proto.InternalMessageInfo

func (m *Event) GetType() Event_EventType {
	if m != nil {
		return m.Type
	}
	return Event_RESERVED
}

func (m *Event) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Event) GetPayload() *any.Any {
	if m != nil {
		return m.Payload
	}
	return nil
}

type Request struct {
	Type                 Request_RequestType `protobuf:"varint,1,opt,name=type,proto3,enum=Request_RequestType" json:"type,omitempty"`
	Id                   []byte              `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Payload              *any.Any            `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *Request) Reset()         { *m = Request{} }
func (m *Request) String() string { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()    {}
func (*Request) Descriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{1}
}

func (m *Request) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Request.Unmarshal(m, b)
}
func (m *Request) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Request.Marshal(b, m, deterministic)
}
func (m *Request) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Request.Merge(m, src)
}
func (m *Request) XXX_Size() int {
	return xxx_messageInfo_Request.Size(m)
}
func (m *Request) XXX_DiscardUnknown() {
	xxx_messageInfo_Request.DiscardUnknown(m)
}

var xxx_messageInfo_Request proto.InternalMessageInfo

func (m *Request) GetType() Request_RequestType {
	if m != nil {
		return m.Type
	}
	return Request_ACCOUNT
}

func (m *Request) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Request) GetPayload() *any.Any {
	if m != nil {
		return m.Payload
	}
	return nil
}

type EpochStartedPayload struct {
	Epoch                int32    `protobuf:"varint,1,opt,name=epoch,proto3" json:"epoch,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EpochStartedPayload) Reset()         { *m = EpochStartedPayload{} }
func (m *EpochStartedPayload) String() string { return proto.CompactTextString(m) }
func (*EpochStartedPayload) ProtoMessage()    {}
func (*EpochStartedPayload) Descriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{2}
}

func (m *EpochStartedPayload) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EpochStartedPayload.Unmarshal(m, b)
}
func (m *EpochStartedPayload) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EpochStartedPayload.Marshal(b, m, deterministic)
}
func (m *EpochStartedPayload) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EpochStartedPayload.Merge(m, src)
}
func (m *EpochStartedPayload) XXX_Size() int {
	return xxx_messageInfo_EpochStartedPayload.Size(m)
}
func (m *EpochStartedPayload) XXX_DiscardUnknown() {
	xxx_messageInfo_EpochStartedPayload.DiscardUnknown(m)
}

var xxx_messageInfo_EpochStartedPayload proto.InternalMessageInfo

func (m *EpochStartedPayload) GetEpoch() int32 {
	if m != nil {
		return m.Epoch
	}
	return 0
}

type ViewChangedPayload struct {
	View                 int32    `protobuf:"varint,1,opt,name=view,proto3" json:"view,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ViewChangedPayload) Reset()         { *m = ViewChangedPayload{} }
func (m *ViewChangedPayload) String() string { return proto.CompactTextString(m) }
func (*ViewChangedPayload) ProtoMessage()    {}
func (*ViewChangedPayload) Descriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{3}
}

func (m *ViewChangedPayload) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ViewChangedPayload.Unmarshal(m, b)
}
func (m *ViewChangedPayload) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ViewChangedPayload.Marshal(b, m, deterministic)
}
func (m *ViewChangedPayload) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ViewChangedPayload.Merge(m, src)
}
func (m *ViewChangedPayload) XXX_Size() int {
	return xxx_messageInfo_ViewChangedPayload.Size(m)
}
func (m *ViewChangedPayload) XXX_DiscardUnknown() {
	xxx_messageInfo_ViewChangedPayload.DiscardUnknown(m)
}

var xxx_messageInfo_ViewChangedPayload proto.InternalMessageInfo

func (m *ViewChangedPayload) GetView() int32 {
	if m != nil {
		return m.View
	}
	return 0
}

type CommittedPayload struct {
	Hash                 []byte   `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CommittedPayload) Reset()         { *m = CommittedPayload{} }
func (m *CommittedPayload) String() string { return proto.CompactTextString(m) }
func (*CommittedPayload) ProtoMessage()    {}
func (*CommittedPayload) Descriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{4}
}

func (m *CommittedPayload) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CommittedPayload.Unmarshal(m, b)
}
func (m *CommittedPayload) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CommittedPayload.Marshal(b, m, deterministic)
}
func (m *CommittedPayload) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CommittedPayload.Merge(m, src)
}
func (m *CommittedPayload) XXX_Size() int {
	return xxx_messageInfo_CommittedPayload.Size(m)
}
func (m *CommittedPayload) XXX_DiscardUnknown() {
	xxx_messageInfo_CommittedPayload.DiscardUnknown(m)
}

var xxx_messageInfo_CommittedPayload proto.InternalMessageInfo

func (m *CommittedPayload) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type AccountRequestPayload struct {
	Address              []byte   `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Block                []byte   `protobuf:"bytes,2,opt,name=block,proto3" json:"block,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AccountRequestPayload) Reset()         { *m = AccountRequestPayload{} }
func (m *AccountRequestPayload) String() string { return proto.CompactTextString(m) }
func (*AccountRequestPayload) ProtoMessage()    {}
func (*AccountRequestPayload) Descriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{5}
}

func (m *AccountRequestPayload) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AccountRequestPayload.Unmarshal(m, b)
}
func (m *AccountRequestPayload) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AccountRequestPayload.Marshal(b, m, deterministic)
}
func (m *AccountRequestPayload) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccountRequestPayload.Merge(m, src)
}
func (m *AccountRequestPayload) XXX_Size() int {
	return xxx_messageInfo_AccountRequestPayload.Size(m)
}
func (m *AccountRequestPayload) XXX_DiscardUnknown() {
	xxx_messageInfo_AccountRequestPayload.DiscardUnknown(m)
}

var xxx_messageInfo_AccountRequestPayload proto.InternalMessageInfo

func (m *AccountRequestPayload) GetAddress() []byte {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *AccountRequestPayload) GetBlock() []byte {
	if m != nil {
		return m.Block
	}
	return nil
}

type AccountResponsePayload struct {
	Address              []byte   `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Block                []byte   `protobuf:"bytes,2,opt,name=block,proto3" json:"block,omitempty"`
	Nonce                uint64   `protobuf:"varint,3,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Value                uint64   `protobuf:"varint,4,opt,name=value,proto3" json:"value,omitempty"`
	Proof                [][]byte `protobuf:"bytes,5,rep,name=proof,proto3" json:"proof,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AccountResponsePayload) Reset()         { *m = AccountResponsePayload{} }
func (m *AccountResponsePayload) String() string { return proto.CompactTextString(m) }
func (*AccountResponsePayload) ProtoMessage()    {}
func (*AccountResponsePayload) Descriptor() ([]byte, []int) {
	return fileDescriptor_2d17a9d3f0ddf27e, []int{6}
}

func (m *AccountResponsePayload) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AccountResponsePayload.Unmarshal(m, b)
}
func (m *AccountResponsePayload) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AccountResponsePayload.Marshal(b, m, deterministic)
}
func (m *AccountResponsePayload) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccountResponsePayload.Merge(m, src)
}
func (m *AccountResponsePayload) XXX_Size() int {
	return xxx_messageInfo_AccountResponsePayload.Size(m)
}
func (m *AccountResponsePayload) XXX_DiscardUnknown() {
	xxx_messageInfo_AccountResponsePayload.DiscardUnknown(m)
}

var xxx_messageInfo_AccountResponsePayload proto.InternalMessageInfo

func (m *AccountResponsePayload) GetAddress() []byte {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *AccountResponsePayload) GetBlock() []byte {
	if m != nil {
		return m.Block
	}
	return nil
}

func (m *AccountResponsePayload) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *AccountResponsePayload) GetValue() uint64 {
	if m != nil {
		return m.Value
	}
	return 0
}

func (m *AccountResponsePayload) GetProof() [][]byte {
	if m != nil {
		return m.Proof
	}
	return nil
}

func init() {
	proto.RegisterEnum("Event_EventType", Event_EventType_name, Event_EventType_value)
	proto.RegisterEnum("Request_RequestType", Request_RequestType_name, Request_RequestType_value)
	proto.RegisterType((*Event)(nil), "Event")
	proto.RegisterType((*Request)(nil), "Request")
	proto.RegisterType((*EpochStartedPayload)(nil), "EpochStartedPayload")
	proto.RegisterType((*ViewChangedPayload)(nil), "ViewChangedPayload")
	proto.RegisterType((*CommittedPayload)(nil), "CommittedPayload")
	proto.RegisterType((*AccountRequestPayload)(nil), "AccountRequestPayload")
	proto.RegisterType((*AccountResponsePayload)(nil), "AccountResponsePayload")
}

func init() { proto.RegisterFile("event.proto", fileDescriptor_2d17a9d3f0ddf27e) }

var fileDescriptor_2d17a9d3f0ddf27e = []byte{
	// 444 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x91, 0xd1, 0x6e, 0x9b, 0x30,
	0x18, 0x85, 0x6b, 0x02, 0xcd, 0xf2, 0x93, 0x76, 0x9e, 0x97, 0x4d, 0x6c, 0x57, 0x11, 0xda, 0x26,
	0xa4, 0x49, 0x54, 0xea, 0x9e, 0x80, 0x82, 0xd5, 0x56, 0x5b, 0x9b, 0xca, 0x61, 0x99, 0xb4, 0x9b,
	0x88, 0x80, 0x9b, 0xa0, 0xa5, 0x98, 0x05, 0x27, 0x15, 0xaf, 0xb0, 0x47, 0xd8, 0xfb, 0xed, 0x3d,
	0x26, 0x6c, 0x68, 0x9b, 0xdb, 0x69, 0x37, 0xe0, 0x73, 0xfc, 0x71, 0x7c, 0xfc, 0x03, 0x36, 0xdf,
	0xf1, 0x42, 0xfa, 0xe5, 0x46, 0x48, 0xf1, 0xf6, 0xcd, 0x52, 0x88, 0xe5, 0x9a, 0x9f, 0x28, 0xb5,
	0xd8, 0xde, 0x9e, 0x24, 0x45, 0xad, 0xb7, 0xdc, 0x3f, 0x08, 0x2c, 0xda, 0xa0, 0xe4, 0x1d, 0x98,
	0xb2, 0x2e, 0xb9, 0x83, 0xc6, 0xc8, 0x3b, 0x3e, 0xc5, 0xbe, 0x72, 0xf5, 0x33, 0xae, 0x4b, 0xce,
	0xd4, 0x2e, 0x39, 0x06, 0x23, 0xcf, 0x1c, 0x63, 0x8c, 0xbc, 0x21, 0x33, 0xf2, 0x8c, 0xf8, 0xd0,
	0x2f, 0x93, 0x7a, 0x2d, 0x92, 0xcc, 0xe9, 0x8d, 0x91, 0x67, 0x9f, 0x8e, 0x7c, 0x7d, 0x98, 0xdf,
	0x1d, 0xe6, 0x07, 0x45, 0xcd, 0x3a, 0xc8, 0xdd, 0xc1, 0xe0, 0x21, 0x92, 0x0c, 0xe1, 0x19, 0xa3,
	0x53, 0xca, 0x66, 0x34, 0xc2, 0x07, 0xe4, 0x39, 0xd8, 0x67, 0x5f, 0x26, 0xe1, 0xe7, 0x79, 0x10,
	0x45, 0x34, 0xc2, 0x88, 0xbc, 0x80, 0x23, 0x7a, 0x33, 0x09, 0x2f, 0xe6, 0xd3, 0x38, 0x60, 0x31,
	0x8d, 0xb0, 0x41, 0x30, 0x0c, 0x67, 0x97, 0xf4, 0xdb, 0x3c, 0xbc, 0x08, 0xae, 0xcf, 0x69, 0x84,
	0x7b, 0xe4, 0x08, 0x06, 0xe1, 0xe4, 0xea, 0xea, 0x32, 0x6e, 0x00, 0x93, 0xd8, 0xd0, 0x0f, 0xc2,
	0x70, 0xf2, 0xf5, 0x3a, 0xc6, 0x16, 0x19, 0x80, 0xa5, 0x12, 0xf1, 0xa1, 0xfb, 0x1b, 0x41, 0x9f,
	0xf1, 0x9f, 0x5b, 0x5e, 0x49, 0xe2, 0xed, 0xdd, 0x74, 0xe4, 0xb7, 0x7e, 0xf7, 0xfe, 0x8f, 0xb7,
	0x7d, 0x0f, 0xf6, 0x93, 0xd0, 0xa7, 0xe5, 0x0e, 0x1e, 0xcb, 0x21, 0xf7, 0x23, 0xbc, 0xa4, 0xa5,
	0x48, 0x57, 0x53, 0x99, 0x6c, 0x24, 0xcf, 0x6e, 0xf4, 0xd7, 0x64, 0x04, 0x16, 0x6f, 0x6c, 0x55,
	0xd4, 0x62, 0x5a, 0xb8, 0x1e, 0x90, 0x59, 0xce, 0xef, 0xc3, 0x55, 0x52, 0x2c, 0x1f, 0x59, 0x02,
	0xe6, 0x2e, 0xe7, 0xf7, 0x2d, 0xaa, 0xd6, 0xee, 0x07, 0xc0, 0xa1, 0xb8, 0xbb, 0xcb, 0xa5, 0xdc,
	0xe3, 0x56, 0x49, 0xa5, 0x23, 0x87, 0x4c, 0xad, 0xdd, 0x73, 0x78, 0x15, 0xa4, 0xa9, 0xd8, 0x16,
	0xb2, 0x2d, 0xdb, 0xc1, 0x0e, 0xf4, 0x93, 0x2c, 0xdb, 0xf0, 0xaa, 0x6a, 0xf9, 0x4e, 0x36, 0xd5,
	0x16, 0x6b, 0x91, 0xfe, 0x68, 0x67, 0xa3, 0x85, 0xfb, 0x0b, 0xc1, 0xeb, 0x87, 0xa4, 0xaa, 0x14,
	0x45, 0xc5, 0xff, 0x31, 0xaa, 0x71, 0x0b, 0x51, 0xa4, 0x5c, 0xcd, 0xd9, 0x64, 0x5a, 0x34, 0xee,
	0x2e, 0x59, 0x6f, 0xb9, 0x63, 0x6a, 0x57, 0x89, 0xc6, 0x2d, 0x37, 0x42, 0xdc, 0x3a, 0xd6, 0xb8,
	0xd7, 0x24, 0x28, 0x71, 0x66, 0x7e, 0x37, 0xca, 0xc5, 0xe2, 0x50, 0xfd, 0x98, 0x4f, 0x7f, 0x03,
	0x00, 0x00, 0xff, 0xff, 0x53, 0x6c, 0x7c, 0x9d, 0x10, 0x03, 0x00, 0x00,
}
