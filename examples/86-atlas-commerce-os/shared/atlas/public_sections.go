package atlas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func renderLandingContent() ui.Node {
	return html.Section(html.Props{Class: "grid gap-8"},
		html.Div(html.Props{Class: "grid gap-6 lg:grid-cols-[minmax(0,1.25fr)_minmax(18rem,0.8fr)]"},
			publicLandingIntroCard(),
			publicLandingMetrics(),
		),
		publicLandingFeatures(),
	)
}

func publicLandingIntroCard() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.9rem] border border-white/10 bg-[linear-gradient(145deg,rgba(13,18,30,0.96),rgba(18,25,40,0.9))] p-6 shadow-[0_22px_50px_rgba(0,0,0,0.22)] sm:p-7"},
		html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text(publicWhyAtlasFeelsReady)),
		html.H2(html.Props{Class: "text-3xl font-black tracking-[-0.03em] text-white"}, html.Text("A darker, cleaner storefront for complex workspace buying.")),
		html.P(html.Props{Class: "text-base leading-8 text-stone-300"}, html.Text("Atlas keeps pricing, delivery timing, and next-step actions visible in a standard commerce layout so buyers can compare products without translating warehouse language.")),
		html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-3"},
			publicSignalPill("Fast regional promise"),
			publicSignalPill("Project quote capture"),
			publicSignalPill("Warehouse-aware recovery"),
		),
	)
}

func publicLandingMetrics() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		publicMetricCard("Regional promise", "Delivery timing stays visible throughout the buying journey so teams can judge confidence before requesting pricing."),
		publicMetricCard("Buying confidence", "Pricing requests, delivery questions, and availability follow-up stay tied to the same product context."),
	)
}

func publicLandingFeatures() ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 md:grid-cols-3"},
		publicFeatureCard("Workspace systems", "Merchandising stays focused on whole setups instead of disconnected utility parts."),
		publicFeatureCard("Beautifully direct UX", "Primary actions stay visible, labels remain explicit, and the reading order is stable across route entry."),
		publicFeatureCard("Server-backed confidence", "Quotes, availability capture, and product questions share the same request-time contract as the internal ops console."),
	)
}

func renderCatalogContent(page catalogPage) ui.Node {
	items := make([]ui.Node, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, publicCatalogCard(item))
	}
	return html.Section(html.Props{Class: "grid gap-8"},
		publicCatalogOverview(page),
		storeCatalogControls(page),
		html.Div(html.Props{Class: "grid gap-5 md:grid-cols-2 xl:grid-cols-3"}, items...),
	)
}

func publicCatalogOverview(page catalogPage) ui.Node {
	return html.Div(html.Props{Class: "flex flex-col gap-4 rounded-[1.9rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_40px_rgba(0,0,0,0.18)] backdrop-blur-sm lg:flex-row lg:items-end lg:justify-between"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text(publicCatalogOverviewLabel)),
			html.H2(html.Props{Class: "text-3xl font-black tracking-[-0.03em] text-white"}, html.Text("Modern workspace systems, organized for quick decisions.")),
			html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300"}, html.Text("The catalog follows a more standard browse flow: category, price, stock posture, and next step all stay in the same place for faster scanning.")),
		),
		html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2 lg:min-w-[20rem]"},
			publicMetricCard(fmt.Sprintf("%d products", len(page.Items)), "Workspace systems stay organized for quick comparison and shortlist building."),
			publicMetricCard("Quote-ready", "Pricing help and delivery follow-up stay one step away on every product page."),
		),
	)
}

func publicCatalogCard(item productCard) ui.Node {
	actionLabel, actionCopy := catalogActionPlan(item.Status)
	return html.A(html.Props{Href: RouteCatalog + "/" + item.Slug, Class: "group grid gap-5 rounded-[1.9rem] border border-white/10 bg-[linear-gradient(180deg,rgba(15,20,33,0.98),rgba(10,15,24,0.95))] p-5 shadow-[0_20px_48px_rgba(0,0,0,0.22)] transition duration-200 hover:-translate-y-1 hover:border-amber-300/35 hover:bg-[linear-gradient(180deg,rgba(18,24,38,1),rgba(12,17,28,0.98))] hover:shadow-[0_28px_65px_rgba(0,0,0,0.28)]"},
		html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(item.Category)),
				html.P(html.Props{Class: "text-[0.68rem] font-medium uppercase tracking-[0.28em] text-stone-500"}, html.Text(item.SKU)),
			),
			html.Span(html.Props{Class: publicStatusClass(item.Status)}, html.Text(publicStatusLabel(item.Status))),
		),
		publicCatalogStory(item),
		publicCatalogPriceRail(item, actionLabel, actionCopy),
	)
}

func publicCatalogStory(item productCard) ui.Node {
	return html.Div(html.Props{Class: "relative overflow-hidden rounded-[1.55rem] border border-white/8 bg-[linear-gradient(160deg,rgba(255,255,255,0.04),rgba(255,255,255,0.02)_58%,rgba(245,158,11,0.05))] px-5 py-6"},
		html.Div(html.Props{Class: "pointer-events-none absolute -right-6 top-5 h-24 w-24 rounded-full bg-amber-300/12 blur-2xl"}),
		html.Div(html.Props{Class: "relative z-[1] grid gap-4"},
			html.Div(html.Props{Class: "flex flex-wrap items-center gap-2 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"},
				html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-stone-300"}, html.Text(publicAtlasSystemLabel)),
				html.Span(html.Props{}, html.Text(catalogPromiseCopy(item.Status))),
			),
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.04em] text-white transition group-hover:text-stone-100"}, html.Text(item.Title)),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(item.Summary)),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-stone-400"}, html.Text(catalogEditorialCopy(item))),
		),
	)
}

func publicCatalogPriceRail(item productCard, actionLabel string, actionCopy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-white/8 bg-white/5 p-4"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"}, html.Text(publicStartingAtLabel)),
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.04em] text-white"}, html.Text(formatPrice(item.PriceCents))),
				html.P(html.Props{Class: "text-sm leading-6 text-stone-400"}, html.Text(actionCopy)),
			),
			html.Span(html.Props{Class: "inline-flex items-center rounded-full border border-amber-300/50 bg-amber-300/12 px-4 py-2 text-sm font-semibold text-amber-100 transition group-hover:border-amber-300/70 group-hover:bg-amber-300/18"}, html.Text(actionLabel)),
		),
		html.Div(html.Props{Class: "grid gap-2 text-sm text-stone-300 sm:grid-cols-2"},
			html.P(html.Props{Class: "rounded-full border border-white/10 bg-white/6 px-3 py-2"}, html.Text(productCategoryCue(item.Category))),
			html.P(html.Props{Class: "rounded-full border border-white/10 bg-white/6 px-3 py-2"}, html.Text(productSupportCue(item.Status))),
		),
	)
}

