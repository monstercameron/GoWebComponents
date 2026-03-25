//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Attrs = router.Attrs

const (
	themeAtom  = "omi-theme"
	searchAtom = "omi-search"
	noticeAtom = "omi-notice"
)

type Notice struct {
	Title string
	Body  string
	Level string
}

type ShellProps struct {
	ActivePath string
	Title      string
	Page       ui.Node
}

type NavLinkProps struct {
	Label  string
	Path   string
	Active bool
}

type HookRecord struct {
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Summary  string   `json:"summary"`
	Hooks    []string `json:"hooks"`
}

func filterHookRecords(parseRecords []HookRecord, parseQuery string) []HookRecord {
	parseTrimmedQuery := strings.TrimSpace(strings.ToLower(parseQuery))
	if parseTrimmedQuery == "" {
		return append([]HookRecord(nil), parseRecords...)
	}

	parseMatches := make([]HookRecord, 0, len(parseRecords))
	for _, parseRecord := range parseRecords {
		isParseHit := strings.Contains(strings.ToLower(parseRecord.Name), parseTrimmedQuery) ||
			strings.Contains(strings.ToLower(parseRecord.Category), parseTrimmedQuery) ||
			strings.Contains(strings.ToLower(parseRecord.Summary), parseTrimmedQuery)
		if !isParseHit {
			for _, parseHook := range parseRecord.Hooks {
				if strings.Contains(strings.ToLower(parseHook), parseTrimmedQuery) {
					isParseHit = true
					break
				}
			}
		}

		if isParseHit {
			parseMatches = append(parseMatches, parseRecord)
		}
	}

	return parseMatches
}

func recordSlug(parseName string) string {
	parseSlug := strings.ToLower(strings.TrimSpace(parseName))
	parseReplacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ":", "", ",", "", ".", "")
	parseSlug = parseReplacer.Replace(parseSlug)
	for strings.Contains(parseSlug, "--") {
		parseSlug = strings.ReplaceAll(parseSlug, "--", "-")
	}
	return strings.Trim(parseSlug, "-")
}

func normalizePlaygroundMode(parseMode string) string {
	switch strings.ToLower(strings.TrimSpace(parseMode)) {
	case "agent", "review", "benchmark":
		return strings.ToLower(strings.TrimSpace(parseMode))
	default:
		return "agent"
	}
}

func loadHookRecords(parseCtx context.Context) ([]HookRecord, error) {
	parseResultCh := fetch.Fetch("./fixtures.json", fetch.Options{})
	select {
	case <-parseCtx.Done():
		return nil, parseCtx.Err()
	case parseResult := <-parseResultCh:
		if parseResult.Err != nil {
			return nil, parseResult.Err
		}

		parsePayload, parseOk := parseResult.Data.(string)
		if !parseOk || parsePayload == "" {
			return nil, fmt.Errorf("fixtures payload unavailable")
		}

		var parseParsed []HookRecord
		if parseErr := json.Unmarshal([]byte(parsePayload), &parseParsed); parseErr != nil {
			return nil, parseErr
		}
		return parseParsed, nil
	}
}

