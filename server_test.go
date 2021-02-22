package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vadimi/grpc-ditto/internal/dittomock"
	"github.com/vadimi/grpc-ditto/internal/logger"
	"github.com/vadimi/grpc-ditto/testdata/greet"
	_ "github.com/vadimi/grpc-ditto/testdata/greet"
	"github.com/vadimi/grpc-ditto/testdata/hello"
	apicode "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	testServer *grpc.Server
	testAddr   string
)

func TestMain(m *testing.M) {
	server, addr, err := startTestServer()
	if err != nil {
		panic(err)
	}

	testServer = server
	testAddr = addr

	defer stopTestServer(testServer)
	os.Exit(m.Run())
}

func TestMockServerUnaryErr(t *testing.T) {
	cc, err := grpc.Dial(testAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := greet.NewGreeterClient(cc)
	_, err = client.SayHello(context.Background(), &greet.HelloRequest{
		Name: "John",
	})

	require.Error(t, err)
	errStatus, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, errStatus.Code())
}

func TestMockServerUnarySuccess(t *testing.T) {
	cc, err := grpc.Dial(testAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := greet.NewGreeterClient(cc)
	resp, _ := client.SayHello(context.Background(), &greet.HelloRequest{
		Name: "Bob",
	})

	assert.Equal(t, "hello Bob", resp.Message)
}

func TestMockServerStreamingSuccess(t *testing.T) {
	cc, err := grpc.Dial(testAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := hello.NewHelloServiceClient(cc)
	resp, _ := client.Hello(context.Background(), &hello.HelloRequest{
		Name: "all",
	})

	var messages []*hello.HelloResponse
	for {
		msg, err := resp.Recv()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)

		messages = append(messages, msg)
	}

	require.Len(t, messages, 2)
	assert.Equal(t, "hello Bob", messages[0].GetName())
	assert.Equal(t, "hello John", messages[1].GetName())
}

func TestMockServerBidiStreamingSuccess(t *testing.T) {
	cc, err := grpc.Dial(testAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := hello.NewHelloServiceClient(cc)
	stream, err := client.HelloMulti(context.Background())
	require.NoError(t, err)

	require.NoError(t, stream.Send(&hello.HelloRequest{Name: "Bob"}))
	require.NoError(t, stream.Send(&hello.HelloRequest{Name: "John"}))
	stream.CloseSend()

	var messages []*hello.HelloResponse
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)

		messages = append(messages, msg)
	}

	require.Len(t, messages, 2)
	assert.Equal(t, "hello Bob", messages[0].GetName())
	assert.Equal(t, "hello John", messages[1].GetName())
}

