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

func EvaluateExpression(parseExpression string, parseAngleMode string, parsePrecision int) Evaluation {
	parseTrimmed := strings.TrimSpace(parseExpression)
	if parseTrimmed == "" {
		return Evaluation{Expression: parseExpression, Error: "Enter an expression"}
	}

	parseParser := parser{
		input:     parseTrimmed,
		angleMode: parseAngleMode,
	}

	parseValue, parseErr := parseParser.parse()
	if parseErr != nil {
		return Evaluation{Expression: parseExpression, Error: parseErr.Error()}
	}
	if math.IsNaN(parseValue) || math.IsInf(parseValue, 0) {
		return Evaluation{Expression: parseExpression, Error: "Expression produced a non-finite result"}
	}

	return Evaluation{
		Expression: parseExpression,
		Value:      parseValue,
		Formatted:  formatNumber(parseValue, parsePrecision),
	}
}

func formatNumber(parseValue float64, parsePrecision int) string {
	if parsePrecision < 0 {
		parsePrecision = 0
	}

	if math.Abs(parseValue-math.Round(parseValue)) < 1e-9 {
		return strconv.FormatInt(int64(math.Round(parseValue)), 10)
	}

	parseText := strconv.FormatFloat(parseValue, 'f', parsePrecision, 64)
	parseText = strings.TrimRight(parseText, "0")
	parseText = strings.TrimRight(parseText, ".")
	if parseText == "-0" {
		return "0"
	}
	return parseText
}

type parser struct {
	input     string
	pos       int
	angleMode string
}

func (parseP *parser) parse() (float64, error) {
	parseValue, parseErr := parseP.parseExpression()
	if parseErr != nil {
		return 0, parseErr
	}
	parseP.skipSpaces()
	if parseP.pos < len(parseP.input) {
		return 0, fmt.Errorf("unexpected token %q", parseP.input[parseP.pos:])
	}
	return parseValue, nil
}

func (parseP *parser) parseExpression() (float64, error) {
	parseLeft, parseErr := parseP.parseTerm()
	if parseErr != nil {
		return 0, parseErr
	}

	for {
		parseP.skipSpaces()
		switch parseP.peek() {
		case '+':
			parseP.pos++
			parseRight, parseErr2 := parseP.parseTerm()
			if parseErr2 != nil {
				return 0, parseErr2
			}
			parseLeft += parseRight
		case '-':
			parseP.pos++
			parseRight2, parseErr3 := parseP.parseTerm()
			if parseErr3 != nil {
				return 0, parseErr3
			}
			parseLeft -= parseRight2
		default:
			return parseLeft, nil
		}
	}
}

func (parseP *parser) parseTerm() (float64, error) {
	parseLeft, parseErr := parseP.parsePower()
	if parseErr != nil {
		return 0, parseErr
	}

	for {
		parseP.skipSpaces()
		switch parseP.peek() {
		case '*':
			parseP.pos++
			parseRight, parseErr2 := parseP.parsePower()
			if parseErr2 != nil {
				return 0, parseErr2
			}
			parseLeft *= parseRight
		case '/':
			parseP.pos++
			parseRight2, parseErr3 := parseP.parsePower()
			if parseErr3 != nil {
				return 0, parseErr3
			}
			if math.Abs(parseRight2) < 1e-12 {
				return 0, fmt.Errorf("division by zero")
			}
			parseLeft /= parseRight2
		default:
			return parseLeft, nil
		}
	}
}

func (parseP *parser) parsePower() (float64, error) {
	parseLeft, parseErr := parseP.parseUnary()
	if parseErr != nil {
		return 0, parseErr
	}

	parseP.skipSpaces()
	if parseP.peek() == '^' {
		parseP.pos++
		parseRight, parseErr2 := parseP.parsePower()
		if parseErr2 != nil {
			return 0, parseErr2
		}
		return math.Pow(parseLeft, parseRight), nil
	}

	return parseLeft, nil
}

func (parseP *parser) parseUnary() (float64, error) {
	parseP.skipSpaces()
	switch parseP.peek() {
	case '+':
		parseP.pos++
		return parseP.parseUnary()
	case '-':
		parseP.pos++
		parseValue, parseErr := parseP.parseUnary()
		if parseErr != nil {
			return 0, parseErr
		}
		return -parseValue, nil
	default:
		return parseP.parsePrimary()
	}
}

func (parseP *parser) parsePrimary() (float64, error) {
	parseP.skipSpaces()
	parseCh := parseP.peek()
	switch {
	case parseCh == '(':
		parseP.pos++
		parseValue, parseErr := parseP.parseExpression()
		if parseErr != nil {
			return 0, parseErr
		}
		parseP.skipSpaces()
		if parseP.peek() != ')' {
			return 0, fmt.Errorf("expected )")
		}
		parseP.pos++
		return parseValue, nil
	case unicode.IsDigit(rune(parseCh)) || parseCh == '.':
		return parseP.parseNumber()
	case unicode.IsLetter(rune(parseCh)):
		return parseP.parseIdentifier()
	default:
		return 0, fmt.Errorf("unexpected character %q", string(parseCh))
	}
}

