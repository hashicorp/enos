// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: hashicorp/enos/v1/enos.proto

package pb

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

const (
	EnosService_GetVersion_FullMethodName                     = "/hashicorp.enos.v1.EnosService/GetVersion"
	EnosService_ValidateScenariosConfiguration_FullMethodName = "/hashicorp.enos.v1.EnosService/ValidateScenariosConfiguration"
	EnosService_ListScenarios_FullMethodName                  = "/hashicorp.enos.v1.EnosService/ListScenarios"
	EnosService_CheckScenarios_FullMethodName                 = "/hashicorp.enos.v1.EnosService/CheckScenarios"
	EnosService_GenerateScenarios_FullMethodName              = "/hashicorp.enos.v1.EnosService/GenerateScenarios"
	EnosService_LaunchScenarios_FullMethodName                = "/hashicorp.enos.v1.EnosService/LaunchScenarios"
	EnosService_DestroyScenarios_FullMethodName               = "/hashicorp.enos.v1.EnosService/DestroyScenarios"
	EnosService_RunScenarios_FullMethodName                   = "/hashicorp.enos.v1.EnosService/RunScenarios"
	EnosService_ExecScenarios_FullMethodName                  = "/hashicorp.enos.v1.EnosService/ExecScenarios"
	EnosService_OutputScenarios_FullMethodName                = "/hashicorp.enos.v1.EnosService/OutputScenarios"
	EnosService_Format_FullMethodName                         = "/hashicorp.enos.v1.EnosService/Format"
	EnosService_OperationEventStream_FullMethodName           = "/hashicorp.enos.v1.EnosService/OperationEventStream"
	EnosService_Operation_FullMethodName                      = "/hashicorp.enos.v1.EnosService/Operation"
	EnosService_ListSamples_FullMethodName                    = "/hashicorp.enos.v1.EnosService/ListSamples"
	EnosService_ObserveSample_FullMethodName                  = "/hashicorp.enos.v1.EnosService/ObserveSample"
	EnosService_OutlineScenarios_FullMethodName               = "/hashicorp.enos.v1.EnosService/OutlineScenarios"
)

// EnosServiceClient is the client API for EnosService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EnosServiceClient interface {
	GetVersion(ctx context.Context, in *GetVersionRequest, opts ...grpc.CallOption) (*GetVersionResponse, error)
	ValidateScenariosConfiguration(ctx context.Context, in *ValidateScenariosConfigurationRequest, opts ...grpc.CallOption) (*ValidateScenariosConfigurationResponse, error)
	ListScenarios(ctx context.Context, in *ListScenariosRequest, opts ...grpc.CallOption) (EnosService_ListScenariosClient, error)
	CheckScenarios(ctx context.Context, in *CheckScenariosRequest, opts ...grpc.CallOption) (*CheckScenariosResponse, error)
	GenerateScenarios(ctx context.Context, in *GenerateScenariosRequest, opts ...grpc.CallOption) (*GenerateScenariosResponse, error)
	LaunchScenarios(ctx context.Context, in *LaunchScenariosRequest, opts ...grpc.CallOption) (*LaunchScenariosResponse, error)
	DestroyScenarios(ctx context.Context, in *DestroyScenariosRequest, opts ...grpc.CallOption) (*DestroyScenariosResponse, error)
	RunScenarios(ctx context.Context, in *RunScenariosRequest, opts ...grpc.CallOption) (*RunScenariosResponse, error)
	ExecScenarios(ctx context.Context, in *ExecScenariosRequest, opts ...grpc.CallOption) (*ExecScenariosResponse, error)
	OutputScenarios(ctx context.Context, in *OutputScenariosRequest, opts ...grpc.CallOption) (*OutputScenariosResponse, error)
	Format(ctx context.Context, in *FormatRequest, opts ...grpc.CallOption) (*FormatResponse, error)
	OperationEventStream(ctx context.Context, in *OperationEventStreamRequest, opts ...grpc.CallOption) (EnosService_OperationEventStreamClient, error)
	Operation(ctx context.Context, in *OperationRequest, opts ...grpc.CallOption) (*OperationResponse, error)
	ListSamples(ctx context.Context, in *ListSamplesRequest, opts ...grpc.CallOption) (*ListSamplesResponse, error)
	ObserveSample(ctx context.Context, in *ObserveSampleRequest, opts ...grpc.CallOption) (*ObserveSampleResponse, error)
	OutlineScenarios(ctx context.Context, in *OutlineScenariosRequest, opts ...grpc.CallOption) (*OutlineScenariosResponse, error)
}

type enosServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewEnosServiceClient(cc grpc.ClientConnInterface) EnosServiceClient {
	return &enosServiceClient{cc}
}

