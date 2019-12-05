package dittomock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"grpc-ditto/internal/logger"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spyzhov/ajson"
)

var (
	ErrNotMatched = errors.New("dittomock: request not matched")
)

type RequestMatherOption func(*RequestMatcher)

func WithMocksPath(mocksPath string) RequestMatherOption {
	return func(rm *RequestMatcher) {
		rm.mocksPath = mocksPath
	}
}

func WithLogger(l logger.Logger) RequestMatherOption {
	return func(rm *RequestMatcher) {
		rm.logger = l
	}
}

func WithMocks(mocks []DittoMock) RequestMatherOption {
	return func(rm *RequestMatcher) {
		rules := map[string][]DittoMock{}
		mergeMocks(mocks, rules)
		rm.rules = rules
	}
}

type RequestMatcher struct {
	rules     map[string][]DittoMock
	logger    logger.Logger
	mocksPath string
}

func (rm *RequestMatcher) Match(method string, json []byte) (*DittoResponse, error) {
	mocks, ok := rm.rules[method]
	if !ok {
		return nil, ErrNotMatched
	}

	for _, mock := range mocks {
		res, err := rm.matches(json, mock.Request)
		if err != nil {
			rm.logger.Warnw("matching error", "err", err)
			continue
		}

		if res {
			return mock.Response, nil
		}
	}

	return nil, ErrNotMatched
}

func NewRequestMatcher(opts ...RequestMatherOption) (*RequestMatcher, error) {
	matcher := &RequestMatcher{
		rules: map[string][]DittoMock{},
	}

	for _, opt := range opts {
		opt(matcher)
	}

	if matcher.logger == nil {
		matcher.logger = logger.NewLogger()
	}

	if matcher.mocksPath != "" {
		err := filepath.Walk(matcher.mocksPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".json" {
				mocks, err := matcher.loadMock(path)
				if err != nil {
					return err
				}
				mergeMocks(mocks, matcher.rules)
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return matcher, nil
}

func (rm *RequestMatcher) loadMock(mockJson string) ([]DittoMock, error) {
	mocks := []DittoMock{}
	js, err := ioutil.ReadFile(mockJson)
	if err != nil {
		return mocks, err
	}
	msg := json.RawMessage{}
	err = json.Unmarshal(js, &msg)
	if err != nil {
		return mocks, err
	}
	if msg[0] == '[' {
		err := json.Unmarshal(msg, &mocks)
		if err != nil {
			return mocks, err
		}
	} else {
		var m DittoMock
		err := json.Unmarshal(msg, &m)
		if err != nil {
			return mocks, err
		}

		mocks = append(mocks, m)

	}

	return mocks, nil
}

func (rm *RequestMatcher) matches(json []byte, req *DittoRequest) (bool, error) {
	result := false
	for _, pattern := range req.BodyPatterns {
		if len(pattern.EqualToJson) > 0 {
			val, err := jsonMatcher(json, pattern.EqualToJson)
			if err != nil || !val {
				return false, err
			}

			result = true
		}
		if pattern.MatchesJsonPath != nil {
			val, err := jsonPathMatcher(json, pattern.MatchesJsonPath)
			if err != nil || !val {
				return false, err
			}

			result = true
		}

	}

	return result, nil
}

func jsonPathMatcher(jsonSrc []byte, pattern *JSONPathWrapper) (bool, error) {
	nodes, err := ajson.JSONPath(jsonSrc, pattern.Expression)
	if err != nil {
		return false, fmt.Errorf("jsonpath matching: %w, expr: %s", err, pattern.Expression)
	}

	if len(nodes) == 0 {
		return false, nil
	}

	if pattern.Partial {
		return len(nodes) > 0, nil
	}

	if pattern.Contains != "" {
		return strings.Contains(nodes[0].String(), pattern.Contains), nil
	}

	patternVal := pattern.Equals
	isRegexp := false
	if patternVal == "" {
		patternVal = pattern.Regexp
		isRegexp = true
	}
	result := false

	if patternVal == "" {
		return result, errors.New("matching expressions cannot be empty")
	}

	for _, node := range nodes {

		switch node.Type() {
		case ajson.String:
			strVal, _ := node.GetString()
			if isRegexp {
				re, err := regexp.Compile(patternVal)
				if err != nil {
					return result, err
				}
				result = re.MatchString(strVal)
			} else {
				result = strings.EqualFold(strVal, patternVal)
			}
		case ajson.Numeric:
			floatVal, err := strconv.ParseFloat(patternVal, 64)
			if err != nil {
				return result, err
			}
			n, _ := node.GetNumeric()
			result = (n == floatVal)
		case ajson.Bool:
			pbVal, err := strconv.ParseBool(patternVal)
			if err != nil {
				return result, err
			}
			bVal, _ := node.GetBool()
			result = pbVal == bVal
		case ajson.Object, ajson.Array:
			result, err = jsonMatcher([]byte(node.String()), []byte(patternVal))
		default:
			result = false
		}

		if !result {
			break
		}
	}

	return result, nil
}

func jsonMatcher(jsonVal []byte, expetedJson json.RawMessage) (bool, error) {
	src, err := canonicalJSON(jsonVal)
	if err != nil {
		return false, err
	}

	expected, err := canonicalJSON(expetedJson)
	if err != nil {
		return false, err
	}

	return bytes.Equal(src, expected), nil
}

func mergeMocks(mocks []DittoMock, group map[string][]DittoMock) {
	for _, m := range mocks {
		methodMocks, ok := group[m.Request.Method]
		if !ok {
			methodMocks = []DittoMock{}
		}
		methodMocks = append(methodMocks, m)
		group[m.Request.Method] = methodMocks
	}
}

func canonicalJSON(src []byte) ([]byte, error) {
	var val interface{}
	err := json.Unmarshal(src, &val)
	if err != nil {
		return nil, err
	}

	canonicalJson, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}

	return canonicalJson, nil
}