func TestMockServerBidiStreamingError(t *testing.T) {
	cc, err := grpc.Dial(testAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := hello.NewHelloServiceClient(cc)
	stream, err := client.HelloMulti(context.Background())
	require.NoError(t, err)

	require.NoError(t, stream.Send(&hello.HelloRequest{Name: "Tom"}))
	require.NoError(t, stream.Send(&hello.HelloRequest{Name: "Jerry"}))
	stream.CloseSend()

	index := 0
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if index == 0 {
			require.NoError(t, err)
			assert.Equal(t, "hello Tom", msg.GetName())
		} else {
			require.Error(t, err)
			errStatus, _ := status.FromError(err)
			assert.Equal(t, codes.NotFound, errStatus.Code())
			break
		}

		index++
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

func greetNotFoundMock() dittomock.DittoMock {
	return dittomock.DittoMock{
		Request: &dittomock.DittoRequest{
			Method: "/greet.Greeter/SayHello",
			BodyPatterns: []dittomock.DittoBodyPattern{
				{
					MatchesJsonPath: &dittomock.JSONPathWrapper{
						JSONPathMessage: dittomock.JSONPathMessage{
							Expression: "$.name",
							Equals:     "John",
						},
					},
				},
			},
		},
		Response: []*dittomock.DittoResponse{
			{
				Status: &dittomock.RpcStatus{
					Code:    codes.Code(apicode.Code_NOT_FOUND),
					Message: "user not found",
				},
			},
		},
	}
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
		Response: []*dittomock.DittoResponse{
			{
				Body: []byte(`{ "message": "hello Bob" }`),
			},
		},
	}
}

func helloStreamMock() dittomock.DittoMock {
	return dittomock.DittoMock{
		Request: &dittomock.DittoRequest{
			Method: "/ditto.example.HelloService/Hello",
			BodyPatterns: []dittomock.DittoBodyPattern{
				{
					MatchesJsonPath: &dittomock.JSONPathWrapper{
						JSONPathMessage: dittomock.JSONPathMessage{
							Expression: "$.name",
							Equals:     "all",
						},
					},
				},
			},
		},
		Response: []*dittomock.DittoResponse{
			{
				Body: []byte(`{ "name": "hello Bob" }`),
			},
			{
				Body: []byte(`{ "name": "hello John" }`),
			},
		},
	}
}

func helloBidiStreamMock() dittomock.DittoMock {
	return dittomock.DittoMock{
		Request: &dittomock.DittoRequest{
			Method: "/ditto.example.HelloService/HelloMulti",
			BodyPatterns: []dittomock.DittoBodyPattern{
				{
					MatchesJsonPath: &dittomock.JSONPathWrapper{
						JSONPathMessage: dittomock.JSONPathMessage{
							Expression: "$[0].name",
							Equals:     "Bob",
						},
					},
				},
				{
					MatchesJsonPath: &dittomock.JSONPathWrapper{
						JSONPathMessage: dittomock.JSONPathMessage{
							Expression: "$[1].name",
							Equals:     "John",
						},
					},
				},
			},
		},
		Response: []*dittomock.DittoResponse{
			{
				Body: []byte(`{ "name": "hello Bob" }`),
			},
			{
				Body: []byte(`{ "name": "hello John" }`),
			},
		},
	}
}

func helloBidiStreamMockErr() dittomock.DittoMock {
	return dittomock.DittoMock{
		Request: &dittomock.DittoRequest{
			Method: "/ditto.example.HelloService/HelloMulti",
			BodyPatterns: []dittomock.DittoBodyPattern{
				{
					MatchesJsonPath: &dittomock.JSONPathWrapper{
						JSONPathMessage: dittomock.JSONPathMessage{
							Expression: "$[0].name",
							Equals:     "Tom",
						},
					},
				},
				{
					MatchesJsonPath: &dittomock.JSONPathWrapper{
						JSONPathMessage: dittomock.JSONPathMessage{
							Expression: "$[1].name",
							Equals:     "Jerry",
						},
					},
				},
			},
		},
		Response: []*dittomock.DittoResponse{
			{
				Body: []byte(`{ "name": "hello Tom" }`),
			},
			{
				Status: &dittomock.RpcStatus{
					Code: codes.NotFound,
				},
			},
		},
	}
}

func startTestServer() (*grpc.Server, string, error) {
	log := logger.NewLogger()
	requestMatcher, err := dittomock.NewRequestMatcher(
		dittomock.WithMocks([]dittomock.DittoMock{
			greetMock(),
			greetNotFoundMock(),
			helloStreamMock(),
			helloBidiStreamMock(),
			helloBidiStreamMockErr(),
		}),
		dittomock.WithLogger(log),
	)

	if err != nil {
		return nil, "", err
	}

	greetDescr, err := findFileDescriptor("greet.proto")
	if err != nil {
		return nil, "", err
	}

	helloDescr, err := findFileDescriptor("hello.proto")
	if err != nil {
		return nil, "", err
	}

	s := &mockServer{
		descrs:  []*desc.FileDescriptor{greetDescr, helloDescr},
		logger:  log,
		matcher: requestMatcher,
	}

	server := grpc.NewServer()
	for _, mockService := range s.serviceDescriptors() {
		server.RegisterService(mockService, s)
	}

	_, addr, err := createListener(server)
	if err != nil {
		return nil, "", err
	}

	return server, addr, nil
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
