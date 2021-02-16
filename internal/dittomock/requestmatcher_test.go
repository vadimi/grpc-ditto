package dittomock

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
