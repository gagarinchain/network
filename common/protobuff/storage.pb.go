// Code generated by protoc-gen-go. DO NOT EDIT.
// source: storage.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

type Tx struct {
	Type                 int32    `protobuf:"varint,1,opt,name=type,proto3" json:"type,omitempty"`
	To                   []byte   `protobuf:"bytes,2,opt,name=to,proto3" json:"to,omitempty"`
	From                 []byte   `protobuf:"bytes,3,opt,name=from,proto3" json:"from,omitempty"`
	Nonce                uint64   `protobuf:"varint,4,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Value                []byte   `protobuf:"bytes,5,opt,name=value,proto3" json:"value,omitempty"`
	Fee                  []byte   `protobuf:"bytes,6,opt,name=fee,proto3" json:"fee,omitempty"`
	Signature            *Sign    `protobuf:"bytes,7,opt,name=signature,proto3" json:"signature,omitempty"`
	HashKey              []byte   `protobuf:"bytes,8,opt,name=hashKey,proto3" json:"hashKey,omitempty"`
	Data                 []byte   `protobuf:"bytes,9,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Tx) Reset()         { *m = Tx{} }
func (m *Tx) String() string { return proto.CompactTextString(m) }
func (*Tx) ProtoMessage()    {}
func (*Tx) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d2c4ccf1453ffdb, []int{0}
}

func (m *Tx) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Tx.Unmarshal(m, b)
}
func (m *Tx) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Tx.Marshal(b, m, deterministic)
}
func (m *Tx) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Tx.Merge(m, src)
}
func (m *Tx) XXX_Size() int {
	return xxx_messageInfo_Tx.Size(m)
}
func (m *Tx) XXX_DiscardUnknown() {
	xxx_messageInfo_Tx.DiscardUnknown(m)
}

var xxx_messageInfo_Tx proto.InternalMessageInfo

func (m *Tx) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *Tx) GetTo() []byte {
	if m != nil {
		return m.To
	}
	return nil
}

func (m *Tx) GetFrom() []byte {
	if m != nil {
		return m.From
	}
	return nil
}

func (m *Tx) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *Tx) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Tx) GetFee() []byte {
	if m != nil {
		return m.Fee
	}
	return nil
}

func (m *Tx) GetSignature() *Sign {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *Tx) GetHashKey() []byte {
	if m != nil {
		return m.HashKey
	}
	return nil
}

func (m *Tx) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type Sign struct {
	From                 []byte   `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	Signature            []byte   `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Sign) Reset()         { *m = Sign{} }
func (m *Sign) String() string { return proto.CompactTextString(m) }
func (*Sign) ProtoMessage()    {}
func (*Sign) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d2c4ccf1453ffdb, []int{1}
}

func (m *Sign) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Sign.Unmarshal(m, b)
}
func (m *Sign) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Sign.Marshal(b, m, deterministic)
}
func (m *Sign) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Sign.Merge(m, src)
}
func (m *Sign) XXX_Size() int {
	return xxx_messageInfo_Sign.Size(m)
}
func (m *Sign) XXX_DiscardUnknown() {
	xxx_messageInfo_Sign.DiscardUnknown(m)
}

var xxx_messageInfo_Sign proto.InternalMessageInfo

func (m *Sign) GetFrom() []byte {
	if m != nil {
		return m.From
	}
	return nil
}

func (m *Sign) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type Account struct {
	Nonce                uint64   `protobuf:"varint,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Value                []byte   `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Origin               []byte   `protobuf:"bytes,3,opt,name=origin,proto3" json:"origin,omitempty"`
	Voters               [][]byte `protobuf:"bytes,4,rep,name=voters,proto3" json:"voters,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Account) Reset()         { *m = Account{} }
func (m *Account) String() string { return proto.CompactTextString(m) }
func (*Account) ProtoMessage()    {}
func (*Account) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d2c4ccf1453ffdb, []int{2}
}

func (m *Account) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Account.Unmarshal(m, b)
}
func (m *Account) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Account.Marshal(b, m, deterministic)
}
func (m *Account) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Account.Merge(m, src)
}
func (m *Account) XXX_Size() int {
	return xxx_messageInfo_Account.Size(m)
}
func (m *Account) XXX_DiscardUnknown() {
	xxx_messageInfo_Account.DiscardUnknown(m)
}

var xxx_messageInfo_Account proto.InternalMessageInfo

func (m *Account) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *Account) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Account) GetOrigin() []byte {
	if m != nil {
		return m.Origin
	}
	return nil
}

func (m *Account) GetVoters() [][]byte {
	if m != nil {
		return m.Voters
	}
	return nil
}

type Record struct {
	Snap                 []byte   `protobuf:"bytes,1,opt,name=snap,proto3" json:"snap,omitempty"`
	Parent               []byte   `protobuf:"bytes,2,opt,name=parent,proto3" json:"parent,omitempty"`
	Siblings             [][]byte `protobuf:"bytes,3,rep,name=siblings,proto3" json:"siblings,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Record) Reset()         { *m = Record{} }
func (m *Record) String() string { return proto.CompactTextString(m) }
func (*Record) ProtoMessage()    {}
func (*Record) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d2c4ccf1453ffdb, []int{3}
}

