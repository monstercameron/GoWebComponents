//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

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

func Shell(props ShellProps) ui.Node {
	theme := state.UseAtom(themeAtom, "aurora")
	search := state.UseAtom(searchAtom, "")
	notice := state.UseAtom(noticeAtom, Notice{})

	toggleTheme := ui.UseEvent(func() {
		theme.Update(func(current string) string {
			if current == "aurora" {
				return "signal"
			}
			return "aurora"
		})
	})

	clearNotice := ui.UseEvent(func() {
		notice.Set(Notice{})
	})

	ui.UseEffect(func() func() {
		document := js.Global().Get("document")
		if document.Truthy() {
			document.Set("title", "OMI Example - "+props.Title)
		}

		body := document.Get("body")
		if body.Truthy() {
			body.Call("setAttribute", "data-theme", theme.Get())
		}

		storage := js.Global().Get("localStorage")
		if storage.Truthy() {
			storage.Call("setItem", "omi-theme", theme.Get())
		}
		return nil
	}, props.Title, theme.Get())

	themePanelClass := "min-h-screen text-slate-100 selection:bg-cyan-400/20 "
	if theme.Get() == "signal" {
		themePanelClass += "bg-[#0d1319]"
	} else {
		themePanelClass += "bg-[#071018]"
	}

	return html.Div(html.Props{
		Class: themePanelClass,
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
					ui.CreateElement(NavLink, NavLinkProps{Label: "Overview", Path: "/", Active: props.ActivePath == "/"}),
					ui.CreateElement(NavLink, NavLinkProps{Label: "Data Lab", Path: "/data", Active: props.ActivePath == "/data"}),
					ui.CreateElement(NavLink, NavLinkProps{Label: "Playground", Path: "/playground", Active: props.ActivePath == "/playground"}),
				),
				html.Div(html.Props{
					Class: "flex items-center gap-3",
				},
					html.Small(html.Props{
						Class: "hidden text-right text-xs uppercase tracking-[0.25em] text-slate-400 md:block",
					}, html.Text("Shared search: "+search.Get())),
					html.Button(html.Props{
						OnClick: toggleTheme,
						Class:   "rounded-full border border-slate-700/80 bg-slate-900/80 px-4 py-2 text-sm font-semibold text-slate-200 hover:bg-slate-800/90",
					}, html.Text("Theme: "+strings.ToUpper(theme.Get()))),
				),
			),
		),
		func() ui.Node {
			if notice.Get().Title == "" {
				return nil
			}

			accent := "border-cyan-900/80 bg-cyan-950/50 text-cyan-100"
			if notice.Get().Level == "warn" {
				accent = "border-amber-900/80 bg-amber-950/45 text-amber-100"
			}

			return html.Div(html.Props{
				Class: "mx-auto mt-6 max-w-6xl rounded-2xl border px-5 py-4 " + accent,
			},
				html.Div(html.Props{
					Class: "flex items-start justify-between gap-4",
				},
					html.Div(html.Props{},
						html.Strong(html.Props{Class: "block text-sm uppercase tracking-[0.25em]"}, html.Text(notice.Get().Title)),
						html.P(html.Props{Class: "mt-2 text-sm leading-6"}, html.Text(notice.Get().Body)),
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
		}, props.Page),
	)
}

func NavLink(props NavLinkProps) ui.Node {
	navigate := ui.UseEvent(func() {
		router.Navigate(props.Path)
	})

	className := "rounded-full border border-slate-700/80 bg-slate-950/30 px-4 py-2 text-sm font-semibold text-slate-300 hover:bg-slate-800/80"
	if props.Active {
		className = "rounded-full border border-cyan-900/90 bg-cyan-950/70 px-4 py-2 text-sm font-semibold text-cyan-100"
	}

	return html.Button(html.Props{
		OnClick: navigate,
		Class:   className,
	}, html.Text(props.Label))
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

func PlaygroundPage() ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/playground",
		Title:      "Playground",
		Page:       ui.CreateElement(PlaygroundContent),
	})
}

func NotFoundPage() ui.Node {
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
				OnClick: ui.UseEvent(func() { router.Navigate("/") }),
				Class:   "mt-8 rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold hover:bg-slate-800/90",
			}, html.Text("Return home")),
		),
	})
}