func Shell(parseProps ShellProps) ui.Node {
	parseTheme := state.UseAtom(themeAtom, "aurora")
	parseSearch := state.UseAtom(searchAtom, "")
	parseNotice := state.UseAtom(noticeAtom, Notice{})

	parseToggleTheme := ui.UseEvent(func() {
		parseTheme.Update(func(parseCurrent string) string {
			if parseCurrent == "aurora" {
				return "signal"
			}
			return "aurora"
		})
	})

	clearNotice := ui.UseEvent(func() {
		parseNotice.Set(Notice{})
	})

	ui.UseEffect(func() func() {
		parseDocument := js.Global().Get("document")
		if parseDocument.Truthy() {
			parseDocument.Set("title", "OMI Example - "+parseProps.Title)
		}

		parseBody := parseDocument.Get("body")
		if parseBody.Truthy() {
			parseBody.Call("setAttribute", "data-theme", parseTheme.Get())
		}

		parseStorage := js.Global().Get("localStorage")
		if parseStorage.Truthy() {
			parseStorage.Call("setItem", "omi-theme", parseTheme.Get())
		}
		return nil
	}, parseProps.Title, parseTheme.Get())

	parseThemePanelClass := "min-h-screen text-slate-100 selection:bg-cyan-400/20 "
	if parseTheme.Get() == "signal" {
		parseThemePanelClass += "bg-[#0d1319]"
	} else {
		parseThemePanelClass += "bg-[#071018]"
	}

	return html.Div(html.Props{
		Class: parseThemePanelClass,
	},
		html.Header(html.Props{
			Class: "border-b border-slate-800/80 bg-[#0d1722]/88 backdrop-blur-xl sticky top-0 z-50",
		},
			html.Div(html.Props{
				Class: "mx-auto flex max-w-6xl items-center justify-between gap-6 px-6 py-4",
			},
				html.Div(html.Props{},
					html.P(html.Props{
						Class: "text-xs uppercase tracking-[0.35em] text-cyan-300",
					}, html.Text("OMI")),
					html.H1(html.Props{
						Class: "text-2xl font-black tracking-tight",
					}, html.Text("Omni Hooks Showcase")),
				),
				html.Nav(html.Props{
					Class: "flex flex-wrap items-center gap-3",
				},
					ui.CreateElement(NavLink, NavLinkProps{Label: "Overview", Path: "/", Active: parseProps.ActivePath == "/"}),
					ui.CreateElement(NavLink, NavLinkProps{Label: "Data Lab", Path: "/data", Active: parseProps.ActivePath == "/data"}),
					ui.CreateElement(NavLink, NavLinkProps{Label: "Search", Path: "/search", Active: parseProps.ActivePath == "/search"}),
					ui.CreateElement(NavLink, NavLinkProps{Label: "Secure", Path: "/secure", Active: parseProps.ActivePath == "/secure"}),
					ui.CreateElement(NavLink, NavLinkProps{Label: "Playground", Path: "/playground", Active: parseProps.ActivePath == "/playground"}),
				),
				html.Div(html.Props{
					Class: "flex items-center gap-3",
				},
					html.Small(html.Props{
						Class: "hidden text-right text-xs uppercase tracking-[0.25em] text-slate-400 md:block",
					}, html.Text("Shared search: "+parseSearch.Get())),
					html.Button(html.Props{
						OnClick: parseToggleTheme,
						Class:   "rounded-full border border-slate-700/80 bg-slate-900/80 px-4 py-2 text-sm font-semibold text-slate-200 hover:bg-slate-800/90",
					}, html.Text("Theme: "+strings.ToUpper(parseTheme.Get()))),
				),
			),
		),
		func() ui.Node {
			if parseNotice.Get().Title == "" {
				return nil
			}

			parseAccent := "border-cyan-900/80 bg-cyan-950/50 text-cyan-100"
			if parseNotice.Get().Level == "warn" {
				parseAccent = "border-amber-900/80 bg-amber-950/45 text-amber-100"
			}

			return html.Div(html.Props{
				Class: "mx-auto mt-6 max-w-6xl rounded-2xl border px-5 py-4 " + parseAccent,
			},
				html.Div(html.Props{
					Class: "flex items-start justify-between gap-4",
				},
					html.Div(html.Props{},
						html.Strong(html.Props{Class: "block text-sm uppercase tracking-[0.25em]"}, html.Text(parseNotice.Get().Title)),
						html.P(html.Props{Class: "mt-2 text-sm leading-6"}, html.Text(parseNotice.Get().Body)),
					),
					html.Button(html.Props{
						OnClick: clearNotice,
						Class:   "rounded-full border border-slate-700/80 bg-slate-900/60 px-3 py-1 text-xs font-bold hover:bg-slate-800/90",
					}, html.Text("Dismiss")),
				),
			)
		}(),
		html.Main(html.Props{
			Class: "mx-auto max-w-6xl px-6 py-10",
		}, parseProps.Page),
		ui.CreateElement(devtools.Panel, devtools.PanelProps{
			Title:           "OMI Devtools",
			InitiallyOpen:   false,
			RefreshInterval: 750 * time.Millisecond,
			MaxDepth:        5,
		}),
	)
}

func NavLink(parseProps NavLinkProps) ui.Node {
	parseNav := router.UseNavigate()
	parseNavigate := ui.UseEvent(func() {
		parseNav.Navigate(parseProps.Path)
	})

	parseClassName := "rounded-full border border-slate-700/80 bg-slate-950/30 px-4 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-800/80"
	if parseProps.Active {
		parseClassName = "rounded-full border border-cyan-900/90 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100"
	}

	return html.Button(html.Props{
		OnClick: parseNavigate,
		Class:   parseClassName,
	}, html.Text(parseProps.Label))
}

func OverviewPage() ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/",
		Title:      "Overview",
		Page:       ui.CreateElement(OverviewContent),
	})
}

func DataPage() ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/data",
		Title:      "Data Lab",
		Page:       ui.CreateElement(DataContent),
	})
}

func DataDetailPage(parseProps Attrs) ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/data",
		Title:      "Record Detail",
		Page:       ui.CreateElement(DataDetailContent, parseProps),
	})
}

func SearchPage(parseProps Attrs) ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/search",
		Title:      "Search",
		Page:       ui.CreateElement(SearchContent, parseProps),
	})
}

func SecurePage(parseProps Attrs) ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/secure",
		Title:      "Secure",
		Page:       ui.CreateElement(SecureContent, parseProps),
	})
}

func PlaygroundPage() ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/playground",
		Title:      "Playground",
		Page:       ui.CreateElement(PlaygroundContent),
	})
}

func DataDetailRouteLoading(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseSlug, _ := parseProps["id"].(string)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Route Loader")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Loading record detail")),
				html.P(html.Props{Class: "mt-2 text-slate-300"}, html.Text("The router is loading route-scoped data before rendering the detail page.")),
			),
			html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/data") }), Class: "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90"}, html.Text("Back to Data Lab")),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.P(html.Props{Class: "text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Route slug: "+parseSlug)),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("Loading record detail...")),
		),
	)
}

