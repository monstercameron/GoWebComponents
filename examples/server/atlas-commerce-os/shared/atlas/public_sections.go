package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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
	return html.Div(html.Props{Class: "grid gap-4 " + publicHeroSurfaceClass() + " shadow-[0_22px_50px_rgba(0,0,0,0.22)] sm:p-7"},
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

func renderCatalogContent(parsePage catalogPage) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseSearch := useAtlasSearchParams()
		parseInitial := atlasCatalogFilterState(parsePage.Query)
		parseForm := ui.UseForm(parseInitial)
		useAtlasEffect(func() func() {
			if !atlasSameListFilterState(parseForm.Get(), parseInitial) {
				parseForm.Reset(parseInitial)
			}
			return nil
		}, parseInitial)
		parseValue := parseForm.Get()
		parseDeferred := ui.UseDeferredValue(parseValue)
		parseDebounced := ui.UseDebounced(parseValue, atlasFilterSyncDelay)
		useAtlasEffect(func() func() {
			parseNext := atlasBuildListFilterQuery(parseSearch.Values(), parseDebounced.Get())
			if parseNext.Encode() != parseSearch.Values().Encode() {
				parseSearch.ReplaceAll(parseNext)
			}
			return nil
		}, parseDebounced.Get())
		parseSubmit := ui.UseEvent(func(parseEvent ui.FormEvent) {
			parseEvent.PreventDefault()
			parseSearch.ReplaceAll(atlasBuildListFilterQuery(parseSearch.Values(), parseForm.Get()))
		})
		parseFilteredItems := filterCatalogItems(parsePage.Items, parseDeferred)
		parseViewPage := parsePage
		parseViewPage.Items = parseFilteredItems
		parseItems := make([]ui.Node, 0, len(parseFilteredItems))
		for _, parseItem := range parseFilteredItems {
			parseItems = append(parseItems, publicCatalogCard(parseItem))
		}
		return html.Section(html.Props{Class: "grid gap-8"},
			publicCatalogOverview(parseViewPage),
			storeCatalogControls(parseForm, parseDebounced.Pending(), parseSubmit),
			html.Div(html.Props{Class: "grid gap-5 md:grid-cols-2 xl:grid-cols-3"}, parseItems...),
		)
	})
}

func publicCatalogOverview(parsePage catalogPage) ui.Node {
	return html.Div(html.Props{Class: "flex flex-col gap-4 " + publicGlassCardClass() + " lg:flex-row lg:items-end lg:justify-between"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text(publicCatalogOverviewLabel)),
			html.H2(html.Props{Class: "text-3xl font-black tracking-[-0.03em] text-white"}, html.Text("Modern workspace systems, organized for quick decisions.")),
			html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300"}, html.Text("The catalog follows a more standard browse flow: category, price, stock posture, and next step all stay in the same place for faster scanning.")),
		),
		html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2 lg:min-w-[20rem]"},
			publicMetricCard(fmt.Sprintf("%d products", len(parsePage.Items)), "Workspace systems stay organized for quick comparison and shortlist building."),
			publicMetricCard("Quote-ready", "Pricing help and delivery follow-up stay one step away on every product page."),
		),
	)
}

func publicCatalogCard(parseItem productCard) ui.Node {
	parseActionLabel, parseActionCopy := catalogActionPlan(parseItem.Status)
	return html.A(html.Props{Href: RouteCatalog + "/" + parseItem.Slug, Class: publicCatalogCardClass()},
		html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseItem.Category)),
				html.P(html.Props{Class: "text-[0.68rem] font-medium uppercase tracking-[0.28em] text-stone-500"}, html.Text(parseItem.SKU)),
			),
			html.Span(html.Props{Class: publicStatusClass(parseItem.Status)}, html.Text(publicStatusLabel(parseItem.Status))),
		),
		publicCatalogStory(parseItem),
		publicCatalogPriceRail(parseItem, parseActionLabel, parseActionCopy),
	)
}

func publicCatalogStory(parseItem productCard) ui.Node {
	return html.Div(html.Props{Class: "relative overflow-hidden rounded-[1.55rem] border border-white/8 bg-[linear-gradient(160deg,rgba(255,255,255,0.04),rgba(255,255,255,0.02)_58%,rgba(245,158,11,0.05))] px-5 py-6"},
		html.Div(html.Props{Class: "pointer-events-none absolute -right-6 top-5 h-24 w-24 rounded-full bg-amber-300/12 blur-2xl"}),
		html.Div(html.Props{Class: "relative z-[1] grid gap-4"},
			html.Div(html.Props{Class: "flex flex-wrap items-center gap-2 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"},
				html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-stone-300"}, html.Text(publicAtlasSystemLabel)),
				html.Span(html.Props{}, html.Text(catalogPromiseCopy(parseItem.Status))),
			),
			html.Div(html.Props{Class: "grid gap-3"},
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.04em] text-white transition group-hover:text-stone-100"}, html.Text(parseItem.Title)),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseItem.Summary)),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-stone-400"}, html.Text(catalogEditorialCopy(parseItem))),
		),
	)
}

func publicCatalogPriceRail(parseItem productCard, parseActionLabel string, parseActionCopy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-white/8 bg-white/5 p-4"},
		html.Div(html.Props{Class: "flex items-end justify-between gap-4"},
			html.Div(html.Props{Class: "grid gap-1"},
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"}, html.Text(publicStartingAtLabel)),
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.04em] text-white"}, html.Text(formatPrice(parseItem.PriceCents))),
				html.P(html.Props{Class: "text-sm leading-6 text-stone-400"}, html.Text(parseActionCopy)),
			),
			html.Span(html.Props{Class: "inline-flex items-center rounded-full border border-amber-300/50 bg-amber-300/12 px-4 py-2 text-sm font-semibold text-amber-100 transition group-hover:border-amber-300/70 group-hover:bg-amber-300/18"}, html.Text(parseActionLabel)),
		),
		html.Div(html.Props{Class: "grid gap-2 text-sm text-stone-300 sm:grid-cols-2"},
			html.P(html.Props{Class: "rounded-full border border-white/10 bg-white/6 px-3 py-2"}, html.Text(productCategoryCue(parseItem.Category))),
			html.P(html.Props{Class: "rounded-full border border-white/10 bg-white/6 px-3 py-2"}, html.Text(productSupportCue(parseItem.Status))),
		),
	)
}

func publicLazySectionFallback(parseEyebrow, parseTitle, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + publicGlassCardClass()},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseEyebrow)),
		html.H3(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(parseTitle)),
		html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseCopy)),
	)
}

func renderProductContent(parsePage productDetailPage, parsePayload Payload) ui.Node {
	parseProduct := parsePage.Product
	return html.Section(html.Props{Class: "grid gap-8 xl:grid-cols-[minmax(0,1.5fr)_minmax(21rem,0.78fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-6"},
			publicProductHeroCard(parseProduct),
			html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]"},
				publicProductMetrics(),
				publicProductFeatureStrip(),
			),
			atlasLazySection(func() ui.Node {
				return publicProductFeedbackSection(parseProduct, parsePage.Comments, parsePayload)
			}, publicLazySectionFallback("Customer reviews and questions", "Loading buyer feedback", "Atlas waits until the primary product story is stable before mounting the heavier review and question workflow."), parseProduct.Slug, commentRecordsSignature(parsePage.Comments)),
			publicProductPromiseLanesIsland(parseProduct),
		),
		publicProductActionRail(parseProduct, parsePayload),
	)
}

type relatedProductRecord struct {
	SKU           string `json:"sku"`
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	Category      string `json:"category"`
	Summary       string `json:"summary"`
	WarehouseID   string `json:"warehouseId"`
	WarehouseName string `json:"warehouseName"`
	Reason        string `json:"reason"`
}

