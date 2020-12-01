package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/vadimi/grpc-ditto/internal/dittomock"
	"github.com/vadimi/grpc-ditto/internal/logger"
	"github.com/vadimi/grpc-ditto/testdata/greet"
	_ "github.com/vadimi/grpc-ditto/testdata/greet"
	"google.golang.org/grpc"
)

func TestMockServer(t *testing.T) {
	d, err := findFileDescriptor("greet.proto")
	if err != nil {
		t.Fatal(err)
	}

	log := logger.NewLogger()
	requestMatcher, err := dittomock.NewRequestMatcher(
		dittomock.WithMocks([]dittomock.DittoMock{greetMock()}),
		dittomock.WithLogger(log),
	)

	if err != nil {
		t.Fatal(err)
	}

	s := &mockServer{
		descrs:  []*desc.FileDescriptor{d},
		logger:  log,
		matcher: requestMatcher,
	}

	server := grpc.NewServer()
	for _, mockService := range s.serviceDescriptors() {
		server.RegisterService(mockService, s)
	}

	_, addr, err := createListener(server)
	if err != nil {
		t.Fatal(err)
	}
	defer stopTestServer(server)

	cc, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := greet.NewGreeterClient(cc)
	resp, _ := client.SayHello(context.Background(), &greet.HelloRequest{
		Name: "Bob",
	})

	if resp.Message != "hello Bob" {
		t.Error("invalid resp")
	}
}

func createListener(server *grpc.Server) (*grpc.Server, string, error) {
	port := 0
	if l, err := net.Listen("tcp", "127.0.0.1:0"); err != nil {
		return nil, "", err
	} else {
		port = l.Addr().(*net.TCPAddr).Port
		go server.Serve(l)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	return server, addr, nil
}

func greetMock() dittomock.DittoMock {
	return dittomock.DittoMock{
		Request: &dittomock.DittoRequest{
			Method: "/greet.Greeter/SayHello",
			BodyPatterns: []dittomock.DittoBodyPattern{
				{
					MatchesJsonPath: &dittomock.JSONPathWrapper{
						JSONPathMessage: dittomock.JSONPathMessage{
							Expression: "$.name",
							Equals:     "Bob",
						},
					},
				},
			},
		},
		Response: &dittomock.DittoResponse{
			Body: []byte(`{ "message": "hello Bob" }`),
		},
	}
}

func stopTestServer(s *grpc.Server) {
	if s == nil {
		return
	}

	timer := time.AfterFunc(time.Duration(15*time.Second), func() {
		s.Stop()
	})
	defer timer.Stop()
	s.GracefulStop()
}