func renderProductContent(page productDetailPage, payload Payload) ui.Node {
	product := page.Product
	return html.Section(html.Props{Class: "grid gap-8 xl:grid-cols-[minmax(0,1.5fr)_minmax(21rem,0.78fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-6"},
			publicProductHeroCard(product),
			html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]"},
				publicProductMetrics(),
				publicProductFeatureStrip(),
			),
			publicProductFeedbackSection(product, page.Comments, payload),
		),
		publicProductActionRail(product, payload),
	)
}

func publicProductFeedbackSection(product productCard, comments []commentRecord, payload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		form := ui.UseForm(publicCommentFormState{Reaction: "up"})
		commentsState := useAtlasState(cloneCommentRecords(comments))
		refreshingState := useAtlasState(false)
		submittingState := useAtlasState(false)
		submissionMessageState := useAtlasState("")
		useAtlasEffect(func() func() {
			commentsState.Set(cloneCommentRecords(comments))
			refreshingState.Set(false)
			return nil
		}, product.Slug, commentRecordsSignature(comments))
		value := form.Get()
		comments := commentsState.Get()
		submitting := submittingState.Get()
		submissionMessage := submissionMessageState.Get()
		countLabel := fmt.Sprintf("%d buyer notes", len(comments))
		if len(comments) == 1 {
			countLabel = "1 buyer note"
		}
		setAuthorName := ui.UseEvent(func(event ui.InputEvent) { form.SetField("AuthorName", event.GetValue()) })
		setSubject := ui.UseEvent(func(event ui.InputEvent) { form.SetField("Subject", event.GetValue()) })
		setBody := ui.UseEvent(func(event ui.InputEvent) { form.SetField("Body", event.GetValue()) })
		setReaction := ui.UseEvent(func(event ui.ChangeEvent) { form.SetField("Reaction", event.GetValue()) })
		submit := ui.UseEvent(func(event ui.FormEvent) {
			event.PreventDefault()
			if !form.Validate(validatePublicCommentForm) {
				return
			}
			if submittingState.Get() {
				return
			}
			submittingState.Set(true)
			submissionMessageState.Set("")
			form.SetFormError("")
			snapshot := form.Get()
			go func() {
				created, serverErrors, err := submitPublicComment(product.Slug, snapshot, payload.CSRF)
				if err != nil {
					form.SetFormError("Comment submit failed. Retry in a moment.")
					submittingState.Set(false)
					return
				}
				if !form.ApplyServerErrors(serverErrors) {
					submittingState.Set(false)
					return
				}
				form.Reset(publicCommentFormState{Reaction: "up"})
				form.SetErrors(nil)
				form.SetFormError("")
				submissionMessageState.Set("Your comment was submitted for review.")
				commentsState.Set(mergePublicCommentList(commentsState.Get(), created))
				refreshingState.Set(true)
				submittingState.Set(false)
				go func(created commentRecord) {
					updatedComments, err := fetchPublicProductComments(product.Slug)
					if err == nil {
						commentsState.Set(mergePublicCommentList(updatedComments, created))
					}
					refreshingState.Set(false)
				}(created)
			}()
		})
		return html.Div(html.Props{Class: "grid gap-5 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
			html.Div(html.Props{Class: "grid gap-2 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Customer reviews and questions")),
					html.H3(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("What buyers are asking before they commit.")),
					html.P(html.Props{Class: "max-w-3xl text-sm leading-7 text-stone-300"}, html.Text("Reviews, questions, and sentiment stay below the core product information in a standard decision flow, so buyers can scan the essentials first and validate with social proof second.")),
				),
				html.Div(html.Props{Class: "rounded-[1.2rem] border border-white/10 bg-white/8 px-4 py-3 text-sm font-semibold text-stone-200"}, html.Text(countLabel)),
			),
			html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,0.95fr)_minmax(19rem,0.8fr)] lg:items-start"},
				publicProductFeedbackList(comments, refreshingState.Get()),
				publicProductFeedbackForm(product, payload, form, value, submitting, submissionMessage, setAuthorName, setSubject, setBody, setReaction, submit),
			),
		)
	})
}

func publicProductFeedbackList(comments []commentRecord, refreshing bool) ui.Node {
	thumbsUp, thumbsDown := publicCommentReactionCounts(comments)
	ratioLabel, ratioValue := publicCommentRatioSummary(thumbsUp, thumbsDown)
	nodes := make([]ui.Node, 0, len(comments))
	for _, item := range comments {
		nodes = append(nodes, html.Div(html.Props{Class: "grid gap-3 rounded-[1.4rem] border border-white/10 bg-white/6 p-5 shadow-[0_14px_30px_rgba(0,0,0,0.14)]"},
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-base font-semibold text-white"}, html.Text(item.Subject)),
					publicCommentReactionBadge(item.Reaction),
				),
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
					html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-stone-400"}, html.Text(strings.ReplaceAll(item.AuthorType, "_", " "))),
					publicCommentStatusBadge(item.Status),
				),
			),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(item.Body)),
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-amber-700"}, html.Text(item.AuthorName+" · shared on "+formatPublicCommentDate(item.CreatedAt))),
		))
	}
	if len(nodes) == 0 {
		nodes = append(nodes, html.Div(html.Props{Class: "rounded-[1.4rem] border border-white/10 bg-white/6 p-5 text-sm leading-7 text-stone-300 shadow-[0_14px_30px_rgba(0,0,0,0.14)]"}, html.Text("No buyer notes have been shared yet. The first approved question or review will appear here once Atlas has it.")))
	}
	children := []ui.Node{
		html.Div(html.Props{Class: "grid gap-4 rounded-[1.4rem] border border-white/10 bg-white/6 p-5 shadow-[0_14px_30px_rgba(0,0,0,0.14)]"},
			html.Div(html.Props{Class: "grid gap-2 sm:grid-cols-[auto_minmax(0,1fr)] sm:items-end sm:gap-4"},
				html.Div(html.Props{Class: "text-3xl font-black tracking-[-0.05em] text-white"}, html.Text(ratioLabel)),
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text("Buyer sentiment snapshot")),
					html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("A quick read on whether approved feedback leans positive or points to delivery and fit concerns buyers should consider.")),
				),
			),
			html.Div(html.Props{Class: "h-3 overflow-hidden rounded-full bg-white/10"},
				html.Div(html.Props{Class: "h-full rounded-full bg-emerald-600", Style: map[string]string{"width": fmt.Sprintf("%d%%", ratioValue)}}),
			),
			html.Div(html.Props{Class: "grid gap-3 text-sm text-stone-300 sm:grid-cols-2"},
				html.P(html.Props{Class: "rounded-[1.1rem] border border-emerald-400/25 bg-emerald-400/10 px-4 py-3 font-semibold text-emerald-200"}, html.Text(fmt.Sprintf("%d thumbs up", thumbsUp))),
				html.P(html.Props{Class: "rounded-[1.1rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 font-semibold text-rose-200"}, html.Text(fmt.Sprintf("%d thumbs down", thumbsDown))),
			),
		),
	}
	if refreshing {
		children = append(children, html.P(html.Props{Class: "rounded-[1.1rem] border border-white/10 bg-white/8 px-4 py-3 text-sm font-medium text-stone-200"}, html.Text("Refreshing buyer notes...")))
	}
	children = append(children, html.Div(html.Props{Class: "grid gap-4"}, nodes...))
	return html.Div(html.Props{Class: "grid gap-4"}, children...)
}

