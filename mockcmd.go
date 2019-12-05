package main

import (
	"fmt"
	"grpc-ditto/internal/dittomock"
	"grpc-ditto/internal/fs"
	"grpc-ditto/internal/logger"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
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
	grpclog.SetLoggerV2(logger.NewGrpcLogger("error"))

	descrs, err := parseProtoFiles(ctx)
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
		return err
	}

	mockServer := &mockServer{
		descrs:  descrs,
		logger:  log,
		matcher: requestMatcher,
	}

	fileDescrs, err := mockServer.fileDescriptors()
	if err != nil {
		return fmt.Errorf("cannot parse file descriptors: %w", err)
	}

	// registering files is required to setup reflection service
	for name, fd := range fileDescrs {
		log.Infow("register mock file", "name", name)
		proto.RegisterFile(name, fd)
	}

	server := grpc.NewServer(grpc.UnknownServiceHandler(unknownHandler))
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

func parseProtoFiles(ctx *cli.Context) ([]*desc.FileDescriptor, error) {
	protoDirs := ctx.StringSlice("proto")
	protofiles, err := findProtoFiles(protoDirs)
	if err != nil {
		return nil, err
	}
	if len(protofiles) == 0 {
		return nil, fmt.Errorf("no proto files found in %s", protoDirs)
	}

	// additional directories to look for dependencies
	for _, d := range ctx.StringSlice("protoimports") {
		protoDirs = append(protoDirs, d)
	}

	p := protoparse.Parser{
		ImportPaths: protoDirs,
		Accessor: func(filename string) (io.ReadCloser, error) {
			return fs.NewFileReader(filename)
		},
	}

	resolvedFiles, err := protoparse.ResolveFilenames(protoDirs, protofiles...)
	if err != nil {
		return nil, err
	}

	return p.ParseFiles(resolvedFiles...)
}

func findProtoFiles(dirs []string) ([]string, error) {
	protofiles := []string{}
	for _, dir := range dirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			if filepath.Ext(f.Name()) == ".proto" {
				protofiles = append(protofiles, f.Name())
			}
		}
	}

	return protofiles, nil
}

func unknownHandler(srv interface{}, stream grpc.ServerStream) error {
	return status.Error(codes.Unimplemented, "unimplemented mock")
}
