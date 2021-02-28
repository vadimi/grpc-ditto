package main

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/vadimi/grpc-ditto/internal/dittomock"
)

type mockValidator struct {
	findMethodFunc func(methodName string) *desc.MethodDescriptor
}

// Validate that all methods in mocks have protos loaded in memory.
// also verifies that mock responses can be successfully marshalled into method response message
func (v *mockValidator) Validate(mocks map[string][]dittomock.DittoMock) error {
	for methodName, mocks := range mocks {
		method := v.findMethodFunc(methodName)
		fmt.Println(methodName, method)
		if method == nil {
			return fmt.Errorf("method %s not found in registered proto files", methodName)
		}

		for i, m := range mocks {
			err := v.ValidateMock(m)
			if err != nil {
				return fmt.Errorf("invalid mock [%d]: %w", i, err)
			}
		}
	}
	return nil
}

func (v *mockValidator) ValidateMock(mock dittomock.DittoMock) error {
	methodName := mock.Request.Method
	method := v.findMethodFunc(methodName)
	if method == nil {
		return fmt.Errorf("method %s not found in registered proto files", methodName)
	}

	for _, resp := range mock.Response {
		if resp.Status != nil {
			continue
		}

		output := dynamic.NewMessage(method.GetOutputType())
		err := output.UnmarshalJSON(resp.Body)
		if err != nil {
			return fmt.Errorf("invalid response for method %s: %w", methodName, err)
		}
	}

	return nil
}