func publicCommentReactionCounts(comments []commentRecord) (int, int) {
	thumbsUp := 0
	thumbsDown := 0
	for _, item := range comments {
		if strings.TrimSpace(strings.ToLower(item.Reaction)) == "down" {
			thumbsDown++
			continue
		}
		thumbsUp++
	}
	return thumbsUp, thumbsDown
}

func publicCommentRatioSummary(thumbsUp int, thumbsDown int) (string, int) {
	total := thumbsUp + thumbsDown
	if total == 0 {
		return "No ratings yet", 0
	}
	percentage := int(float64(thumbsUp)*100/float64(total) + 0.5)
	return fmt.Sprintf("%d%% thumbs up", percentage), percentage
}

func publicCommentReactionBadge(reaction string) ui.Node {
	label := "Thumbs up"
	className := "inline-flex items-center rounded-full border border-emerald-400/25 bg-emerald-400/10 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-emerald-200"
	if strings.TrimSpace(strings.ToLower(reaction)) == "down" {
		label = "Thumbs down"
		className = "inline-flex items-center rounded-full border border-rose-400/25 bg-rose-400/10 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-rose-200"
	}
	return html.Span(html.Props{Class: className}, html.Text(label))
}

type publicCommentFormState struct {
	AuthorName string
	Reaction   string
	Subject    string
	Body       string
}

func validatePublicCommentForm(value publicCommentFormState) ui.FieldErrors {
	errors := ui.FieldErrors{}
	if strings.TrimSpace(value.AuthorName) == "" {
		errors["AuthorName"] = "Enter your name before sharing feedback."
	}
	reaction := strings.TrimSpace(strings.ToLower(value.Reaction))
	if reaction != "up" && reaction != "down" {
		errors["Reaction"] = "Choose thumbs up or thumbs down."
	}
	if strings.TrimSpace(value.Subject) == "" {
		errors["Subject"] = "Add a short headline for your review or question."
	}
	if len(strings.TrimSpace(value.Body)) < 8 {
		errors["Body"] = "Share a more specific note so other buyers get useful context."
	}
	return errors
}

func publicProductFeedbackForm(product productCard, payload Payload, form ui.Form[publicCommentFormState], value publicCommentFormState, submitting bool, submissionMessage string, setAuthorName ui.Handler, setSubject ui.Handler, setBody ui.Handler, setReaction ui.Handler, submit ui.Handler) ui.Node {
	children := []ui.Node{
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Share your review or question")),
		html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("Use the same product page to leave a quick review or ask a buying question without opening a separate support flow.")),
	}
	if strings.TrimSpace(submissionMessage) != "" {
		children = append(children, html.P(html.Props{Class: "rounded-[1.1rem] border border-emerald-400/25 bg-emerald-400/10 px-4 py-3 text-sm font-medium text-emerald-100"}, html.Text(submissionMessage)))
	}
	if strings.TrimSpace(form.FormError()) != "" {
		children = append(children, html.P(html.Props{Class: "rounded-[1.1rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 text-sm font-medium text-rose-100"}, html.Text(form.FormError())))
	}
	children = append(children, prependCSRFToken(payload.CSRF)...)
	children = append(children,
		html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
			html.Span(html.Props{}, html.Text("Name")),
			html.Input(html.Props{Name: "author_name", Value: value.AuthorName, AutoComplete: "name", OnInput: setAuthorName, Class: publicCommentFieldClass(form.Error("AuthorName") != "")}),
			publicCommentFieldError(form.Error("AuthorName")),
		),
		publicCommentReactionInput(value.Reaction, setReaction, form.Error("Reaction")),
		html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
			html.Span(html.Props{}, html.Text("Headline")),
			html.Input(html.Props{Name: "subject", Value: value.Subject, OnInput: setSubject, Class: publicCommentFieldClass(form.Error("Subject") != "")}),
			publicCommentFieldError(form.Error("Subject")),
		),
		html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
			html.Span(html.Props{}, html.Text("Comment")),
			html.Textarea(html.Props{Name: "body", OnInput: setBody, Class: publicCommentTextareaClass(form.Error("Body") != "")}, html.Text(value.Body)),
			publicCommentFieldError(form.Error("Body")),
		),
		html.Button(html.Props{Type: "submit", Disabled: submitting, Class: publicCommentSubmitClass(submitting)}, html.Text(publicCommentSubmitLabel(submitting))),
	)
	return html.Form(html.Props{Action: "/api/public/products/" + product.Slug + "/comments", Method: "post", OnSubmit: submit, Class: "grid gap-4 rounded-[1.6rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"}, children...)
}

func cloneCommentRecords(items []commentRecord) []commentRecord {
	if len(items) == 0 {
		return []commentRecord{}
	}
	cloned := make([]commentRecord, len(items))
	copy(cloned, items)
	return cloned
}

func commentRecordsSignature(items []commentRecord) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item.ID+":"+item.Status+":"+item.UpdatedAt)
	}
	return strings.Join(parts, "|")
}