func (m *Record) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Record.Unmarshal(m, b)
}
func (m *Record) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Record.Marshal(b, m, deterministic)
}
func (m *Record) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Record.Merge(m, src)
}
func (m *Record) XXX_Size() int {
	return xxx_messageInfo_Record.Size(m)
}
func (m *Record) XXX_DiscardUnknown() {
	xxx_messageInfo_Record.DiscardUnknown(m)
}

var xxx_messageInfo_Record proto.InternalMessageInfo

func (m *Record) GetSnap() []byte {
	if m != nil {
		return m.Snap
	}
	return nil
}

func (m *Record) GetParent() []byte {
	if m != nil {
		return m.Parent
	}
	return nil
}

func (m *Record) GetSiblings() [][]byte {
	if m != nil {
		return m.Siblings
	}
	return nil
}

type Snapshot struct {
	Hash                 []byte   `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Proposer             []byte   `protobuf:"bytes,2,opt,name=proposer,proto3" json:"proposer,omitempty"`
	Entries              []*Entry `protobuf:"bytes,3,rep,name=entries,proto3" json:"entries,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Snapshot) Reset()         { *m = Snapshot{} }
func (m *Snapshot) String() string { return proto.CompactTextString(m) }
func (*Snapshot) ProtoMessage()    {}
func (*Snapshot) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d2c4ccf1453ffdb, []int{4}
}

func (m *Snapshot) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Snapshot.Unmarshal(m, b)
}
func (m *Snapshot) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Snapshot.Marshal(b, m, deterministic)
}
func (m *Snapshot) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Snapshot.Merge(m, src)
}
func (m *Snapshot) XXX_Size() int {
	return xxx_messageInfo_Snapshot.Size(m)
}
func (m *Snapshot) XXX_DiscardUnknown() {
	xxx_messageInfo_Snapshot.DiscardUnknown(m)
}

var xxx_messageInfo_Snapshot proto.InternalMessageInfo

func (m *Snapshot) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *Snapshot) GetProposer() []byte {
	if m != nil {
		return m.Proposer
	}
	return nil
}

func (m *Snapshot) GetEntries() []*Entry {
	if m != nil {
		return m.Entries
	}
	return nil
}

type Entry struct {
	Address              []byte   `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Account              []byte   `protobuf:"bytes,2,opt,name=account,proto3" json:"account,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Entry) Reset()         { *m = Entry{} }
func (m *Entry) String() string { return proto.CompactTextString(m) }
func (*Entry) ProtoMessage()    {}
func (*Entry) Descriptor() ([]byte, []int) {
	return fileDescriptor_0d2c4ccf1453ffdb, []int{5}
}

func (m *Entry) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Entry.Unmarshal(m, b)
}
func (m *Entry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Entry.Marshal(b, m, deterministic)
}
func (m *Entry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Entry.Merge(m, src)
}
func (m *Entry) XXX_Size() int {
	return xxx_messageInfo_Entry.Size(m)
}
func (m *Entry) XXX_DiscardUnknown() {
	xxx_messageInfo_Entry.DiscardUnknown(m)
}

var xxx_messageInfo_Entry proto.InternalMessageInfo

func (m *Entry) GetAddress() []byte {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *Entry) GetAccount() []byte {
	if m != nil {
		return m.Account
	}
	return nil
}

func init() {
	proto.RegisterType((*Tx)(nil), "Tx")
	proto.RegisterType((*Sign)(nil), "Sign")
	proto.RegisterType((*Account)(nil), "Account")
	proto.RegisterType((*Record)(nil), "Record")
	proto.RegisterType((*Snapshot)(nil), "Snapshot")
	proto.RegisterType((*Entry)(nil), "Entry")
}

func init() { proto.RegisterFile("storage.proto", fileDescriptor_0d2c4ccf1453ffdb) }

