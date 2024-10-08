// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.21.9
// source: mocking_service.proto

package api

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	MockingService_AddMock_FullMethodName = "/grpcditto.api.MockingService/AddMock"
	MockingService_Clear_FullMethodName   = "/grpcditto.api.MockingService/Clear"
)

// MockingServiceClient is the client API for MockingService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MockingServiceClient interface {
	// AddMock adds new mock to the server
	AddMock(ctx context.Context, in *AddMockRequest, opts ...grpc.CallOption) (*AddMockResponse, error)
	// Delete all mocks
	Clear(ctx context.Context, in *ClearRequest, opts ...grpc.CallOption) (*ClearResponse, error)
}

type mockingServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMockingServiceClient(cc grpc.ClientConnInterface) MockingServiceClient {
	return &mockingServiceClient{cc}
}

func (c *mockingServiceClient) AddMock(ctx context.Context, in *AddMockRequest, opts ...grpc.CallOption) (*AddMockResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AddMockResponse)
	err := c.cc.Invoke(ctx, MockingService_AddMock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mockingServiceClient) Clear(ctx context.Context, in *ClearRequest, opts ...grpc.CallOption) (*ClearResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ClearResponse)
	err := c.cc.Invoke(ctx, MockingService_Clear_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MockingServiceServer is the server API for MockingService service.
// All implementations must embed UnimplementedMockingServiceServer
// for forward compatibility.
type MockingServiceServer interface {
	// AddMock adds new mock to the server
	AddMock(context.Context, *AddMockRequest) (*AddMockResponse, error)
	// Delete all mocks
	Clear(context.Context, *ClearRequest) (*ClearResponse, error)
	mustEmbedUnimplementedMockingServiceServer()
}

// UnimplementedMockingServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMockingServiceServer struct{}

func (UnimplementedMockingServiceServer) AddMock(context.Context, *AddMockRequest) (*AddMockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddMock not implemented")
}
func (UnimplementedMockingServiceServer) Clear(context.Context, *ClearRequest) (*ClearResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Clear not implemented")
}
func (UnimplementedMockingServiceServer) mustEmbedUnimplementedMockingServiceServer() {}
func (UnimplementedMockingServiceServer) testEmbeddedByValue()                        {}

// UnsafeMockingServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MockingServiceServer will
// result in compilation errors.
type UnsafeMockingServiceServer interface {
	mustEmbedUnimplementedMockingServiceServer()
}

func RegisterMockingServiceServer(s grpc.ServiceRegistrar, srv MockingServiceServer) {
	// If the following call pancis, it indicates UnimplementedMockingServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&MockingService_ServiceDesc, srv)
}

func _MockingService_AddMock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddMockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MockingServiceServer).AddMock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MockingService_AddMock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MockingServiceServer).AddMock(ctx, req.(*AddMockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MockingService_Clear_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClearRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MockingServiceServer).Clear(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MockingService_Clear_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MockingServiceServer).Clear(ctx, req.(*ClearRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// MockingService_ServiceDesc is the grpc.ServiceDesc for MockingService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MockingService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "grpcditto.api.MockingService",
	HandlerType: (*MockingServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddMock",
			Handler:    _MockingService_AddMock_Handler,
		},
		{
			MethodName: "Clear",
			Handler:    _MockingService_Clear_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "mocking_service.proto",
}
