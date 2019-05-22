package rpn

import (
	"errors"
	"strings"
	"github.com/shopspring/decimal"
)

// Evaluate an input string as a space-separated list of RPN commands.
func Evaluate(input string) (decimal.Decimal, error) {
	var stack []decimal.Decimal
	inputs := strings.Split(input, " ")

	for _, command := range inputs {
		switch command {
		case "+", "-", "*", "/", "%", "^":
			if len(stack) < 2 {
				return decimal.Zero, errors.New("stack overflow")
			}
			lhs := stack[len(stack)-2]
			rhs := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			switch command {
			case "+":
				rhs = lhs.Add(rhs)
			case "-":
				rhs = lhs.Sub(rhs)
			case "*":
				rhs = lhs.Mul(rhs)
			case "/":
				rhs = lhs.Div(rhs)
			case "%":
				rhs = lhs.Mod(rhs)
			case "^":
				rhs = lhs.Pow(rhs)
			}
			stack[len(stack)-1] = rhs
		case "abs", "atan", "ceil", "cos", "floor", "neg", "sin", "tan":
			if len(stack) < 1 {
				return decimal.Zero, errors.New("stack overflow")
			}
			val := stack[len(stack)-1]
			switch command {
			case "abs":
				val = val.Abs()
			case "atan":
				val = val.Atan()
			case "ceil":
				val = val.Ceil()
			case "cos":
				val = val.Cos()
			case "floor":
				val = val.Floor()
			case "neg":
				val = val.Neg()
			case "sin":
				val = val.Sin()
			case "tan":
				val = val.Tan()
			}
			stack[len(stack)-1] = val
		default:
			val, err := decimal.NewFromString(command)
			if err != nil {
				return val, err
			}
			stack = append(stack, val)
		}
	}

	if len(stack) != 1 {
		return decimal.Zero, errors.New("unclean stack")
	}
	return stack[0], nil
}


