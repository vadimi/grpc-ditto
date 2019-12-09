package services

import (
	"context"
	"grpc-ditto/api"
	"grpc-ditto/internal/dittomock"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockingServiceImpl struct {
	matcher *dittomock.RequestMatcher
}

func NewMockingService(matcher *dittomock.RequestMatcher) api.MockingServiceServer {
	return &mockingServiceImpl{
		matcher: matcher,
	}
}

func (s *mockingServiceImpl) AddMock(ctx context.Context, req *api.AddMockRequest) (*api.AddMockResponse, error) {
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

	s.matcher.AddMock(dittoMock(req))

	return &api.AddMockResponse{}, nil
}

func dittoMock(req *api.AddMockRequest) dittomock.DittoMock {
	respBody := req.Mock.Response.Body
	if respBody == "" {
		respBody = "{}"
	}

	m := dittomock.DittoMock{}
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
			p.EqualToJson = []byte(reqPattern.GetEqualToJson())
		case *api.DittoBodyPattern_MatchesJsonpath:
			p.MatchesJsonPath = jsonPathWrapper(reqPattern.GetMatchesJsonpath())
		}

		m.Request.BodyPatterns = append(m.Request.BodyPatterns, p)
	}

	return m
}

func jsonPathWrapper(p *api.JSONPathPattern) *dittomock.JSONPathWrapper {
	w := &dittomock.JSONPathWrapper{
		JSONPathMessage: dittomock.JSONPathMessage{
			Expression: p.GetExpression(),
		},
	}

	switch p.GetOperator().(type) {
	case *api.JSONPathPattern_Equals:
		w.JSONPathMessage.Equals = p.GetEquals()
	case *api.JSONPathPattern_Regexp:
		w.JSONPathMessage.Regexp = p.GetRegexp()
	case *api.JSONPathPattern_Contains:
		w.JSONPathMessage.Contains = p.GetContains()
	default:
		w.Partial = true
	}

	return w
}
