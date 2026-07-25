//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

type accessibilityForm struct {
	Name    string
	Email   string
	Channel string
}

func validateAccessibilityForm(parseValue accessibilityForm) ui.FieldErrors {
	parseErrors := ui.FieldErrors{}
	if len(strings.TrimSpace(parseValue.Name)) < 2 {
		parseErrors["Name"] = "Name must be at least 2 characters."
	}
	if !strings.Contains(parseValue.Email, "@") {
		parseErrors["Email"] = "Email must include @."
	}
	if strings.TrimSpace(parseValue.Channel) == "" {
		parseErrors["Channel"] = "Choose an announcement channel."
	}
	return parseErrors
}

func formAccessibilityExample() ui.Node {
	parseForm := ui.UseForm(accessibilityForm{})
	parseFocus := ui.UseFocusManager()
	parseAnnouncer := ui.UseAnnouncer()
	parseVisibleAnnouncement := ui.UseState("Waiting for validation.")

	parseAnnouncePolite := func(parseMessage2 string) {
		parseAnnouncer.Polite(parseMessage2)
		parseVisibleAnnouncement.Set(parseMessage2)
	}
	parseAnnounceAssertive := func(parseMessage3 string) {
		parseAnnouncer.Assertive(parseMessage3)
		parseVisibleAnnouncement.Set(parseMessage3)
	}

	parseNameID := ui.UseId() + "-name"
	parseEmailID := ui.UseId() + "-email"
	parseChannelID := ui.UseId() + "-channel"
	parseFieldIDs := map[string]string{"Name": parseNameID, "Email": parseEmailID, "Channel": parseChannelID}
	parseFieldOrder := []string{"Name", "Email", "Channel"}

	setName := ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField("Name", parseEvent.GetValue()) })
	setEmail := ui.UseEvent(func(parseEvent2 ui.InputEvent) { parseForm.SetField("Email", parseEvent2.GetValue()) })
	setChannel := ui.UseEvent(func(parseEvent3 ui.ChangeEvent) { parseForm.SetField("Channel", parseEvent3.GetValue()) })

	parseSubmit := ui.UseEvent(func(parseEvent4 ui.FormEvent) {
		parseEvent4.PreventDefault()
		parseErrors := validateAccessibilityForm(parseForm.Get())
		if len(parseErrors) > 0 {
			parseForm.SetErrors(parseErrors)
			parseMessage := fmt.Sprintf("Please correct %d field errors.", len(parseErrors))
			parseAnnounceAssertive(parseMessage)
			parseFocus.FocusFirstError(parseErrors, parseFieldIDs, parseFieldOrder...)
			return
		}
		parseAnnouncePolite("Submitting the accessibility review request.")
		parseForm.Submit(func(parseValue2 accessibilityForm) error {
			time.Sleep(220 * time.Millisecond)
			if strings.HasSuffix(strings.ToLower(parseValue2.Email), "@blocked.test") {
				return fmt.Errorf("server rejected %s", parseValue2.Email)
			}
			return nil
		})
	})

	parsePreviousSubmitted := ui.UsePrevious(parseForm.Submitted())
	ui.UseEffect(func() func() {
		if parseForm.Submitting() {
			return nil
		}
		if parseForm.SubmitError() != nil {
			parseAnnounceAssertive("Submission failed: " + parseForm.SubmitError().Error())
			return nil
		}
		if parseForm.Submitted() && (!parsePreviousSubmitted.Ok() || !parsePreviousSubmitted.Get()) {
			parseAnnouncePolite("Accessibility review request sent.")
		}
		return nil
	}, parseForm.Submitting(), parseForm.Submitted(), parseForm.SubmitError())

	parseValue := parseForm.Get()
	parseNameError := parseForm.Error("Name")
	parseEmailError := parseForm.Error("Email")
	parseChannelError := parseForm.Error("Channel")

	return shared.ExamplePage(
		"ui.UseAnnouncer with form accessibility",
		"Validation announcements, aria wiring, pending state semantics, and focus-to-error behavior",
		"This form uses ui.UseAnnouncer for live-region updates and ui.UseFocusManager to move focus to the first invalid field. Errors are wired through aria-invalid and aria-describedby rather than only color changes.",
		parseAnnouncer.Region(),
		shared.ExamplePanel("Accessible validation flow",
			html.Form(html.Props{OnSubmit: parseSubmit, Class: "mt-3 grid gap-5", Aria: map[string]string{"busy": map[bool]string{true: "true", false: "false"}[parseForm.Submitting()]}},
				html.Div(html.Props{},
					html.Label(html.Props{For: parseNameID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Reviewer name")),
					html.Input(html.Props{
						ID:          parseNameID,
						Value:       parseValue.Name,
						OnInput:     setName,
						Class:       "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
						Placeholder: "Ada Lovelace",
						Aria: map[string]string{
							"invalid":     map[bool]string{true: "true", false: "false"}[parseNameError != ""],
							"describedby": parseNameID + "-help " + parseNameID + "-error",
						},
					}),
					html.P(html.Props{ID: parseNameID + "-help", Class: "mt-2 text-xs text-slate-500"}, html.Text("Use the same name that appears on the review checklist.")),
					html.P(html.Props{ID: parseNameID + "-error", Class: "mt-1 text-sm text-rose-300"}, html.Text(parseNameError)),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: parseEmailID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Notification email")),
					html.Input(html.Props{
						ID:          parseEmailID,
						Value:       parseValue.Email,
						OnInput:     setEmail,
						Class:       "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
						Placeholder: "ada@example.com",
						Aria: map[string]string{
							"invalid":     map[bool]string{true: "true", false: "false"}[parseEmailError != ""],
							"describedby": parseEmailID + "-help " + parseEmailID + "-error",
						},
					}),
					html.P(html.Props{ID: parseEmailID + "-help", Class: "mt-2 text-xs text-slate-500"}, html.Text("We announce async submission results to this address.")),
					html.P(html.Props{ID: parseEmailID + "-error", Class: "mt-1 text-sm text-rose-300"}, html.Text(parseEmailError)),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: parseChannelID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Announcement channel")),
					html.Select(html.Props{
						ID:       parseChannelID,
						Value:    parseValue.Channel,
						OnChange: setChannel,
						Class:    "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
						Aria: map[string]string{
							"invalid":     map[bool]string{true: "true", false: "false"}[parseChannelError != ""],
							"describedby": parseChannelID + "-error",
						},
					},
						html.Option(html.Props{Value: ""}, html.Text("Select a channel")),
						html.Option(html.Props{Value: "slack"}, html.Text("Slack")),
						html.Option(html.Props{Value: "email"}, html.Text("Email digest")),
						html.Option(html.Props{Value: "status-page"}, html.Text("Status page")),
					),
					html.P(html.Props{ID: parseChannelID + "-error", Class: "mt-1 text-sm text-rose-300"}, html.Text(parseChannelError)),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					html.Button(html.Props{ID: "submit-accessibility-form", Type: "submit", Disabled: parseForm.Submitting(), Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100"}, html.Text("Submit review request")),
				),
			),
		),
		shared.ExamplePanel("Announcement state",
			html.Div(html.Props{ID: "form-announcement", Class: "mt-3 rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5 text-sm leading-7 text-slate-200"}, html.Text(parseVisibleAnnouncement.Get())),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Submitting", fmt.Sprintf("%t", parseForm.Submitting())),
				shared.ExampleStat("Submitted", fmt.Sprintf("%t", parseForm.Submitted())),
				shared.ExampleStat("Has errors", fmt.Sprintf("%t", parseForm.HasErrors())),
				shared.ExampleStat("Channel", parseValue.Channel),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(formAccessibilityExample))
	exampleboot.WaitExampleRuntime()
}