func DataDetailRouteError(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseMessage, _ := parseProps["error"].(string)
	parseSlug, _ := parseProps["id"].(string)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Route Loader Error")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Record detail failed")),
			),
			html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/data") }), Class: "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90"}, html.Text("Back to Data Lab")),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-rose-400/20 bg-rose-400/10 p-8 text-rose-100"},
			html.P(html.Props{Class: "text-sm uppercase tracking-[0.28em] text-rose-200/80"}, html.Text("Route slug: "+parseSlug)),
			html.P(html.Props{Class: "mt-4"}, html.Text(parseMessage)),
		),
	)
}

func SearchRouteLoading(parseProps Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Route Loader")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Loading search results")),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("The router is revalidating search results for the current query.")),
		),
	)
}

func SearchRouteError(parseProps Attrs) ui.Node {
	parseMessage, _ := parseProps["error"].(string)
	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-rose-400/20 bg-rose-400/10 p-8 text-rose-100"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Route Loader Error")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Search route failed")),
			html.P(html.Props{Class: "mt-4"}, html.Text(parseMessage)),
		),
	)
}

func SecureRouteLoading(parseProps Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Protected Loader")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Checking session access")),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("The router is checking whether the current request should be allowed into the protected route.")),
		),
	)
}

func SecureRouteError(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseMessage, _ := parseProps["error"].(string)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Protected Loader")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Access denied")),
			),
			html.Div(html.Props{Class: "flex gap-3"},
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/secure?auth=true&role=admin") }), Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Grant demo access")),
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/") }), Class: "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90"}, html.Text("Back home")),
			),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-rose-400/20 bg-rose-400/10 p-8 text-rose-100"},
			html.P(html.Props{Class: "mt-1"}, html.Text(parseMessage)),
			html.P(html.Props{Class: "mt-4 text-rose-100/80"}, html.Text("Use the demo access button to reload this protected route with a simulated authenticated session.")),
		),
	)
}

func NotFoundPage() ui.Node {
	parseNav := router.UseNavigate()
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "*",
		Title:      "Not Found",
		Page: html.Div(html.Props{
			Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-10 text-center",
		},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("404")),
			html.H2(html.Props{Class: "mt-4 text-4xl font-black"}, html.Text("Route not found")),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("The requested OMI panel does not exist.")),
			html.Button(html.Props{
				OnClick: ui.UseEvent(func() { parseNav.Navigate("/") }),
				Class:   "mt-8 rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold hover:bg-slate-800/90",
			}, html.Text("Return home")),
		),
	})
}