func publicProductFeedbackSection(parseProduct productCard, parseComments []commentRecord, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseForm := ui.UseForm(publicCommentFormState{Reaction: "up"})
		parsePendingCommentsState := useAtlasState([]commentRecord{})
		parseSubmittingState := useAtlasState(false)
		parseSubmissionMessageState := useAtlasState("")
		parseCommentsResource := useAtlasCachedResource(CachedRequestResourceKey("/api/public/products/"+parseProduct.Slug+"/comments", "items"), func(parseCtx context.Context) ([]commentRecord, error) {
			return fetchPublicProductComments(parseCtx, parseProduct.Slug)
		})
		parseResourceState := parseCommentsResource.Get()
		useAtlasEffect(func() func() {
			parseCommentsResource.Set(cloneCommentRecords(parseComments))
			parsePendingCommentsState.Set([]commentRecord{})
			return nil
		}, parseProduct.Slug, commentRecordsSignature(parseComments))
		parseValue := parseForm.Get()
		parseVisibleComments := cloneCommentRecords(parseComments)
		if parseResourceState.Ready {
			parseVisibleComments = cloneCommentRecords(parseResourceState.Value)
		}
		for _, parsePending := range parsePendingCommentsState.Get() {
			parseVisibleComments = mergePublicCommentList(parseVisibleComments, parsePending)
		}
		parseSubmitting := parseSubmittingState.Get()
		parseSubmissionMessage := parseSubmissionMessageState.Get()
		isParseRefreshing := parseResourceState.Loading && parseResourceState.Ready
		parseCountLabel := fmt.Sprintf("%d buyer notes", len(parseVisibleComments))
		if len(parseVisibleComments) == 1 {
			parseCountLabel = "1 buyer note"
		}
		setAuthorName := ui.UseEvent(func(parseEvent ui.InputEvent) { parseForm.SetField("AuthorName", parseEvent.GetValue()) })
		setSubject := ui.UseEvent(func(parseEvent2 ui.InputEvent) { parseForm.SetField("Subject", parseEvent2.GetValue()) })
		setBody := ui.UseEvent(func(parseEvent3 ui.InputEvent) { parseForm.SetField("Body", parseEvent3.GetValue()) })
		setReaction := ui.UseEvent(func(parseEvent4 ui.ChangeEvent) { parseForm.SetField("Reaction", parseEvent4.GetValue()) })
		parseSubmit := ui.UseEvent(func(parseEvent5 ui.FormEvent) {
			parseEvent5.PreventDefault()
			if !parseForm.Validate(validatePublicCommentForm) {
				return
			}
			if parseSubmittingState.Get() {
				return
			}
			parseSubmittingState.Set(true)
			parseSubmissionMessageState.Set("")
			parseForm.SetFormError("")
			parseSnapshot := parseForm.Get()
			go func() {
				parseCreated, parseServerErrors, parseErr := submitPublicComment(parseProduct.Slug, parseSnapshot, parsePayload.CSRF)
				if parseErr != nil {
					parseForm.SetFormError("Comment submit failed. Retry in a moment.")
					parseSubmittingState.Set(false)
					return
				}
				if !parseForm.ApplyServerErrors(parseServerErrors) {
					parseSubmittingState.Set(false)
					return
				}
				parseForm.Reset(publicCommentFormState{Reaction: "up"})
				parseForm.SetErrors(nil)
				parseForm.SetFormError("")
				parseSubmissionMessageState.Set("Your comment was submitted for review.")
				dispatchAtlasShellToast(atlasShellToast{
					Title:  "Comment queued",
					Detail: "Atlas submitted the buyer comment and refreshed the thread without leaving the product route.",
					Tone:   "success",
				})
				parsePendingCommentsState.Set(mergePublicCommentList(parsePendingCommentsState.Get(), parseCreated))
				parseCommentsResource.Update(func(parseItems []commentRecord) []commentRecord {
					return mergePublicCommentList(parseItems, parseCreated)
				})
				parseCommentsResource.Reload()
				parseSubmittingState.Set(false)
			}()
		})
		return html.Div(html.Props{Class: "grid gap-5 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
			html.Div(html.Props{Class: "grid gap-2 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Customer reviews and questions")),
					html.H3(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("What buyers are asking before they commit.")),
					html.P(html.Props{Class: "max-w-3xl text-sm leading-7 text-stone-300"}, html.Text("Reviews, questions, and sentiment stay below the core product information in a standard decision flow, so buyers can scan the essentials first and validate with social proof second.")),
				),
				html.Div(html.Props{Class: "rounded-[1.2rem] border border-white/10 bg-white/8 px-4 py-3 text-sm font-semibold text-stone-200"}, html.Text(parseCountLabel)),
			),
			html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,0.95fr)_minmax(19rem,0.8fr)] lg:items-start"},
				publicProductFeedbackList(parseVisibleComments, isParseRefreshing),
				publicProductFeedbackForm(parseProduct, parsePayload, parseForm, parseValue, parseSubmitting, parseSubmissionMessage, setAuthorName, setSubject, setBody, setReaction, parseSubmit),
			),
		)
	})
}

func publicProductFeedbackList(parseComments []commentRecord, isRefreshing bool) ui.Node {
	parseThumbsUp, parseThumbsDown := publicCommentReactionCounts(parseComments)
	parseRatioLabel, parseRatioValue := publicCommentRatioSummary(parseThumbsUp, parseThumbsDown)
	parseNodes := make([]ui.Node, 0, len(parseComments))
	for _, parseItem := range parseComments {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "grid gap-3 rounded-[1.4rem] border border-white/10 bg-white/6 p-5 shadow-[0_14px_30px_rgba(0,0,0,0.14)]"},
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-base font-semibold text-white"}, html.Text(parseItem.Subject)),
					publicCommentReactionBadge(parseItem.Reaction),
				),
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-2"},
					html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-stone-400"}, html.Text(strings.ReplaceAll(parseItem.AuthorType, "_", " "))),
					publicCommentStatusBadge(parseItem.Status),
				),
			),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseItem.Body)),
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-amber-700"}, html.Text(parseItem.AuthorName+" · shared on "+formatPublicCommentDate(parseItem.CreatedAt))),
		))
	}
	if len(parseNodes) == 0 {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "rounded-[1.4rem] border border-white/10 bg-white/6 p-5 text-sm leading-7 text-stone-300 shadow-[0_14px_30px_rgba(0,0,0,0.14)]"}, html.Text("No buyer notes have been shared yet. The first approved question or review will appear here once Atlas has it.")))
	}
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: "grid gap-4 rounded-[1.4rem] border border-white/10 bg-white/6 p-5 shadow-[0_14px_30px_rgba(0,0,0,0.14)]"},
			html.Div(html.Props{Class: "grid gap-2 sm:grid-cols-[auto_minmax(0,1fr)] sm:items-end sm:gap-4"},
				html.Div(html.Props{Class: "text-3xl font-black tracking-[-0.05em] text-white"}, html.Text(parseRatioLabel)),
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text("Buyer sentiment snapshot")),
					html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("A quick read on whether approved feedback leans positive or points to delivery and fit concerns buyers should consider.")),
				),
			),
			html.Div(html.Props{Class: "h-3 overflow-hidden rounded-full bg-white/10"},
				html.Div(html.Props{Class: "h-full rounded-full bg-emerald-600", Style: map[string]string{"width": fmt.Sprintf("%d%%", parseRatioValue)}}),
			),
			html.Div(html.Props{Class: "grid gap-3 text-sm text-stone-300 sm:grid-cols-2"},
				html.P(html.Props{Class: "rounded-[1.1rem] border border-emerald-400/25 bg-emerald-400/10 px-4 py-3 font-semibold text-emerald-200"}, html.Text(fmt.Sprintf("%d thumbs up", parseThumbsUp))),
				html.P(html.Props{Class: "rounded-[1.1rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 font-semibold text-rose-200"}, html.Text(fmt.Sprintf("%d thumbs down", parseThumbsDown))),
			),
		),
	}
	if isRefreshing {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.1rem] border border-white/10 bg-white/8 px-4 py-3 text-sm font-medium text-stone-200"}, html.Text("Refreshing buyer notes...")))
	}
	parseChildren = append(parseChildren, html.Div(html.Props{Class: "grid gap-4"}, parseNodes...))
	return html.Div(html.Props{Class: "grid gap-4"}, parseChildren...)
}

func publicCommentReactionCounts(parseComments []commentRecord) (int, int) {
	parseThumbsUp := 0
	parseThumbsDown := 0
	for _, parseItem := range parseComments {
		if strings.TrimSpace(strings.ToLower(parseItem.Reaction)) == "down" {
			parseThumbsDown++
			continue
		}
		parseThumbsUp++
	}
	return parseThumbsUp, parseThumbsDown
}

func publicCommentRatioSummary(parseThumbsUp int, parseThumbsDown int) (string, int) {
	parseTotal := parseThumbsUp + parseThumbsDown
	if parseTotal == 0 {
		return "No ratings yet", 0
	}
	parsePercentage := int(float64(parseThumbsUp)*100/float64(parseTotal) + 0.5)
	return fmt.Sprintf("%d%% thumbs up", parsePercentage), parsePercentage
}

func publicCommentReactionBadge(parseReaction string) ui.Node {
	parseLabel := "Thumbs up"
	parseClassName := "inline-flex items-center rounded-full border border-emerald-400/25 bg-emerald-400/10 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-emerald-200"
	if strings.TrimSpace(strings.ToLower(parseReaction)) == "down" {
		parseLabel = "Thumbs down"
		parseClassName = "inline-flex items-center rounded-full border border-rose-400/25 bg-rose-400/10 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-rose-200"
	}
	return html.Span(html.Props{Class: parseClassName}, html.Text(parseLabel))
}

type publicCommentFormState struct {
	AuthorName string
	Reaction   string
	Subject    string
	Body       string
}

func validatePublicCommentForm(parseValue publicCommentFormState) ui.FieldErrors {
	parseErrors := ui.FieldErrors{}
	if strings.TrimSpace(parseValue.AuthorName) == "" {
		parseErrors["AuthorName"] = "Enter your name before sharing feedback."
	}
	parseReaction := strings.TrimSpace(strings.ToLower(parseValue.Reaction))
	if parseReaction != "up" && parseReaction != "down" {
		parseErrors["Reaction"] = "Choose thumbs up or thumbs down."
	}
	if strings.TrimSpace(parseValue.Subject) == "" {
		parseErrors["Subject"] = "Add a short headline for your review or question."
	}
	if len(strings.TrimSpace(parseValue.Body)) < 8 {
		parseErrors["Body"] = "Share a more specific note so other buyers get useful context."
	}
	return parseErrors
}