func (parseP *parser) parseNumber() (float64, error) {
	parseStart := parseP.pos
	isParseDotSeen := false
	for parseP.pos < len(parseP.input) {
		parseCh := parseP.input[parseP.pos]
		if parseCh == '.' {
			if isParseDotSeen {
				break
			}
			isParseDotSeen = true
			parseP.pos++
			continue
		}
		if !unicode.IsDigit(rune(parseCh)) {
			break
		}
		parseP.pos++
	}

	parseValue, parseErr := strconv.ParseFloat(parseP.input[parseStart:parseP.pos], 64)
	if parseErr != nil {
		return 0, fmt.Errorf("invalid number")
	}
	return parseValue, nil
}

func (parseP *parser) parseIdentifier() (float64, error) {
	parseStart := parseP.pos
	for parseP.pos < len(parseP.input) && (unicode.IsLetter(rune(parseP.input[parseP.pos])) || unicode.IsDigit(rune(parseP.input[parseP.pos]))) {
		parseP.pos++
	}

	parseName := strings.ToLower(parseP.input[parseStart:parseP.pos])
	parseP.skipSpaces()
	if parseP.peek() != '(' {
		switch parseName {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		default:
			return 0, fmt.Errorf("unknown identifier %q", parseName)
		}
	}

	parseP.pos++
	parseArgs := []float64{}
	for {
		parseP.skipSpaces()
		if parseP.peek() == ')' {
			parseP.pos++
			break
		}

		parseValue, parseErr := parseP.parseExpression()
		if parseErr != nil {
			return 0, parseErr
		}
		parseArgs = append(parseArgs, parseValue)

		parseP.skipSpaces()
		if parseP.peek() == ',' {
			parseP.pos++
			continue
		}
		if parseP.peek() != ')' {
			return 0, fmt.Errorf("expected , or )")
		}
		parseP.pos++
		break
	}

	return parseP.callFunction(parseName, parseArgs)
}

func (parseP *parser) callFunction(parseName string, parseArgs []float64) (float64, error) {
	switch parseName {
	case "sin":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("sin expects 1 argument")
		}
		return math.Sin(parseP.toRadians(parseArgs[0])), nil
	case "cos":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("cos expects 1 argument")
		}
		return math.Cos(parseP.toRadians(parseArgs[0])), nil
	case "tan":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("tan expects 1 argument")
		}
		return math.Tan(parseP.toRadians(parseArgs[0])), nil
	case "sqrt":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("sqrt expects 1 argument")
		}
		if parseArgs[0] < 0 {
			return 0, fmt.Errorf("sqrt expects a non-negative value")
		}
		return math.Sqrt(parseArgs[0]), nil
	case "abs":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("abs expects 1 argument")
		}
		return math.Abs(parseArgs[0]), nil
	case "log":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("log expects 1 argument")
		}
		if parseArgs[0] <= 0 {
			return 0, fmt.Errorf("log expects a positive value")
		}
		return math.Log10(parseArgs[0]), nil
	case "ln":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("ln expects 1 argument")
		}
		if parseArgs[0] <= 0 {
			return 0, fmt.Errorf("ln expects a positive value")
		}
		return math.Log(parseArgs[0]), nil
	case "round":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("round expects 1 argument")
		}
		return math.Round(parseArgs[0]), nil
	case "floor":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("floor expects 1 argument")
		}
		return math.Floor(parseArgs[0]), nil
	case "ceil":
		if len(parseArgs) != 1 {
			return 0, fmt.Errorf("ceil expects 1 argument")
		}
		return math.Ceil(parseArgs[0]), nil
	case "pow":
		if len(parseArgs) != 2 {
			return 0, fmt.Errorf("pow expects 2 arguments")
		}
		return math.Pow(parseArgs[0], parseArgs[1]), nil
	case "max":
		if len(parseArgs) != 2 {
			return 0, fmt.Errorf("max expects 2 arguments")
		}
		return math.Max(parseArgs[0], parseArgs[1]), nil
	case "min":
		if len(parseArgs) != 2 {
			return 0, fmt.Errorf("min expects 2 arguments")
		}
		return math.Min(parseArgs[0], parseArgs[1]), nil
	default:
		return 0, fmt.Errorf("unknown function %q", parseName)
	}
}

func (parseP *parser) toRadians(parseValue float64) float64 {
	if strings.ToLower(parseP.angleMode) == "deg" {
		return parseValue * math.Pi / 180
	}
	return parseValue
}

func (parseP *parser) skipSpaces() {
	for parseP.pos < len(parseP.input) && unicode.IsSpace(rune(parseP.input[parseP.pos])) {
		parseP.pos++
	}
}

func (parseP *parser) peek() byte {
	if parseP.pos >= len(parseP.input) {
		return 0
	}
	return parseP.input[parseP.pos]
}
