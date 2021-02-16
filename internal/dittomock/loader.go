package dittomock

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	pstruct "github.com/golang/protobuf/ptypes/struct"
	"github.com/vadimi/grpc-ditto/api"
	"google.golang.org/grpc/codes"
)

func FromProto(req *api.DittoMock) (DittoMock, error) {
	m := DittoMock{
		Response: make([]*DittoResponse, 0, len(req.Response)),
	}

	for _, src := range req.Response {
		var respBody []byte
		var err error
		if src.Response != nil {
			switch src.GetResponse().(type) {
			case *api.DittoResponse_Body:
				respBody, err = structToBytes(src.GetBody())
				if err != nil {
					return m, fmt.Errorf("structToBytes: %w", err)
				}

				if len(respBody) == 0 {
					respBody = []byte("{}")
				}

				m.Response = append(m.Response, &DittoResponse{
					Body: respBody,
				})
			case *api.DittoResponse_Status:
				status := src.GetStatus()
				m.Response = append(m.Response, &DittoResponse{
					Body: respBody,
					Status: &RpcStatus{
						Code:    codes.Code(status.GetCode()),
						Message: status.GetMessage(),
					},
				})
			}
		}
	}

	m.Request = &DittoRequest{
		Method:       req.Request.GetMethod(),
		BodyPatterns: make([]DittoBodyPattern, 0, len(req.Request.BodyPatterns)),
	}

	for _, reqPattern := range req.Request.GetBodyPatterns() {
		p := DittoBodyPattern{}

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

func jsonPathWrapper(p *api.JSONPathPattern) *JSONPathWrapper {
	w := &JSONPathWrapper{
		JSONPathMessage: JSONPathMessage{
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
