package dittomock

import (
	"encoding/json"

	"google.golang.org/grpc/codes"
)

type DittoRequest struct {
	Method       string
	BodyPatterns []DittoBodyPattern `json:"bodyPatterns"`
}

func (dr *DittoRequest) String() string {
	res, err := json.Marshal(dr)
	if err != nil {
		return ""
	}
	return string(res)
}

type DittoResponse struct {
	Body   json.RawMessage
	Status *RpcStatus
}

type RpcStatus struct {
	Code    codes.Code
	Message string
}

type DittoMock struct {
	Request  *DittoRequest
	Response []*DittoResponse
}

type DittoBodyPattern struct {
	EqualToJson     json.RawMessage  `json:"equalToJson,omitempty"`
	MatchesJsonPath *JSONPathWrapper `json:"matchesJsonPath,omitempty"`
}

type JSONPathMessage struct {
	Expression string `json:"expression,omitempty"`
	Contains   string `json:"contains,omitempty"`
	Equals     string `json:"eq,omitempty"`
	Regexp     string `json:"regexp,omitempty"`
}

type JSONPathWrapper struct {
	JSONPathMessage
	Partial bool `json:"-"`
}

func (w *JSONPathWrapper) UnmarshalJSON(data []byte) error {
	var m interface{}
	m = &w.Expression
	w.Partial = true
	if data[0] == '{' {
		m = &w.JSONPathMessage
		w.Partial = false
	}
	return json.Unmarshal(data, m)
}