func OverviewContent() ui.Node {
	search := state.UseAtom(searchAtom, "")
	notice := state.UseAtom(noticeAtom, Notice{})

	sessionStart := ui.UseRef(time.Now().Format("3:04:05 PM"))
	tickCount := ui.UseState(0)
	searchID := ui.UseId()

	broadcast := ui.UseCallback(func(title, body string) {
		notice.Set(Notice{Title: title, Body: body, Level: "info"})
	}, search.Get())

	announce := ui.UseEvent(func() {
		broadcast("Overview", "Shared search is set to '"+search.Get()+"'.")
	})

	updateSearch := ui.UseEvent(func(event ui.InputEvent) {
		search.Set(event.GetValue())
	})

	ui.UseEffect(func() func() {
		stop := make(chan struct{})
		var stopOnce sync.Once
		ticker := time.NewTicker(time.Second)

		go func() {
			for {
				select {
				case <-ticker.C:
					tickCount.Update(func(v int) int { return v + 1 })
				case <-stop:
					return
				}
			}
		}()

		return func() {
			stopOnce.Do(func() {
				ticker.Stop()
				close(stop)
			})
		}
	}, "overview-ticker")

	highlights := html.Fragment(
		html.Div(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-slate-400"}, html.Text("UseRef")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(sessionStart.Get())),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text("Session start is stored in a ref so it stays stable across re-renders.")),
		),
		html.Div(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-slate-400"}, html.Text("UseEffect")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(fmt.Sprintf("%d ticks", tickCount.Get()))),
			html.P(html.Props{Class: "mt-2 text-sm text-slate-300"}, html.Text("A ticker effect increments local state every second and cleans up on route changes.")),
		),
		html.Div(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Small(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-slate-400"}, html.Text("Shared Atoms")),
			html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(strings.TrimSpace(search.Get()+" ")+func() string {
				if search.Get() == "" {
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
						For:   searchID,
						Class: "mb-2 block text-xs uppercase tracking-[0.28em] text-slate-400",
					}, html.Text("Global search atom")),
					html.Input(html.Props{
						ID:          searchID,
						Value:       search.Get(),
						OnInput:     updateSearch,
						Placeholder: "Type once here, reuse it on the data route",
						Class:       "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
					}),
				),
				html.Button(html.Props{
					OnClick: announce,
					Class:   "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90",
				}, html.Text("Broadcast notice")),
			),
		),
		html.Section(html.Props{
			Class: "grid gap-6 md:grid-cols-3",
		}, highlights),
		func() ui.Node {
			if notice.Get().Title == "" {
				return nil
			}
			return html.Blockquote(html.Props{
				Class: "rounded-3xl border border-slate-800/80 bg-slate-950/65 p-6 text-slate-200",
			}, html.Text("Latest notice: "+notice.Get().Title+" - "+notice.Get().Body))
		}(),
	)
}

