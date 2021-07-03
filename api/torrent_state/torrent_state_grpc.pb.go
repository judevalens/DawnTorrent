// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package torrent_state

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ControlTorrentClient is the client API for ControlTorrent service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ControlTorrentClient interface {
	AddTorrent(ctx context.Context, in *TorrentInfo, opts ...grpc.CallOption) (*SingleTorrentState, error)
	RemoveTorrent(ctx context.Context, in *TorrentInfo, opts ...grpc.CallOption) (*SingleTorrentState, error)
	ControlTorrent(ctx context.Context, in *Control, opts ...grpc.CallOption) (*SingleTorrentState, error)
	Subscribe(ctx context.Context, in *Subscription, opts ...grpc.CallOption) (ControlTorrent_SubscribeClient, error)
}

type controlTorrentClient struct {
	cc grpc.ClientConnInterface
}

func NewControlTorrentClient(cc grpc.ClientConnInterface) ControlTorrentClient {
	return &controlTorrentClient{cc}
}

func (c *controlTorrentClient) AddTorrent(ctx context.Context, in *TorrentInfo, opts ...grpc.CallOption) (*SingleTorrentState, error) {
	out := new(SingleTorrentState)
	err := c.cc.Invoke(ctx, "/main.ControlTorrent/AddTorrent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlTorrentClient) RemoveTorrent(ctx context.Context, in *TorrentInfo, opts ...grpc.CallOption) (*SingleTorrentState, error) {
	out := new(SingleTorrentState)
	err := c.cc.Invoke(ctx, "/main.ControlTorrent/RemoveTorrent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlTorrentClient) ControlTorrent(ctx context.Context, in *Control, opts ...grpc.CallOption) (*SingleTorrentState, error) {
	out := new(SingleTorrentState)
	err := c.cc.Invoke(ctx, "/main.ControlTorrent/ControlTorrent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controlTorrentClient) Subscribe(ctx context.Context, in *Subscription, opts ...grpc.CallOption) (ControlTorrent_SubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &ControlTorrent_ServiceDesc.Streams[0], "/main.ControlTorrent/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &controlTorrentSubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type ControlTorrent_SubscribeClient interface {
	Recv() (*SingleTorrentState, error)
	grpc.ClientStream
}

type controlTorrentSubscribeClient struct {
	grpc.ClientStream
}

func (x *controlTorrentSubscribeClient) Recv() (*SingleTorrentState, error) {
	m := new(SingleTorrentState)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ControlTorrentServer is the server API for ControlTorrent service.
// All implementations must embed UnimplementedControlTorrentServer
// for forward compatibility
type ControlTorrentServer interface {
	AddTorrent(context.Context, *TorrentInfo) (*SingleTorrentState, error)
	RemoveTorrent(context.Context, *TorrentInfo) (*SingleTorrentState, error)
	ControlTorrent(context.Context, *Control) (*SingleTorrentState, error)
	Subscribe(*Subscription, ControlTorrent_SubscribeServer) error
	mustEmbedUnimplementedControlTorrentServer()
}

// UnimplementedControlTorrentServer must be embedded to have forward compatible implementations.
type UnimplementedControlTorrentServer struct {
}

func (UnimplementedControlTorrentServer) AddTorrent(context.Context, *TorrentInfo) (*SingleTorrentState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddTorrent not implemented")
}
func (UnimplementedControlTorrentServer) RemoveTorrent(context.Context, *TorrentInfo) (*SingleTorrentState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveTorrent not implemented")
}
func (UnimplementedControlTorrentServer) ControlTorrent(context.Context, *Control) (*SingleTorrentState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ControlTorrent not implemented")
}
func (UnimplementedControlTorrentServer) Subscribe(*Subscription, ControlTorrent_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}
func (UnimplementedControlTorrentServer) mustEmbedUnimplementedControlTorrentServer() {}

// UnsafeControlTorrentServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ControlTorrentServer will
// result in compilation errors.
type UnsafeControlTorrentServer interface {
	mustEmbedUnimplementedControlTorrentServer()
}

func RegisterControlTorrentServer(s grpc.ServiceRegistrar, srv ControlTorrentServer) {
	s.RegisterService(&ControlTorrent_ServiceDesc, srv)
}

func _ControlTorrent_AddTorrent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TorrentInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlTorrentServer).AddTorrent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/main.ControlTorrent/AddTorrent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlTorrentServer).AddTorrent(ctx, req.(*TorrentInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlTorrent_RemoveTorrent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TorrentInfo)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlTorrentServer).RemoveTorrent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/main.ControlTorrent/RemoveTorrent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlTorrentServer).RemoveTorrent(ctx, req.(*TorrentInfo))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlTorrent_ControlTorrent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Control)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControlTorrentServer).ControlTorrent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/main.ControlTorrent/ControlTorrent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControlTorrentServer).ControlTorrent(ctx, req.(*Control))
	}
	return interceptor(ctx, in, info, handler)
}

func _ControlTorrent_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Subscription)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ControlTorrentServer).Subscribe(m, &controlTorrentSubscribeServer{stream})
}

type ControlTorrent_SubscribeServer interface {
	Send(*SingleTorrentState) error
	grpc.ServerStream
}

type controlTorrentSubscribeServer struct {
	grpc.ServerStream
}

func (x *controlTorrentSubscribeServer) Send(m *SingleTorrentState) error {
	return x.ServerStream.SendMsg(m)
}

// ControlTorrent_ServiceDesc is the grpc.ServiceDesc for ControlTorrent service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ControlTorrent_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "main.ControlTorrent",
	HandlerType: (*ControlTorrentServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddTorrent",
			Handler:    _ControlTorrent_AddTorrent_Handler,
		},
		{
			MethodName: "RemoveTorrent",
			Handler:    _ControlTorrent_RemoveTorrent_Handler,
		},
		{
			MethodName: "ControlTorrent",
			Handler:    _ControlTorrent_ControlTorrent_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Subscribe",
			Handler:       _ControlTorrent_Subscribe_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "torrent_state.proto",
}
