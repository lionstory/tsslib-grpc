// Code generated by protoc-gen-go. DO NOT EDIT.
// source: vss.proto

package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type CurveType int32

const (
	CurveType_secp256k1 CurveType = 0
	CurveType_ed25519   CurveType = 1
	CurveType_p256sm2   CurveType = 2
	CurveType_p256      CurveType = 3
)

var CurveType_name = map[int32]string{
	0: "secp256k1",
	1: "ed25519",
	2: "p256sm2",
	3: "p256",
}
var CurveType_value = map[string]int32{
	"secp256k1": 0,
	"ed25519":   1,
	"p256sm2":   2,
	"p256":      3,
}

func (x CurveType) String() string {
	return proto.EnumName(CurveType_name, int32(x))
}
func (CurveType) EnumDescriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

type Share struct {
	Threshold int32  `protobuf:"varint,1,opt,name=Threshold" json:"Threshold,omitempty"`
	ID        []byte `protobuf:"bytes,2,opt,name=ID,proto3" json:"ID,omitempty"`
	Share     []byte `protobuf:"bytes,3,opt,name=Share,proto3" json:"Share,omitempty"`
}

func (m *Share) Reset()                    { *m = Share{} }
func (m *Share) String() string            { return proto.CompactTextString(m) }
func (*Share) ProtoMessage()               {}
func (*Share) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *Share) GetThreshold() int32 {
	if m != nil {
		return m.Threshold
	}
	return 0
}

func (m *Share) GetID() []byte {
	if m != nil {
		return m.ID
	}
	return nil
}

func (m *Share) GetShare() []byte {
	if m != nil {
		return m.Share
	}
	return nil
}

type PublicKey struct {
	X []byte `protobuf:"bytes,1,opt,name=X,proto3" json:"X,omitempty"`
	Y []byte `protobuf:"bytes,2,opt,name=Y,proto3" json:"Y,omitempty"`
}

func (m *PublicKey) Reset()                    { *m = PublicKey{} }
func (m *PublicKey) String() string            { return proto.CompactTextString(m) }
func (*PublicKey) ProtoMessage()               {}
func (*PublicKey) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *PublicKey) GetX() []byte {
	if m != nil {
		return m.X
	}
	return nil
}

func (m *PublicKey) GetY() []byte {
	if m != nil {
		return m.Y
	}
	return nil
}

type Partner struct {
	ID      int64  `protobuf:"varint,1,opt,name=ID" json:"ID,omitempty"`
	Address string `protobuf:"bytes,2,opt,name=Address" json:"Address,omitempty"`
	Status  string `protobuf:"bytes,3,opt,name=Status" json:"Status,omitempty"`
}

func (m *Partner) Reset()                    { *m = Partner{} }
func (m *Partner) String() string            { return proto.CompactTextString(m) }
func (*Partner) ProtoMessage()               {}
func (*Partner) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{2} }

func (m *Partner) GetID() int64 {
	if m != nil {
		return m.ID
	}
	return 0
}

func (m *Partner) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *Partner) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

