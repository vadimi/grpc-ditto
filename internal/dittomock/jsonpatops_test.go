package dittomock

import "testing"

import "github.com/spyzhov/ajson"

func TestRegexOperationSuccess(t *testing.T) {
	pattern := "^ap.*$"
	value := "apple"

	match, err := regexMatchOp(ajson.StringNode("val", value), ajson.StringNode("pattern", pattern))
	if err != nil {
		t.Errorf("no error expected, got: %s", err)
		return
	}

	res, err := match.GetBool()
	if err != nil {
		t.Errorf("no error expected, got: %s", err)
		return
	}

	if !res {
		t.Error("matching should return success")
	}
}
