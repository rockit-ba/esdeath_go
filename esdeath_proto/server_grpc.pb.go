// 语法版本是protobuf3

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.1
// source: server.proto

// service 的 root path,客户端的protocol文件的package 的此值和server端的要一致，不然路径会匹配不上

package esdeath_proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Esdeath_AddDelayMsg_FullMethodName    = "/esdeath.Esdeath/AddDelayMsg"
	Esdeath_PullDelayMsg_FullMethodName   = "/esdeath.Esdeath/PullDelayMsg"
	Esdeath_AckDelayMsg_FullMethodName    = "/esdeath.Esdeath/AckDelayMsg"
	Esdeath_CancelDelayMsg_FullMethodName = "/esdeath.Esdeath/CancelDelayMsg"
	Esdeath_GetRole_FullMethodName        = "/esdeath.Esdeath/GetRole"
)

// EsdeathClient is the client API for Esdeath service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EsdeathClient interface {
	// 新增延迟消息
	AddDelayMsg(ctx context.Context, in *DelayMsgAdd, opts ...grpc.CallOption) (*AddDelayMsgResult, error)
	// 拉取延迟消息
	PullDelayMsg(ctx context.Context, in *DelayMsgPull, opts ...grpc.CallOption) (*PullDelayMsgResult, error)
	// 消息消费响应
	AckDelayMsg(ctx context.Context, in *DelayMsgAck, opts ...grpc.CallOption) (*AckDelayMsgResult, error)
	// 取消延迟消息
	CancelDelayMsg(ctx context.Context, in *DelayMsgCancel, opts ...grpc.CallOption) (*CancelDelayMsgResult, error)
	// 获取角色信息（leader or follower）
	GetRole(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*RoleResult, error)
}

type esdeathClient struct {
	cc grpc.ClientConnInterface
}

func NewEsdeathClient(cc grpc.ClientConnInterface) EsdeathClient {
	return &esdeathClient{cc}
}

func (c *esdeathClient) AddDelayMsg(ctx context.Context, in *DelayMsgAdd, opts ...grpc.CallOption) (*AddDelayMsgResult, error) {
	out := new(AddDelayMsgResult)
	err := c.cc.Invoke(ctx, Esdeath_AddDelayMsg_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *esdeathClient) PullDelayMsg(ctx context.Context, in *DelayMsgPull, opts ...grpc.CallOption) (*PullDelayMsgResult, error) {
	out := new(PullDelayMsgResult)
	err := c.cc.Invoke(ctx, Esdeath_PullDelayMsg_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *esdeathClient) AckDelayMsg(ctx context.Context, in *DelayMsgAck, opts ...grpc.CallOption) (*AckDelayMsgResult, error) {
	out := new(AckDelayMsgResult)
	err := c.cc.Invoke(ctx, Esdeath_AckDelayMsg_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *esdeathClient) CancelDelayMsg(ctx context.Context, in *DelayMsgCancel, opts ...grpc.CallOption) (*CancelDelayMsgResult, error) {
	out := new(CancelDelayMsgResult)
	err := c.cc.Invoke(ctx, Esdeath_CancelDelayMsg_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *esdeathClient) GetRole(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*RoleResult, error) {
	out := new(RoleResult)
	err := c.cc.Invoke(ctx, Esdeath_GetRole_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EsdeathServer is the server API for Esdeath service.
// All implementations must embed UnimplementedEsdeathServer
// for forward compatibility
type EsdeathServer interface {
	// 新增延迟消息
	AddDelayMsg(context.Context, *DelayMsgAdd) (*AddDelayMsgResult, error)
	// 拉取延迟消息
	PullDelayMsg(context.Context, *DelayMsgPull) (*PullDelayMsgResult, error)
	// 消息消费响应
	AckDelayMsg(context.Context, *DelayMsgAck) (*AckDelayMsgResult, error)
	// 取消延迟消息
	CancelDelayMsg(context.Context, *DelayMsgCancel) (*CancelDelayMsgResult, error)
	// 获取角色信息（leader or follower）
	GetRole(context.Context, *emptypb.Empty) (*RoleResult, error)
	mustEmbedUnimplementedEsdeathServer()
}

// UnimplementedEsdeathServer must be embedded to have forward compatible implementations.
type UnimplementedEsdeathServer struct {
}

func (UnimplementedEsdeathServer) AddDelayMsg(context.Context, *DelayMsgAdd) (*AddDelayMsgResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddDelayMsg not implemented")
}
func (UnimplementedEsdeathServer) PullDelayMsg(context.Context, *DelayMsgPull) (*PullDelayMsgResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PullDelayMsg not implemented")
}
func (UnimplementedEsdeathServer) AckDelayMsg(context.Context, *DelayMsgAck) (*AckDelayMsgResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AckDelayMsg not implemented")
}
func (UnimplementedEsdeathServer) CancelDelayMsg(context.Context, *DelayMsgCancel) (*CancelDelayMsgResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelDelayMsg not implemented")
}
func (UnimplementedEsdeathServer) GetRole(context.Context, *emptypb.Empty) (*RoleResult, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRole not implemented")
}
func (UnimplementedEsdeathServer) mustEmbedUnimplementedEsdeathServer() {}

// UnsafeEsdeathServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EsdeathServer will
// result in compilation errors.
type UnsafeEsdeathServer interface {
	mustEmbedUnimplementedEsdeathServer()
}

func RegisterEsdeathServer(s grpc.ServiceRegistrar, srv EsdeathServer) {
	s.RegisterService(&Esdeath_ServiceDesc, srv)
}

func _Esdeath_AddDelayMsg_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DelayMsgAdd)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EsdeathServer).AddDelayMsg(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Esdeath_AddDelayMsg_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EsdeathServer).AddDelayMsg(ctx, req.(*DelayMsgAdd))
	}
	return interceptor(ctx, in, info, handler)
}

func _Esdeath_PullDelayMsg_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DelayMsgPull)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EsdeathServer).PullDelayMsg(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Esdeath_PullDelayMsg_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EsdeathServer).PullDelayMsg(ctx, req.(*DelayMsgPull))
	}
	return interceptor(ctx, in, info, handler)
}

func _Esdeath_AckDelayMsg_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DelayMsgAck)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EsdeathServer).AckDelayMsg(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Esdeath_AckDelayMsg_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EsdeathServer).AckDelayMsg(ctx, req.(*DelayMsgAck))
	}
	return interceptor(ctx, in, info, handler)
}

func _Esdeath_CancelDelayMsg_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DelayMsgCancel)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EsdeathServer).CancelDelayMsg(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Esdeath_CancelDelayMsg_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EsdeathServer).CancelDelayMsg(ctx, req.(*DelayMsgCancel))
	}
	return interceptor(ctx, in, info, handler)
}

func _Esdeath_GetRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EsdeathServer).GetRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Esdeath_GetRole_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EsdeathServer).GetRole(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// Esdeath_ServiceDesc is the grpc.ServiceDesc for Esdeath service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Esdeath_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "esdeath.Esdeath",
	HandlerType: (*EsdeathServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddDelayMsg",
			Handler:    _Esdeath_AddDelayMsg_Handler,
		},
		{
			MethodName: "PullDelayMsg",
			Handler:    _Esdeath_PullDelayMsg_Handler,
		},
		{
			MethodName: "AckDelayMsg",
			Handler:    _Esdeath_AckDelayMsg_Handler,
		},
		{
			MethodName: "CancelDelayMsg",
			Handler:    _Esdeath_CancelDelayMsg_Handler,
		},
		{
			MethodName: "GetRole",
			Handler:    _Esdeath_GetRole_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "server.proto",
}
