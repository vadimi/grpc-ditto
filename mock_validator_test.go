package main

import (
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vadimi/grpc-ditto/internal/dittomock"
	"github.com/vadimi/grpc-ditto/internal/logger"
)

func TestMockServiceValidateSuccess(t *testing.T) {
	log := logger.NewLogger()
	requestMatcher, err := dittomock.NewRequestMatcher(
		dittomock.WithMocks([]dittomock.DittoMock{
			greetMock(),
		}),
		dittomock.WithLogger(log),
	)

	require.NoError(t, err)

	greetDescr, err := findFileDescriptor("greet.proto")
	require.NoError(t, err)

	s := &mockServer{
		descrs:  []*desc.FileDescriptor{greetDescr},
		logger:  log,
		matcher: requestMatcher,
	}

	validator := &mockValidator{
		findMethodFunc: s.findMethodByName,
	}

	err = validator.Validate(requestMatcher.Mocks())
	assert.NoError(t, err)
}

func TestMockServiceValidateFailureInvalidMethod(t *testing.T) {
	log := logger.NewLogger()
	invalidMock := dittomock.DittoMock{
		Request: &dittomock.DittoRequest{
			Method: "/greet.Greeter111/SayHello",
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

	requestMatcher, err := dittomock.NewRequestMatcher(
		dittomock.WithMocks([]dittomock.DittoMock{
			invalidMock,
		}),
		dittomock.WithLogger(log),
	)

	require.NoError(t, err)

	greetDescr, err := findFileDescriptor("greet.proto")
	require.NoError(t, err)

	s := &mockServer{
		descrs:  []*desc.FileDescriptor{greetDescr},
		logger:  log,
		matcher: requestMatcher,
	}

	validator := &mockValidator{
		findMethodFunc: s.findMethodByName,
	}

	err = validator.Validate(requestMatcher.Mocks())
	assert.Error(t, err)
}

func TestMockServiceValidateFailureInvalidResponse(t *testing.T) {
	log := logger.NewLogger()
	invalidMock := dittomock.DittoMock{
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
				Body: []byte(`{ "message1": "hello Bob" }`),
			},
		},
	}

	requestMatcher, err := dittomock.NewRequestMatcher(
		dittomock.WithMocks([]dittomock.DittoMock{
			invalidMock,
		}),
		dittomock.WithLogger(log),
	)

	require.NoError(t, err)

	greetDescr, err := findFileDescriptor("greet.proto")
	require.NoError(t, err)

	s := &mockServer{
		descrs:  []*desc.FileDescriptor{greetDescr},
		logger:  log,
		matcher: requestMatcher,
	}

	validator := &mockValidator{
		findMethodFunc: s.findMethodByName,
	}

	err = validator.Validate(requestMatcher.Mocks())
	assert.Error(t, err)
}