func publicProductFeedbackForm(parseProduct productCard, parsePayload Payload, parseForm ui.Form[publicCommentFormState], parseValue publicCommentFormState, isSubmitting bool, parseSubmissionMessage string, setAuthorName ui.Handler, setSubject ui.Handler, setBody ui.Handler, setReaction ui.Handler, parseSubmit ui.Handler) ui.Node {
	parseAuthorInputID := ui.UseId()
	parseAuthorErrorID := parseAuthorInputID + "-error"
	parseReactionFieldID := ui.UseId()
	parseReactionErrorID := parseReactionFieldID + "-error"
	parseSubjectInputID := ui.UseId()
	parseSubjectErrorID := parseSubjectInputID + "-error"
	parseBodyInputID := ui.UseId()
	parseBodyErrorID := parseBodyInputID + "-error"
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Share your review or question")),
		html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("Use the same product page to leave a quick review or ask a buying question without opening a separate support flow.")),
	}
	if strings.TrimSpace(parseSubmissionMessage) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.1rem] border border-emerald-400/25 bg-emerald-400/10 px-4 py-3 text-sm font-medium text-emerald-100"}, html.Text(parseSubmissionMessage)))
	}
	if strings.TrimSpace(parseForm.FormError()) != "" {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.1rem] border border-rose-400/25 bg-rose-400/10 px-4 py-3 text-sm font-medium text-rose-100"}, html.Text(parseForm.FormError())))
	}
	parseChildren = append(parseChildren, prependCSRFToken(parsePayload.CSRF)...)
	parseChildren = append(parseChildren,
		html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
			html.Span(html.Props{ID: parseAuthorInputID + "-label"}, html.Text("Name")),
			html.Input(html.Props{ID: parseAuthorInputID, Name: "author_name", Value: parseValue.AuthorName, AutoComplete: "name", OnInput: setAuthorName, Class: publicCommentFieldClass(parseForm.Error("AuthorName") != ""), Raw: map[string]any{"aria-labelledby": parseAuthorInputID + "-label", "aria-describedby": parseAuthorErrorID, "aria-invalid": parseForm.Error("AuthorName") != ""}}),
			publicCommentFieldError(parseAuthorErrorID, parseForm.Error("AuthorName")),
		),
		publicCommentReactionInput(parseReactionFieldID, parseReactionErrorID, parseValue.Reaction, setReaction, parseForm.Error("Reaction")),
		html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
			html.Span(html.Props{ID: parseSubjectInputID + "-label"}, html.Text("Headline")),
			html.Input(html.Props{ID: parseSubjectInputID, Name: "subject", Value: parseValue.Subject, OnInput: setSubject, Class: publicCommentFieldClass(parseForm.Error("Subject") != ""), Raw: map[string]any{"aria-labelledby": parseSubjectInputID + "-label", "aria-describedby": parseSubjectErrorID, "aria-invalid": parseForm.Error("Subject") != ""}}),
			publicCommentFieldError(parseSubjectErrorID, parseForm.Error("Subject")),
		),
		html.Label(html.Props{Class: "grid gap-2 text-sm font-medium text-stone-300"},
			html.Span(html.Props{ID: parseBodyInputID + "-label"}, html.Text("Comment")),
			html.Textarea(html.Props{ID: parseBodyInputID, Name: "body", OnInput: setBody, Class: publicCommentTextareaClass(parseForm.Error("Body") != ""), Raw: map[string]any{"aria-labelledby": parseBodyInputID + "-label", "aria-describedby": parseBodyErrorID, "aria-invalid": parseForm.Error("Body") != ""}}, html.Text(parseValue.Body)),
			publicCommentFieldError(parseBodyErrorID, parseForm.Error("Body")),
		),
		html.Button(html.Props{Type: "submit", Disabled: isSubmitting, Class: publicCommentSubmitClass(isSubmitting)}, html.Text(publicCommentSubmitLabel(isSubmitting))),
	)
	return html.Form(html.Props{Action: "/api/public/products/" + parseProduct.Slug + "/comments", Method: "post", OnSubmit: parseSubmit, Class: "grid gap-4 rounded-[1.6rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"}, parseChildren...)
}

func cloneCommentRecords(parseItems []commentRecord) []commentRecord {
	if len(parseItems) == 0 {
		return []commentRecord{}
	}
	parseCloned := make([]commentRecord, len(parseItems))
	copy(parseCloned, parseItems)
	return parseCloned
}

func commentRecordsSignature(parseItems []commentRecord) string {
	if len(parseItems) == 0 {
		return ""
	}
	parseParts := make([]string, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseParts = append(parseParts, parseItem.ID+":"+parseItem.Status+":"+parseItem.UpdatedAt)
	}
	return strings.Join(parseParts, "|")
}

func submitPublicComment(parseSlug string, parseInput publicCommentFormState, parseCsrfToken string) (commentRecord, ui.ServerFormErrors, error) {
	parseBody := map[string]string{
		"author_name": parseInput.AuthorName,
		"reaction":    parseInput.Reaction,
		"subject":     parseInput.Subject,
		"body":        parseInput.Body,
	}
	parseHeaderName, parseHeaderValue := ui.NewCSRFToken(parseCsrfToken).Header()
	parseResult := <-atlasFetch("/api/public/products/"+parseSlug+"/comments", atlasFetchOptions{
		Method: http.MethodPost,
		Headers: map[string]any{
			"Content-Type":  "application/json",
			"Accept":        "application/json",
			parseHeaderName: parseHeaderValue,
		},
		Body: parseBody,
	})
	if strings.TrimSpace(parseResult.Error) != "" {
		return commentRecord{}, ui.ServerFormErrors{}, fmt.Errorf("%s", parseResult.Error)
	}
	if parseResult.Status >= 200 && parseResult.Status < 300 {
		var parseCreated commentRecord
		if parseErr := json.Unmarshal([]byte(parseResult.Data), &parseCreated); parseErr != nil {
			return commentRecord{}, ui.ServerFormErrors{}, parseErr
		}
		return parseCreated, ui.ServerFormErrors{}, nil
	}
	var parseFailure ui.ServerFormErrors
	if parseErr2 := json.Unmarshal([]byte(parseResult.Data), &parseFailure); parseErr2 != nil {
		return commentRecord{}, ui.ServerFormErrors{}, parseErr2
	}
	parseFailure.Fields = normalizePublicCommentFieldErrors(parseFailure.Fields)
	if parseFailure.FormMessage() == "" {
		parseFailure.Message = "Fix the highlighted fields and try again."
	}
	return commentRecord{}, parseFailure, nil
}

func mergePublicCommentList(parseItems []commentRecord, parsePendingComment commentRecord) []commentRecord {
	parseMerged := cloneCommentRecords(parseItems)
	if strings.TrimSpace(parsePendingComment.ID) == "" {
		return parseMerged
	}
	for _, parseItem := range parseMerged {
		if parseItem.ID == parsePendingComment.ID {
			return parseMerged
		}
	}
	return append([]commentRecord{parsePendingComment}, parseMerged...)
}

func publicCommentStatusBadge(parseStatus string) ui.Node {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseStatus))
	if parseTrimmed == "" || parseTrimmed == "approved" {
		return nil
	}
	parseClassName := "rounded-full border px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em]"
	switch parseTrimmed {
	case "pending":
		parseClassName += " border-amber-400/25 bg-amber-400/10 text-amber-200"
	case "flagged", "rejected":
		parseClassName += " border-rose-400/25 bg-rose-400/10 text-rose-200"
	default:
		parseClassName += " border-white/10 bg-white/8 text-stone-300"
	}
	return html.Span(html.Props{Class: parseClassName}, html.Text(strings.ReplaceAll(parseTrimmed, "_", " ")))
}

func fetchPublicProductComments(parseCtx context.Context, parseSlug string) ([]commentRecord, error) {
	parsePayload, parseErr := fetchAtlasJSON[struct {
		Items []commentRecord `json:"items"`
	}](parseCtx, "/api/public/products/"+parseSlug+"/comments")
	if parseErr != nil {
		return nil, parseErr
	}
	return cloneCommentRecords(parsePayload.Items), nil
}

func fetchPublicRelatedProducts(parseCtx context.Context, parseSlug string) ([]relatedProductRecord, error) {
	parsePayload, parseErr := fetchAtlasJSON[struct {
		Items []relatedProductRecord `json:"items"`
	}](parseCtx, "/api/public/products/"+parseSlug+"/related-products")
	if parseErr != nil {
		return nil, parseErr
	}
	parseItems := make([]relatedProductRecord, len(parsePayload.Items))
	copy(parseItems, parsePayload.Items)
	return parseItems, nil
}

func normalizePublicCommentFieldErrors(parseFields ui.FieldErrors) ui.FieldErrors {
	if len(parseFields) == 0 {
		return nil
	}
	parseNormalized := ui.FieldErrors{}
	for parseKey, parseValue := range parseFields {
		switch strings.TrimSpace(strings.ToLower(parseKey)) {
		case "author_name":
			parseNormalized["AuthorName"] = parseValue
		case "reaction":
			parseNormalized["Reaction"] = parseValue
		case "subject":
			parseNormalized["Subject"] = parseValue
		case "body":
			parseNormalized["Body"] = parseValue
		default:
			parseNormalized[parseKey] = parseValue
		}
	}
	return parseNormalized
}

func publicCommentSubmitClass(isSubmitting bool) string {
	parseClassName := "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"
	if isSubmitting {
		return parseClassName + " cursor-wait opacity-70"
	}
	return parseClassName
}

func publicCommentSubmitLabel(isSubmitting bool) string {
	if isSubmitting {
		return "Sending..."
	}
	return "Share feedback"
}