func (c *enosServiceClient) GetVersion(ctx context.Context, in *GetVersionRequest, opts ...grpc.CallOption) (*GetVersionResponse, error) {
	out := new(GetVersionResponse)
	err := c.cc.Invoke(ctx, EnosService_GetVersion_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) ValidateScenariosConfiguration(ctx context.Context, in *ValidateScenariosConfigurationRequest, opts ...grpc.CallOption) (*ValidateScenariosConfigurationResponse, error) {
	out := new(ValidateScenariosConfigurationResponse)
	err := c.cc.Invoke(ctx, EnosService_ValidateScenariosConfiguration_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) ListScenarios(ctx context.Context, in *ListScenariosRequest, opts ...grpc.CallOption) (EnosService_ListScenariosClient, error) {
	stream, err := c.cc.NewStream(ctx, &EnosService_ServiceDesc.Streams[0], EnosService_ListScenarios_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &enosServiceListScenariosClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type EnosService_ListScenariosClient interface {
	Recv() (*EnosServiceListScenariosResponse, error)
	grpc.ClientStream
}

type enosServiceListScenariosClient struct {
	grpc.ClientStream
}

func (x *enosServiceListScenariosClient) Recv() (*EnosServiceListScenariosResponse, error) {
	m := new(EnosServiceListScenariosResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *enosServiceClient) CheckScenarios(ctx context.Context, in *CheckScenariosRequest, opts ...grpc.CallOption) (*CheckScenariosResponse, error) {
	out := new(CheckScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_CheckScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) GenerateScenarios(ctx context.Context, in *GenerateScenariosRequest, opts ...grpc.CallOption) (*GenerateScenariosResponse, error) {
	out := new(GenerateScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_GenerateScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) LaunchScenarios(ctx context.Context, in *LaunchScenariosRequest, opts ...grpc.CallOption) (*LaunchScenariosResponse, error) {
	out := new(LaunchScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_LaunchScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) DestroyScenarios(ctx context.Context, in *DestroyScenariosRequest, opts ...grpc.CallOption) (*DestroyScenariosResponse, error) {
	out := new(DestroyScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_DestroyScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) RunScenarios(ctx context.Context, in *RunScenariosRequest, opts ...grpc.CallOption) (*RunScenariosResponse, error) {
	out := new(RunScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_RunScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) ExecScenarios(ctx context.Context, in *ExecScenariosRequest, opts ...grpc.CallOption) (*ExecScenariosResponse, error) {
	out := new(ExecScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_ExecScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) OutputScenarios(ctx context.Context, in *OutputScenariosRequest, opts ...grpc.CallOption) (*OutputScenariosResponse, error) {
	out := new(OutputScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_OutputScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) Format(ctx context.Context, in *FormatRequest, opts ...grpc.CallOption) (*FormatResponse, error) {
	out := new(FormatResponse)
	err := c.cc.Invoke(ctx, EnosService_Format_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) OperationEventStream(ctx context.Context, in *OperationEventStreamRequest, opts ...grpc.CallOption) (EnosService_OperationEventStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &EnosService_ServiceDesc.Streams[1], EnosService_OperationEventStream_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &enosServiceOperationEventStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type EnosService_OperationEventStreamClient interface {
	Recv() (*OperationEventStreamResponse, error)
	grpc.ClientStream
}

type enosServiceOperationEventStreamClient struct {
	grpc.ClientStream
}

func (x *enosServiceOperationEventStreamClient) Recv() (*OperationEventStreamResponse, error) {
	m := new(OperationEventStreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *enosServiceClient) Operation(ctx context.Context, in *OperationRequest, opts ...grpc.CallOption) (*OperationResponse, error) {
	out := new(OperationResponse)
	err := c.cc.Invoke(ctx, EnosService_Operation_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) ListSamples(ctx context.Context, in *ListSamplesRequest, opts ...grpc.CallOption) (*ListSamplesResponse, error) {
	out := new(ListSamplesResponse)
	err := c.cc.Invoke(ctx, EnosService_ListSamples_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) ObserveSample(ctx context.Context, in *ObserveSampleRequest, opts ...grpc.CallOption) (*ObserveSampleResponse, error) {
	out := new(ObserveSampleResponse)
	err := c.cc.Invoke(ctx, EnosService_ObserveSample_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *enosServiceClient) OutlineScenarios(ctx context.Context, in *OutlineScenariosRequest, opts ...grpc.CallOption) (*OutlineScenariosResponse, error) {
	out := new(OutlineScenariosResponse)
	err := c.cc.Invoke(ctx, EnosService_OutlineScenarios_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EnosServiceServer is the server API for EnosService service.
// All implementations should embed UnimplementedEnosServiceServer
// for forward compatibility
type EnosServiceServer interface {
	GetVersion(context.Context, *GetVersionRequest) (*GetVersionResponse, error)
	ValidateScenariosConfiguration(context.Context, *ValidateScenariosConfigurationRequest) (*ValidateScenariosConfigurationResponse, error)
	ListScenarios(*ListScenariosRequest, EnosService_ListScenariosServer) error
	CheckScenarios(context.Context, *CheckScenariosRequest) (*CheckScenariosResponse, error)
	GenerateScenarios(context.Context, *GenerateScenariosRequest) (*GenerateScenariosResponse, error)
	LaunchScenarios(context.Context, *LaunchScenariosRequest) (*LaunchScenariosResponse, error)
	DestroyScenarios(context.Context, *DestroyScenariosRequest) (*DestroyScenariosResponse, error)
	RunScenarios(context.Context, *RunScenariosRequest) (*RunScenariosResponse, error)
	ExecScenarios(context.Context, *ExecScenariosRequest) (*ExecScenariosResponse, error)
	OutputScenarios(context.Context, *OutputScenariosRequest) (*OutputScenariosResponse, error)
	Format(context.Context, *FormatRequest) (*FormatResponse, error)
	OperationEventStream(*OperationEventStreamRequest, EnosService_OperationEventStreamServer) error
	Operation(context.Context, *OperationRequest) (*OperationResponse, error)
	ListSamples(context.Context, *ListSamplesRequest) (*ListSamplesResponse, error)
	ObserveSample(context.Context, *ObserveSampleRequest) (*ObserveSampleResponse, error)
	OutlineScenarios(context.Context, *OutlineScenariosRequest) (*OutlineScenariosResponse, error)
}

// UnimplementedEnosServiceServer should be embedded to have forward compatible implementations.
type UnimplementedEnosServiceServer struct {
}

func (UnimplementedEnosServiceServer) GetVersion(context.Context, *GetVersionRequest) (*GetVersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVersion not implemented")
}
func (UnimplementedEnosServiceServer) ValidateScenariosConfiguration(context.Context, *ValidateScenariosConfigurationRequest) (*ValidateScenariosConfigurationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ValidateScenariosConfiguration not implemented")
}
func (UnimplementedEnosServiceServer) ListScenarios(*ListScenariosRequest, EnosService_ListScenariosServer) error {
	return status.Errorf(codes.Unimplemented, "method ListScenarios not implemented")
}
func (UnimplementedEnosServiceServer) CheckScenarios(context.Context, *CheckScenariosRequest) (*CheckScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckScenarios not implemented")
}
func (UnimplementedEnosServiceServer) GenerateScenarios(context.Context, *GenerateScenariosRequest) (*GenerateScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateScenarios not implemented")
}
func (UnimplementedEnosServiceServer) LaunchScenarios(context.Context, *LaunchScenariosRequest) (*LaunchScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LaunchScenarios not implemented")
}
func (UnimplementedEnosServiceServer) DestroyScenarios(context.Context, *DestroyScenariosRequest) (*DestroyScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DestroyScenarios not implemented")
}
func (UnimplementedEnosServiceServer) RunScenarios(context.Context, *RunScenariosRequest) (*RunScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RunScenarios not implemented")
}
func (UnimplementedEnosServiceServer) ExecScenarios(context.Context, *ExecScenariosRequest) (*ExecScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExecScenarios not implemented")
}
func (UnimplementedEnosServiceServer) OutputScenarios(context.Context, *OutputScenariosRequest) (*OutputScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OutputScenarios not implemented")
}
func (UnimplementedEnosServiceServer) Format(context.Context, *FormatRequest) (*FormatResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Format not implemented")
}
func (UnimplementedEnosServiceServer) OperationEventStream(*OperationEventStreamRequest, EnosService_OperationEventStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method OperationEventStream not implemented")
}
func (UnimplementedEnosServiceServer) Operation(context.Context, *OperationRequest) (*OperationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Operation not implemented")
}
func (UnimplementedEnosServiceServer) ListSamples(context.Context, *ListSamplesRequest) (*ListSamplesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListSamples not implemented")
}
func (UnimplementedEnosServiceServer) ObserveSample(context.Context, *ObserveSampleRequest) (*ObserveSampleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ObserveSample not implemented")
}
func (UnimplementedEnosServiceServer) OutlineScenarios(context.Context, *OutlineScenariosRequest) (*OutlineScenariosResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OutlineScenarios not implemented")
}

// UnsafeEnosServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EnosServiceServer will
// result in compilation errors.
type UnsafeEnosServiceServer interface {
	mustEmbedUnimplementedEnosServiceServer()
}

func RegisterEnosServiceServer(s grpc.ServiceRegistrar, srv EnosServiceServer) {
	s.RegisterService(&EnosService_ServiceDesc, srv)
}

func _EnosService_GetVersion_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetVersionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).GetVersion(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_GetVersion_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).GetVersion(ctx, req.(*GetVersionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_ValidateScenariosConfiguration_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateScenariosConfigurationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).ValidateScenariosConfiguration(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_ValidateScenariosConfiguration_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).ValidateScenariosConfiguration(ctx, req.(*ValidateScenariosConfigurationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_ListScenarios_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ListScenariosRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(EnosServiceServer).ListScenarios(m, &enosServiceListScenariosServer{stream})
}

type EnosService_ListScenariosServer interface {
	Send(*EnosServiceListScenariosResponse) error
	grpc.ServerStream
}

type enosServiceListScenariosServer struct {
	grpc.ServerStream
}

func (x *enosServiceListScenariosServer) Send(m *EnosServiceListScenariosResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _EnosService_CheckScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).CheckScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_CheckScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).CheckScenarios(ctx, req.(*CheckScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_GenerateScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GenerateScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).GenerateScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_GenerateScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).GenerateScenarios(ctx, req.(*GenerateScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_LaunchScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LaunchScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).LaunchScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_LaunchScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).LaunchScenarios(ctx, req.(*LaunchScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_DestroyScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DestroyScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).DestroyScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_DestroyScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).DestroyScenarios(ctx, req.(*DestroyScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_RunScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).RunScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_RunScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).RunScenarios(ctx, req.(*RunScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_ExecScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ExecScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).ExecScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_ExecScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).ExecScenarios(ctx, req.(*ExecScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_OutputScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OutputScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).OutputScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_OutputScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).OutputScenarios(ctx, req.(*OutputScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_Format_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FormatRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).Format(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_Format_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).Format(ctx, req.(*FormatRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_OperationEventStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(OperationEventStreamRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(EnosServiceServer).OperationEventStream(m, &enosServiceOperationEventStreamServer{stream})
}

type EnosService_OperationEventStreamServer interface {
	Send(*OperationEventStreamResponse) error
	grpc.ServerStream
}

type enosServiceOperationEventStreamServer struct {
	grpc.ServerStream
}

func (x *enosServiceOperationEventStreamServer) Send(m *OperationEventStreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _EnosService_Operation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OperationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).Operation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_Operation_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).Operation(ctx, req.(*OperationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_ListSamples_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListSamplesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).ListSamples(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_ListSamples_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).ListSamples(ctx, req.(*ListSamplesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_ObserveSample_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ObserveSampleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).ObserveSample(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_ObserveSample_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).ObserveSample(ctx, req.(*ObserveSampleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnosService_OutlineScenarios_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OutlineScenariosRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnosServiceServer).OutlineScenarios(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnosService_OutlineScenarios_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnosServiceServer).OutlineScenarios(ctx, req.(*OutlineScenariosRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// EnosService_ServiceDesc is the grpc.ServiceDesc for EnosService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var EnosService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "hashicorp.enos.v1.EnosService",
	HandlerType: (*EnosServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetVersion",
			Handler:    _EnosService_GetVersion_Handler,
		},
		{
			MethodName: "ValidateScenariosConfiguration",
			Handler:    _EnosService_ValidateScenariosConfiguration_Handler,
		},
		{
			MethodName: "CheckScenarios",
			Handler:    _EnosService_CheckScenarios_Handler,
		},
		{
			MethodName: "GenerateScenarios",
			Handler:    _EnosService_GenerateScenarios_Handler,
		},
		{
			MethodName: "LaunchScenarios",
			Handler:    _EnosService_LaunchScenarios_Handler,
		},
		{
			MethodName: "DestroyScenarios",
			Handler:    _EnosService_DestroyScenarios_Handler,
		},
		{
			MethodName: "RunScenarios",
			Handler:    _EnosService_RunScenarios_Handler,
		},
		{
			MethodName: "ExecScenarios",
			Handler:    _EnosService_ExecScenarios_Handler,
		},
		{
			MethodName: "OutputScenarios",
			Handler:    _EnosService_OutputScenarios_Handler,
		},
		{
			MethodName: "Format",
			Handler:    _EnosService_Format_Handler,
		},
		{
			MethodName: "Operation",
			Handler:    _EnosService_Operation_Handler,
		},
		{
			MethodName: "ListSamples",
			Handler:    _EnosService_ListSamples_Handler,
		},
		{
			MethodName: "ObserveSample",
			Handler:    _EnosService_ObserveSample_Handler,
		},
		{
			MethodName: "OutlineScenarios",
			Handler:    _EnosService_OutlineScenarios_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ListScenarios",
			Handler:       _EnosService_ListScenarios_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "OperationEventStream",
			Handler:       _EnosService_OperationEventStream_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "hashicorp/enos/v1/enos.proto",
}