func submitPublicComment(slug string, input publicCommentFormState, csrfToken string) (commentRecord, ui.ServerFormErrors, error) {
	body, err := json.Marshal(map[string]string{
		"author_name": input.AuthorName,
		"reaction":    input.Reaction,
		"subject":     input.Subject,
		"body":        input.Body,
	})
	if err != nil {
		return commentRecord{}, ui.ServerFormErrors{}, err
	}
	request, err := http.NewRequest(http.MethodPost, "/api/public/products/"+slug+"/comments", bytes.NewReader(body))
	if err != nil {
		return commentRecord{}, ui.ServerFormErrors{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	headerName, headerValue := ui.NewCSRFToken(csrfToken).Header()
	request.Header.Set(headerName, headerValue)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return commentRecord{}, ui.ServerFormErrors{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		var created commentRecord
		if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
			return commentRecord{}, ui.ServerFormErrors{}, err
		}
		return created, ui.ServerFormErrors{}, nil
	}
	var failure ui.ServerFormErrors
	if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
		return commentRecord{}, ui.ServerFormErrors{}, err
	}
	failure.Fields = normalizePublicCommentFieldErrors(failure.Fields)
	if failure.FormMessage() == "" {
		failure.Message = "Fix the highlighted fields and try again."
	}
	return commentRecord{}, failure, nil
}

func mergePublicCommentList(items []commentRecord, pendingComment commentRecord) []commentRecord {
	merged := cloneCommentRecords(items)
	if strings.TrimSpace(pendingComment.ID) == "" {
		return merged
	}
	for _, item := range merged {
		if item.ID == pendingComment.ID {
			return merged
		}
	}
	return append([]commentRecord{pendingComment}, merged...)
}

func publicCommentStatusBadge(status string) ui.Node {
	trimmed := strings.TrimSpace(strings.ToLower(status))
	if trimmed == "" || trimmed == "approved" {
		return nil
	}
	className := "rounded-full border px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em]"
	switch trimmed {
	case "pending":
		className += " border-amber-400/25 bg-amber-400/10 text-amber-200"
	case "flagged", "rejected":
		className += " border-rose-400/25 bg-rose-400/10 text-rose-200"
	default:
		className += " border-white/10 bg-white/8 text-stone-300"
	}
	return html.Span(html.Props{Class: className}, html.Text(strings.ReplaceAll(trimmed, "_", " ")))
}