func publicCommentReactionInput(parseFieldID string, parseErrorID string, parseSelected string, parseHandler ui.Handler, parseErrorText string) ui.Node {
	if strings.TrimSpace(parseSelected) == "" {
		parseSelected = "up"
	}
	return html.Fieldset(html.Props{Class: "grid gap-3 rounded-[1.35rem] border border-white/10 bg-white/5 p-4", Raw: map[string]any{"aria-describedby": parseErrorID, "aria-invalid": strings.TrimSpace(parseErrorText) != ""}},
		html.Legend(html.Props{ID: parseFieldID + "-legend", Class: "px-1 text-sm font-medium text-stone-300"}, html.Text("Your reaction")),
		html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
			html.Label(html.Props{Class: "flex cursor-pointer items-start gap-3 rounded-[1.1rem] border border-emerald-400/25 bg-white/6 px-4 py-3 text-sm text-stone-300"},
				html.Input(html.Props{ID: parseFieldID + "-up", Type: "radio", Name: "reaction", Value: "up", Checked: strings.TrimSpace(parseSelected) == "up", OnChange: parseHandler, Class: "mt-1 h-4 w-4 border-stone-300 text-emerald-600", Raw: map[string]any{"aria-labelledby": parseFieldID + "-legend", "aria-describedby": parseErrorID}}),
				html.Span(html.Props{Class: "grid gap-1"},
					html.Span(html.Props{Class: "font-semibold text-white"}, html.Text("Thumbs up")),
					html.Span(html.Props{Class: "text-xs leading-6 text-stone-400"}, html.Text("Recommend it or confirm the setup met expectations.")),
				),
			),
			html.Label(html.Props{Class: "flex cursor-pointer items-start gap-3 rounded-[1.1rem] border border-rose-400/25 bg-white/6 px-4 py-3 text-sm text-stone-300"},
				html.Input(html.Props{ID: parseFieldID + "-down", Type: "radio", Name: "reaction", Value: "down", Checked: strings.TrimSpace(parseSelected) == "down", OnChange: parseHandler, Class: "mt-1 h-4 w-4 border-stone-300 text-rose-600", Raw: map[string]any{"aria-labelledby": parseFieldID + "-legend", "aria-describedby": parseErrorID}}),
				html.Span(html.Props{Class: "grid gap-1"},
					html.Span(html.Props{Class: "font-semibold text-white"}, html.Text("Thumbs down")),
					html.Span(html.Props{Class: "text-xs leading-6 text-stone-400"}, html.Text("Call out delivery friction, finish issues, or fit concerns buyers should know.")),
				),
			),
		),
		publicCommentFieldError(parseErrorID, parseErrorText),
	)
}

func publicCommentFieldClass(hasError bool) string {
	parseClassName := "rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"
	if hasError {
		return parseClassName + " border-rose-300 bg-rose-50/60 focus:border-rose-400"
	}
	return parseClassName
}

func publicCommentTextareaClass(hasError bool) string {
	parseClassName := "min-h-28 rounded-[1.1rem] border border-white/10 bg-[rgba(8,12,20,0.9)] px-4 py-3 text-white outline-none transition focus:border-amber-300/60 focus:bg-[rgba(10,15,24,1)]"
	if hasError {
		return parseClassName + " border-rose-300 bg-rose-50/60 focus:border-rose-400"
	}
	return parseClassName
}

func publicCommentFieldError(parseId string, parseMessage string) ui.Node {
	if strings.TrimSpace(parseMessage) == "" {
		return nil
	}
	return html.P(html.Props{ID: parseId, Class: "text-sm font-medium text-rose-600"}, html.Text(parseMessage))
}

func formatPublicCommentDate(parseValue string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return "recently"
	}
	if len(parseTrimmed) >= 10 {
		return parseTrimmed[:10]
	}
	return parseTrimmed
}

func publicProductHeroCard(parseProduct productCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 " + publicHeroSurfaceClass() + " lg:p-7"},
		html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,1.3fr)_minmax(18rem,0.8fr)] lg:items-start"},
			html.Div(html.Props{Class: "grid gap-5"},
				publicProductIdentity(parseProduct),
				publicProductStory(parseProduct),
				html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-3"},
					publicSignalPill(productCategoryCue(parseProduct.Category)),
					publicSignalPill(productSupportCue(parseProduct.Status)),
					publicSignalPill(productBuyingMotion(parseProduct.Status)),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/6 p-5"},
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"}, html.Text(publicStartingAtLabel)),
				html.P(html.Props{Class: "text-4xl font-black tracking-[-0.05em] text-white"}, html.Text(formatPrice(parseProduct.PriceCents))),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("A standard product summary keeps price, stock posture, and next steps visible before buyers move into reviews or delivery planning.")),
				html.Div(html.Props{Class: "grid gap-3"},
					html.P(html.Props{Class: "rounded-[1.1rem] border border-white/10 bg-white/6 px-4 py-3 text-sm text-stone-300"}, html.Text(catalogEditorialCopy(parseProduct))),
				),
			),
		),
	)
}

func publicProductIdentity(parseProduct productCard) ui.Node {
	return html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseProduct.Category)),
		html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-300"}, html.Text(parseProduct.SKU)),
		html.Span(html.Props{Class: publicStatusClass(parseProduct.Status)}, html.Text(publicStatusLabel(parseProduct.Status))),
	)
}

func publicProductStory(parseProduct productCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.H2(html.Props{Class: "max-w-4xl text-3xl font-black tracking-[-0.04em] text-white sm:text-4xl lg:text-[3rem]"}, html.Text(parseProduct.Title)),
		html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300"}, html.Text(parseProduct.Summary)),
	)
}

func publicProductContextColumn(parseLabel string, parseCopy string) ui.Node {
	return html.Div(html.Props{Class: "grid gap-3 rounded-[1.25rem] border border-white/10 bg-white/6 p-4 shadow-[0_14px_30px_rgba(0,0,0,0.12)]"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "text-sm leading-6 text-stone-200"}, html.Text(parseCopy)),
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

func publicProductPromiseLanesIsland(parseProduct productCard) ui.Node {
	return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ResetKeys: []any{parseProduct.Slug, parseProduct.Status},
		ErrorFallback: func(parseErr error, reset func()) ui.Node {
			return publicProductPromiseLanesError(parseProduct, parseErr, reset)
		},
		Child: ui.CreateElement(func() ui.Node {
			parseResource := useAtlasResource(func(parseCtx context.Context) (warehouseDirectoryPage, error) {
				return fetchAtlasJSON[warehouseDirectoryPage](parseCtx, "/api/public/warehouses")
			}, parseProduct.Slug)
			parseState := parseResource.Get()
			parseContent := publicProductPromiseLanesCard(parseProduct, parseState.Value.Items, parseState.Loading && parseState.Ready)
			return ui.CreateElement(ui.AsyncBoundary, ui.AsyncBoundaryProps{
				Pending:  !parseState.Ready && parseState.Error == nil,
				Error:    parseState.Error,
				Fallback: publicProductPromiseLanesFallback(parseProduct),
				ErrorFallback: func(parseErr2 error) ui.Node {
					return publicProductPromiseLanesError(parseProduct, parseErr2, parseResource.Reload)
				},
				Content: parseContent,
			})
		}),
	})
}

func publicProductPromiseLanesCard(parseProduct productCard, parseWarehouses []warehouseCard, isRefreshing bool) ui.Node {
	parseNodes := make([]ui.Node, 0, 3)
	for _, parseItem := range parseWarehouses {
		parseNodes = append(parseNodes, html.A(html.Props{Href: RouteWarehouses + "/" + parseItem.Slug + "/availability/" + parseProduct.Slug, Class: "grid gap-2 rounded-[1.3rem] border border-white/10 bg-white/6 px-4 py-4 transition hover:border-amber-300/35 hover:bg-white/10"},
			html.Div(html.Props{Class: "flex items-center justify-between gap-3"},
				html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.Name)),
				html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-stone-300"}, html.Text(parseItem.Region)),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-stone-300"}, html.Text(fallback(parseItem.ServiceLevel, "Regional service posture"))),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-amber-200"}, html.Text("Open "+parseItem.Name+" availability")),
		))
		if len(parseNodes) == 3 {
			break
		}
	}
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text("Regional promise lanes")),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("Check the warehouse route that matches this product.")),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("This below-the-fold module loads after the main product story, so route-critical content stays stable while the regional availability lanes resolve independently.")),
		),
	}
	if isRefreshing {
		parseChildren = append(parseChildren, html.P(html.Props{Class: "rounded-[1.2rem] border border-cyan-300/30 bg-cyan-300/10 px-4 py-3 text-sm text-cyan-100"}, html.Text("Refreshing the regional lane list in the background while the current panel stays visible.")))
	}
	if len(parseNodes) == 0 {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "rounded-[1.3rem] border border-white/10 bg-white/6 px-4 py-4 text-sm leading-7 text-stone-300"}, html.Text("Atlas is still resolving warehouse lanes for this product.")))
	}
	parseChildren = append(parseChildren, parseNodes...)
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"}, parseChildren...)
}

func publicProductPromiseLanesFallback(parseProduct productCard) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text("Regional promise lanes")),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("Check the warehouse route that matches this product.")),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("Atlas defers this secondary lane module until after hydration so loading stays local to the panel instead of blocking the product route.")),
		),
		html.Div(html.Props{Class: "grid gap-3 md:grid-cols-3"},
			html.Div(html.Props{Class: "h-24 rounded-[1.3rem] border border-white/10 bg-white/6"}),
			html.Div(html.Props{Class: "h-24 rounded-[1.3rem] border border-white/10 bg-white/6"}),
			html.Div(html.Props{Class: "h-24 rounded-[1.3rem] border border-white/10 bg-white/6"}),
		),
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-stone-500"}, html.Text("Product: "+parseProduct.Title)),
	)
}

func publicProductPromiseLanesError(parseProduct productCard, parseErr error, parseRetry func()) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-rose-400/25 bg-rose-400/10 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)]"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-rose-200"}, html.Text("Regional promise lanes")),
			html.P(html.Props{Class: "text-xl font-black tracking-[-0.03em] text-white"}, html.Text("Atlas could not load the lane panel.")),
			html.P(html.Props{Class: "text-sm leading-7 text-rose-100"}, html.Text(parseErr.Error())),
		),
		html.Button(html.Props{
			Type:    "button",
			Class:   "inline-flex items-center justify-center rounded-full border border-rose-200/40 bg-rose-200/10 px-4 py-3 text-sm font-semibold text-white transition hover:bg-rose-200/20",
			OnClick: ui.UseEvent(func() { parseRetry() }),
		}, html.Text("Retry lane panel")),
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-rose-100/80"}, html.Text("Product: "+parseProduct.Title)),
	)
}

