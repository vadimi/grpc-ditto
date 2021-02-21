package dittomock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/vadimi/grpc-ditto/api"
	"github.com/vadimi/grpc-ditto/internal/logger"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/spyzhov/ajson"
	"sigs.k8s.io/yaml"
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
	rw        sync.RWMutex
}

func (rm *RequestMatcher) Match(method string, js []byte) (*DittoMock, error) {
	rm.rw.RLock()
	defer rm.rw.RUnlock()

	mocks, ok := rm.rules[method]
	if !ok {
		return nil, ErrNotMatched
	}

	for _, mock := range mocks {
		res, err := rm.matches(js, mock.Request)
		if err != nil {
			rm.logger.Warnw("matching error", "err", err)
			continue
		}

		if res {
			rm.logger.Debugw("match found", "expr", mock.Request.String())
			return &mock, nil
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
			ext := strings.ToLower(filepath.Ext(path))
			var loadMockFn func(io.Reader) ([]DittoMock, error)
			switch ext {
			case ".json":
				loadMockFn = matcher.loadMockJSON
			case ".yaml", ".yml":
				loadMockFn = matcher.loadMockYAML
			}

			if loadMockFn != nil {
				f, err := os.Open(path)
				defer f.Close()
				if err != nil {
					return err
				}

				mocks, err := loadMockFn(f)
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

func (rm *RequestMatcher) Clear() {
	rm.rw.Lock()
	defer rm.rw.Unlock()

	rm.rules = map[string][]DittoMock{}
}

func (rm *RequestMatcher) AddMock(mock DittoMock) {
	rm.rw.Lock()
	defer rm.rw.Unlock()

	mergeMocks([]DittoMock{mock}, rm.rules)
}

func (rm *RequestMatcher) loadMockYAML(mockYAML io.Reader) ([]DittoMock, error) {
	y, err := io.ReadAll(mockYAML)
	if err != nil {
		return []DittoMock{}, err
	}
	js, err := yaml.YAMLToJSON(y)
	if err != nil {
		return []DittoMock{}, err
	}
	return rm.loadMock(js)
}

func (rm *RequestMatcher) loadMockJSON(mockJson io.Reader) ([]DittoMock, error) {
	js, err := io.ReadAll(mockJson)
	if err != nil {
		return []DittoMock{}, err
	}

	return rm.loadMock(js)
}

func (rm *RequestMatcher) loadMock(js []byte) ([]DittoMock, error) {
	mocks := []DittoMock{}
	msgs := []json.RawMessage{}
	err := json.Unmarshal(js, &msgs)
	if err != nil {
		return mocks, err
	}

	for _, msg := range msgs {
		m := &api.DittoMock{}
		err := protojson.Unmarshal(msg, m)
		if err != nil {
			return mocks, err
		}

		dm, err := FromProto(m)
		if err != nil {
			return mocks, err
		}
		mocks = append(mocks, dm)
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

	patternVal := pattern.Regexp
	isRegexp := true
	if patternVal == "" {
		patternVal = pattern.Equals
		isRegexp = false
	}
	result := false

	if isRegexp {
		return regexpMatcher(patternVal, nodes)
	}

	for _, node := range nodes {
		switch node.Type() {
		case ajson.String:
			strVal, _ := node.GetString()
			result = strings.EqualFold(strVal, patternVal)
		case ajson.Numeric:
			n, _ := node.GetNumeric()
			floatVal, err := strconv.ParseFloat(patternVal, 64)
			if err != nil {
				return result, err
			}
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

func regexpMatcher(regexpExpr string, nodes []*ajson.Node) (bool, error) {
	result := false
	re, err := regexp.Compile(regexpExpr)
	if err != nil {
		return result, err
	}
	for _, node := range nodes {
		strVal := ""
		switch node.Type() {
		case ajson.String:
			strVal, _ = node.GetString()
		default:
			strVal = node.String()
		}
		result = re.MatchString(strVal)
		if !result {
			break
		}
		continue
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