func fetchPublicProductComments(slug string) ([]commentRecord, error) {
	request, err := http.NewRequest(http.MethodGet, "/api/public/products/"+slug+"/comments", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var payload struct {
		Items []commentRecord `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return cloneCommentRecords(payload.Items), nil
}

func normalizePublicCommentFieldErrors(fields ui.FieldErrors) ui.FieldErrors {
	if len(fields) == 0 {
		return nil
	}
	normalized := ui.FieldErrors{}
	for key, value := range fields {
		switch strings.TrimSpace(strings.ToLower(key)) {
		case "author_name":
			normalized["AuthorName"] = value
		case "reaction":
			normalized["Reaction"] = value
		case "subject":
			normalized["Subject"] = value
		case "body":
			normalized["Body"] = value
		default:
			normalized[key] = value
		}
	}
	return normalized
}

func publicCommentSubmitClass(submitting bool) string {
	className := "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"
	if submitting {
		return className + " cursor-wait opacity-70"
	}
	return className
}

func publicCommentSubmitLabel(submitting bool) string {
	if submitting {
		return "Sending..."
	}
	return "Share feedback"
}

func publicCommentReactionInput(selected string, handler ui.Handler, errorText string) ui.Node {
	if strings.TrimSpace(selected) == "" {
		selected = "up"
	}
	return html.Fieldset(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-white/5 p-4"},
		html.Legend(html.Props{Class: "px-1 text-sm font-medium text-stone-300"}, html.Text("Your reaction")),
		html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
			html.Label(html.Props{Class: "flex cursor-pointer items-start gap-3 rounded-[1.1rem] border border-emerald-400/25 bg-white/6 px-4 py-3 text-sm text-stone-300"},
				html.Input(html.Props{Type: "radio", Name: "reaction", Value: "up", Checked: strings.TrimSpace(selected) == "up", OnChange: handler, Class: "mt-1 h-4 w-4 border-stone-300 text-emerald-600"}),
				html.Span(html.Props{Class: "grid gap-1"},
					html.Span(html.Props{Class: "font-semibold text-white"}, html.Text("Thumbs up")),
					html.Span(html.Props{Class: "text-xs leading-6 text-stone-400"}, html.Text("Recommend it or confirm the setup met expectations.")),
				),
			),
			html.Label(html.Props{Class: "flex cursor-pointer items-start gap-3 rounded-[1.1rem] border border-rose-400/25 bg-white/6 px-4 py-3 text-sm text-stone-300"},
				html.Input(html.Props{Type: "radio", Name: "reaction", Value: "down", Checked: strings.TrimSpace(selected) == "down", OnChange: handler, Class: "mt-1 h-4 w-4 border-stone-300 text-rose-600"}),
				html.Span(html.Props{Class: "grid gap-1"},
					html.Span(html.Props{Class: "font-semibold text-white"}, html.Text("Thumbs down")),
					html.Span(html.Props{Class: "text-xs leading-6 text-stone-400"}, html.Text("Call out delivery friction, finish issues, or fit concerns buyers should know.")),
				),
			),
		),
		publicCommentFieldError(errorText),
	)
}

func publicCommentFieldClass(hasError bool) string {
	className := "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"
	if hasError {
		return className + " border-rose-300 bg-rose-50/60 focus:border-rose-400"
	}
	return className
}

func publicCommentTextareaClass(hasError bool) string {
	className := "min-h-28 rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"
	if hasError {
		return className + " border-rose-300 bg-rose-50/60 focus:border-rose-400"
	}
	return className
}

func publicCommentFieldError(message string) ui.Node {
	if strings.TrimSpace(message) == "" {
		return nil
	}
	return html.P(html.Props{Class: "text-sm font-medium text-rose-600"}, html.Text(message))
}

func formatPublicCommentDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "recently"
	}
	if len(trimmed) >= 10 {
		return trimmed[:10]
	}
	return trimmed
}

func publicProductHeroCard(product productCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 rounded-[1.9rem] border border-white/10 bg-[linear-gradient(145deg,rgba(13,18,30,0.96),rgba(18,25,40,0.9))] p-6 shadow-[0_28px_65px_rgba(0,0,0,0.22)] lg:p-7"},
		html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,1.3fr)_minmax(18rem,0.8fr)] lg:items-start"},
			html.Div(html.Props{Class: "grid gap-5"},
				publicProductIdentity(product),
				publicProductStory(product),
				html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-3"},
					publicSignalPill(productCategoryCue(product.Category)),
					publicSignalPill(productSupportCue(product.Status)),
					publicSignalPill(productBuyingMotion(product.Status)),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/6 p-5"},
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"}, html.Text(publicStartingAtLabel)),
				html.P(html.Props{Class: "text-4xl font-black tracking-[-0.05em] text-white"}, html.Text(formatPrice(product.PriceCents))),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("A standard product summary keeps price, stock posture, and next steps visible before buyers move into reviews or delivery planning.")),
				html.Div(html.Props{Class: "grid gap-3"},
					html.P(html.Props{Class: "rounded-[1.1rem] border border-white/10 bg-white/6 px-4 py-3 text-sm text-stone-300"}, html.Text(catalogEditorialCopy(product))),
				),
			),
		),
	)
}

func publicProductIdentity(product productCard) ui.Node {
	return html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(product.Category)),
		html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-300"}, html.Text(product.SKU)),
		html.Span(html.Props{Class: publicStatusClass(product.Status)}, html.Text(publicStatusLabel(product.Status))),
	)
}

func publicProductStory(product productCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.H2(html.Props{Class: "max-w-4xl text-3xl font-black tracking-[-0.04em] text-white sm:text-4xl lg:text-[3rem]"}, html.Text(product.Title)),
		html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300"}, html.Text(product.Summary)),
	)
}

func publicProductContextColumn(label string, copy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-3 rounded-[1.25rem] border border-white/10 bg-white/6 p-4 shadow-[0_14px_30px_rgba(0,0,0,0.12)]"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"}, html.Text(label)),
		html.P(html.Props{Class: "text-sm leading-6 text-stone-200"}, html.Text(copy)),
	)
}

func publicProductMetrics() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text("Why this product page is easier to use")),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			publicMetricCard(publicRegionalPromiseLabel, "Availability, delivery timing, and status stay grouped together instead of being buried below the fold."),
			publicMetricCard(publicCommercialSupportLabel, "Primary quote and availability actions stay in a dedicated rail that matches common commerce layouts."),
			publicMetricCard(publicOperationalContinuityLabel, "The same summary, pricing, and review signals stay visible from SSR through client takeover."),
		),
	)
}

func publicProductFeatureStrip() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text("Quick buying notes")),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			publicFeatureCard(publicQuietHierarchyLabel, "Core product details lead, support actions stay to the side, and reviews sit below in a familiar reading order."),
			publicFeatureCard(publicProgressiveFormsLabel, "Pricing, availability, and feedback all stay attached to the product instead of splitting across routes."),
			publicFeatureCard(publicPremiumUtilityLabel, "The darker palette reduces visual noise and puts emphasis on product title, price, and action labels."),
		),
	)
}

func publicProductActionRail(product productCard, payload Payload) ui.Node {
	productSupportTitle, productSupportCopy, productSupportPoints := productSupportPlan(product.Status)
	return html.Div(html.Props{Class: "grid gap-5 xl:sticky xl:top-24"},
		html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(publicBuyerNextStepLabel)),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(productSupportTitle)),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(productSupportCopy)),
			html.Div(html.Props{Class: "grid gap-3 text-sm text-stone-300"}, publicSupportPoints(productSupportPoints)...),
		),
		productPrimaryActionForm(product, payload),
		productSecondaryActionCard(product),
	)
}

func publicSupportPoints(points []string) []ui.Node {
	nodes := make([]ui.Node, 0, len(points))
	for _, point := range points {
		nodes = append(nodes, html.P(html.Props{Class: "rounded-[1.2rem] border border-white/10 bg-white/6 px-4 py-3"}, html.Text(point)))
	}
	return nodes
}

func renderWarehouseListContent(page catalogPage) ui.Node {
	nodes := make([]ui.Node, 0, len(page.Items))
	for _, item := range page.Items {
		nodes = append(nodes, publicWarehouseProductCard(item))
	}
	if len(nodes) == 0 {
		nodes = append(nodes, html.Div(html.Props{Class: "rounded-[1.8rem] border border-stone-200/80 bg-white/80 p-6 text-sm leading-7 text-stone-600 shadow-[0_18px_40px_rgba(120,107,82,0.08)]"}, html.Text("No products matched the current filters. Clear the query or widen the status filter to bring products back into view.")))
	}
	return html.Section(html.Props{Class: "grid gap-8"},
		publicWarehouseListOverview(page),
		publicWarehouseProductControls(page),
		html.Div(html.Props{Class: "flex flex-col gap-4"}, nodes...),
	)
}

func publicWarehouseListOverview(page catalogPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 rounded-[2rem] border border-stone-200/80 bg-white/65 p-6 shadow-[0_18px_40px_rgba(120,107,82,0.08)] lg:grid-cols-[minmax(0,1.2fr)_minmax(18rem,0.8fr)] lg:items-end"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.3em] text-stone-500"}, html.Text("Atlas product volume")),
			html.H2(html.Props{Class: "text-3xl font-black tracking-[-0.03em] text-stone-950"}, html.Text("Sorted products and live volume, without warehouse picking.")),
			html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-600"}, html.Text("This route now starts with products. Search, filter, and sort the catalog by product status and total volume instead of choosing a warehouse first.")),
		),
		html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
			publicMetricCard(fmt.Sprintf("%d products", len(page.Items)), "The route now stays product-first and drops the warehouse directory cards entirely."),
			publicMetricCard(fmt.Sprintf("%d total volume", totalPublicWarehouseVolume(page.Items)), fmt.Sprintf("%d low-stock products still need attention.", countPublicLowStockProducts(page.Items))),
		),
	)
}

func publicWarehouseProductControls(page catalogPage) ui.Node {
	return html.Form(html.Props{Action: RouteWarehouses, Method: "get", Class: "flex flex-col gap-4 rounded-[1.8rem] border border-stone-200/80 bg-white/72 p-5 shadow-[0_18px_40px_rgba(120,107,82,0.08)] lg:flex-row lg:flex-wrap lg:items-end"},
		publicWarehouseProductInput("q", "Search", page.Query.Search),
		publicWarehouseProductSelect("category", "Category", page.Query.Category, append([]optionItem{{"", "All categories"}}, productCategoryOptions()...)),
		publicWarehouseProductSelect("status", "Status", page.Query.Status, append([]optionItem{{"", "All statuses"}}, productStatusOptions()...)),
		publicWarehouseProductSelect("sort", "Sort", page.Query.Sort, []optionItem{{"volume", "Highest volume"}, {"updated", "Recently updated"}, {"price", "Highest price"}, {"status", "Status"}}),
		html.Button(html.Props{Type: "submit", Class: "rounded-full bg-stone-950 px-5 py-3 text-sm font-semibold text-stone-50 transition hover:bg-stone-800"}, html.Text("Apply")),
	)
}

func publicWarehouseProductInput(name, label, value string) ui.Node {
	return html.Label(html.Props{Class: "flex flex-col gap-2 text-sm font-medium text-stone-700 lg:min-w-[16rem] lg:flex-[1.4]"},
		html.Span(html.Props{}, html.Text(label)),
		html.Input(html.Props{Name: name, Value: value, Class: "rounded-[1.1rem] border border-stone-200 bg-white px-4 py-3 text-stone-950 outline-none transition focus:border-amber-500/70 focus:bg-white"}),
	)
}

func publicWarehouseProductSelect(name, label, value string, options []optionItem) ui.Node {
	children := make([]ui.Node, 0, len(options))
	for _, option := range options {
		selected := strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(option.Value)) || (strings.TrimSpace(value) == "" && option.Value == "")
		children = append(children, html.Option(html.Props{Value: option.Value, Selected: selected}, html.Text(option.Label)))
	}
	return html.Label(html.Props{Class: "flex flex-col gap-2 text-sm font-medium text-stone-700 lg:min-w-[11rem] lg:flex-1"},
		html.Span(html.Props{}, html.Text(label)),
		html.Select(html.Props{Name: name, Class: "rounded-[1.1rem] border border-stone-200 bg-white px-4 py-3 text-stone-950 outline-none transition focus:border-amber-500/70 focus:bg-white"}, children...),
	)
}

func publicWarehouseProductCard(item productCard) ui.Node {
	return html.A(html.Props{Href: publicWarehouseProductHref(item), Class: "group flex flex-col gap-5 rounded-[2rem] border border-stone-200/80 bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(248,244,238,0.9))] p-6 shadow-[0_20px_48px_rgba(120,107,82,0.09)] transition hover:border-stone-300 hover:bg-white hover:shadow-[0_28px_65px_rgba(120,107,82,0.14)] lg:flex-row lg:items-start lg:justify-between"},
		html.Div(html.Props{Class: "flex flex-1 flex-col gap-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
				html.Div(html.Props{Class: "flex flex-col gap-2"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(item.Category)),
					html.P(html.Props{Class: "text-[0.68rem] font-medium uppercase tracking-[0.28em] text-stone-500"}, html.Text(item.SKU)),
				),
				html.Span(html.Props{Class: strings.Replace(publicStatusClass(item.Status), "text-white", "text-stone-900", 1)}, html.Text(publicStatusLabel(item.Status))),
			),
			html.Div(html.Props{Class: "flex flex-col gap-3"},
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-stone-950 transition group-hover:text-stone-800"}, html.Text(item.Title)),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text(item.Summary)),
			),
			html.Div(html.Props{Class: "flex items-center justify-between gap-4 rounded-[1.45rem] border border-stone-200/75 bg-white/75 p-4 text-sm text-stone-600"},
				html.P(html.Props{Class: "leading-6"}, html.Text("Open the warehouse item profile for the full CRUD controls, stock edits, and replenishment elements tied to this row.")),
				html.Span(html.Props{Class: "font-semibold text-stone-900 transition group-hover:text-stone-700"}, html.Text("Open item profile")),
			),
		),
		html.Div(html.Props{Class: "flex flex-col gap-3 lg:min-w-[22rem] lg:max-w-[24rem]"},
			html.Div(html.Props{Class: "flex flex-wrap gap-3 lg:flex-col"},
				publicWarehouseProductMetric("Available", fmt.Sprintf("%d", item.Available)),
				publicWarehouseProductMetric("Inbound", fmt.Sprintf("%d", item.Inbound)),
				publicWarehouseProductMetric("Volume", fmt.Sprintf("%d", item.Volume)),
				publicWarehouseProductMetric("Price", formatPrice(item.PriceCents)),
			),
		),
	)
}

func publicWarehouseProductHref(item productCard) string {
	if strings.TrimSpace(item.WarehouseID) != "" && strings.TrimSpace(item.SKU) != "" {
		return RouteWarehouseOps + "/" + item.WarehouseID + "/items/" + item.SKU
	}
	return "/app/products/" + item.Slug
}

func publicWarehouseProductMetric(label string, value string) ui.Node {
	return html.Div(html.Props{Class: "flex min-w-[10rem] flex-1 items-center justify-between gap-4 rounded-[1.2rem] border border-stone-200/75 bg-white/75 px-4 py-3 lg:min-w-0"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(label)),
		html.P(html.Props{Class: "text-lg font-black tracking-[-0.03em] text-stone-950"}, html.Text(value)),
	)
}

func totalPublicWarehouseVolume(items []productCard) int {
	total := 0
	for _, item := range items {
		total += item.Volume
	}
	return total
}

func countPublicLowStockProducts(items []productCard) int {
	count := 0
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Status), "low_stock") || item.Available <= 3 {
			count++
		}
	}
	return count
}

func renderWarehouseDetailContent(page warehouseDetailPage) ui.Node {
	warehouse := page.Warehouse
	return html.Section(html.Props{Class: "grid gap-8 lg:grid-cols-[minmax(0,1.15fr)_minmax(18rem,0.78fr)]"},
		publicWarehouseDetailHero(warehouse),
		publicWarehouseDetailFeatureStrip(),
		listCard(publicRegionalAvailabilityPicksLabel, publicWarehouseRegionalProducts(warehouse, page.Products)...),
	)
}

func publicWarehouseDetailHero(warehouse warehouseCard) ui.Node {
	return html.Div(html.Props{Class: "relative overflow-hidden rounded-[2.3rem] border border-stone-200/80 bg-[linear-gradient(145deg,rgba(255,255,255,0.96),rgba(244,236,224,0.94)_55%,rgba(231,220,202,0.92))] p-7 shadow-[0_28px_65px_rgba(120,107,82,0.14)]"},
		html.Div(html.Props{Class: "pointer-events-none absolute -right-10 top-6 h-36 w-36 rounded-full bg-amber-200/35 blur-3xl"}),
		html.Div(html.Props{Class: "relative z-[1] grid gap-7 lg:grid-cols-[minmax(0,1.05fr)_minmax(15rem,0.72fr)]"},
			html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(warehouse.Region)),
					html.Span(html.Props{Class: "rounded-full border border-white/80 bg-white/70 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(publicRegionalHubLabel)),
					html.Span(html.Props{Class: "rounded-full border border-stone-200 bg-white/85 px-4 py-2 text-sm font-semibold text-stone-800"}, html.Text(warehouse.ServiceLevel)),
				),
				html.H2(html.Props{Class: "text-4xl font-black tracking-[-0.04em] text-stone-950"}, html.Text(warehouse.Name)),
				html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-600"}, html.Text(warehouse.PublicSummary)),
				html.Div(html.Props{Class: "grid gap-4 rounded-[1.75rem] border border-white/75 bg-white/55 p-5 backdrop-blur-sm sm:grid-cols-2"},
					publicProductContextColumn(publicRegionalReadLabel, warehouseRegionCue(warehouse.Region)),
					publicProductContextColumn(publicServicePostureLabel, warehouseServiceTone(warehouse.ServiceLevel)),
				),
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.A(html.Props{Href: RouteCatalog + "?warehouse=" + url.QueryEscape(warehouse.ID), Class: "rounded-full bg-stone-950 px-5 py-3 text-sm font-semibold text-stone-50 transition hover:bg-stone-800"}, html.Text(publicBrowseRegionalProductsLabel)),
					html.A(html.Props{Href: RouteCatalog, Class: "rounded-full border border-stone-300 bg-white/75 px-5 py-3 text-sm font-semibold text-stone-900 transition hover:border-stone-500 hover:bg-white"}, html.Text(publicCompareAllSystemsLabel)),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 rounded-[1.9rem] border border-stone-200/80 bg-white/78 p-5 shadow-[0_18px_40px_rgba(120,107,82,0.08)]"},
				publicMetricCard(publicRegionalFocusLabel, warehouse.Region),
				publicMetricCard(publicServiceLevelLabel, warehouse.ServiceLevel),
				publicMetricCard(publicPromiseLensLabel, "Use this region when you want the clearest path to timing and availability for nearby projects."),
			),
		),
	)
}

func publicWarehouseDetailFeatureStrip() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3 lg:col-span-2"},
		publicFeatureCard("Clear regional choices", "Regional pages stay calm and readable instead of collapsing into shipping jargon."),
		publicFeatureCard("Regional trust", "Coverage, speed, and product fit stay legible before a buyer ever opens a detailed delivery route."),
		publicFeatureCard("Connected discovery", "You can move from region details into matching products without losing pricing or delivery context."),
	)
}

func publicWarehouseRegionalProducts(warehouse warehouseCard, products []productCard) []ui.Node {
	nodes := make([]ui.Node, 0, len(products))
	for _, item := range products {
		nodes = append(nodes, html.A(html.Props{Href: RouteWarehouses + "/" + warehouse.Slug + "/availability/" + item.Slug, Class: "grid gap-3 rounded-[1.6rem] border border-stone-200/75 bg-white/75 p-5 transition hover:border-stone-300 hover:bg-white"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-lg font-semibold text-stone-950"}, html.Text(item.Title)),
					html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(item.SKU+" · "+item.Category)),
				),
				html.Span(html.Props{Class: publicStatusClass(item.Status)}, html.Text(publicStatusLabel(item.Status))),
			),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text(item.Summary)),
			html.Div(html.Props{Class: "flex items-end justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-1"},
					html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(publicStartingAtLabel)),
					html.P(html.Props{Class: "text-xl font-black tracking-[-0.03em] text-stone-950"}, html.Text(formatPrice(item.PriceCents))),
				),
				html.Span(html.Props{Class: "text-sm font-semibold text-stone-900"}, html.Text(publicOpenRegionalAvailabilityLabel)),
			),
		))
	}
	return nodes
}

func renderAvailabilityContent(availability availabilityPage, payload Payload) ui.Node {
	availabilityTitle, availabilityCopy := availabilitySupportPlan(availability.Available, availability.Inbound)
	return html.Section(html.Props{Class: "grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.8fr)]"},
		html.Div(html.Props{Class: "grid gap-5"},
			publicAvailabilityHero(availability),
			publicAvailabilityFeatureStrip(),
		),
		html.Div(html.Props{Class: "grid gap-5"},
			html.Div(html.Props{Class: "grid gap-4 rounded-[2rem] border border-stone-200/80 bg-white/78 p-6 shadow-[0_18px_45px_rgba(120,107,82,0.08)]"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(publicRegionalNextStepLabel)),
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-stone-950"}, html.Text(availabilityTitle)),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text(availabilityCopy)),
			),
			availabilityPrimaryActionForm(availability, payload),
			availabilityQuestionActionForm(availability, payload),
		),
	)
}

func publicAvailabilityHero(availability availabilityPage) ui.Node {
	return html.Div(html.Props{Class: "relative overflow-hidden rounded-[2.3rem] border border-stone-200/80 bg-[linear-gradient(145deg,rgba(255,255,255,0.96),rgba(244,236,224,0.94)_55%,rgba(231,220,202,0.92))] p-7 shadow-[0_28px_65px_rgba(120,107,82,0.14)]"},
		html.Div(html.Props{Class: "pointer-events-none absolute -right-10 top-6 h-36 w-36 rounded-full bg-amber-200/35 blur-3xl"}),
		html.Div(html.Props{Class: "relative z-[1] grid gap-7 lg:grid-cols-[minmax(0,1.05fr)_minmax(15rem,0.72fr)]"},
			html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(availability.Warehouse.Name)),
					html.Span(html.Props{Class: "rounded-full border border-white/80 bg-white/70 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(availability.Warehouse.Region)),
					html.Span(html.Props{Class: publicStatusClass(availability.Status)}, html.Text(publicStatusLabel(availability.Status))),
				),
				html.H2(html.Props{Class: "text-4xl font-black tracking-[-0.04em] text-stone-950"}, html.Text(availability.Product.Title)),
				html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-600"}, html.Text(fmt.Sprintf("%d available now with %d inbound at %s.", availability.Available, availability.Inbound, availability.Warehouse.Name))),
				html.Div(html.Props{Class: "grid gap-4 rounded-[1.75rem] border border-white/75 bg-white/55 p-5 backdrop-blur-sm sm:grid-cols-2"},
					publicProductContextColumn(publicAvailabilityStoryLabel, availabilityStoryCopy(availability.Available, availability.Inbound)),
					publicProductContextColumn(publicWhyThisMattersLabel, "Product demand stays tied to a named hub, so promise language feels concrete instead of generic."),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 rounded-[1.9rem] border border-stone-200/80 bg-white/78 p-5 shadow-[0_18px_40px_rgba(120,107,82,0.08)]"},
				publicMetricCard(fmt.Sprintf("%d available", availability.Available), "Available now in this region."),
				publicMetricCard(fmt.Sprintf("%d inbound", availability.Inbound), "More units already scheduled for upcoming orders."),
				publicMetricCard(publicStatusLabel(availability.Status), "Status stays explicit so buyers know whether to quote now or plan ahead."),
			),
		),
	)
}

func publicAvailabilityFeatureStrip() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
		publicFeatureCard("Concrete promise", "Availability is framed around a real delivery region, which makes timing confidence easier to evaluate."),
		publicFeatureCard("Keep your place", "Availability capture stays next to the delivery story so buyer intent does not vanish when stock tightens."),
		publicFeatureCard("Clear next steps", "The route stays calm and practical without forcing buyers to learn Atlas operations language."),
	)
}

func renderPublicHero(payload Payload) ui.Node {
	config := publicHeroConfig(payload.Route.Path)
	if isPublicDetailRoute(payload.Route.Path) {
		return html.Section(html.Props{Class: "relative overflow-hidden rounded-[1.9rem] border border-white/10 bg-[linear-gradient(140deg,rgba(12,17,28,0.98),rgba(15,21,34,0.95)_58%,rgba(21,28,44,0.92)_100%)] px-6 py-6 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:px-8 lg:px-10"},
			html.Div(html.Props{Class: "pointer-events-none absolute inset-y-0 right-0 hidden w-[28%] bg-[radial-gradient(circle_at_center,_rgba(245,158,11,0.12),_transparent_70%)] lg:block"}),
			html.Div(html.Props{Class: "relative z-[1] grid gap-5"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3 text-[0.72rem] font-semibold uppercase tracking-[0.34em] text-amber-300"},
					html.Span(html.Props{Class: "rounded-full border border-amber-300/25 bg-amber-300/10 px-3 py-1"}, html.Text(config.kicker)),
					html.Span(html.Props{}, html.Text(publicRequestTimeSSRLabel)),
				),
				html.Div(html.Props{Class: "grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end lg:gap-6"},
					html.Div(html.Props{Class: "grid gap-3"},
						html.H1(html.Props{Class: "max-w-4xl text-4xl font-black tracking-[-0.04em] text-white sm:text-[3rem] lg:text-[3.35rem]"}, html.Text(fallback(payload.Route.Title, "Atlas"))),
						html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300"}, html.Text(routeSummary(payload.Route.Path))),
					),
					html.Div(html.Props{Class: "flex flex-col gap-3 sm:flex-row sm:flex-wrap lg:justify-end"},
						html.A(html.Props{Href: config.primaryHref, Class: "inline-flex items-center justify-center rounded-full bg-amber-300 px-6 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(config.primaryLabel)),
						html.A(html.Props{Href: config.secondaryHref, Class: "inline-flex items-center justify-center rounded-full border border-white/12 bg-white/6 px-6 py-3 text-sm font-semibold text-stone-200 transition hover:border-amber-300/35 hover:bg-white/10 hover:text-white"}, html.Text(config.secondaryLabel)),
					),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					publicSignalPill("24 workspace products"),
					publicSignalPill("3 delivery regions"),
					publicSignalPill(publicProgressiveFormsLabel),
				),
			),
		)
	}
	return html.Section(html.Props{Class: "relative overflow-hidden rounded-[2.1rem] border border-white/10 bg-[linear-gradient(140deg,rgba(12,17,28,0.98),rgba(16,22,36,0.95)_52%,rgba(20,27,42,0.92)_100%)] px-6 py-8 shadow-[0_35px_80px_rgba(0,0,0,0.24)] sm:px-8 lg:px-10 lg:py-10"},
		html.Div(html.Props{Class: "pointer-events-none absolute inset-y-0 right-0 hidden w-[36%] bg-[radial-gradient(circle_at_center,_rgba(245,158,11,0.18),_transparent_68%)] lg:block"}),
		html.Div(html.Props{Class: "pointer-events-none absolute -left-10 top-10 h-28 w-28 rounded-full bg-amber-300/14 blur-3xl"}),
		html.Div(html.Props{Class: "grid gap-8 lg:grid-cols-[minmax(0,1.35fr)_minmax(18rem,0.8fr)] lg:items-end"},
			html.Div(html.Props{Class: "relative z-[1] grid gap-5"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3 text-[0.72rem] font-semibold uppercase tracking-[0.34em] text-amber-300"},
					html.Span(html.Props{Class: "rounded-full border border-amber-300/25 bg-amber-300/10 px-3 py-1"}, html.Text(config.kicker)),
					html.Span(html.Props{}, html.Text(publicRequestTimeSSRLabel)),
				),
				html.H1(html.Props{Class: "max-w-4xl text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl lg:text-6xl"}, html.Text(fallback(payload.Route.Title, "Atlas"))),
				html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300 sm:text-lg"}, html.Text(routeSummary(payload.Route.Path))),
				html.Div(html.Props{Class: "flex flex-col gap-3 sm:flex-row sm:flex-wrap"},
					html.A(html.Props{Href: config.primaryHref, Class: "inline-flex items-center justify-center rounded-full bg-amber-300 px-6 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(config.primaryLabel)),
					html.A(html.Props{Href: config.secondaryHref, Class: "inline-flex items-center justify-center rounded-full border border-white/12 bg-white/6 px-6 py-3 text-sm font-semibold text-stone-200 transition hover:border-amber-300/35 hover:bg-white/10 hover:text-white"}, html.Text(config.secondaryLabel)),
				),
			),
			html.Div(html.Props{Class: "relative z-[1] grid gap-4 lg:justify-self-end lg:min-w-[19rem]"},
				publicMetricCard("24 workspace products", "Desks, storage, seating, and accessories organized for quick scanability."),
				publicMetricCard("3 delivery regions", "Nevada, Illinois, and New Jersey stay visible across delivery and availability flows."),
				publicMetricCard(publicProgressiveFormsLabel, "Quotes, availability requests, and product questions stay close to the buying decision."),
			),
		),
	)
}

func isPublicDetailRoute(path string) bool {
	if strings.HasPrefix(path, RouteCatalog+"/") {
		return true
	}
	if strings.Contains(path, "/availability/") {
		return true
	}
	return strings.HasPrefix(path, RouteWarehouses+"/") && path != RouteWarehouses
}

type publicHeroState struct {
	kicker         string
	primaryLabel   string
	primaryHref    string
	secondaryLabel string
	secondaryHref  string
}

func publicHeroConfig(path string) publicHeroState {
	state := publicHeroState{
		kicker:         "Warehouse-backed design systems",
		primaryLabel:   "Shop workspace systems",
		primaryHref:    RouteCatalog,
		secondaryLabel: "Explore warehouse network",
		secondaryHref:  RouteWarehouses,
	}
	switch {
	case path == RouteCatalog:
		state.kicker = "Curated modular workspace catalog"
	case strings.HasPrefix(path, RouteCatalog+"/"):
		state.kicker = "Product detail with delivery context"
		state.primaryLabel = "Browse more products"
		state.secondaryLabel = "See delivery by region"
	case path == RouteWarehouses:
		state.kicker = "Regional delivery options"
		state.primaryLabel = "See stocked products"
	case strings.HasPrefix(path, RouteWarehouses+"/"):
		state.kicker = "Regional delivery and availability"
		state.primaryLabel = "Browse the catalog"
	}
	return state
}