func publicProductActionRail(parseProduct productCard, parsePayload Payload) ui.Node {
	parseProductSupportTitle, parseProductSupportCopy, parseProductSupportPoints := productSupportPlan(parseProduct.Status)
	return ui.CreateElement(func() ui.Node {
		parseOpen := ui.UseState(false)
		parseSheetID := ui.UseId() + "-public-action-rail"
		parseTitleID := parseSheetID + "-title"
		parseDescriptionID := parseSheetID + "-description"
		parseCloseID := parseSheetID + "-close"
		parseOpenDrawer := ui.UseEvent(func() { parseOpen.Set(true) })
		parseCloseDrawer := func() { parseOpen.Set(false) }
		buildRailChildren := func() []ui.Node {
			return []ui.Node{
				html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(publicBuyerNextStepLabel)),
					html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(parseProductSupportTitle)),
					html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseProductSupportCopy)),
					html.Div(html.Props{Class: "grid gap-3 text-sm text-stone-300"}, publicSupportPoints(parseProductSupportPoints)...),
				),
				productPrimaryActionForm(parseProduct, parsePayload),
				publicRelatedProductsCard(parseProduct),
				productSecondaryActionCard(parseProduct),
			}
		}
		return html.Div(html.Props{Class: "grid gap-5 xl:sticky xl:top-24"},
			html.Button(html.Props{Type: "button", Class: "inline-flex w-fit items-center rounded-full border border-amber-300/45 bg-amber-300/10 px-4 py-3 text-sm font-semibold text-amber-100 xl:hidden", OnClick: parseOpenDrawer}, html.Text("Open buying drawer")),
			html.Div(html.Props{Class: "hidden gap-5 xl:grid"}, buildRailChildren()...),
			atlasDismissibleSheet(parseOpen.Get(), parseSheetID, parseTitleID, parseDescriptionID, "#"+parseCloseID, parseCloseDrawer, html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
					html.Div(html.Props{Class: "grid gap-2"},
						html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(publicBuyerNextStepLabel)),
						html.P(html.Props{ID: parseTitleID, Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(parseProductSupportTitle)),
						html.P(html.Props{ID: parseDescriptionID, Class: "text-sm leading-7 text-stone-300"}, html.Text(parseProductSupportCopy)),
					),
					html.Button(html.Props{ID: parseCloseID, Type: "button", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-stone-200", OnClick: ui.UseEvent(func() { parseCloseDrawer() })}, html.Text("Close")),
				),
				html.Div(html.Props{Class: "grid gap-5"}, buildRailChildren()...),
			)),
		)
	})
}

func publicRelatedProductsCard(parseProduct productCard) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseResource := useAtlasCachedResource(CachedRequestResourceKey("/api/public/products/"+parseProduct.Slug+"/related-products", "items"), func(parseCtx context.Context) ([]relatedProductRecord, error) {
			return fetchPublicRelatedProducts(parseCtx, parseProduct.Slug)
		})
		parseState := parseResource.Get()
		parseItems := []relatedProductRecord{}
		if parseState.Ready {
			parseItems = parseState.Value
		}
		parseDescription := "Open adjacent Atlas systems without leaving the same buying context."
		if parseState.Loading && !parseState.Ready {
			parseDescription = "Loading adjacent systems for this product..."
		} else if parseState.Error != nil && !parseState.Ready {
			parseDescription = "Atlas could not load related systems right now. Reopen the route to retry."
		}
		parseChildren := []ui.Node{
			html.Div(html.Props{Class: "grid gap-2"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Related systems")),
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text("Stay inside the same Atlas family.")),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseDescription)),
			),
		}
		parseChildren = append(parseChildren, publicRelatedProductNodes(parseItems)...)
		return html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"}, parseChildren...)
	})
}

func publicRelatedProductNodes(parseItems []relatedProductRecord) []ui.Node {
	if len(parseItems) == 0 {
		return []ui.Node{
			html.Div(html.Props{Class: "rounded-[1.3rem] border border-white/10 bg-white/6 px-4 py-4 text-sm leading-7 text-stone-300"}, html.Text("Repeat-open visits can reuse the cached related-product list after the first lookup, so Atlas does not need to rebuild this secondary panel every time.")),
		}
	}
	parseNodes := make([]ui.Node, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseNodes = append(parseNodes, html.A(html.Props{Href: RouteCatalog + "/" + parseItem.Slug, Class: "grid gap-2 rounded-[1.3rem] border border-white/10 bg-white/6 px-4 py-4 transition hover:border-amber-300/35 hover:bg-white/10"},
			html.P(html.Props{Class: "text-sm font-semibold text-white"}, html.Text(parseItem.Title)),
			html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-400"}, html.Text(parseItem.Category+" · "+parseItem.SKU)),
			html.P(html.Props{Class: "text-sm leading-6 text-stone-300"}, html.Text(fallback(parseItem.Reason, parseItem.Summary))),
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-amber-200"}, html.Text(fallback(parseItem.WarehouseName, parseItem.WarehouseID))),
		))
	}
	return parseNodes
}

func publicSupportPoints(parsePoints []string) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parsePoints))
	for _, parsePoint := range parsePoints {
		parseNodes = append(parseNodes, html.P(html.Props{Class: "rounded-[1.2rem] border border-white/10 bg-white/6 px-4 py-3"}, html.Text(parsePoint)))
	}
	return parseNodes
}

func renderWarehouseDirectoryContent(parsePage warehouseDirectoryPage) ui.Node {
	parseNodes := make([]ui.Node, 0, len(parsePage.Items))
	for _, parseItem := range parsePage.Items {
		parseNodes = append(parseNodes, publicWarehouseDirectoryCard(parseItem))
	}
	if len(parseNodes) == 0 {
		parseNodes = append(parseNodes, html.Div(html.Props{Class: "rounded-[1.8rem] border border-stone-200/80 bg-white/80 p-6 text-sm leading-7 text-stone-600 shadow-[0_18px_40px_rgba(120,107,82,0.08)]"}, html.Text("No delivery regions are available right now. Retry the page or return to the storefront.")))
	}
	return html.Section(html.Props{Class: "grid gap-8"},
		publicWarehouseDirectoryOverview(parsePage),
		html.Div(html.Props{Class: "flex flex-col gap-4"}, parseNodes...),
	)
}

func publicWarehouseDirectoryOverview(parsePage warehouseDirectoryPage) ui.Node {
	parsePrimaryHref := RouteCatalog
	if len(parsePage.Items) > 0 {
		parsePrimaryHref = RouteWarehouses + "/" + parsePage.Items[0].Slug
	}
	return html.Div(html.Props{Class: "relative overflow-hidden grid gap-5 rounded-[2.35rem] border border-stone-200/80 bg-[linear-gradient(145deg,rgba(255,255,255,0.96),rgba(246,238,227,0.94)_55%,rgba(233,222,205,0.9))] p-7 shadow-[0_26px_60px_rgba(120,107,82,0.12)] lg:grid-cols-[minmax(0,1.15fr)_minmax(20rem,0.85fr)] lg:items-end"},
		html.Div(html.Props{Class: "pointer-events-none absolute -right-12 top-0 h-44 w-44 rounded-full bg-amber-200/35 blur-3xl"}),
		html.Div(html.Props{Class: "pointer-events-none absolute bottom-0 left-10 h-32 w-32 rounded-full bg-stone-200/40 blur-3xl"}),
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.3em] text-stone-500"}, html.Text("Atlas delivery regions")),
			html.H2(html.Props{Class: "text-3xl font-black tracking-[-0.03em] text-stone-950"}, html.Text("Choose the warehouse route that matches your delivery window.")),
			html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-600"}, html.Text("Compare regional service posture, stocked volume, and warehouse-specific focus before you open the route that best fits the project timeline in front of you.")),
			html.Div(html.Props{Class: "flex flex-wrap items-center gap-3 pt-2"},
				html.A(html.Props{Href: parsePrimaryHref, Class: "rounded-full bg-stone-950 px-5 py-3 text-sm font-semibold text-stone-50 transition hover:bg-stone-800"}, html.Text("Open featured region")),
				html.A(html.Props{Href: RouteCatalog, Class: "rounded-full border border-stone-300 bg-white/75 px-5 py-3 text-sm font-semibold text-stone-900 transition hover:border-stone-500 hover:bg-white"}, html.Text("Browse all systems")),
			),
		),
		html.Div(html.Props{Class: "relative z-[1] grid gap-3"},
			html.Div(html.Props{Class: "grid gap-3 rounded-[1.8rem] border border-white/75 bg-white/68 p-5 backdrop-blur-sm"},
				html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.26em] text-amber-700"}, html.Text("Regional commerce board")),
				html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
					publicMetricCard(fmt.Sprintf("%d delivery regions", len(parsePage.Items)), "Each route ties promise language to a real Atlas facility rather than a generic shipping estimate."),
					publicMetricCard(primaryWarehouseServiceLevel(parsePage.Items), "Use service level as the first cue, then open the warehouse detail for stocked highlights and product-specific availability."),
					publicMetricCard(fmt.Sprintf("%d stocked units", publicWarehouseUnitsTotal(parsePage.Items)), "Available volume stays visible so the directory feels like a real merchandised network, not only a list of names."),
					publicMetricCard(fmt.Sprintf("%d inbound units", publicWarehouseInboundTotal(parsePage.Items)), "Inbound posture makes the next-best regional choice visible before the buyer drills into product-level availability."),
				),
			),
		),
	)
}

