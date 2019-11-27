package main

import (
	"fmt"
	"grpc-ditto/internal/dittomock"
	"grpc-ditto/internal/logger"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

func mockCmd(ctx *cli.Context) error {
	log := logger.NewLogger()

	protoDir := ctx.String("proto")
	protofiles, err := findProtoFiles(protoDir)
	if err != nil {
		return err
	}
	if len(protofiles) == 0 {
		return fmt.Errorf("no proto files found in %s", protoDir)
	}

	p := protoparse.Parser{}
	descrs, err := p.ParseFiles(protofiles...)
	if err != nil {
		return err
	}

	mocksPath := ctx.String("mocks")
	log.Infow("loading mocks", "path", mocksPath)
	requestMatcher, err := dittomock.NewRequestMatcher(
		dittomock.WithMocksPath(mocksPath),
		dittomock.WithLogger(log),
	)
	if err != nil {
		log.Fatal(err)
	}
	grpclog.SetLoggerV2(logger.NewGrpcLogger("error"))

	server := grpc.NewServer(grpc.UnknownServiceHandler(unknownHandler))
	mockServer := &mockServer{
		descrs:  descrs,
		logger:  log,
		matcher: requestMatcher,
	}

	fileDescrs, err := mockServer.fileDescriptors()
	if err != nil {
		return fmt.Errorf("cannot parse file descriptors: %w", err)
	}
	for name, fd := range fileDescrs {
		log.Infow("register mock file", "name", name)
		proto.RegisterFile(name, fd)
	}
	for _, mockService := range mockServer.serviceDescriptors() {
		log.Infow("register mock service", "service", mockService.ServiceName)
		server.RegisterService(mockService, mockServer)
	}
	reflection.Register(server)

	port := ctx.Int("port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		_ = <-sigs
		server.GracefulStop()
	}()

	log.Infow("start server", "port", port)
	if err := server.Serve(lis); err != nil {
		return err
	}

	return nil
}

func findProtoFiles(dir string) ([]string, error) {
	protofiles := []string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".proto" {
			protofiles = append(protofiles, path)
		}
		return nil
	})

	return protofiles, err
}

func unknownHandler(srv interface{}, stream grpc.ServerStream) error {
	return status.Error(codes.Unimplemented, "unimplemented mock")
}
