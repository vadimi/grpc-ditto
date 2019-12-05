package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"grpc-ditto/internal/dittomock"
	"grpc-ditto/internal/logger"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockServer interface {
}

type mockServer struct {
	logger  logger.Logger
	descrs  []*desc.FileDescriptor
	matcher *dittomock.RequestMatcher
}

func (s *mockServer) findMessageByMethod(method string) *desc.MethodDescriptor {
	name := method[strings.LastIndex(method, "/")+1:]
	for _, d := range s.descrs {
		for _, s := range d.GetServices() {
			methodDesc := s.FindMethodByName(name)
			if methodDesc != nil {
				return methodDesc
			}
		}
	}
	return nil
}

func (s *mockServer) fileDescriptors() (map[string][]byte, error) {
	result := map[string][]byte{}
	err := s.processDescriptiors(s.descrs, result)
	return result, err
}

func (s *mockServer) processDescriptiors(descrs []*desc.FileDescriptor, compressed map[string][]byte) error {
	for _, d := range descrs {
		if _, ok := compressed[d.GetName()]; ok {
			continue
		}
		fd := d.AsFileDescriptorProto()
		fDescBytes, err := proto.Marshal(fd)
		if err != nil {
			return err
		}
		zipFd, err := compressBytes(fDescBytes)
		if err != nil {
			return err
		}
		compressed[fd.GetName()] = zipFd

		err = s.processDescriptiors(d.GetDependencies(), compressed)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *mockServer) serviceDescriptors() []*grpc.ServiceDesc {
	result := []*grpc.ServiceDesc{}
	for _, d := range s.descrs {
		for _, service := range d.GetServices() {
			grpcSvcDesc := &grpc.ServiceDesc{
				ServiceName: service.GetFullyQualifiedName(),
				HandlerType: (*MockServer)(nil),
				Metadata:    d.AsFileDescriptorProto().GetName(),
				Methods:     []grpc.MethodDesc{},
			}

			for _, m := range service.GetMethods() {
				grpcSvcDesc.Methods = append(grpcSvcDesc.Methods, grpc.MethodDesc{
					MethodName: m.GetName(),
					Handler:    mockHandler,
				})
			}

			result = append(result, grpcSvcDesc)
		}
	}

	return result
}

func mockHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	fullMethodName, ok := grpc.Method(ctx)
	if !ok {
		return nil, errors.New("something is really wrong, method name not found in the request")
	}

	mockSrv := srv.(*mockServer)
	mockSrv.logger.Debugw("grpc call", "method", fullMethodName)

	methodDesc := mockSrv.findMessageByMethod(fullMethodName)
	if methodDesc == nil {
		return nil, status.Errorf(codes.Unimplemented, "unimplemented mock for method: %s", fullMethodName)
	}

	in := dynamic.NewMessage(methodDesc.GetInputType())
	if err := dec(in); err != nil {
		return nil, err
	}

	js, err := in.MarshalJSON()
	if err != nil {
		mockSrv.logger.Error(fmt.Errorf("input message json marshaling: %w", err))
		return nil, status.Errorf(codes.Unknown, "input message json marshaling: %s", err)
	}
	mockSrv.logger.Debugw("matching request", "req", string(js))
	respMock, err := mockSrv.matcher.Match(fullMethodName, js)
	if err != nil {
		if errors.Is(err, dittomock.ErrNotMatched) {
			mockSrv.logger.Warn("no match found")
		}
		return nil, status.Errorf(codes.Unimplemented, "unimplemented mock for method: %s", fullMethodName)
	}

	output := dynamic.NewMessage(methodDesc.GetOutputType())
	err = output.UnmarshalJSON(respMock.Body)
	return output, err
}

func compressBytes(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(src)
	if err != nil {
		return nil, err
	}
	err = zw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
