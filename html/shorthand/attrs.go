package shorthand

//go:generate go run ../internal/attrgen ../..

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// InputFn aliases [html.InputFn], the callback shapes OnInput and OnChange accept.
type InputFn = html.InputFn

// ChangeFn aliases [html.ChangeFn].
type ChangeFn = html.ChangeFn

// MarkdownOption aliases [html.MarkdownOption].
type MarkdownOption = html.MarkdownOption

// OnInput delegates to [html.OnInput].
func OnInput[F InputFn](parseCallback F) PropOption { return html.OnInput(parseCallback) }

// OnInputHandler delegates to [html.OnInputHandler].
func OnInputHandler(parseHandler ui.Handler) PropOption { return html.OnInputHandler(parseHandler) }

// OnChange delegates to [html.OnChange].
func OnChange[F ChangeFn](parseCallback F) PropOption { return html.OnChange(parseCallback) }

// OnChangeHandler delegates to [html.OnChangeHandler].
func OnChangeHandler(parseHandler ui.Handler) PropOption { return html.OnChangeHandler(parseHandler) }

// CheckedIf delegates to [html.CheckedIf].
func CheckedIf(isCondition bool) PropOption { return html.CheckedIf(isCondition) }

// RequiredIf delegates to [html.RequiredIf].
func RequiredIf(isCondition bool) PropOption { return html.RequiredIf(isCondition) }

// IfProp delegates to [html.IfProp].
func IfProp(isCondition bool, parseOptions ...PropOption) PropOption {
	return html.IfProp(isCondition, parseOptions...)
}

// UnlessProp delegates to [html.UnlessProp].
func UnlessProp(isCondition bool, parseOptions ...PropOption) PropOption {
	return html.UnlessProp(isCondition, parseOptions...)
}

// WithURLSchemes delegates to [html.WithURLSchemes].
func WithURLSchemes(parseSchemes ...string) MarkdownOption {
	return html.WithURLSchemes(parseSchemes...)
}