func publicWarehouseDirectoryCard(parseItem warehouseCard) ui.Node {
	return html.A(html.Props{Href: RouteWarehouses + "/" + parseItem.Slug, Class: "group flex flex-col gap-5 rounded-[2rem] border border-stone-200/80 bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(248,244,238,0.9))] p-6 shadow-[0_20px_48px_rgba(120,107,82,0.09)] transition hover:border-stone-300 hover:bg-white hover:shadow-[0_28px_65px_rgba(120,107,82,0.14)] lg:flex-row lg:items-start lg:justify-between"},
		html.Div(html.Props{Class: "flex flex-1 flex-col gap-4"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
				html.Div(html.Props{Class: "flex flex-col gap-2"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseItem.Region)),
					html.P(html.Props{Class: "text-[0.68rem] font-medium uppercase tracking-[0.28em] text-stone-500"}, html.Text(parseItem.ServiceLevel)),
				),
				html.Span(html.Props{Class: "rounded-full border border-stone-200 bg-white/85 px-4 py-2 text-sm font-semibold text-stone-800"}, html.Text(parseItem.Name)),
			),
			html.Div(html.Props{Class: "flex flex-col gap-3"},
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-stone-950 transition group-hover:text-stone-800"}, html.Text(parseItem.Name)),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text(parseItem.PublicSummary)),
			),
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-3"},
				publicWarehouseDirectoryMetric("Available", fmt.Sprintf("%d units", parseItem.Available)),
				publicWarehouseDirectoryMetric("Inbound", fmt.Sprintf("%d units", parseItem.Inbound)),
				publicWarehouseDirectoryMetric("Pressure", fallback(parseItem.Pressure, "Balanced posture")),
			),
			html.Div(html.Props{Class: "flex items-center justify-between gap-4 rounded-[1.45rem] border border-stone-200/75 bg-white/75 p-4 text-sm text-stone-600"},
				html.Div(html.Props{Class: "grid gap-1"},
					html.P(html.Props{Class: "leading-6"}, html.Text("Open the warehouse route for stocked highlights, regional service details, and product-by-product availability.")),
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-stone-500"}, html.Text(fallback(parseItem.Focus, "Regional project fit")+" | "+fallback(parseItem.Backlog, "Backlog controlled"))),
				),
				html.Span(html.Props{Class: "font-semibold text-stone-900 transition group-hover:text-stone-700"}, html.Text("Open warehouse route")),
			),
		),
		html.Div(html.Props{Class: "flex flex-col gap-3 lg:min-w-[22rem] lg:max-w-[24rem]"},
			html.Div(html.Props{Class: "flex flex-wrap gap-3 lg:flex-col"},
				publicWarehouseDirectoryMetric("Region", fallback(parseItem.Region, "Regional lane")),
				publicWarehouseDirectoryMetric("Service level", fallback(parseItem.ServiceLevel, "Standard coverage")),
				publicWarehouseDirectoryMetric("Best for", warehouseRegionCue(parseItem.Region)),
			),
		),
	)
}

func publicWarehouseDirectoryMetric(parseLabel string, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "flex min-w-[10rem] flex-1 items-center justify-between gap-4 rounded-[1.2rem] border border-stone-200/75 bg-white/75 px-4 py-3 lg:min-w-0"},
		html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "text-right text-sm font-semibold leading-6 text-stone-900"}, html.Text(parseValue)),
	)
}

func primaryWarehouseServiceLevel(parseItems []warehouseCard) string {
	if len(parseItems) == 0 {
		return "Regional service posture"
	}
	return fallback(parseItems[0].ServiceLevel, "Regional service posture")
}

func publicWarehouseUnitsTotal(parseItems []warehouseCard) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.Available
	}
	return parseTotal
}

func publicWarehouseInboundTotal(parseItems []warehouseCard) int {
	parseTotal := 0
	for _, parseItem := range parseItems {
		parseTotal += parseItem.Inbound
	}
	return parseTotal
}

func renderWarehouseDetailContent(parsePage warehouseDetailPage, parsePayload Payload) ui.Node {
	parseWarehouse := parsePage.Warehouse
	return html.Section(html.Props{Class: "grid gap-8 lg:grid-cols-[minmax(0,1.12fr)_minmax(19rem,0.82fr)] lg:items-start"},
		html.Div(html.Props{Class: "grid gap-8"},
			publicWarehouseDetailHero(parseWarehouse),
			publicWarehouseDetailFeatureStrip(),
			publicWarehouseStoryBand(parsePage),
			publicWarehouseRegionalProductShowcase(parseWarehouse, parsePage.Products),
		),
		publicWarehouseSideDataCard(parsePage, parsePayload),
	)
}

func publicWarehouseDetailHero(parseWarehouse warehouseCard) ui.Node {
	return html.Div(html.Props{Class: "relative overflow-hidden rounded-[2.3rem] border border-stone-200/80 bg-[linear-gradient(145deg,rgba(255,255,255,0.96),rgba(244,236,224,0.94)_55%,rgba(231,220,202,0.92))] p-7 shadow-[0_28px_65px_rgba(120,107,82,0.14)]"},
		html.Div(html.Props{Class: "pointer-events-none absolute -right-10 top-6 h-36 w-36 rounded-full bg-amber-200/35 blur-3xl"}),
		html.Div(html.Props{Class: "relative z-[1] grid gap-7 lg:grid-cols-[minmax(0,1.05fr)_minmax(15rem,0.72fr)]"},
			html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseWarehouse.Region)),
					html.Span(html.Props{Class: "rounded-full border border-white/80 bg-white/70 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(publicRegionalHubLabel)),
					html.Span(html.Props{Class: "rounded-full border border-stone-200 bg-white/85 px-4 py-2 text-sm font-semibold text-stone-800"}, html.Text(parseWarehouse.ServiceLevel)),
				),
				html.H2(html.Props{Class: "text-4xl font-black tracking-[-0.04em] text-stone-950"}, html.Text(parseWarehouse.Name)),
				html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-600"}, html.Text(parseWarehouse.PublicSummary)),
				html.Div(html.Props{Class: "grid gap-4 rounded-[1.75rem] border border-white/75 bg-white/55 p-5 backdrop-blur-sm sm:grid-cols-2"},
					publicProductContextColumn(publicRegionalReadLabel, warehouseRegionCue(parseWarehouse.Region)),
					publicProductContextColumn(publicServicePostureLabel, warehouseServiceTone(parseWarehouse.ServiceLevel)),
				),
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.A(html.Props{Href: RouteCatalog + "?warehouse=" + url.QueryEscape(parseWarehouse.ID), Class: "rounded-full bg-stone-950 px-5 py-3 text-sm font-semibold text-stone-50 transition hover:bg-stone-800"}, html.Text(publicBrowseRegionalProductsLabel)),
					html.A(html.Props{Href: RouteCatalog, Class: "rounded-full border border-stone-300 bg-white/75 px-5 py-3 text-sm font-semibold text-stone-900 transition hover:border-stone-500 hover:bg-white"}, html.Text(publicCompareAllSystemsLabel)),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 rounded-[1.9rem] border border-stone-200/80 bg-white/78 p-5 shadow-[0_18px_40px_rgba(120,107,82,0.08)]"},
				publicMetricCard(publicRegionalFocusLabel, parseWarehouse.Region),
				publicMetricCard(publicServiceLevelLabel, parseWarehouse.ServiceLevel),
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

func publicWarehouseStoryBand(parsePage warehouseDetailPage) ui.Node {
	parseWarehouse := parsePage.Warehouse
	return html.Div(html.Props{Class: "grid gap-4 lg:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]"},
		html.Div(html.Props{Class: "grid gap-4 rounded-[1.95rem] border border-stone-200/80 bg-white/82 p-6 shadow-[0_18px_45px_rgba(120,107,82,0.08)]"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Regional capability summary")),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-stone-950"}, html.Text(fallback(parseWarehouse.Focus, "Regional warehouse posture"))),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text("This warehouse route should feel like a merchandised regional story: clear service posture, visible stocked volume, and one obvious next step into product-specific availability.")),
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-3"},
				publicWarehouseDirectoryMetric("Staffing", fallback(parseWarehouse.Staffing, "Core team assigned")),
				publicWarehouseDirectoryMetric("Backlog", fallback(parseWarehouse.Backlog, "Backlog under control")),
				publicWarehouseDirectoryMetric("Risk lanes", fmt.Sprintf("%d flagged", parseWarehouse.RiskCount)),
			),
		),
		html.Div(html.Props{Class: "grid gap-4"},
			html.Div(html.Props{Class: "grid gap-3 rounded-[1.95rem] border border-stone-200/80 bg-[linear-gradient(180deg,rgba(255,255,255,0.96),rgba(246,238,227,0.92))] p-6 shadow-[0_18px_45px_rgba(120,107,82,0.08)]"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-stone-500"}, html.Text("Product volume story")),
				html.P(html.Props{Class: "text-3xl font-black tracking-[-0.03em] text-stone-950"}, html.Text(fmt.Sprintf("%d stocked highlights", len(parsePage.Products)))),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text("Use this route when the buyer wants a region-first answer before choosing the exact product availability lane.")),
			),
			html.Div(html.Props{Class: "flex flex-wrap gap-3 rounded-[1.8rem] border border-stone-200/80 bg-white/75 p-5 shadow-[0_18px_40px_rgba(120,107,82,0.08)]"},
				html.A(html.Props{Href: RouteCatalog + "?warehouse=" + url.QueryEscape(parseWarehouse.ID), Class: "rounded-full bg-stone-950 px-5 py-3 text-sm font-semibold text-stone-50 transition hover:bg-stone-800"}, html.Text(publicBrowseRegionalProductsLabel)),
				html.A(html.Props{Href: RouteCatalog, Class: "rounded-full border border-stone-300 bg-white/85 px-5 py-3 text-sm font-semibold text-stone-900 transition hover:border-stone-500 hover:bg-white"}, html.Text(publicCompareAllSystemsLabel)),
			),
		),
	)
}

