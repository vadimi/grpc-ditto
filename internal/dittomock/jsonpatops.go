package dittomock

import (
	"regexp"

	"github.com/spyzhov/ajson"
)

func init() {
	ajson.AddOperation("=~", 3, false, regexMatchOp)
}

func regexMatchOp(left *ajson.Node, right *ajson.Node) (node *ajson.Node, err error) {
	pattern, err := right.GetString()
	if err != nil {
		return nil, err
	}
	val, err := left.GetString()
	if err != nil {
		return nil, err
	}
	res, err := regexp.MatchString(pattern, val)
	if err != nil {
		return nil, err
	}
	return ajson.BoolNode("eq", res), nil
}
