package services

import (
	"bytes"
	"context"
	"fmt"
	"github.com/videa-tv/grpc-ditto/api"
	"github.com/videa-tv/grpc-ditto/internal/dittomock"
	"github.com/videa-tv/grpc-ditto/internal/logger"

	"github.com/golang/protobuf/jsonpb"
	pstruct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockingServiceImpl struct {
	matcher *dittomock.RequestMatcher
	log     logger.Logger
}

func NewMockingService(matcher *dittomock.RequestMatcher, log logger.Logger) api.MockingServiceServer {
	return &mockingServiceImpl{
		matcher: matcher,
		log:     log,
	}
}

func (s *mockingServiceImpl) Clear(ctx context.Context, req *api.ClearRequest) (*api.ClearResponse, error) {
	s.log.Info("clear all mocks")
	s.matcher.Clear()
	return &api.ClearResponse{}, nil
}

func (s *mockingServiceImpl) AddMock(ctx context.Context, req *api.AddMockRequest) (*api.AddMockResponse, error) {
	s.log.Infow("add new mock", "method", req.Mock.Request.Method)
	resp := &api.AddMockResponse{}

	if req.Mock == nil {
		return resp, status.Error(codes.InvalidArgument, "mock is required")
	}

	if req.Mock.Request == nil {
		return resp, status.Error(codes.InvalidArgument, "mock request is required")
	}

	if req.Mock.Request.Method == "" {
		return resp, status.Error(codes.InvalidArgument, "mock request method is required")
	}

	msgJS, _ := (&jsonpb.Marshaler{}).MarshalToString(req)
	s.log.Debugw("adding mock", "method", req.Mock.Request.Method, "msgJS", msgJS)

	mock, err := dittoMock(req)
	if err != nil {
		s.log.Errorw("converting mock", "err", err)
		return nil, err
	}

	s.matcher.AddMock(mock)

	return &api.AddMockResponse{}, nil
}

func dittoMock(req *api.AddMockRequest) (dittomock.DittoMock, error) {
	m := dittomock.DittoMock{}

	var respBody []byte
	var err error
	if req.Mock.Response != nil {
		respBody, err = structToBytes(req.Mock.Response.Body)
		if err != nil {
			return m, fmt.Errorf("structToBytes: %w", err)
		}
	}

	if len(respBody) == 0 {
		respBody = []byte("{}")
	}

	m.Response = &dittomock.DittoResponse{
		Body: []byte(respBody),
	}

	m.Request = &dittomock.DittoRequest{
		Method:       req.Mock.Request.GetMethod(),
		BodyPatterns: make([]dittomock.DittoBodyPattern, 0, len(req.Mock.Request.BodyPatterns)),
	}

	for _, reqPattern := range req.Mock.Request.GetBodyPatterns() {
		p := dittomock.DittoBodyPattern{}

		switch reqPattern.GetPattern().(type) {
		case *api.DittoBodyPattern_EqualToJson:
			b, err := structToBytes(reqPattern.GetEqualToJson())
			if err != nil {
				return m, fmt.Errorf("structToBytes conversion of equal_to_json: %w", err)
			}
			p.EqualToJson = b
		case *api.DittoBodyPattern_MatchesJsonpath:
			p.MatchesJsonPath = jsonPathWrapper(reqPattern.GetMatchesJsonpath())
		}

		m.Request.BodyPatterns = append(m.Request.BodyPatterns, p)
	}

	return m, nil
}

func jsonPathWrapper(p *api.JSONPathPattern) *dittomock.JSONPathWrapper {
	w := &dittomock.JSONPathWrapper{
		JSONPathMessage: dittomock.JSONPathMessage{
			Expression: p.GetExpression(),
		},
	}

	switch p.GetOperator().(type) {
	case *api.JSONPathPattern_Eq:
		w.JSONPathMessage.Equals = p.GetEq()
	case *api.JSONPathPattern_Regexp:
		w.JSONPathMessage.Regexp = p.GetRegexp()
	case *api.JSONPathPattern_Contains:
		w.JSONPathMessage.Contains = p.GetContains()
	default:
		w.Partial = true
	}

	return w
}

func structToBytes(msg *pstruct.Struct) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}

	buf := &bytes.Buffer{}
	if err := (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, msg); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