func OverviewContent() ui.Node {
	parseSearch := state.UseAtom(searchAtom, "")
	parseNotice := state.UseAtom(noticeAtom, Notice{})

	parseSessionStart := ui.UseRef(time.Now().Format("3:04:05 PM"))
	parseTickCount := ui.UseState(0)
	parseSearchID := ui.UseId()

	parseBroadcast := ui.UseCallback(func(parseTitle, parseBody string) {
		parseNotice.Set(Notice{Title: parseTitle, Body: parseBody, Level: "info"})
	}, parseSearch.Get())

	parseAnnounce := ui.UseEvent(func() {
		parseBroadcast("Overview", "Shared search is set to '"+parseSearch.Get()+"'.")
	})

	parseUpdateSearch := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseSearch.Set(parseEvent.GetValue())
	})

	ui.UseEffect(func() func() {
		parseStop := make(chan struct{})
		var parseStopOnce sync.Once
		parseTicker := time.NewTicker(time.Second)

		go func() {
			for {
				select {
				case <-parseTicker.C:
					parseTickCount.Update(func(parseV int) int { return parseV + 1 })
				case <-parseStop:
					return
				}
			}
		}()

		return func() {
			parseStopOnce.Do(func() {
				parseTicker.Stop()
				close(parseStop)
			})
		}
	}, "overview-ticker")

	parseHighlights := html.Fragment(
		html.Div(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-slate-400"}, html.Text("UseRef")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(parseSessionStart.Get())),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text("Session start is stored in a ref so it stays stable across re-renders.")),
		),
		html.Div(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-slate-400"}, html.Text("UseEffect")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(fmt.Sprintf("%d ticks", parseTickCount.Get()))),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text("A ticker effect increments local state every second and cleans up on route changes.")),
		),
		html.Div(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-slate-400"}, html.Text("Shared Atoms")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(strings.TrimSpace(parseSearch.Get()+" ")+func() string {
				if parseSearch.Get() == "" {
					return "idle"
				}
				return "active"
			}())),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text("Search text and notifications are shared across routes through atoms.")),
		),
	)

	return html.Div(html.Props{
		Class: "space-y-8",
	},
		html.Section(html.Props{
			Class: "rounded-[2rem] border border-slate-800/80 bg-[radial-gradient(circle_at_top,_rgba(8,145,178,0.18),_transparent_55%),rgba(13,23,34,0.88)] p-8",
		},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Overview")),
			html.H2(html.Props{Class: "mt-4 max-w-3xl text-5xl font-black tracking-tight"}, html.Text("One example that exercises the full public surface.")),
			html.P(html.Props{Class: "mt-4 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("This page demonstrates local state, effects, refs, IDs, callbacks, and shared atoms. The other panels add fetch and routing on top.")),
			html.Div(html.Props{
				Class: "mt-8 flex flex-wrap items-end gap-4",
			},
				html.Div(html.Props{Class: "flex-1 min-w-[16rem]"},
					html.Label(html.Props{
						For:   parseSearchID,
						Class: "mb-2 block text-xs uppercase tracking-[0.28em] text-slate-400",
					}, html.Text("Global search atom")),
					html.Input(html.Props{
						ID:          parseSearchID,
						Value:       parseSearch.Get(),
						OnInput:     parseUpdateSearch,
						Placeholder: "Type once here, reuse it on the data route",
						Class:       "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
					}),
				),
				html.Button(html.Props{
					OnClick: parseAnnounce,
					Class:   "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90",
				}, html.Text("Broadcast notice")),
			),
		),
		html.Section(html.Props{
			Class: "grid gap-6 md:grid-cols-3",
		}, parseHighlights),
		func() ui.Node {
			if parseNotice.Get().Title == "" {
				return nil
			}
			return html.Blockquote(html.Props{
				Class: "rounded-3xl border border-slate-800/80 bg-slate-950/65 p-6 text-slate-200",
			}, html.Text("Latest notice: "+parseNotice.Get().Title+" - "+parseNotice.Get().Body))
		}(),
	)
}

func DataContent() ui.Node {
	parseSearch := state.UseAtom(searchAtom, "")
	parseNotice := state.UseAtom(noticeAtom, Notice{})
	parseNav := router.UseNavigate()
	parseQuery := router.UseQuery()
	parseResource := fetch.UseFetch("./fixtures.json")
	parseRecords := ui.UseState([]HookRecord{})
	parseLoadedAt := ui.UseState("never")
	parseLastQuery := ui.UseRef("")
	parseSearchID := ui.UseId()
	parseUrlFilter := parseQuery.Get("q")

	parseFetchState := parseResource.Get()

	ui.UseEffect(func() func() {
		if strings.TrimSpace(parseSearch.Get()) != strings.TrimSpace(parseUrlFilter) {
			parseSearch.Set(parseUrlFilter)
		}
		return nil
	}, parseUrlFilter)

	parseRefresh := ui.UseEvent(func() {
		parseLastQuery.Set(parseSearch.Get())
		parseResource.Refetch()
	})

	parseUpdateSearch := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseValue := parseEvent.GetValue()
		parseSearch.Set(parseValue)
		parseTrimmed := strings.TrimSpace(parseValue)
		if parseTrimmed == "" {
			parseNav.Replace("/data")
			return
		}
		parseEncoded := url.QueryEscape(parseTrimmed)
		parseNav.Replace("/data?q=" + parseEncoded)
	})

	parseBadgeClass := ui.UseCallback(func(parseCategory string) string {
		switch strings.ToLower(parseCategory) {
		case "hooks":
			return "bg-cyan-950/70 text-cyan-100 border-cyan-900/80"
		case "state":
			return "bg-emerald-400/15 text-emerald-100 border-emerald-400/20"
		case "routing":
			return "bg-fuchsia-400/15 text-fuchsia-100 border-fuchsia-400/20"
		default:
			return "bg-slate-900/70 text-slate-100 border-slate-700/80"
		}
	}, len(parseRecords.Get()))

	ui.UseEffect(func() func() {
		parseResource.Refetch()
		return nil
	}, "load-fixtures")

	ui.UseEffect(func() func() {
		if parseFetchState.Data == nil {
			return nil
		}

		parsePayload, parseOk := parseFetchState.Data.(string)
		if !parseOk || parsePayload == "" {
			return nil
		}

		var parseParsed []HookRecord
		if parseErr := json.Unmarshal([]byte(parsePayload), &parseParsed); parseErr == nil {
			parseRecords.Set(parseParsed)
			parseLoadedAt.Set(time.Now().Format("3:04:05 PM"))
			parseNotice.Set(Notice{
				Title: "Data Lab",
				Body:  fmt.Sprintf("Loaded %d showcase records from local fixtures.", len(parseParsed)),
				Level: "info",
			})
		}
		return nil
	}, parseFetchState.Data)

	parseFiltered := ui.UseMemo(func() []HookRecord {
		return filterHookRecords(parseRecords.Get(), parseSearch.Get())
	}, parseRecords.Get(), parseSearch.Get())

	parseCards := make([]ui.Node, 0, len(parseFiltered))
	for _, parseRecord := range parseFiltered {
		parseRecord2 := parseRecord
		parseHookTags := make([]ui.Node, 0, len(parseRecord2.Hooks))
		for _, parseHook := range parseRecord2.Hooks {
			parseHookTags = append(parseHookTags, html.Code(html.Props{
				Class: "rounded-full border border-slate-700/80 bg-slate-950/65 px-3 py-1 text-xs text-slate-200",
			}, html.Text(parseHook)))
		}

		parseCards = append(parseCards, html.Article(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
				html.Div(html.Props{},
					html.H3(html.Props{Class: "text-2xl font-black"}, html.Text(parseRecord2.Name)),
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(parseRecord2.Summary)),
				),
				html.Span(html.Props{
					Class: "rounded-full border px-3 py-1 text-xs font-bold uppercase tracking-[0.25em] " + parseBadgeClass(parseRecord2.Category),
				}, html.Text(parseRecord2.Category)),
			),
			html.Div(html.Props{
				Class: "mt-5 flex flex-wrap gap-2",
			}, parseHookTags...),
			html.Button(html.Props{
				OnClick: ui.UseEvent(func() { parseNav.Navigate("/data/" + recordSlug(parseRecord2.Name)) }),
				Class:   "mt-5 rounded-full border border-cyan-900/80 bg-cyan-950/60 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80",
			}, html.Text("Open detail")),
		))
	}

	parseStatusText := "Idle"
	if parseFetchState.Loading {
		parseStatusText = "Loading"
	} else if parseFetchState.Error != "" {
		parseStatusText = "Error"
	} else if len(parseRecords.Get()) > 0 {
		parseStatusText = "Ready"
	}

	return html.Div(html.Props{
		Class: "space-y-8",
	},
		html.Section(html.Props{
			Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8",
		},
			html.Div(html.Props{
				Class: "flex flex-wrap items-end justify-between gap-6",
			},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Data Lab")),
					html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Manual fetch, shared filters, memoized results.")),
				),
				html.Button(html.Props{
					OnClick: parseRefresh,
					Class:   "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90",
				}, html.Text("Refresh fixtures")),
				html.Button(html.Props{
					OnClick: ui.UseEvent(func() {
						parseTrimmed2 := strings.TrimSpace(parseSearch.Get())
						if parseTrimmed2 == "" {
							parseNav.Navigate("/search")
							return
						}
						parseNav.Navigate("/search?q=" + url.QueryEscape(parseTrimmed2))
					}),
					Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80",
				}, html.Text("Open route search")),
			),
			html.Div(html.Props{
				Class: "mt-8 grid gap-4 md:grid-cols-[1fr_auto_auto_auto]",
			},
				html.Div(html.Props{},
					html.Label(html.Props{
						For:   parseSearchID,
						Class: "mb-2 block text-xs uppercase tracking-[0.28em] text-slate-400",
					}, html.Text("Search")),
					html.Input(html.Props{
						ID:          parseSearchID,
						Value:       parseSearch.Get(),
						OnInput:     parseUpdateSearch,
						Placeholder: "Filter by hook, category, or summary",
						Class:       "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
					}),
				),
				html.Div(html.Props{},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Status")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(parseStatusText)),
				),
				html.Div(html.Props{},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Loaded at")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(parseLoadedAt.Get())),
				),
				html.Div(html.Props{},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Previous query")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(func() string {
						if parseLastQuery.Get() == "" {
							return "none"
						}
						return parseLastQuery.Get()
					}())),
				),
				html.Div(html.Props{},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("URL filter")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(func() string {
						if parseUrlFilter == "" {
							return "none"
						}
						return parseUrlFilter
					}())),
				),
			),
		),
		func() ui.Node {
			if parseFetchState.Error == "" {
				return nil
			}
			return html.Div(html.Props{
				Class: "rounded-3xl border border-rose-400/20 bg-rose-400/10 p-5 text-rose-100",
			}, html.Text("Fixture load failed: "+parseFetchState.Error))
		}(),
		html.Section(html.Props{
			Class: "grid gap-6 md:grid-cols-2",
		}, parseCards...),
	)
}

