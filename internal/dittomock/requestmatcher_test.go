package dittomock

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestMockLoaderJSON(t *testing.T) {
	js := `[
  {
    "request": {
      "method": "/greet.Greeter/SayHello",
      "body_patterns": [
        {
          "matches_jsonpath": { "expression": "$.name", "eq": "Bob" }
        }
      ]
    },
    "response": [
      {
        "body": { "message": "ok" }
      }
    ]
  }
]
`
	r := strings.NewReader(js)

	rm, err := NewRequestMatcher()
	require.NoError(t, err)

	mocks, err := rm.loadMockJSON(r)
	require.NoError(t, err)
	assert.NotEmpty(t, mocks)
	assert.Equal(t, "/greet.Greeter/SayHello", mocks[0].Request.Method)
	assert.Equal(t, "$.name", mocks[0].Request.BodyPatterns[0].MatchesJsonPath.Expression)
	assert.Equal(t, "Bob", mocks[0].Request.BodyPatterns[0].MatchesJsonPath.Equals)

	var body map[string]string
	err = json.Unmarshal(mocks[0].Response[0].Body, &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["message"])
}

func TestMockLoaderMultipleJSON(t *testing.T) {
	js := `[
  {
    "request": {
      "method": "/greet.Greeter/SayHello",
      "body_patterns": [
        {
          "matches_jsonpath": { "expression": "$.name", "eq": "Bob" }
        }
      ]
    },
    "response": [
      {
        "body": { "message": "ok" }
      }
    ]
  },
  {
    "request": {
      "method": "/greet.Greeter/SayHello",
      "body_patterns": [
        {
          "matches_jsonpath": { "expression": "$.name", "eq": "John" }
        }
      ]
    },
    "response": [
      {
        "body": { "message": "ok" }
      },
      {
        "body": { "message": "ok" }
      }
    ]
  }
]
`
	r := strings.NewReader(js)

	rm, err := NewRequestMatcher()
	require.NoError(t, err)

	mocks, err := rm.loadMockJSON(r)
	require.NoError(t, err)
	require.Len(t, mocks, 2)
	assert.Equal(t, "/greet.Greeter/SayHello", mocks[1].Request.Method)
	assert.Equal(t, "$.name", mocks[1].Request.BodyPatterns[0].MatchesJsonPath.Expression)
	assert.Equal(t, "John", mocks[1].Request.BodyPatterns[0].MatchesJsonPath.Equals)

	require.Len(t, mocks[1].Response, 2)
	var body map[string]string
	err = json.Unmarshal(mocks[1].Response[0].Body, &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["message"])
}

func TestMockLoaderYAML(t *testing.T) {
	js := `---
- request:
    method: "/greet.Greeter/SayHello"
    body_patterns:
    - matches_jsonpath:
        expression: "$.name"
        eq: Bob
  response:
  - body:
      message: ok
`
	r := strings.NewReader(js)

	rm, err := NewRequestMatcher()
	require.NoError(t, err)

	mocks, err := rm.loadMockYAML(r)
	require.NoError(t, err)
	assert.NotEmpty(t, mocks)
	assert.Equal(t, "/greet.Greeter/SayHello", mocks[0].Request.Method)
	assert.Equal(t, "$.name", mocks[0].Request.BodyPatterns[0].MatchesJsonPath.Expression)
	assert.Equal(t, "Bob", mocks[0].Request.BodyPatterns[0].MatchesJsonPath.Equals)

	var body map[string]string
	err = json.Unmarshal(mocks[0].Response[0].Body, &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["message"])
}

func TestMockLoaderStatusResponse(t *testing.T) {
	js := `---
- request:
    method: "/greet.Greeter/SayHello"
    body_patterns:
      - matches_jsonpath:
          expression: "$.name"
          eq: Bob
  response:
    - status:
        code: UNKNOWN
        message: oops, something went wrong
`
	r := strings.NewReader(js)

	rm, err := NewRequestMatcher()
	require.NoError(t, err)

	mocks, err := rm.loadMockYAML(r)
	require.NoError(t, err)
	require.NotEmpty(t, mocks)

	require.NotNil(t, mocks[0].Response[0].Status)
	assert.Equal(t, codes.Unknown, mocks[0].Response[0].Status.Code)
}

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
		Response: []*DittoResponse{
			{
				Body: []byte("ok"),
			},
		},
	}
	mocks := []DittoMock{m1}
	rm, _ := NewRequestMatcher(WithMocks(mocks))
	mresp, err := rm.Match("test", []byte("{}"))

	require.NoError(t, err)
	require.NotNil(t, mresp)
	assert.Equal(t, []byte("ok"), []byte(mresp.Response[0].Body))
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
			Response: []*DittoResponse{
				{
					Body: []byte("ok"),
				},
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mresp, err := rm.Match("test", []byte(test.src))

		require.NoError(t, err)
		require.NotNil(t, mresp)
		assert.Equal(t, []byte("ok"), []byte(mresp.Response[0].Body))
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
			Response: []*DittoResponse{
				{
					Body: []byte("ok"),
				},
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mresp, err := rm.Match("test", []byte(test.src))

		require.NoError(t, err)
		require.NotNil(t, mresp)
		assert.Equal(t, []byte("ok"), []byte(mresp.Response[0].Body))
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
			Response: []*DittoResponse{
				{
					Body: []byte("ok"),
				},
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mresp, err := rm.Match("test", []byte(test.src))

		require.NoError(t, err)
		require.NotNil(t, mresp)
		assert.Equal(t, []byte("ok"), []byte(mresp.Response[0].Body))
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
			Response: []*DittoResponse{
				{
					Body: []byte("ok"),
				},
			},
		}
		mocks := []DittoMock{m1}
		rm, _ := NewRequestMatcher(WithMocks(mocks))
		mock, err := rm.Match("test", []byte(test.src))

		require.NoError(t, err)
		require.NotNil(t, mock)
		assert.Equal(t, []byte("ok"), []byte(mock.Response[0].Body))
	}
}

func TestMockLoaderJSON_ResponseBodyTemplate(t *testing.T) {
	js := `[
  {
    "request": {
      "method": "/greet.Greeter/SayHello",
      "body_patterns": [
        {
          "matches_jsonpath": { "expression": "$.name", "eq": "Bob" }
        }
      ]
    },
    "response": [
      {
        "body_template": "{ \"message\": \"{{now_rfc3339}}\" }"
      }
    ]
  }
]
`
	r := strings.NewReader(js)

	rm, err := NewRequestMatcher()
	require.NoError(t, err)

	mocks, err := rm.loadMockJSON(r)
	require.NoError(t, err)
	assert.NotEmpty(t, mocks)
	assert.Equal(t, "/greet.Greeter/SayHello", mocks[0].Request.Method)
	assert.Equal(t, "$.name", mocks[0].Request.BodyPatterns[0].MatchesJsonPath.Expression)
	assert.Equal(t, "Bob", mocks[0].Request.BodyPatterns[0].MatchesJsonPath.Equals)

	var body map[string]string
	err = json.Unmarshal(mocks[0].Response[0].Body, &body)
	require.NoError(t, err)
	_, err = time.Parse(time.RFC3339, body["message"])
	require.NoError(t, err)
}