func DataContent() ui.Node {
	search := state.UseAtom(searchAtom, "")
	notice := state.UseAtom(noticeAtom, Notice{})
	resource := fetch.UseFetch("./fixtures.json")
	records := ui.UseState([]HookRecord{})
	loadedAt := ui.UseState("never")
	lastQuery := ui.UseRef("")
	searchID := ui.UseId()

	fetchState := resource.Get()

	refresh := ui.UseEvent(func() {
		lastQuery.Set(search.Get())
		resource.Refetch()
	})

	updateSearch := ui.UseEvent(func(event ui.InputEvent) {
		search.Set(event.GetValue())
	})

	badgeClass := ui.UseCallback(func(category string) string {
		switch strings.ToLower(category) {
		case "hooks":
			return "bg-cyan-950/70 text-cyan-100 border-cyan-900/80"
		case "state":
			return "bg-emerald-400/15 text-emerald-100 border-emerald-400/20"
		case "routing":
			return "bg-fuchsia-400/15 text-fuchsia-100 border-fuchsia-400/20"
		default:
			return "bg-slate-900/70 text-slate-100 border-slate-700/80"
		}
	}, len(records.Get()))

	ui.UseEffect(func() func() {
		resource.Refetch()
		return nil
	}, "load-fixtures")

	ui.UseEffect(func() func() {
		if fetchState.Data == nil {
			return nil
		}

		payload, ok := fetchState.Data.(string)
		if !ok || payload == "" {
			return nil
		}

		var parsed []HookRecord
		if err := json.Unmarshal([]byte(payload), &parsed); err == nil {
			records.Set(parsed)
			loadedAt.Set(time.Now().Format("3:04:05 PM"))
			notice.Set(Notice{
				Title: "Data Lab",
				Body:  fmt.Sprintf("Loaded %d showcase records from local fixtures.", len(parsed)),
				Level: "info",
			})
		}
		return nil
	}, fetchState.Data)

	filtered := ui.UseMemo(func() []HookRecord {
		query := strings.TrimSpace(strings.ToLower(search.Get()))
		if query == "" {
			return records.Get()
		}

		matches := make([]HookRecord, 0, len(records.Get()))
		for _, record := range records.Get() {
			hit := strings.Contains(strings.ToLower(record.Name), query) ||
				strings.Contains(strings.ToLower(record.Category), query) ||
				strings.Contains(strings.ToLower(record.Summary), query)
			if !hit {
				for _, hook := range record.Hooks {
					if strings.Contains(strings.ToLower(hook), query) {
						hit = true
						break
					}
				}
			}

			if hit {
				matches = append(matches, record)
			}
		}
		return matches
	}, records.Get(), search.Get())

	cards := make([]ui.Node, 0, len(filtered))
	for _, record := range filtered {
		hookTags := make([]ui.Node, 0, len(record.Hooks))
		for _, hook := range record.Hooks {
			hookTags = append(hookTags, html.Code(html.Props{
				Class: "rounded-full border border-slate-700/80 bg-slate-950/65 px-3 py-1 text-xs text-slate-200",
			}, html.Text(hook)))
		}

		cards = append(cards, html.Article(html.Props{
			Class: "rounded-3xl border border-slate-800/80 bg-[#0d1722]/82 p-6",
		},
			html.Div(html.Props{Class: "flex items-start justify-between gap-4"},
				html.Div(html.Props{},
					html.H3(html.Props{Class: "text-2xl font-black"}, html.Text(record.Name)),
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(record.Summary)),
				),
				html.Span(html.Props{
					Class: "rounded-full border px-3 py-1 text-xs font-bold uppercase tracking-[0.25em] " + badgeClass(record.Category),
				}, html.Text(record.Category)),
			),
			html.Div(html.Props{
				Class: "mt-5 flex flex-wrap gap-2",
			}, hookTags...),
		))
	}

	statusText := "Idle"
	if fetchState.Loading {
		statusText = "Loading"
	} else if fetchState.Error != "" {
		statusText = "Error"
	} else if len(records.Get()) > 0 {
		statusText = "Ready"
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
					OnClick: refresh,
					Class:   "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90",
				}, html.Text("Refresh fixtures")),
			),
			html.Div(html.Props{
				Class: "mt-8 grid gap-4 md:grid-cols-[1fr_auto_auto_auto]",
			},
				html.Div(html.Props{},
					html.Label(html.Props{
						For:   searchID,
						Class: "mb-2 block text-xs uppercase tracking-[0.28em] text-slate-400",
					}, html.Text("Search")),
					html.Input(html.Props{
						ID:          searchID,
						Value:       search.Get(),
						OnInput:     updateSearch,
						Placeholder: "Filter by hook, category, or summary",
						Class:       "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
					}),
				),
				html.Div(html.Props{},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Status")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(statusText)),
				),
				html.Div(html.Props{},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Loaded at")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(loadedAt.Get())),
				),
				html.Div(html.Props{},
					html.Small(html.Props{Class: "block text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Previous query")),
					html.P(html.Props{Class: "mt-2 text-lg font-bold"}, html.Text(func() string {
						if lastQuery.Get() == "" {
							return "none"
						}
						return lastQuery.Get()
					}())),
				),
			),
		),
		func() ui.Node {
			if fetchState.Error == "" {
				return nil
			}
			return html.Div(html.Props{
				Class: "rounded-3xl border border-rose-400/20 bg-rose-400/10 p-5 text-rose-100",
			}, html.Text("Fixture load failed: "+fetchState.Error))
		}(),
		html.Section(html.Props{
			Class: "grid gap-6 md:grid-cols-2",
		}, cards...),
	)
}