func DataDetailContent(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseRevalidator := router.UseRevalidator()
	parseRouteData := router.UseRouteData()
	parseSlug, _ := parseProps["slug"].(string)
	if parseRouteData != nil {
		if parseValue, parseOk := parseRouteData["slug"].(string); parseOk && parseValue != "" {
			parseSlug = parseValue
		}
	}

	parseSelected := HookRecord{}
	if parseName, parseOk2 := parseProps["name"].(string); parseOk2 {
		parseSelected.Name = parseName
	}
	if parseCategory, parseOk3 := parseProps["category"].(string); parseOk3 {
		parseSelected.Category = parseCategory
	}
	if parseSummary, parseOk4 := parseProps["summary"].(string); parseOk4 {
		parseSelected.Summary = parseSummary
	}
	if parseHooks, parseOk5 := parseProps["hooks"].([]string); parseOk5 {
		parseSelected.Hooks = parseHooks
	}

	parseHookTags := make([]ui.Node, 0, len(parseSelected.Hooks))
	for _, parseHook := range parseSelected.Hooks {
		parseHookTags = append(parseHookTags, html.Code(html.Props{
			Class: "rounded-full border border-slate-700/80 bg-slate-950/65 px-3 py-1 text-xs text-slate-200",
		}, html.Text(parseHook)))
	}

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Route Loader")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Loader-backed detail route")),
				html.P(html.Props{Class: "mt-2 text-slate-300"}, html.Text("This page resolves fixture data in `router.Options{Loader: ...}` before rendering the final route component.")),
			),
			html.Div(html.Props{Class: "flex gap-3"},
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseRevalidator.Revalidate() }), Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(func() string {
					if parseRevalidator.Loading() {
						return "Revalidating..."
					}
					return "Revalidate route"
				}())),
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/data") }), Class: "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90"}, html.Text("Back to Data Lab")),
			),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Data Detail")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text(parseSelected.Name)),
			func() ui.Node {
				if parseSelected.Category == "" {
					return nil
				}
				return html.P(html.Props{Class: "mt-3 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Category: "+parseSelected.Category))
			}(),
			html.P(html.Props{Class: "mt-4 text-lg leading-8 text-slate-300"}, html.Text(parseSelected.Summary)),
			html.P(html.Props{Class: "mt-4 text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Route slug: "+parseSlug)),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-500"}, html.Text("Data came from the router loader cache for the current route key.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-2"}, parseHookTags...),
		),
	)
}