func publicWarehouseSideDataCard(parsePage warehouseDetailPage, parsePayload Payload) ui.Node {
	return ui.CreateElement(func() ui.Node {
		parseResource := useAtlasStartupPageResource(parsePayload)
		parseState := parseResource.Get()
		parseCurrent := parsePage
		if parseState.Ready {
			parseDecoded := decode[warehouseDetailPage](parseState.Value)
			if strings.TrimSpace(parseDecoded.Warehouse.ID) != "" {
				parseCurrent = parseDecoded
			}
		}
		parseWarehouse := parseCurrent.Warehouse
		parseStatusCopy := "Warehouse posture stays cached for repeat-open reviews of staffing, backlog, and regional focus."
		if parseState.Loading && parseState.Ready {
			parseStatusCopy = "Refreshing the latest warehouse posture..."
		} else if parseState.Error != nil && !parseState.Ready {
			parseStatusCopy = "Showing the SSR warehouse snapshot until Atlas can reload the side data."
		}
		return html.Div(html.Props{Class: "grid gap-4 lg:sticky lg:top-24"},
			html.Div(html.Props{Class: "grid gap-3 rounded-[1.9rem] border border-stone-200/80 bg-white/80 p-6 shadow-[0_18px_40px_rgba(120,107,82,0.08)]"},
				html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Warehouse side data")),
				html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-stone-950"}, html.Text(fallback(parseWarehouse.Name, "Regional hub posture"))),
				html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text(parseStatusCopy)),
			),
			publicMetricCard("Pressure", fallback(parseWarehouse.Pressure, "Balanced regional pressure")),
			publicMetricCard("Staffing", fallback(parseWarehouse.Staffing, "Core team assigned")),
			publicMetricCard("Backlog", fallback(parseWarehouse.Backlog, "Backlog is under control")),
			publicMetricCard("Regional focus", fallback(parseWarehouse.Focus, "Regional delivery planning")),
			publicMetricCard(fmt.Sprintf("%d stocked highlights", len(parseCurrent.Products)), "Repeat-open visits can reuse this side snapshot without waiting for the whole warehouse route to rebuild."),
		)
	})
}

func publicWarehouseRegionalProducts(parseWarehouse warehouseCard, parseProducts []productCard) []ui.Node {
	parseNodes := make([]ui.Node, 0, len(parseProducts))
	for _, parseItem := range parseProducts {
		parseNodes = append(parseNodes, html.A(html.Props{Href: RouteWarehouses + "/" + parseWarehouse.Slug + "/availability/" + parseItem.Slug, Class: "grid gap-3 rounded-[1.6rem] border border-stone-200/75 bg-white/75 p-5 transition hover:border-stone-300 hover:bg-white"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-2"},
					html.P(html.Props{Class: "text-lg font-semibold text-stone-950"}, html.Text(parseItem.Title)),
					html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(parseItem.SKU+" · "+parseItem.Category)),
				),
				html.Span(html.Props{Class: publicStatusClass(parseItem.Status)}, html.Text(publicStatusLabel(parseItem.Status))),
			),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text(parseItem.Summary)),
			html.Div(html.Props{Class: "flex items-end justify-between gap-3"},
				html.Div(html.Props{Class: "grid gap-1"},
					html.P(html.Props{Class: "text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-500"}, html.Text(publicStartingAtLabel)),
					html.P(html.Props{Class: "text-xl font-black tracking-[-0.03em] text-stone-950"}, html.Text(formatPrice(parseItem.PriceCents))),
				),
				html.Span(html.Props{Class: "text-sm font-semibold text-stone-900"}, html.Text(publicOpenRegionalAvailabilityLabel)),
			),
		))
	}
	return parseNodes
}

func publicWarehouseRegionalProductShowcase(parseWarehouse warehouseCard, parseProducts []productCard) ui.Node {
	parseNodes := publicWarehouseRegionalProducts(parseWarehouse, parseProducts)
	if len(parseNodes) == 0 {
		parseNodes = []ui.Node{
			html.Div(html.Props{Class: "rounded-[1.7rem] border border-stone-200/80 bg-white/80 p-5 text-sm leading-7 text-stone-600"}, html.Text("No stocked highlights are available for this warehouse right now. Return to the catalog or compare another regional route.")),
		}
	}
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(publicRegionalAvailabilityPicksLabel)),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-stone-950"}, html.Text("Open stocked systems that fit this regional route.")),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-600"}, html.Text("These product cards keep the warehouse route feeling merchandised instead of reading like a detached logistics utility page.")),
		),
	}
	parseChildren = append(parseChildren, parseNodes...)
	return html.Div(html.Props{Class: "grid gap-4 rounded-[2rem] border border-stone-200/80 bg-white/78 p-6 shadow-[0_18px_45px_rgba(120,107,82,0.08)]"}, parseChildren...)
}

func renderAvailabilityContent(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	return html.Section(html.Props{Class: "grid gap-8 xl:grid-cols-[minmax(0,1.5fr)_minmax(21rem,0.78fr)] xl:items-start"},
		html.Div(html.Props{Class: "grid gap-6"},
			publicAvailabilityHero(parseAvailability),
			html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]"},
				publicAvailabilityMetrics(parseAvailability),
				publicAvailabilityFeatureStrip(),
			),
			publicAvailabilityPromiseBand(parseAvailability),
		),
		publicAvailabilityActionRail(parseAvailability, parsePayload),
	)
}

func publicAvailabilityHero(parseAvailability availabilityPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-5 rounded-[1.9rem] border border-white/10 bg-[linear-gradient(145deg,rgba(13,18,30,0.96),rgba(18,25,40,0.9))] p-6 shadow-[0_28px_65px_rgba(0,0,0,0.22)] lg:p-7"},
		html.Div(html.Props{Class: "grid gap-5 lg:grid-cols-[minmax(0,1.3fr)_minmax(18rem,0.8fr)] lg:items-start"},
			html.Div(html.Props{Class: "grid gap-5"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(parseAvailability.Warehouse.Name)),
					html.Span(html.Props{Class: "rounded-full border border-white/10 bg-white/8 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-stone-300"}, html.Text(parseAvailability.Warehouse.Region)),
					html.Span(html.Props{Class: publicStatusClass(parseAvailability.Status)}, html.Text(publicStatusLabel(parseAvailability.Status))),
				),
				html.H2(html.Props{Class: "max-w-4xl text-3xl font-black tracking-[-0.04em] text-white sm:text-4xl lg:text-[3rem]"}, html.Text(parseAvailability.Product.Title)),
				html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300"}, html.Text(fmt.Sprintf("%d available now with %d inbound at %s.", parseAvailability.Available, parseAvailability.Inbound, parseAvailability.Warehouse.Name))),
				html.Div(html.Props{Class: "grid gap-4 rounded-[1.75rem] border border-white/10 bg-white/6 p-5 backdrop-blur-sm sm:grid-cols-2"},
					publicProductContextColumn(publicAvailabilityStoryLabel, availabilityStoryCopy(parseAvailability.Available, parseAvailability.Inbound)),
					publicProductContextColumn(publicWhyThisMattersLabel, "Product demand stays tied to a named hub, so promise language feels concrete instead of generic."),
				),
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3"},
					html.A(html.Props{Href: RouteCatalog + "/" + parseAvailability.Product.Slug, Class: "rounded-full bg-amber-300 px-5 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text("Back to product detail")),
					html.A(html.Props{Href: RouteWarehouses + "/" + parseAvailability.Warehouse.Slug, Class: "rounded-full border border-white/12 bg-white/6 px-5 py-3 text-sm font-semibold text-stone-200 transition hover:border-amber-300/35 hover:bg-white/10 hover:text-white"}, html.Text("See warehouse route")),
				),
			),
			html.Div(html.Props{Class: "grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/6 p-5"},
				publicMetricCard(fmt.Sprintf("%d available", parseAvailability.Available), "Available now in this region."),
				publicMetricCard(fmt.Sprintf("%d inbound", parseAvailability.Inbound), "More units already scheduled for upcoming orders."),
				publicMetricCard(publicStatusLabel(parseAvailability.Status), "Status stays explicit so buyers know whether to quote now or plan ahead."),
			),
		),
	)
}

func publicAvailabilityFeatureStrip() ui.Node {
	return html.Div(html.Props{Class: "grid gap-4"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text("Quick buying notes")),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			publicFeatureCard("Concrete promise", "Availability is framed around a real delivery region, which makes timing confidence easier to evaluate."),
			publicFeatureCard("Keep your place", "Availability capture stays next to the delivery story so buyer intent does not vanish when stock tightens."),
			publicFeatureCard("Clear next steps", "The route stays calm and practical without forcing buyers to learn Atlas operations language."),
		),
	)
}

func publicAvailabilityMetrics(parseAvailability availabilityPage) ui.Node {
	_ = parseAvailability
	return html.Div(html.Props{Class: "grid gap-4"},
		html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-stone-400"}, html.Text("Why this availability page is easier to use")),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			publicMetricCard("Regional promise", "The route keeps one warehouse, one SKU, and one timing read in the same first screenful."),
			publicMetricCard("Action rail", "Quote, reserve, and support actions stay in a dedicated side rail instead of being split into detached utility blocks."),
			publicMetricCard("Consistent scaffold", "The same hero, metrics, and side-rail reading order as product detail reduces route-switching friction."),
		),
	)
}

