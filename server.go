package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vadimi/grpc-ditto/internal/dittomock"
	"github.com/vadimi/grpc-ditto/internal/logger"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"

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

func (s *mockServer) processDescriptors(descrs []*desc.FileDescriptor, result map[string][]byte) error {
	for _, d := range descrs {
		if _, ok := result[d.GetName()]; ok {
			continue
		}
		fd := d.AsFileDescriptorProto()

		fDescBytes, err := proto.MarshalOptions{}.Marshal(fd)
		if err != nil {
			return err
		}

		result[fd.GetName()] = fDescBytes

		err = s.processDescriptors(d.GetDependencies(), result)
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

	inputJS, err := readInput(stream, methodDesc)
	if err != nil {
		mockSrv.logger.Error(fmt.Errorf("input message json marshaling: %w", err))
		return err
	}

	mockSrv.logger.Debugw("matching request", "req", string(inputJS))
	mock, err := mockSrv.matcher.Match(fullMethodName, inputJS)
	if err != nil {
		if errors.Is(err, dittomock.ErrNotMatched) {
			mockSrv.logger.Warn("no match found")
		} else {
			mockSrv.logger.Error(err)
		}
		return status.Errorf(codes.Unimplemented, "unimplemented mock for method: %s", fullMethodName)
	}

	// if methodDesc.IsServerStreaming() {
	// 	if respMock.Body[0] != '[' {
	// 		err := fmt.Errorf("server streaming method requires array in response body: %s", fullMethodName)
	// 		mockSrv.logger.Error(err)
	// 		return status.Error(codes.Unimplemented, err.Error())
	// 	}

	// 	var arr []json.RawMessage
	// 	if err := json.Unmarshal(respMock.Body, &arr); err != nil {
	// 		return status.Errorf(codes.Unknown, "output message json unmarshaling: %s", err)
	// 	}
	// 	outputMessages = arr
	// }

	for _, resp := range mock.Response {
		if resp.Status != nil {
			return status.Error(resp.Status.Code, resp.Status.Message)
		}

		output := dynamic.NewMessage(methodDesc.GetOutputType())
		err = output.UnmarshalJSON(resp.Body)
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

func readInput(stream grpc.ServerStream, methodDesc *desc.MethodDescriptor) ([]byte, error) {
	// for loop supports both client streaming and unary messages
	// io.EOF means it's the last message on the stream
	var inMessages []json.RawMessage
	for {
		in := dynamic.NewMessage(methodDesc.GetInputType())

		err := stream.RecvMsg(in)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		js, err := in.MarshalJSONPB(&jsonpb.Marshaler{OrigName: true, EmitDefaults: true})
		if err != nil {
			return nil, status.Errorf(codes.Unknown, "input message json marshaling: %s", err)
		}

		inMessages = append(inMessages, js)
	}

	var inputJS []byte
	if methodDesc.IsClientStreaming() {
		res, err := json.Marshal(inMessages)
		if err != nil {
			return nil, err
		}
		inputJS = res
	} else {
		inputJS = inMessages[0]
	}

	return inputJS, nil
}

func healthCheckFileDescriptor() (*desc.FileDescriptor, error) {
	return findFileDescriptor("grpc/health/v1/health.proto")
}

func findFileDescriptor(name string) (*desc.FileDescriptor, error) {
	fileDesc, err := protoregistry.GlobalFiles.FindFileByPath(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	fdproto := protodesc.ToFileDescriptorProto(fileDesc)

	return desc.CreateFileDescriptor(fdproto)
}