func SearchContent(parseProps Attrs) ui.Node {
	parseSearch := state.UseAtom(searchAtom, "")
	parseNav := router.UseNavigate()
	parseRevalidator := router.UseRevalidator()
	parseQuery := router.UseQuery()
	parseRouteData := router.UseRouteData()
	parseSearchID := ui.UseId()
	parseQueryTerm := parseQuery.Get("q")

	ui.UseEffect(func() func() {
		if strings.TrimSpace(parseSearch.Get()) != strings.TrimSpace(parseQueryTerm) {
			parseSearch.Set(parseQueryTerm)
		}
		return nil
	}, parseQueryTerm)

	parseUpdateSearch := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseValue := parseEvent.GetValue()
		parseSearch.Set(parseValue)
		parseTrimmed := strings.TrimSpace(parseValue)
		if parseTrimmed == "" {
			parseNav.Replace("/search")
			return
		}
		parseNav.Replace("/search?q=" + url.QueryEscape(parseTrimmed))
	})

	parseResults, _ := parseProps["results"].([]HookRecord)
	if parseRouteData != nil {
		if parseLoadedResults, parseOk := parseRouteData["results"].([]HookRecord); parseOk {
			parseResults = parseLoadedResults
		}
	}

	parseCount := len(parseResults)
	parseResultCards := make([]ui.Node, 0, parseCount)
	for _, parseRecord := range parseResults {
		parseRecord2 := parseRecord
		parseHookTags := make([]ui.Node, 0, len(parseRecord2.Hooks))
		for _, parseHook := range parseRecord2.Hooks {
			parseHookTags = append(parseHookTags, html.Code(html.Props{
				Class: "rounded-full border border-slate-700/80 bg-slate-950/65 px-3 py-1 text-xs text-slate-200",
			}, html.Text(parseHook)))
		}

		parseResultCards = append(parseResultCards, html.Article(html.Props{Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6"},
			html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
				html.Div(html.Props{},
					html.H3(html.Props{Class: "text-2xl font-black"}, html.Text(parseRecord2.Name)),
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(parseRecord2.Summary)),
				),
				html.Button(html.Props{
					OnClick: ui.UseEvent(func() { parseNav.Navigate("/data/" + recordSlug(parseRecord2.Name)) }),
					Class:   "rounded-full border border-cyan-900/80 bg-cyan-950/60 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80",
				}, html.Text("Open detail")),
			),
			html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-2"}, parseHookTags...),
		))
	}

	return html.Div(html.Props{Class: "space-y-8"},
		html.Section(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.Div(html.Props{Class: "flex flex-wrap items-end justify-between gap-6"},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Route Loader Search")),
					html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Query-driven route data")),
					html.P(html.Props{Class: "mt-4 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("This page reloads route data when the `q` query param changes, using the router loader key instead of page-local fetch state.")),
				),
				html.Div(html.Props{Class: "flex items-end gap-4"},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Matches")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(fmt.Sprintf("%d", parseCount))),
					html.Button(html.Props{OnClick: ui.UseEvent(func() { parseRevalidator.Revalidate() }), Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(func() string {
						if parseRevalidator.Loading() {
							return "Revalidating..."
						}
						return "Revalidate route"
					}())),
				),
			),
			html.Div(html.Props{Class: "mt-8"},
				html.Label(html.Props{For: parseSearchID, Class: "mb-2 block text-xs uppercase tracking-[0.28em] text-slate-400"}, html.Text("Query")),
				html.Input(html.Props{
					ID:          parseSearchID,
					Value:       parseQueryTerm,
					OnInput:     parseUpdateSearch,
					Placeholder: "Search hooks, summaries, or categories",
					Class:       "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
				}),
			),
		),
		func() ui.Node {
			if parseCount == 0 {
				return html.Div(html.Props{Class: "rounded-3xl border border-slate-800/80 bg-slate-950/65 p-8 text-slate-300"},
					html.P(html.Props{}, html.Text("No route-loaded results matched the current query.")),
				)
			}
			return html.Section(html.Props{Class: "grid gap-6 md:grid-cols-2"}, parseResultCards...)
		}(),
	)
}

func SecureContent(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseRevalidator := router.UseRevalidator()
	parseRouteData := router.UseRouteData()
	parseUserName, _ := parseProps["userName"].(string)
	parseRole, _ := parseProps["role"].(string)
	parseGrantedBy, _ := parseProps["grantedBy"].(string)
	if parseRouteData != nil {
		if parseValue, parseOk := parseRouteData["userName"].(string); parseOk && parseValue != "" {
			parseUserName = parseValue
		}
		if parseValue2, parseOk2 := parseRouteData["role"].(string); parseOk2 && parseValue2 != "" {
			parseRole = parseValue2
		}
		if parseValue3, parseOk3 := parseRouteData["grantedBy"].(string); parseOk3 && parseValue3 != "" {
			parseGrantedBy = parseValue3
		}
	}

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Protected Route Loader")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Protected session granted")),
				html.P(html.Props{Class: "mt-2 text-slate-300"}, html.Text("This page only renders after the route loader approves the simulated session state.")),
			),
			html.Div(html.Props{Class: "flex gap-3"},
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseRevalidator.Revalidate() }), Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text(func() string {
					if parseRevalidator.Loading() {
						return "Revalidating..."
					}
					return "Revalidate route"
				}())),
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Replace("/secure") }), Class: "rounded-full border border-rose-400/30 bg-rose-400/10 px-5 py-3 font-semibold text-rose-100 hover:bg-rose-400/20"}, html.Text("Revoke access")),
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Replace("/secure?auth=true&role=auditor") }), Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Switch role")),
			),
		),
		html.Div(html.Props{Class: "grid gap-6 md:grid-cols-3"},
			html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-6"},
				html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("User")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(parseUserName)),
			),
			html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-6"},
				html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Role")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(strings.ToUpper(parseRole))),
			),
			html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-6"},
				html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Granted by")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(parseGrantedBy)),
			),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-slate-950/65 p-8"},
			html.P(html.Props{Class: "text-slate-300 leading-7"}, html.Text("Protected-route loaders are useful when access checks and route data should happen before the final page renders. This keeps denial, loading, and successful route states at the router layer rather than scattering them through page-local effects.")),
		),
	)
}

