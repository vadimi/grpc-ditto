package dittomock

import (
	"bytes"
	"testing"
)

func TestSimpleJSONMatching(t *testing.T) {
	m1 := DittoMock{
		Request: &DittoRequest{
			Method: "test",
			BodyPatterns: []DittoBodyPattern{
				{
					EqualToJson: []byte("{}"),
				},
			},
		},
		Response: &DittoResponse{
			Body: []byte("ok"),
		},
	}
	mocks := []DittoMock{m1}
	rm, _ := NewRequestMatcher(WithMocks(mocks))
	mresp, err := rm.Match("test", []byte("{}"))
	if err != nil {
		t.Errorf("matching error not expected, got %w", err)
		return
	}

	if mresp == nil || !bytes.Equal([]byte("ok"), mresp.Body) {
		t.Errorf("Expected 'ok', got: %s", mresp.Body)
	}
}

func TestRegexpMatch(t *testing.T) {
	tests := []struct {
		expr        string
		regexpMatch string
		src         string
	}{
		{"$.name", "^to.*$", `{"name": "tofu"}`},
		{"$.name", "^callback[-]svc.*$", `{"name": "callback-svc"}`},
		{"$.meal[?(@.name =~ '^tof.*$')].name", "tofu", `{ "meal": [{"name": "apple"},{"name": "tofu"}] }`},
	}

	for _, test := range tests {
		m1 := DittoMock{
			Request: &DittoRequest{
				Method: "test",
				BodyPatterns: []DittoBodyPattern{
					{
						MatchesJsonPath: &JSONPathWrapper{
							JSONPathMessage: JSONPathMessage{
								Expression: test.expr,
								Regexp:     test.regexpMatch,
							},
						},
					},
				},
			},
			Response: &DittoResponse{
				Body: []byte("ok"),
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mresp, err := rm.Match("test", []byte(test.src))
		if err != nil {
			t.Errorf("matching error not expected for expected result '%s', got %s", test.regexpMatch, err)
			return
		}

		if mresp == nil || !bytes.Equal([]byte("ok"), mresp.Body) {
			t.Errorf("Expected 'ok', got: %s", mresp.Body)
		}
	}
}

func TestSimpleJSONPathEqualsMatching(t *testing.T) {
	tests := []struct {
		expr   string
		equals string
		src    string
	}{
		{"$[0].name", "tofu", `[{"name": "tofu"}]`},
		{"$.name", "tofu", `{"name": "tofu"}`},
		{"$.meal.name", "tofu", `{ "meal": {"name": "tofu"} }`},
		{"$.meal[1].name", "tofu", `{ "meal": [{"name": "apple"},{"name": "tofu"}] }`},
		{"$.meal[?(@.name == 'tofu')].name", "tofu", `{ "meal": [{"name": "apple"},{"name": "tofu"}] }`},
		{"$.meal[?(@.name =~ '^tof.*$')].name", "tofu", `{ "meal": [{"name": "apple"},{"name": "tofu"}] }`},
		{"$.stationIds", "[1, 2]", `{ "stationIds": [1, 2] }`},
		{"$.result", "true", `{ "result": true }`},
	}

	for _, test := range tests {
		m1 := DittoMock{
			Request: &DittoRequest{
				Method: "test",
				BodyPatterns: []DittoBodyPattern{
					{
						MatchesJsonPath: &JSONPathWrapper{
							JSONPathMessage: JSONPathMessage{
								Expression: test.expr,
								Equals:     test.equals,
							},
						},
					},
				},
			},
			Response: &DittoResponse{
				Body: []byte("ok"),
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mresp, err := rm.Match("test", []byte(test.src))
		if err != nil {
			t.Errorf("matching error not expected for expected result '%s', got %s, expr: %s", test.equals, err, test.expr)
			return
		}

		if mresp == nil || !bytes.Equal([]byte("ok"), mresp.Body) {
			t.Errorf("Expected 'ok', got: %s", mresp.Body)
		}
	}
}

func TestMultipleJSONPathMatching(t *testing.T) {
	tests := []struct {
		exprs []JSONPathMessage
		src   string
	}{
		{
			[]JSONPathMessage{
				{Expression: "$.name", Equals: "lock1"},
				{Expression: "$.duration", Equals: "100"},
			}, `{"name": "lock1", "duration": 100}`,
		},
		{
			[]JSONPathMessage{
				{Expression: "$.name", Equals: "n1"},
				{Expression: "$.station", Equals: `{"prop1": "val1", "name": "s1"}`},
			}, `{"name": "n1", "station": {"name": "s1", "prop1": "val1"}}`,
		},
	}

	for _, test := range tests {
		patterns := []DittoBodyPattern{}
		for i := range test.exprs {
			wrapper := &JSONPathWrapper{JSONPathMessage: test.exprs[i]}
			patterns = append(patterns, DittoBodyPattern{MatchesJsonPath: wrapper})
		}

		m1 := DittoMock{
			Request: &DittoRequest{
				Method:       "test",
				BodyPatterns: patterns,
			},
			Response: &DittoResponse{
				Body: []byte("ok"),
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mresp, err := rm.Match("test", []byte(test.src))
		if err != nil {
			t.Errorf("matching error not expected, got %s", err)
			return
		}

		if mresp == nil || !bytes.Equal([]byte("ok"), mresp.Body) {
			t.Errorf("Expected 'ok', got: %s", mresp.Body)
		}
	}
}

func TestPartialJSONPathEqualsMatching(t *testing.T) {
	tests := []struct {
		expr string
		src  string
	}{
		{"$.name", `{"name": "tofu"}`},
		{"$.meal[?(@.name == 'tofu')].name", `{ "meal": [{"name": "apple"},{"name": "tofu"}] }`},
		{"$.stationIds", `{ "stationIds": [1, 2] }`},
	}

	for _, test := range tests {
		m1 := DittoMock{
			Request: &DittoRequest{
				Method: "test",
				BodyPatterns: []DittoBodyPattern{
					{
						MatchesJsonPath: &JSONPathWrapper{
							JSONPathMessage: JSONPathMessage{
								Expression: test.expr,
							},
							Partial: true,
						},
					},
				},
			},
			Response: &DittoResponse{
				Body: []byte("ok"),
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mresp, err := rm.Match("test", []byte(test.src))
		if err != nil {
			t.Errorf("matching error not expected for expression '%s', got %s", test.expr, err)
			return
		}

		if mresp == nil || !bytes.Equal([]byte("ok"), mresp.Body) {
			t.Errorf("Expected 'ok', got: %s", mresp.Body)
		}
	}
}
