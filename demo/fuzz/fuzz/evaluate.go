package fuzzevaluate

import (
	"skylight.example/rpn"

	// go-fuzz support
	_ "github.com/dvyukov/go-fuzz/go-fuzz-dep"
)

// FuzzEvaluate checks if rpn.Evaluate hits all covered lines
func FuzzEvaluate(data []byte) int {
	_, err := rpn.Evaluate(string(data))
	if err == nil {
		return 1
	}
	return 0
}