func PlaygroundContent() ui.Node {
	parseNotice := state.UseAtom(noticeAtom, Notice{})
	parseParams := router.UseParams()
	parseNav := router.UseNavigate()
	parseTitle := ui.UseState("")
	parseNotes := ui.UseState("")
	parseRouteMode := normalizePlaygroundMode(parseParams.Get("mode"))
	parseMode := ui.UseState(parseRouteMode)
	parseSavedCount := ui.UseState(0)
	parseSessionCode := ui.UseRef("omi-" + time.Now().Format("150405"))

	parseTitleID := ui.UseId()
	parseNotesID := ui.UseId()
	parseModeID := ui.UseId()

	parsePreview := ui.UseMemo(func() string {
		parseTrimmed := strings.TrimSpace(parseNotes.Get())
		if parseTrimmed == "" {
			parseTrimmed = "No notes yet."
		}
		return fmt.Sprintf("%s | %s | %s", parseTitle.Get(), strings.ToUpper(parseMode.Get()), parseTrimmed)
	}, parseTitle.Get(), parseNotes.Get(), parseMode.Get())

	parseSaveCallback := ui.UseCallback(func() {
		parseSavedCount.Update(func(parseV int) int { return parseV + 1 })
		parseNotice.Set(Notice{
			Title: "Playground",
			Body:  "Saved session " + parseSessionCode.Get() + " with mode " + strings.ToUpper(parseMode.Get()) + ".",
			Level: "warn",
		})
	}, parseTitle.Get(), parseNotes.Get(), parseMode.Get(), parseSavedCount.Get())

	parseSave := ui.UseEvent(parseSaveCallback)
	reset := ui.UseEvent(func() {
		parseTitle.Set("")
		parseNotes.Set("")
		parseMode.Set("agent")
		parseNav.Replace("/playground/agent")
	})

	parseUpdateTitle := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseTitle.Set(parseEvent.GetValue())
	})
	parseUpdateNotes := ui.UseEvent(func(parseEvent2 ui.InputEvent) {
		parseNotes.Set(parseEvent2.GetValue())
	})
	parseUpdateMode := ui.UseEvent(func(parseEvent3 ui.ChangeEvent) {
		parseNextMode := normalizePlaygroundMode(parseEvent3.GetValue())
		parseMode.Set(parseNextMode)
		parseNav.Replace("/playground/" + parseNextMode)
	})

	ui.UseEffect(func() func() {
		if parseMode.Get() != parseRouteMode {
			parseMode.Set(parseRouteMode)
		}
		return nil
	}, parseRouteMode)

	ui.UseEffect(func() func() {
		parseStorage := js.Global().Get("localStorage")
		if parseStorage.Truthy() {
			parseStorage.Call("setItem", "omi-draft", parsePreview)
		}
		return nil
	}, parsePreview)

	return html.Div(html.Props{
		Class: "grid gap-8 lg:grid-cols-[1.15fr_0.85fr]",
	},
		html.Section(html.Props{
			Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8",
		},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Playground")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Forms, IDs, callbacks, refs, and effect persistence.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/playground/agent") }), Class: func() string {
					if parseMode.Get() == "agent" {
						return "rounded-full border border-cyan-900/90 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100"
					}
					return "rounded-full border border-slate-700/80 bg-slate-900/70 px-4 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-800/90"
				}()}, html.Text("/playground/agent")),
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/playground/review") }), Class: func() string {
					if parseMode.Get() == "review" {
						return "rounded-full border border-cyan-900/90 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100"
					}
					return "rounded-full border border-slate-700/80 bg-slate-900/70 px-4 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-800/90"
				}()}, html.Text("/playground/review")),
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/playground/benchmark") }), Class: func() string {
					if parseMode.Get() == "benchmark" {
						return "rounded-full border border-cyan-900/90 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100"
					}
					return "rounded-full border border-slate-700/80 bg-slate-900/70 px-4 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-800/90"
				}()}, html.Text("/playground/benchmark")),
			),
			html.P(html.Props{Class: "mt-4 text-sm text-slate-400"}, html.Text("These controls demonstrate nested route navigation by switching child paths under /playground.")),
			html.Form(html.Props{
				Class: "mt-8 space-y-6",
			},
				html.Fieldset(html.Props{
					Class: "space-y-5",
				},
					html.Legend(html.Props{
						Class: "text-sm font-bold uppercase tracking-[0.25em] text-slate-300",
					}, html.Text("Compose a session")),
					html.Div(html.Props{},
						html.Label(html.Props{
							For:   parseTitleID,
							Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400",
						}, html.Text("Title")),
						html.Input(html.Props{
							ID:          parseTitleID,
							Value:       parseTitle.Get(),
							OnInput:     parseUpdateTitle,
							Placeholder: "Agent coders and durable UI APIs",
							Class:       "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
						}),
					),
					html.Div(html.Props{},
						html.Label(html.Props{
							For:   parseModeID,
							Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400",
						}, html.Text("Mode")),
						html.Select(html.Props{
							ID:       parseModeID,
							Value:    parseMode.Get(),
							OnChange: parseUpdateMode,
							Class:    "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 focus:outline-none",
						},
							html.Option(html.Props{Value: "agent"}, html.Text("Agent")),
							html.Option(html.Props{Value: "review"}, html.Text("Review")),
							html.Option(html.Props{Value: "benchmark"}, html.Text("Benchmark")),
						),
					),
					html.Div(html.Props{},
						html.Label(html.Props{
							For:   parseNotesID,
							Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400",
						}, html.Text("Notes")),
						html.Textarea(html.Props{
							ID:          parseNotesID,
							Value:       parseNotes.Get(),
							OnInput:     parseUpdateNotes,
							Placeholder: "Describe the scenario you want this session to cover...",
							Class:       "min-h-[10rem] w-full rounded-3xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
						}),
					),
				),
				html.Div(html.Props{
					Class: "flex flex-wrap gap-3",
				},
					html.Button(html.Props{
						Type:    "button",
						OnClick: parseSave,
						Class:   "rounded-full border border-cyan-900/90 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80",
					}, html.Text("Save session")),
					html.Button(html.Props{
						Type:    "button",
						OnClick: reset,
						Class:   "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90",
					}, html.Text("Reset")),
				),
			),
		),
		html.Aside(html.Props{
			Class: "space-y-6",
		},
			html.Div(html.Props{
				Class: "rounded-[2rem] border border-slate-800/80 bg-slate-950/65 p-6",
			},
				html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Live preview")),
				html.Pre(html.Props{
					Class: "mt-4 whitespace-pre-wrap break-words text-sm leading-7 text-slate-200",
				}, html.Text(parsePreview)),
			),
			html.Div(html.Props{
				Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/82 p-6",
			},
				html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Session ref")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(parseSessionCode.Get())),
				html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(fmt.Sprintf("Saved %d times in this browser session.", parseSavedCount.Get()))),
				func() ui.Node {
					if parseNotice.Get().Title == "" {
						return nil
					}
					return html.P(html.Props{
						Class: "mt-4 rounded-2xl border border-slate-700/80 bg-slate-950/65 px-4 py-3 text-sm text-slate-200",
					}, html.Text("Latest global notice: "+parseNotice.Get().Body))
				}(),
			),
		),
	)
}
