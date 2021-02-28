package services

import (
	"context"

	"github.com/vadimi/grpc-ditto/api"
	"github.com/vadimi/grpc-ditto/internal/dittomock"
	"github.com/vadimi/grpc-ditto/internal/logger"

	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockingServiceImpl struct {
	matcher   *dittomock.RequestMatcher
	log       logger.Logger
	validator MockValidator

	api.UnimplementedMockingServiceServer
}

type MockValidator interface {
	ValidateMock(dittomock.DittoMock) error
}

func NewMockingService(matcher *dittomock.RequestMatcher, validator MockValidator, log logger.Logger) api.MockingServiceServer {
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

	if err := s.validator.ValidateMock(mock); err != nil {
		s.log.Errorw("mock validation failed", "err", err)
		return nil, err
	}

	s.matcher.AddMock(mock)

	return &api.AddMockResponse{}, nil
}

func dittoMock(req *api.AddMockRequest) (dittomock.DittoMock, error) {
	return dittomock.FromProto(req.Mock)
}