func publicAvailabilityActionRail(parseAvailability availabilityPage, parsePayload Payload) ui.Node {
	parseAvailabilityTitle, parseAvailabilityCopy := availabilitySupportPlan(parseAvailability.Available, parseAvailability.Inbound)
	return html.Div(html.Props{Class: "grid gap-5 xl:sticky xl:top-24"},
		html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text(publicRegionalNextStepLabel)),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(parseAvailabilityTitle)),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(parseAvailabilityCopy)),
			html.Div(html.Props{Class: "grid gap-3 text-sm text-stone-300"}, publicSupportPoints(publicAvailabilitySupportPoints(parseAvailability))...),
		),
		availabilityPrimaryActionForm(parseAvailability, parsePayload),
		availabilityQuestionActionForm(parseAvailability, parsePayload),
		html.Div(html.Props{Class: "grid gap-3 rounded-[1.7rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
			html.P(html.Props{Class: "text-lg font-semibold text-white"}, html.Text("Keep warehouse context visible")),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text("Use the product route for broader comparison or move into the warehouse route if the buyer needs more regional confidence before requesting follow-up.")),
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				html.A(html.Props{Href: RouteCatalog + "/" + parseAvailability.Product.Slug, Class: "rounded-full border border-white/12 bg-white/6 px-4 py-3 text-sm font-semibold text-stone-200 transition hover:border-amber-300/35 hover:bg-white/10 hover:text-white"}, html.Text("Product route")),
				html.A(html.Props{Href: RouteWarehouses + "/" + parseAvailability.Warehouse.Slug, Class: "rounded-full border border-white/12 bg-white/6 px-4 py-3 text-sm font-semibold text-stone-200 transition hover:border-amber-300/35 hover:bg-white/10 hover:text-white"}, html.Text("Warehouse route")),
			),
		),
	)
}

func publicAvailabilityPromiseBand(parseAvailability availabilityPage) ui.Node {
	return html.Div(html.Props{Class: "grid gap-4 rounded-[1.8rem] border border-white/10 bg-white/6 p-6 shadow-[0_18px_45px_rgba(0,0,0,0.18)] backdrop-blur-sm"},
		html.Div(html.Props{Class: "grid gap-2"},
			html.P(html.Props{Class: "text-[0.72rem] font-semibold uppercase tracking-[0.3em] text-amber-700"}, html.Text("Warehouse-specific promise")),
			html.P(html.Props{Class: "text-2xl font-black tracking-[-0.03em] text-white"}, html.Text(fallback(parseAvailability.Warehouse.Focus, "Regional delivery posture"))),
			html.P(html.Props{Class: "text-sm leading-7 text-stone-300"}, html.Text(fallback(parseAvailability.Warehouse.PublicSummary, "This route should explain what this warehouse can realistically support for this product before the buyer submits follow-up."))),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			publicMetricCard(publicRegionalReadLabel, warehouseRegionCue(parseAvailability.Warehouse.Region)),
			publicMetricCard(publicServicePostureLabel, warehouseServiceTone(parseAvailability.Warehouse.ServiceLevel)),
			publicMetricCard(publicPromiseLensLabel, availabilityStoryCopy(parseAvailability.Available, parseAvailability.Inbound)),
		),
	)
}

func publicAvailabilitySupportPoints(parseAvailability availabilityPage) []string {
	parsePoints := []string{
		warehouseRegionCue(parseAvailability.Warehouse.Region),
		warehouseServiceTone(parseAvailability.Warehouse.ServiceLevel),
	}
	if strings.TrimSpace(parseAvailability.Warehouse.Backlog) != "" {
		parsePoints = append(parsePoints, "Warehouse posture: "+parseAvailability.Warehouse.Backlog+".")
	}
	if parseAvailability.Available > 0 {
		parsePoints = append(parsePoints, "Quote now if the project can move on this region's current stock posture.")
	} else if parseAvailability.Inbound > 0 {
		parsePoints = append(parsePoints, "Use reserve capture to hold buyer intent against the inbound recovery window for this specific hub.")
	} else {
		parsePoints = append(parsePoints, "Use the support path to discuss substitutions or a different warehouse before promising timing.")
	}
	return parsePoints
}

func renderPublicHero(parsePayload Payload) ui.Node {
	parseConfig := publicHeroConfig(parsePayload.Route.Path)
	if isPublicDetailRoute(parsePayload.Route.Path) {
		return html.Section(html.Props{Class: "relative overflow-hidden rounded-[1.9rem] border border-white/10 bg-[linear-gradient(140deg,rgba(12,17,28,0.98),rgba(15,21,34,0.95)_58%,rgba(21,28,44,0.92)_100%)] px-6 py-6 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:px-8 lg:px-10"},
			html.Div(html.Props{Class: "pointer-events-none absolute inset-y-0 right-0 hidden w-[28%] bg-[radial-gradient(circle_at_center,_rgba(245,158,11,0.12),_transparent_70%)] lg:block"}),
			html.Div(html.Props{Class: "relative z-[1] grid gap-5"},
				html.Div(html.Props{Class: "flex flex-wrap items-center gap-3 text-[0.72rem] font-semibold uppercase tracking-[0.34em] text-amber-300"},
					html.Span(html.Props{Class: "rounded-full border border-amber-300/25 bg-amber-300/10 px-3 py-1"}, html.Text(parseConfig.kicker)),
					html.Span(html.Props{}, html.Text(publicRequestTimeSSRLabel)),
				),
				html.Div(html.Props{Class: "grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end lg:gap-6"},
					html.Div(html.Props{Class: "grid gap-3"},
						html.H1(html.Props{Class: "max-w-4xl text-4xl font-black tracking-[-0.04em] text-white sm:text-[3rem] lg:text-[3.35rem]"}, html.Text(fallback(parsePayload.Route.Title, "Atlas"))),
						html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300"}, html.Text(routeSummary(parsePayload.Route.Path))),
					),
					html.Div(html.Props{Class: "flex flex-col gap-3 sm:flex-row sm:flex-wrap lg:justify-end"},
						html.A(html.Props{Href: parseConfig.primaryHref, Class: "inline-flex items-center justify-center rounded-full bg-amber-300 px-6 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(parseConfig.primaryLabel)),
						html.A(html.Props{Href: parseConfig.secondaryHref, Class: "inline-flex items-center justify-center rounded-full border border-white/12 bg-white/6 px-6 py-3 text-sm font-semibold text-stone-200 transition hover:border-amber-300/35 hover:bg-white/10 hover:text-white"}, html.Text(parseConfig.secondaryLabel)),
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
					html.Span(html.Props{Class: "rounded-full border border-amber-300/25 bg-amber-300/10 px-3 py-1"}, html.Text(parseConfig.kicker)),
					html.Span(html.Props{}, html.Text(publicRequestTimeSSRLabel)),
				),
				html.H1(html.Props{Class: "max-w-4xl text-4xl font-black tracking-[-0.04em] text-white sm:text-5xl lg:text-6xl"}, html.Text(fallback(parsePayload.Route.Title, "Atlas"))),
				html.P(html.Props{Class: "max-w-3xl text-base leading-8 text-stone-300 sm:text-lg"}, html.Text(routeSummary(parsePayload.Route.Path))),
				html.Div(html.Props{Class: "flex flex-col gap-3 sm:flex-row sm:flex-wrap"},
					html.A(html.Props{Href: parseConfig.primaryHref, Class: "inline-flex items-center justify-center rounded-full bg-amber-300 px-6 py-3 text-sm font-semibold text-stone-950 transition hover:bg-amber-200"}, html.Text(parseConfig.primaryLabel)),
					html.A(html.Props{Href: parseConfig.secondaryHref, Class: "inline-flex items-center justify-center rounded-full border border-white/12 bg-white/6 px-6 py-3 text-sm font-semibold text-stone-200 transition hover:border-amber-300/35 hover:bg-white/10 hover:text-white"}, html.Text(parseConfig.secondaryLabel)),
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

func isPublicDetailRoute(parsePath string) bool {
	if strings.HasPrefix(parsePath, RouteCatalog+"/") {
		return true
	}
	if strings.Contains(parsePath, "/availability/") {
		return true
	}
	return strings.HasPrefix(parsePath, RouteWarehouses+"/") && parsePath != RouteWarehouses
}

type publicHeroState struct {
	kicker         string
	primaryLabel   string
	primaryHref    string
	secondaryLabel string
	secondaryHref  string
}

func publicHeroConfig(parsePath string) publicHeroState {
	parseState := publicHeroState{
		kicker:         "Warehouse-backed design systems",
		primaryLabel:   "Shop workspace systems",
		primaryHref:    RouteCatalog,
		secondaryLabel: "Explore warehouse network",
		secondaryHref:  RouteWarehouses,
	}
	switch {
	case parsePath == RouteCatalog:
		parseState.kicker = "Curated modular workspace catalog"
	case strings.HasPrefix(parsePath, RouteCatalog+"/"):
		parseState.kicker = "Product detail with delivery context"
		parseState.primaryLabel = "Browse more products"
		parseState.secondaryLabel = "See delivery by region"
	case parsePath == RouteWarehouses:
		parseState.kicker = "Regional delivery options"
		parseState.primaryLabel = "See stocked products"
	case strings.HasPrefix(parsePath, RouteWarehouses+"/"):
		parseState.kicker = "Regional delivery and availability"
		parseState.primaryLabel = "Browse the catalog"
	}
	return parseState
}
