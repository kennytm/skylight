package rpn_test

import (
	"testing"
	"github.com/shopspring/decimal"
	"skylight.example/rpn"
)

func TestRPN(t *testing.T) {
	result, err := rpn.Evaluate("1.23 4.56 + neg")
	if err != nil {
		t.Error("unexpected error", err)
	}
	if !result.Equal(decimal.RequireFromString("-5.79")) {
		t.Error("wrong result", result)
	}
}