func PlaygroundContent() ui.Node {
	notice := state.UseAtom(noticeAtom, Notice{})
	title := ui.UseState("")
	notes := ui.UseState("")
	mode := ui.UseState("agent")
	savedCount := ui.UseState(0)
	sessionCode := ui.UseRef("omi-" + time.Now().Format("150405"))

	titleID := ui.UseId()
	notesID := ui.UseId()
	modeID := ui.UseId()

	preview := ui.UseMemo(func() string {
		trimmed := strings.TrimSpace(notes.Get())
		if trimmed == "" {
			trimmed = "No notes yet."
		}
		return fmt.Sprintf("%s | %s | %s", title.Get(), strings.ToUpper(mode.Get()), trimmed)
	}, title.Get(), notes.Get(), mode.Get())

	saveCallback := ui.UseCallback(func() {
		savedCount.Update(func(v int) int { return v + 1 })
		notice.Set(Notice{
			Title: "Playground",
			Body:  "Saved session " + sessionCode.Get() + " with mode " + strings.ToUpper(mode.Get()) + ".",
			Level: "warn",
		})
	}, title.Get(), notes.Get(), mode.Get(), savedCount.Get())

	save := ui.UseEvent(saveCallback)
	reset := ui.UseEvent(func() {
		title.Set("")
		notes.Set("")
		mode.Set("agent")
	})

	updateTitle := ui.UseEvent(func(event ui.InputEvent) {
		title.Set(event.GetValue())
	})
	updateNotes := ui.UseEvent(func(event ui.InputEvent) {
		notes.Set(event.GetValue())
	})
	updateMode := ui.UseEvent(func(event ui.ChangeEvent) {
		mode.Set(event.GetValue())
	})

	ui.UseEffect(func() func() {
		storage := js.Global().Get("localStorage")
		if storage.Truthy() {
			storage.Call("setItem", "omi-draft", preview)
		}
		return nil
	}, preview)

	return html.Div(html.Props{
		Class: "grid gap-8 lg:grid-cols-[1.15fr_0.85fr]",
	},
		html.Section(html.Props{
			Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8",
		},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Playground")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Forms, IDs, callbacks, refs, and effect persistence.")),
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
							For:   titleID,
							Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400",
						}, html.Text("Title")),
						html.Input(html.Props{
							ID:          titleID,
							Value:       title.Get(),
							OnInput:     updateTitle,
							Placeholder: "Agent coders and durable UI APIs",
							Class:       "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:outline-none",
						}),
					),
					html.Div(html.Props{},
						html.Label(html.Props{
							For:   modeID,
							Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400",
						}, html.Text("Mode")),
						html.Select(html.Props{
							ID:       modeID,
							Value:    mode.Get(),
							OnChange: updateMode,
							Class:    "w-full rounded-2xl border border-slate-700/80 bg-slate-950/70 px-4 py-3 text-slate-100 focus:outline-none",
						},
							html.Option(html.Props{Value: "agent"}, html.Text("Agent")),
							html.Option(html.Props{Value: "review"}, html.Text("Review")),
							html.Option(html.Props{Value: "benchmark"}, html.Text("Benchmark")),
						),
					),
					html.Div(html.Props{},
						html.Label(html.Props{
							For:   notesID,
							Class: "mb-2 block text-xs uppercase tracking-[0.25em] text-slate-400",
						}, html.Text("Notes")),
						html.Textarea(html.Props{
							ID:          notesID,
							Value:       notes.Get(),
							OnInput:     updateNotes,
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
						OnClick: save,
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
				}, html.Text(preview)),
			),
			html.Div(html.Props{
				Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/82 p-6",
			},
				html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Session ref")),
				html.H3(html.Props{Class: "mt-3 text-2xl font-black"}, html.Text(sessionCode.Get())),
				html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(fmt.Sprintf("Saved %d times in this browser session.", savedCount.Get()))),
				func() ui.Node {
					if notice.Get().Title == "" {
						return nil
					}
					return html.P(html.Props{
						Class: "mt-4 rounded-2xl border border-slate-700/80 bg-slate-950/65 px-4 py-3 text-sm text-slate-200",
					}, html.Text("Latest global notice: "+notice.Get().Body))
				}(),
			),
		),
	)
}
