package dittomock

import (
	"encoding/json"

	"google.golang.org/grpc/codes"
)

type DittoRequest struct {
	Method       string
	BodyPatterns []DittoBodyPattern `json:"bodyPatterns"`
}

type DittoResponse struct {
	Body       json.RawMessage
	StatusCode codes.Code
}

type DittoMock struct {
	Request  *DittoRequest
	Response *DittoResponse
}

type DittoBodyPattern struct {
	EqualToJson     json.RawMessage  `json:"equalToJson"`
	MatchesJsonPath *JSONPathWrapper `json:"matchesJsonPath"`
}

type JSONPathMessage struct {
	Expression string `json:"expression"`
	Contains   string `json:"contains"`
	Equals     string `json:"equals"`
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
