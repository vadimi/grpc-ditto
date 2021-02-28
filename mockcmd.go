package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vadimi/grpc-ditto/api"
	"github.com/vadimi/grpc-ditto/internal/dittomock"
	"github.com/vadimi/grpc-ditto/internal/fs"
	"github.com/vadimi/grpc-ditto/internal/logger"
	"github.com/vadimi/grpc-ditto/internal/services"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoimpl"
)

const (
	maxShutdownTime = 30 * time.Second
)

func newMockCmd(log logger.Logger) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		grpclog.SetLoggerV2(logger.NewGrpcLogger(log, "error"))

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

		// health check service
		// implement it using mocks to allow using/overriding health mocks for other purposes
		healthcheckDescr, err := healthCheckFileDescriptor()
		if err != nil {
			return err
		}
		descrs = append(descrs, healthcheckDescr)
		requestMatcher.AddMock(healthCheckMocks())

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
			_, err := protoregistry.GlobalFiles.FindFileByPath(name)
			if err == protoregistry.NotFound {
				// DescBuilder also registers files
				protoimpl.DescBuilder{RawDescriptor: fd}.Build()
			}
		}

		validator := &mockValidator{
			findMethodFunc: mockServer.findMethodByName,
		}

		if err := validator.Validate(requestMatcher.Mocks()); err != nil {
			return err
		}

		server := grpc.NewServer(grpc.UnknownServiceHandler(unknownHandler))
		for _, mockService := range mockServer.serviceDescriptors() {
			log.Infow("register mock service", "service", mockService.ServiceName)
			server.RegisterService(mockService, mockServer)
		}

		api.RegisterMockingServiceServer(
			server,
			services.NewMockingService(requestMatcher, validator, log),
		)

		reflection.Register(server)

		port := ctx.Int("port")
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return err
		}
		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
			<-sigs
			log.Info("stopping service")
			timer := time.AfterFunc(maxShutdownTime, func() {
				log.Info("force stop gRPC server")
				server.Stop()
			})
			defer timer.Stop()
			server.GracefulStop()
		}()

		log.Infow("start server", "port", port)
		if err := server.Serve(lis); err != nil {
			return err
		}

		return nil
	}
}

func parseProtoFiles(ctx *cli.Context) ([]*desc.FileDescriptor, error) {
	protoPaths := ctx.StringSlice("proto")
	protofiles, err := findProtoFiles(protoPaths)
	if err != nil {
		return nil, err
	}
	if len(protofiles) == 0 {
		return nil, fmt.Errorf("no proto files found in %s", protoPaths)
	}

	protoDirs := resolveProtoDirs(protoPaths)
	// additional directories to look for dependencies
	protoDirs = append(protoDirs, ctx.StringSlice("protoimports")...)

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

func resolveProtoDirs(paths []string) []string {
	tmp := map[string]struct{}{}
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil {
			if fi.IsDir() {
				tmp[p] = struct{}{}
			} else {
				tmp[filepath.Dir(p)] = struct{}{}
			}
		}
	}

	var res []string
	for k := range tmp {
		res = append(res, k)
	}

	return res
}

func findProtoFiles(paths []string) ([]string, error) {
	protofiles := []string{}
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			if isProto(fi.Name()) {
				protofiles = append(protofiles, fi.Name())
			}
			continue
		}

		files, err := ioutil.ReadDir(p)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			if isProto(f.Name()) {
				protofiles = append(protofiles, f.Name())
			}
		}
	}

	return protofiles, nil
}

func unknownHandler(srv interface{}, stream grpc.ServerStream) error {
	return status.Error(codes.Unimplemented, "unimplemented mock")
}

func healthCheckMocks() dittomock.DittoMock {
	return dittomock.DittoMock{
		Request: &dittomock.DittoRequest{
			Method: "/grpc.health.v1.Health/Check",
			BodyPatterns: []dittomock.DittoBodyPattern{
				{
					EqualToJson: []byte("{}"),
				},
			},
		},
		Response: []*dittomock.DittoResponse{
			{
				Body: []byte(`{ "status": "SERVING" }`),
			},
		},
	}
}

func isProto(p string) bool {
	return strings.ToLower(filepath.Ext(p)) == ".proto"
}
