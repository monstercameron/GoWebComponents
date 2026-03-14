//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

type Evaluation struct {
	Expression string
	Value      float64
	Formatted  string
	Error      string
}

func EvaluateExpression(expression string, angleMode string, precision int) Evaluation {
	trimmed := strings.TrimSpace(expression)
	if trimmed == "" {
		return Evaluation{Expression: expression, Error: "Enter an expression"}
	}

	parser := parser{
		input:     trimmed,
		angleMode: angleMode,
	}

	value, err := parser.parse()
	if err != nil {
		return Evaluation{Expression: expression, Error: err.Error()}
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return Evaluation{Expression: expression, Error: "Expression produced a non-finite result"}
	}

	return Evaluation{
		Expression: expression,
		Value:      value,
		Formatted:  formatNumber(value, precision),
	}
}

func formatNumber(value float64, precision int) string {
	if precision < 0 {
		precision = 0
	}

	if math.Abs(value-math.Round(value)) < 1e-9 {
		return strconv.FormatInt(int64(math.Round(value)), 10)
	}

	text := strconv.FormatFloat(value, 'f', precision, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "-0" {
		return "0"
	}
	return text
}

type parser struct {
	input     string
	pos       int
	angleMode string
}

func (p *parser) parse() (float64, error) {
	value, err := p.parseExpression()
	if err != nil {
		return 0, err
	}
	p.skipSpaces()
	if p.pos < len(p.input) {
		return 0, fmt.Errorf("unexpected token %q", p.input[p.pos:])
	}
	return value, nil
}

func (p *parser) parseExpression() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}

	for {
		p.skipSpaces()
		switch p.peek() {
		case '+':
			p.pos++
			right, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			left += right
		case '-':
			p.pos++
			right, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			left -= right
		default:
			return left, nil
		}
	}
}

func (p *parser) parseTerm() (float64, error) {
	left, err := p.parsePower()
	if err != nil {
		return 0, err
	}

	for {
		p.skipSpaces()
		switch p.peek() {
		case '*':
			p.pos++
			right, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			left *= right
		case '/':
			p.pos++
			right, err := p.parsePower()
			if err != nil {
				return 0, err
			}
			if math.Abs(right) < 1e-12 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		default:
			return left, nil
		}
	}
}

func (p *parser) parsePower() (float64, error) {
	left, err := p.parseUnary()
	if err != nil {
		return 0, err
	}

	p.skipSpaces()
	if p.peek() == '^' {
		p.pos++
		right, err := p.parsePower()
		if err != nil {
			return 0, err
		}
		return math.Pow(left, right), nil
	}

	return left, nil
}

func (p *parser) parseUnary() (float64, error) {
	p.skipSpaces()
	switch p.peek() {
	case '+':
		p.pos++
		return p.parseUnary()
	case '-':
		p.pos++
		value, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		return -value, nil
	default:
		return p.parsePrimary()
	}
}

func (p *parser) parsePrimary() (float64, error) {
	p.skipSpaces()
	ch := p.peek()
	switch {
	case ch == '(':
		p.pos++
		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if p.peek() != ')' {
			return 0, fmt.Errorf("expected )")
		}
		p.pos++
		return value, nil
	case unicode.IsDigit(rune(ch)) || ch == '.':
		return p.parseNumber()
	case unicode.IsLetter(rune(ch)):
		return p.parseIdentifier()
	default:
		return 0, fmt.Errorf("unexpected character %q", string(ch))
	}
}

func (p *parser) parseNumber() (float64, error) {
	start := p.pos
	dotSeen := false
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '.' {
			if dotSeen {
				break
			}
			dotSeen = true
			p.pos++
			continue
		}
		if !unicode.IsDigit(rune(ch)) {
			break
		}
		p.pos++
	}

	value, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number")
	}
	return value, nil
}

func (p *parser) parseIdentifier() (float64, error) {
	start := p.pos
	for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos]))) {
		p.pos++
	}

	name := strings.ToLower(p.input[start:p.pos])
	p.skipSpaces()
	if p.peek() != '(' {
		switch name {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		default:
			return 0, fmt.Errorf("unknown identifier %q", name)
		}
	}

	p.pos++
	args := []float64{}
	for {
		p.skipSpaces()
		if p.peek() == ')' {
			p.pos++
			break
		}

		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		args = append(args, value)

		p.skipSpaces()
		if p.peek() == ',' {
			p.pos++
			continue
		}
		if p.peek() != ')' {
			return 0, fmt.Errorf("expected , or )")
		}
		p.pos++
		break
	}

	return p.callFunction(name, args)
}

func (p *parser) callFunction(name string, args []float64) (float64, error) {
	switch name {
	case "sin":
		if len(args) != 1 {
			return 0, fmt.Errorf("sin expects 1 argument")
		}
		return math.Sin(p.toRadians(args[0])), nil
	case "cos":
		if len(args) != 1 {
			return 0, fmt.Errorf("cos expects 1 argument")
		}
		return math.Cos(p.toRadians(args[0])), nil
	case "tan":
		if len(args) != 1 {
			return 0, fmt.Errorf("tan expects 1 argument")
		}
		return math.Tan(p.toRadians(args[0])), nil
	case "sqrt":
		if len(args) != 1 {
			return 0, fmt.Errorf("sqrt expects 1 argument")
		}
		if args[0] < 0 {
			return 0, fmt.Errorf("sqrt expects a non-negative value")
		}
		return math.Sqrt(args[0]), nil
	case "abs":
		if len(args) != 1 {
			return 0, fmt.Errorf("abs expects 1 argument")
		}
		return math.Abs(args[0]), nil
	case "log":
		if len(args) != 1 {
			return 0, fmt.Errorf("log expects 1 argument")
		}
		if args[0] <= 0 {
			return 0, fmt.Errorf("log expects a positive value")
		}
		return math.Log10(args[0]), nil
	case "ln":
		if len(args) != 1 {
			return 0, fmt.Errorf("ln expects 1 argument")
		}
		if args[0] <= 0 {
			return 0, fmt.Errorf("ln expects a positive value")
		}
		return math.Log(args[0]), nil
	case "round":
		if len(args) != 1 {
			return 0, fmt.Errorf("round expects 1 argument")
		}
		return math.Round(args[0]), nil
	case "floor":
		if len(args) != 1 {
			return 0, fmt.Errorf("floor expects 1 argument")
		}
		return math.Floor(args[0]), nil
	case "ceil":
		if len(args) != 1 {
			return 0, fmt.Errorf("ceil expects 1 argument")
		}
		return math.Ceil(args[0]), nil
	case "pow":
		if len(args) != 2 {
			return 0, fmt.Errorf("pow expects 2 arguments")
		}
		return math.Pow(args[0], args[1]), nil
	case "max":
		if len(args) != 2 {
			return 0, fmt.Errorf("max expects 2 arguments")
		}
		return math.Max(args[0], args[1]), nil
	case "min":
		if len(args) != 2 {
			return 0, fmt.Errorf("min expects 2 arguments")
		}
		return math.Min(args[0], args[1]), nil
	default:
		return 0, fmt.Errorf("unknown function %q", name)
	}
}

func (p *parser) toRadians(value float64) float64 {
	if strings.ToLower(p.angleMode) == "deg" {
		return value * math.Pi / 180
	}
	return value
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}