var fileDescriptor_0d2c4ccf1453ffdb = []byte{
	// 367 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x92, 0x3f, 0x6f, 0xdb, 0x30,
	0x10, 0xc5, 0x41, 0xfd, 0xb5, 0xcf, 0x6d, 0x51, 0x10, 0x45, 0x41, 0x14, 0x1d, 0x04, 0x75, 0xd1,
	0xe4, 0xc1, 0x5d, 0x02, 0x64, 0x4a, 0x80, 0x4c, 0x59, 0x02, 0x39, 0x53, 0x90, 0x85, 0x96, 0xce,
	0xb2, 0x00, 0x87, 0x24, 0x48, 0xda, 0x88, 0x3f, 0x64, 0xbe, 0x53, 0x70, 0x94, 0x64, 0x7b, 0xc8,
	0xf6, 0x7e, 0xcf, 0xe4, 0xe3, 0xbb, 0xb3, 0xe0, 0xbb, 0xf3, 0xda, 0xca, 0x0e, 0x97, 0xc6, 0x6a,
	0xaf, 0xcb, 0x0f, 0x06, 0xd1, 0xf3, 0x3b, 0xe7, 0x90, 0xf8, 0x93, 0x41, 0xc1, 0x0a, 0x56, 0xa5,
	0x75, 0xd0, 0xfc, 0x07, 0x44, 0x5e, 0x8b, 0xa8, 0x60, 0xd5, 0xb7, 0x3a, 0xf2, 0x9a, 0xce, 0x6c,
	0xad, 0x7e, 0x13, 0x71, 0x70, 0x82, 0xe6, 0xbf, 0x20, 0x55, 0x5a, 0x35, 0x28, 0x92, 0x82, 0x55,
	0x49, 0x3d, 0x00, 0xb9, 0x47, 0xb9, 0x3f, 0xa0, 0x48, 0xc3, 0xd1, 0x01, 0xf8, 0x4f, 0x88, 0xb7,
	0x88, 0x22, 0x0b, 0x1e, 0x49, 0xfe, 0x0f, 0xe6, 0xae, 0xef, 0x94, 0xf4, 0x07, 0x8b, 0x22, 0x2f,
	0x58, 0xb5, 0x58, 0xa5, 0xcb, 0x75, 0xdf, 0xa9, 0xfa, 0xe2, 0x73, 0x01, 0xf9, 0x4e, 0xba, 0xdd,
	0x23, 0x9e, 0xc4, 0x2c, 0x5c, 0x9d, 0x90, 0x0a, 0xb5, 0xd2, 0x4b, 0x31, 0x1f, 0x0a, 0x91, 0x2e,
	0x6f, 0x20, 0xa1, 0x80, 0x73, 0x59, 0x76, 0x55, 0xf6, 0xef, 0xf5, 0x73, 0xc3, 0x5c, 0x17, 0xa3,
	0x44, 0xc8, 0xef, 0x9a, 0x46, 0x1f, 0x94, 0xbf, 0x4c, 0xc5, 0xbe, 0x9c, 0x2a, 0xba, 0x9e, 0xea,
	0x37, 0x64, 0xda, 0xf6, 0x5d, 0xaf, 0xc6, 0xbd, 0x8c, 0x44, 0xfe, 0x51, 0x7b, 0xb4, 0x4e, 0x24,
	0x45, 0x4c, 0xfe, 0x40, 0xe5, 0x13, 0x64, 0x35, 0x36, 0xda, 0xb6, 0x54, 0xd1, 0x29, 0x69, 0xa6,
	0x8a, 0xa4, 0xe9, 0x96, 0x91, 0x16, 0x95, 0x1f, 0x1f, 0x19, 0x89, 0xff, 0x81, 0x99, 0xeb, 0x37,
	0xfb, 0x5e, 0x75, 0x4e, 0xc4, 0x21, 0xef, 0xcc, 0xe5, 0x2b, 0xcc, 0xd6, 0x4a, 0x1a, 0xb7, 0xd3,
	0x9e, 0x32, 0x69, 0x3b, 0x53, 0x26, 0x69, 0xba, 0x6b, 0xac, 0x36, 0xda, 0xa1, 0x1d, 0x53, 0xcf,
	0xcc, 0x0b, 0xc8, 0x51, 0x79, 0xdb, 0xe3, 0x10, 0xbb, 0x58, 0x65, 0xcb, 0x07, 0xe5, 0xed, 0xa9,
	0x9e, 0xec, 0xf2, 0x16, 0xd2, 0xe0, 0xd0, 0xff, 0x20, 0xdb, 0xd6, 0xa2, 0x73, 0x63, 0xfa, 0x84,
	0xe1, 0x97, 0x61, 0x73, 0x63, 0xfe, 0x84, 0xf7, 0xc9, 0x4b, 0x64, 0x36, 0x9b, 0x2c, 0x7c, 0x6a,
	0xff, 0x3f, 0x03, 0x00, 0x00, 0xff, 0xff, 0xac, 0x80, 0x7f, 0x64, 0x7b, 0x02, 0x00, 0x00,
}
