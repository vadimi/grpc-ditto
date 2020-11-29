package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vadimi/grpc-ditto/internal/dittomock"
	"github.com/vadimi/grpc-ditto/internal/logger"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	_ "google.golang.org/grpc/health/grpc_health_v1"
)

// MockServer is an interface that grpc reflection expects to register types
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
	err := s.processDescriptors(s.descrs, result)
	return result, err
}

func (s *mockServer) processDescriptors(descrs []*desc.FileDescriptor, compressed map[string][]byte) error {
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

		err = s.processDescriptors(d.GetDependencies(), compressed)
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
				Streams:     []grpc.StreamDesc{},
			}

			for _, m := range service.GetMethods() {
				// in grpc-go all methods are implemented as streams
				// so we just need one handler to rule them all
				grpcSvcDesc.Streams = append(grpcSvcDesc.Streams, grpc.StreamDesc{
					StreamName:    m.GetName(),
					Handler:       mockServerStreamHandler,
					ServerStreams: m.IsServerStreaming(),
				})
			}

			result = append(result, grpcSvcDesc)
		}
	}

	return result
}

func mockServerStreamHandler(srv interface{}, stream grpc.ServerStream) error {
	fullMethodName, ok := grpc.Method(stream.Context())
	if !ok {
		return errors.New("something is really wrong, method name not found in the request")
	}

	mockSrv := srv.(*mockServer)
	mockSrv.logger.Debugw("grpc call", "method", fullMethodName)

	methodDesc := mockSrv.findMessageByMethod(fullMethodName)
	if methodDesc == nil {
		return status.Errorf(codes.Unimplemented, "unimplemented mock for method: %s", fullMethodName)
	}

	in := dynamic.NewMessage(methodDesc.GetInputType())

	if err := stream.RecvMsg(in); err != nil {
		return err
	}

	js, err := in.MarshalJSONPB(&jsonpb.Marshaler{OrigName: true})
	if err != nil {
		mockSrv.logger.Error(fmt.Errorf("input message json marshaling: %w", err))
		return status.Errorf(codes.Unknown, "input message json marshaling: %s", err)
	}
	mockSrv.logger.Debugw("matching request", "req", string(js))
	respMock, err := mockSrv.matcher.Match(fullMethodName, js)
	if err != nil {
		if errors.Is(err, dittomock.ErrNotMatched) {
			mockSrv.logger.Warn("no match found")
		} else {

			mockSrv.logger.Error(err)
		}
		return status.Errorf(codes.Unimplemented, "unimplemented mock for method: %s", fullMethodName)
	}

	output := dynamic.NewMessage(methodDesc.GetOutputType())

	outputMessages := []json.RawMessage{respMock.Body}

	if methodDesc.IsServerStreaming() {
		if respMock.Body[0] != '[' {
			err := fmt.Errorf("server streaming method requires array in response body: %s", fullMethodName)
			mockSrv.logger.Error(err)
			return status.Error(codes.Unimplemented, err.Error())
		}

		var arr []json.RawMessage
		if err := json.Unmarshal(respMock.Body, &arr); err != nil {
			return status.Errorf(codes.Unknown, "output message json unmarshaling: %s", err)
		}
		outputMessages = arr
	}

	for _, msg := range outputMessages {
		err = output.UnmarshalJSON(msg)
		if err != nil {
			mockSrv.logger.Error(err)
			return err
		}

		err = stream.SendMsg(output)
		if err != nil {
			return err
		}
	}

	return nil
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

func uncompressBytes(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	zr, err := gzip.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	_, err = io.Copy(&buf, zr)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func healthCheckFileDescriptor() (*desc.FileDescriptor, error) {
	fd := proto.FileDescriptor("grpc/health/v1/health.proto")
	fdRaw, err := uncompressBytes(fd)
	if err != nil {
		return nil, fmt.Errorf("uncompress health check descriptor: %w", err)
	}
	fdp := &descriptor.FileDescriptorProto{}
	err = proto.Unmarshal(fdRaw, fdp)
	if err != nil {
		return nil, fmt.Errorf("proto unmarshal health check descriptor: %w", err)
	}
	return desc.CreateFileDescriptor(fdp)
}