type Project struct {
	ProjectID int64              `protobuf:"varint,1,opt,name=ProjectID" json:"ProjectID,omitempty"`
	PartID    int64              `protobuf:"varint,2,opt,name=PartID" json:"PartID,omitempty"`
	Partners  map[int64]*Partner `protobuf:"bytes,3,rep,name=Partners" json:"Partners,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Curve     CurveType          `protobuf:"varint,4,opt,name=Curve,enum=service.CurveType" json:"Curve,omitempty"`
	PublicKey *PublicKey         `protobuf:"bytes,5,opt,name=PublicKey" json:"PublicKey,omitempty"`
	Share     *Share             `protobuf:"bytes,6,opt,name=Share" json:"Share,omitempty"`
	VS        [][]byte           `protobuf:"bytes,7,rep,name=VS,proto3" json:"VS,omitempty"`
}

func (m *Project) Reset()                    { *m = Project{} }
func (m *Project) String() string            { return proto.CompactTextString(m) }
func (*Project) ProtoMessage()               {}
func (*Project) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{3} }

func (m *Project) GetProjectID() int64 {
	if m != nil {
		return m.ProjectID
	}
	return 0
}

func (m *Project) GetPartID() int64 {
	if m != nil {
		return m.PartID
	}
	return 0
}

func (m *Project) GetPartners() map[int64]*Partner {
	if m != nil {
		return m.Partners
	}
	return nil
}

func (m *Project) GetCurve() CurveType {
	if m != nil {
		return m.Curve
	}
	return CurveType_secp256k1
}

func (m *Project) GetPublicKey() *PublicKey {
	if m != nil {
		return m.PublicKey
	}
	return nil
}

func (m *Project) GetShare() *Share {
	if m != nil {
		return m.Share
	}
	return nil
}

func (m *Project) GetVS() [][]byte {
	if m != nil {
		return m.VS
	}
	return nil
}

type ProjectSetupRequest struct {
	ProjectID int64      `protobuf:"varint,1,opt,name=ProjectID" json:"ProjectID,omitempty"`
	PartID    int64      `protobuf:"varint,2,opt,name=PartID" json:"PartID,omitempty"`
	Partners  []*Partner `protobuf:"bytes,3,rep,name=Partners" json:"Partners,omitempty"`
}

func (m *ProjectSetupRequest) Reset()                    { *m = ProjectSetupRequest{} }
func (m *ProjectSetupRequest) String() string            { return proto.CompactTextString(m) }
func (*ProjectSetupRequest) ProtoMessage()               {}
func (*ProjectSetupRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{4} }

func (m *ProjectSetupRequest) GetProjectID() int64 {
	if m != nil {
		return m.ProjectID
	}
	return 0
}

func (m *ProjectSetupRequest) GetPartID() int64 {
	if m != nil {
		return m.PartID
	}
	return 0
}

func (m *ProjectSetupRequest) GetPartners() []*Partner {
	if m != nil {
		return m.Partners
	}
	return nil
}

type KeyGenerateRequest struct {
	ProjectID int64     `protobuf:"varint,1,opt,name=ProjectID" json:"ProjectID,omitempty"`
	Curve     CurveType `protobuf:"varint,2,opt,name=Curve,enum=service.CurveType" json:"Curve,omitempty"`
	Threshold int64     `protobuf:"varint,3,opt,name=Threshold" json:"Threshold,omitempty"`
}

func (m *KeyGenerateRequest) Reset()                    { *m = KeyGenerateRequest{} }
func (m *KeyGenerateRequest) String() string            { return proto.CompactTextString(m) }
func (*KeyGenerateRequest) ProtoMessage()               {}
func (*KeyGenerateRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{5} }

func (m *KeyGenerateRequest) GetProjectID() int64 {
	if m != nil {
		return m.ProjectID
	}
	return 0
}

func (m *KeyGenerateRequest) GetCurve() CurveType {
	if m != nil {
		return m.Curve
	}
	return CurveType_secp256k1
}

func (m *KeyGenerateRequest) GetThreshold() int64 {
	if m != nil {
		return m.Threshold
	}
	return 0
}

type CmtMsg struct {
	ProjectID int64     `protobuf:"varint,1,opt,name=ProjectID" json:"ProjectID,omitempty"`
	Curve     CurveType `protobuf:"varint,2,opt,name=Curve,enum=service.CurveType" json:"Curve,omitempty"`
	Cmt       []byte    `protobuf:"bytes,3,opt,name=Cmt,proto3" json:"Cmt,omitempty"`
}

func (m *CmtMsg) Reset()                    { *m = CmtMsg{} }
func (m *CmtMsg) String() string            { return proto.CompactTextString(m) }
func (*CmtMsg) ProtoMessage()               {}
func (*CmtMsg) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{6} }

func (m *CmtMsg) GetProjectID() int64 {
	if m != nil {
		return m.ProjectID
	}
	return 0
}

func (m *CmtMsg) GetCurve() CurveType {
	if m != nil {
		return m.Curve
	}
	return CurveType_secp256k1
}

func (m *CmtMsg) GetCmt() []byte {
	if m != nil {
		return m.Cmt
	}
	return nil
}

type ShareMsg struct {
	ProjectID int64  `protobuf:"varint,1,opt,name=ProjectID" json:"ProjectID,omitempty"`
	Share     []byte `protobuf:"bytes,2,opt,name=Share,proto3" json:"Share,omitempty"`
	X         []byte `protobuf:"bytes,3,opt,name=X,proto3" json:"X,omitempty"`
	Y         []byte `protobuf:"bytes,4,opt,name=Y,proto3" json:"Y,omitempty"`
}

func (m *ShareMsg) Reset()                    { *m = ShareMsg{} }
func (m *ShareMsg) String() string            { return proto.CompactTextString(m) }
func (*ShareMsg) ProtoMessage()               {}
func (*ShareMsg) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{7} }

func (m *ShareMsg) GetProjectID() int64 {
	if m != nil {
		return m.ProjectID
	}
	return 0
}

func (m *ShareMsg) GetShare() []byte {
	if m != nil {
		return m.Share
	}
	return nil
}

func (m *ShareMsg) GetX() []byte {
	if m != nil {
		return m.X
	}
	return nil
}

func (m *ShareMsg) GetY() []byte {
	if m != nil {
		return m.Y
	}
	return nil
}

type CommonRequest struct {
	ProjectID int64  `protobuf:"varint,1,opt,name=ProjectID" json:"ProjectID,omitempty"`
	Raw       string `protobuf:"bytes,2,opt,name=Raw" json:"Raw,omitempty"`
}

func (m *CommonRequest) Reset()                    { *m = CommonRequest{} }
func (m *CommonRequest) String() string            { return proto.CompactTextString(m) }
func (*CommonRequest) ProtoMessage()               {}
func (*CommonRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{8} }

func (m *CommonRequest) GetProjectID() int64 {
	if m != nil {
		return m.ProjectID
	}
	return 0
}

func (m *CommonRequest) GetRaw() string {
	if m != nil {
		return m.Raw
	}
	return ""
}

func init() {
	proto.RegisterType((*Share)(nil), "service.Share")
	proto.RegisterType((*PublicKey)(nil), "service.PublicKey")
	proto.RegisterType((*Partner)(nil), "service.Partner")
	proto.RegisterType((*Project)(nil), "service.Project")
	proto.RegisterType((*ProjectSetupRequest)(nil), "service.ProjectSetupRequest")
	proto.RegisterType((*KeyGenerateRequest)(nil), "service.KeyGenerateRequest")
	proto.RegisterType((*CmtMsg)(nil), "service.CmtMsg")
	proto.RegisterType((*ShareMsg)(nil), "service.ShareMsg")
	proto.RegisterType((*CommonRequest)(nil), "service.CommonRequest")
	proto.RegisterEnum("service.CurveType", CurveType_name, CurveType_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for VssService service

type VssServiceClient interface {
	ProjectSetup(ctx context.Context, in *ProjectSetupRequest, opts ...grpc.CallOption) (*CommonResonse, error)
	ProjectSync(ctx context.Context, in *ProjectSetupRequest, opts ...grpc.CallOption) (*CommonResonse, error)
	KeyGenerate(ctx context.Context, in *KeyGenerateRequest, opts ...grpc.CallOption) (*CommonResonse, error)
	CmtProcess(ctx context.Context, in *CmtMsg, opts ...grpc.CallOption) (*CommonResonse, error)
	ShareProcess(ctx context.Context, in *ShareMsg, opts ...grpc.CallOption) (*CommonResonse, error)
	Reconstruct(ctx context.Context, in *CommonRequest, opts ...grpc.CallOption) (*CommonResonse, error)
	Encrypt(ctx context.Context, in *CommonRequest, opts ...grpc.CallOption) (*CommonResonse, error)
	Decrypt(ctx context.Context, in *CommonRequest, opts ...grpc.CallOption) (*CommonResonse, error)
}

type vssServiceClient struct {
	cc *grpc.ClientConn
}

func NewVssServiceClient(cc *grpc.ClientConn) VssServiceClient {
	return &vssServiceClient{cc}
}

func (c *vssServiceClient) ProjectSetup(ctx context.Context, in *ProjectSetupRequest, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/ProjectSetup", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vssServiceClient) ProjectSync(ctx context.Context, in *ProjectSetupRequest, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/ProjectSync", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vssServiceClient) KeyGenerate(ctx context.Context, in *KeyGenerateRequest, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/KeyGenerate", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vssServiceClient) CmtProcess(ctx context.Context, in *CmtMsg, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/CmtProcess", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vssServiceClient) ShareProcess(ctx context.Context, in *ShareMsg, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/ShareProcess", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vssServiceClient) Reconstruct(ctx context.Context, in *CommonRequest, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/Reconstruct", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vssServiceClient) Encrypt(ctx context.Context, in *CommonRequest, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/Encrypt", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *vssServiceClient) Decrypt(ctx context.Context, in *CommonRequest, opts ...grpc.CallOption) (*CommonResonse, error) {
	out := new(CommonResonse)
	err := grpc.Invoke(ctx, "/service.VssService/Decrypt", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for VssService service

type VssServiceServer interface {
	ProjectSetup(context.Context, *ProjectSetupRequest) (*CommonResonse, error)
	ProjectSync(context.Context, *ProjectSetupRequest) (*CommonResonse, error)
	KeyGenerate(context.Context, *KeyGenerateRequest) (*CommonResonse, error)
	CmtProcess(context.Context, *CmtMsg) (*CommonResonse, error)
	ShareProcess(context.Context, *ShareMsg) (*CommonResonse, error)
	Reconstruct(context.Context, *CommonRequest) (*CommonResonse, error)
	Encrypt(context.Context, *CommonRequest) (*CommonResonse, error)
	Decrypt(context.Context, *CommonRequest) (*CommonResonse, error)
}

func RegisterVssServiceServer(s *grpc.Server, srv VssServiceServer) {
	s.RegisterService(&_VssService_serviceDesc, srv)
}

func _VssService_ProjectSetup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProjectSetupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).ProjectSetup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/ProjectSetup",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).ProjectSetup(ctx, req.(*ProjectSetupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VssService_ProjectSync_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ProjectSetupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).ProjectSync(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/ProjectSync",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).ProjectSync(ctx, req.(*ProjectSetupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VssService_KeyGenerate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(KeyGenerateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).KeyGenerate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/KeyGenerate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).KeyGenerate(ctx, req.(*KeyGenerateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VssService_CmtProcess_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CmtMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).CmtProcess(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/CmtProcess",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).CmtProcess(ctx, req.(*CmtMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _VssService_ShareProcess_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShareMsg)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).ShareProcess(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/ShareProcess",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).ShareProcess(ctx, req.(*ShareMsg))
	}
	return interceptor(ctx, in, info, handler)
}

func _VssService_Reconstruct_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommonRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).Reconstruct(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/Reconstruct",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).Reconstruct(ctx, req.(*CommonRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VssService_Encrypt_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommonRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).Encrypt(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/Encrypt",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).Encrypt(ctx, req.(*CommonRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _VssService_Decrypt_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommonRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(VssServiceServer).Decrypt(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/service.VssService/Decrypt",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(VssServiceServer).Decrypt(ctx, req.(*CommonRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _VssService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "service.VssService",
	HandlerType: (*VssServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ProjectSetup",
			Handler:    _VssService_ProjectSetup_Handler,
		},
		{
			MethodName: "ProjectSync",
			Handler:    _VssService_ProjectSync_Handler,
		},
		{
			MethodName: "KeyGenerate",
			Handler:    _VssService_KeyGenerate_Handler,
		},
		{
			MethodName: "CmtProcess",
			Handler:    _VssService_CmtProcess_Handler,
		},
		{
			MethodName: "ShareProcess",
			Handler:    _VssService_ShareProcess_Handler,
		},
		{
			MethodName: "Reconstruct",
			Handler:    _VssService_Reconstruct_Handler,
		},
		{
			MethodName: "Encrypt",
			Handler:    _VssService_Encrypt_Handler,
		},
		{
			MethodName: "Decrypt",
			Handler:    _VssService_Decrypt_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "vss.proto",
}

func init() { proto.RegisterFile("vss.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 639 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x55, 0xff, 0x6b, 0xd3, 0x40,
	0x14, 0x37, 0x49, 0xdb, 0x2c, 0x2f, 0xed, 0x16, 0x4f, 0x19, 0x61, 0x0e, 0x29, 0x41, 0xb4, 0x88,
	0x6c, 0x2e, 0x52, 0x71, 0x13, 0x15, 0xd7, 0x0e, 0x19, 0x65, 0x50, 0xae, 0x63, 0x6c, 0xfb, 0xad,
	0x4d, 0x8f, 0x6d, 0x6e, 0x4d, 0xe2, 0xdd, 0xa5, 0x12, 0x7f, 0xf4, 0xef, 0xf4, 0x8f, 0x91, 0xbb,
	0x5c, 0xd2, 0xd4, 0xb2, 0x2f, 0x14, 0x7f, 0x7b, 0xef, 0xdd, 0xe7, 0xf3, 0xde, 0xdd, 0xe7, 0xbd,
	0x97, 0x80, 0x35, 0x65, 0x6c, 0x2b, 0xa6, 0x11, 0x8f, 0x90, 0xc9, 0x08, 0x9d, 0x5e, 0x05, 0x64,
	0xa3, 0xa1, 0x8c, 0x2c, 0xee, 0xf5, 0xa0, 0x3a, 0xb8, 0x1c, 0x52, 0x82, 0x36, 0xc1, 0x3a, 0xbe,
	0xa4, 0x84, 0x5d, 0x46, 0x37, 0x63, 0x57, 0x6b, 0x6a, 0xad, 0x2a, 0x9e, 0x05, 0xd0, 0x2a, 0xe8,
	0x87, 0x5d, 0x57, 0x6f, 0x6a, 0xad, 0x3a, 0xd6, 0x0f, 0xbb, 0xe8, 0xa9, 0xa2, 0xb9, 0x86, 0x0c,
	0x65, 0x8e, 0xf7, 0x0a, 0xac, 0x7e, 0x32, 0xba, 0xb9, 0x0a, 0x7a, 0x24, 0x45, 0x75, 0xd0, 0x4e,
	0x65, 0xa2, 0x3a, 0xd6, 0x4e, 0x85, 0x77, 0xa6, 0xf8, 0xda, 0x99, 0xd7, 0x03, 0xb3, 0x3f, 0xa4,
	0x3c, 0x24, 0x54, 0x65, 0x16, 0x38, 0x43, 0x66, 0x76, 0xc1, 0xfc, 0x3a, 0x1e, 0x53, 0xc2, 0x98,
	0x84, 0x5b, 0x38, 0x77, 0xd1, 0x3a, 0xd4, 0x06, 0x7c, 0xc8, 0x13, 0x26, 0x8b, 0x5a, 0x58, 0x79,
	0xde, 0x1f, 0x1d, 0xcc, 0x3e, 0x8d, 0xbe, 0x93, 0x80, 0x8b, 0x57, 0x28, 0xb3, 0x48, 0x3a, 0x0b,
	0x88, 0x0c, 0xa2, 0xac, 0x7a, 0x89, 0x81, 0x95, 0x87, 0xf6, 0x60, 0x45, 0x5d, 0x47, 0xe4, 0x36,
	0x5a, 0xb6, 0xff, 0x7c, 0x2b, 0x97, 0x49, 0xb1, 0xb7, 0x72, 0xc0, 0x41, 0xc8, 0x69, 0x8a, 0x0b,
	0x3c, 0x6a, 0x41, 0xb5, 0x93, 0xd0, 0x29, 0x71, 0x2b, 0x4d, 0xad, 0xb5, 0xea, 0xa3, 0x82, 0x28,
	0xa3, 0xc7, 0x69, 0x4c, 0x70, 0x06, 0x40, 0x6f, 0x4b, 0xea, 0xb8, 0xd5, 0xa6, 0xd6, 0xb2, 0x4b,
	0xe8, 0xe2, 0x04, 0x97, 0x24, 0x7c, 0x91, 0xab, 0x5c, 0x93, 0xe8, 0xd5, 0x02, 0x2d, 0xa3, 0x4a,
	0x75, 0xa1, 0xe0, 0xc9, 0xc0, 0x35, 0x9b, 0x86, 0xe8, 0xcd, 0xc9, 0x60, 0xe3, 0x08, 0x1a, 0x73,
	0x97, 0x45, 0x0e, 0x18, 0xd7, 0x24, 0x55, 0x72, 0x08, 0x13, 0xbd, 0x84, 0xea, 0x74, 0x78, 0x93,
	0x10, 0xa9, 0x83, 0xed, 0x3b, 0xb3, 0x6b, 0x64, 0x44, 0x9c, 0x1d, 0xef, 0xe9, 0x1f, 0x34, 0x2f,
	0x85, 0x27, 0x4a, 0x83, 0x01, 0xe1, 0x49, 0x8c, 0xc9, 0x8f, 0x84, 0xb0, 0x65, 0x95, 0x7e, 0xb3,
	0xa0, 0xf4, 0x62, 0xed, 0x02, 0xe1, 0xfd, 0x02, 0xd4, 0x23, 0xe9, 0x37, 0x12, 0x12, 0x3a, 0xe4,
	0xe4, 0x61, 0x95, 0x8b, 0x7e, 0xe8, 0xf7, 0xf5, 0x63, 0x6e, 0xe2, 0x8d, 0x2c, 0x4f, 0x11, 0xf0,
	0x46, 0x50, 0xeb, 0x4c, 0xf8, 0x11, 0xbb, 0xf8, 0x6f, 0xf5, 0x1c, 0x30, 0x3a, 0x13, 0xae, 0x36,
	0x46, 0x98, 0xde, 0x39, 0xac, 0xc8, 0x16, 0xde, 0x5f, 0xa5, 0xd8, 0x37, 0xbd, 0xb4, 0x6f, 0xd9,
	0x8a, 0x19, 0x73, 0x2b, 0x56, 0xc9, 0x57, 0xec, 0x0b, 0x34, 0x3a, 0xd1, 0x64, 0x12, 0x85, 0x0f,
	0x93, 0xcd, 0x01, 0x03, 0x0f, 0x7f, 0xaa, 0x95, 0x13, 0xe6, 0xeb, 0xcf, 0x60, 0x15, 0x4f, 0x40,
	0x0d, 0xb0, 0x18, 0x09, 0x62, 0xbf, 0xfd, 0xfe, 0x7a, 0xc7, 0x79, 0x84, 0x6c, 0x30, 0xc9, 0xd8,
	0x6f, 0xb7, 0x77, 0x76, 0x1d, 0x4d, 0x38, 0xe2, 0x80, 0x4d, 0x7c, 0x47, 0x47, 0x2b, 0x50, 0x11,
	0x8e, 0x63, 0xf8, 0xbf, 0x2b, 0x00, 0x27, 0x8c, 0x0d, 0x32, 0x39, 0x50, 0x17, 0xea, 0xe5, 0x31,
	0x42, 0x9b, 0xff, 0x6e, 0x58, 0x79, 0xba, 0x36, 0xd6, 0x67, 0x32, 0xaa, 0x47, 0xb0, 0x28, 0x64,
	0x04, 0x75, 0xc0, 0xce, 0xe1, 0x69, 0x18, 0x2c, 0x99, 0x64, 0x1f, 0xec, 0xd2, 0x58, 0xa1, 0x67,
	0x05, 0x6c, 0x71, 0xd8, 0x6e, 0xcd, 0xd1, 0x06, 0xe8, 0x4c, 0x78, 0x9f, 0x46, 0x81, 0xf8, 0x34,
	0xad, 0xcd, 0x50, 0x72, 0x66, 0x6e, 0xa5, 0xed, 0x42, 0x5d, 0xb6, 0x2e, 0x27, 0x3e, 0x9e, 0x5f,
	0xe9, 0xbb, 0xa8, 0x9f, 0xc0, 0xc6, 0x24, 0x88, 0x42, 0xc6, 0x69, 0x12, 0x70, 0xb4, 0x08, 0xbb,
	0xfb, 0xc2, 0xbb, 0x60, 0x1e, 0x84, 0x01, 0x4d, 0xe3, 0xa5, 0xa8, 0x5d, 0xb2, 0x14, 0x75, 0x7f,
	0xed, 0xbc, 0x11, 0x5f, 0x5f, 0x6c, 0xcb, 0x7f, 0xcd, 0xf6, 0xc7, 0x78, 0x34, 0xaa, 0x49, 0xf3,
	0xdd, 0xdf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xd7, 0x41, 0xea, 0x04, 0x9b, 0x06, 0x00, 0x00,
}
