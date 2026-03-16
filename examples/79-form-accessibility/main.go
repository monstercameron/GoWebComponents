//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type accessibilityForm struct {
	Name    string
	Email   string
	Channel string
}

func validateAccessibilityForm(value accessibilityForm) ui.FieldErrors {
	errors := ui.FieldErrors{}
	if len(strings.TrimSpace(value.Name)) < 2 {
		errors["Name"] = "Name must be at least 2 characters."
	}
	if !strings.Contains(value.Email, "@") {
		errors["Email"] = "Email must include @."
	}
	if strings.TrimSpace(value.Channel) == "" {
		errors["Channel"] = "Choose an announcement channel."
	}
	return errors
}

func formAccessibilityExample() ui.Node {
	form := ui.UseForm(accessibilityForm{})
	focus := ui.UseFocusManager()
	announcer := ui.UseAnnouncer()
	visibleAnnouncement := ui.UseState("Waiting for validation.")

	announcePolite := func(message string) {
		announcer.Polite(message)
		visibleAnnouncement.Set(message)
	}
	announceAssertive := func(message string) {
		announcer.Assertive(message)
		visibleAnnouncement.Set(message)
	}

	nameID := ui.UseId() + "-name"
	emailID := ui.UseId() + "-email"
	channelID := ui.UseId() + "-channel"
	fieldIDs := map[string]string{"Name": nameID, "Email": emailID, "Channel": channelID}
	fieldOrder := []string{"Name", "Email", "Channel"}

	setName := ui.UseEvent(func(event ui.InputEvent) { form.SetField("Name", event.GetValue()) })
	setEmail := ui.UseEvent(func(event ui.InputEvent) { form.SetField("Email", event.GetValue()) })
	setChannel := ui.UseEvent(func(event ui.ChangeEvent) { form.SetField("Channel", event.GetValue()) })

	submit := ui.UseEvent(func(event ui.FormEvent) {
		event.PreventDefault()
		errors := validateAccessibilityForm(form.Get())
		if len(errors) > 0 {
			form.SetErrors(errors)
			message := fmt.Sprintf("Please correct %d field errors.", len(errors))
			announceAssertive(message)
			focus.FocusFirstError(errors, fieldIDs, fieldOrder...)
			return
		}
		announcePolite("Submitting the accessibility review request.")
		form.Submit(func(value accessibilityForm) error {
			time.Sleep(220 * time.Millisecond)
			if strings.HasSuffix(strings.ToLower(value.Email), "@blocked.test") {
				return fmt.Errorf("server rejected %s", value.Email)
			}
			return nil
		})
	})

	previousSubmitted := ui.UsePrevious(form.Submitted())
	ui.UseEffect(func() func() {
		if form.Submitting() {
			return nil
		}
		if form.SubmitError() != nil {
			announceAssertive("Submission failed: " + form.SubmitError().Error())
			return nil
		}
		if form.Submitted() && (!previousSubmitted.Ok() || !previousSubmitted.Get()) {
			announcePolite("Accessibility review request sent.")
		}
		return nil
	}, form.Submitting(), form.Submitted(), form.SubmitError())

	value := form.Get()
	nameError := form.Error("Name")
	emailError := form.Error("Email")
	channelError := form.Error("Channel")

	return shared.ExamplePage(
		"ui.UseAnnouncer with form accessibility",
		"Validation announcements, aria wiring, pending state semantics, and focus-to-error behavior",
		"This form uses ui.UseAnnouncer for live-region updates and ui.UseFocusManager to move focus to the first invalid field. Errors are wired through aria-invalid and aria-describedby rather than only color changes.",
		announcer.Region(),
		shared.ExamplePanel("Accessible validation flow",
			html.Form(html.Props{OnSubmit: submit, Class: "mt-3 grid gap-5", Aria: map[string]string{"busy": map[bool]string{true: "true", false: "false"}[form.Submitting()]}},
				html.Div(html.Props{},
					html.Label(html.Props{For: nameID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Reviewer name")),
					html.Input(html.Props{
						ID:      nameID,
						Value:   value.Name,
						OnInput: setName,
						Class:   "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
						Placeholder: "Ada Lovelace",
						Aria: map[string]string{
							"invalid":     map[bool]string{true: "true", false: "false"}[nameError != ""],
							"describedby": nameID + "-help " + nameID + "-error",
						},
					}),
					html.P(html.Props{ID: nameID + "-help", Class: "mt-2 text-xs text-slate-500"}, html.Text("Use the same name that appears on the review checklist.")),
					html.P(html.Props{ID: nameID + "-error", Class: "mt-1 text-sm text-rose-300"}, html.Text(nameError)),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: emailID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Notification email")),
					html.Input(html.Props{
						ID:      emailID,
						Value:   value.Email,
						OnInput: setEmail,
						Class:   "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
						Placeholder: "ada@example.com",
						Aria: map[string]string{
							"invalid":     map[bool]string{true: "true", false: "false"}[emailError != ""],
							"describedby": emailID + "-help " + emailID + "-error",
						},
					}),
					html.P(html.Props{ID: emailID + "-help", Class: "mt-2 text-xs text-slate-500"}, html.Text("We announce async submission results to this address.")),
					html.P(html.Props{ID: emailID + "-error", Class: "mt-1 text-sm text-rose-300"}, html.Text(emailError)),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: channelID, Class: "block text-sm uppercase tracking-[0.25em] text-slate-400"}, html.Text("Announcement channel")),
					html.Select(html.Props{
						ID:       channelID,
						Value:    value.Channel,
						OnChange: setChannel,
						Class:    "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
						Aria: map[string]string{
							"invalid":     map[bool]string{true: "true", false: "false"}[channelError != ""],
							"describedby": channelID + "-error",
						},
					},
						html.Option(html.Props{Value: ""}, html.Text("Select a channel")),
						html.Option(html.Props{Value: "slack"}, html.Text("Slack")),
						html.Option(html.Props{Value: "email"}, html.Text("Email digest")),
						html.Option(html.Props{Value: "status-page"}, html.Text("Status page")),
					),
					html.P(html.Props{ID: channelID + "-error", Class: "mt-1 text-sm text-rose-300"}, html.Text(channelError)),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					html.Button(html.Props{ID: "submit-accessibility-form", Type: "submit", Disabled: form.Submitting(), Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100"}, html.Text("Submit review request")),
				),
			),
		),
		shared.ExamplePanel("Announcement state",
			html.Div(html.Props{ID: "form-announcement", Class: "mt-3 rounded-[1.5rem] border border-white/10 bg-slate-950/45 p-5 text-sm leading-7 text-slate-200"}, html.Text(visibleAnnouncement.Get())),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Submitting", fmt.Sprintf("%t", form.Submitting())),
				shared.ExampleStat("Submitted", fmt.Sprintf("%t", form.Submitted())),
				shared.ExampleStat("Has errors", fmt.Sprintf("%t", form.HasErrors())),
				shared.ExampleStat("Channel", value.Channel),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(formAccessibilityExample), "#app")
	select {}
}