//go:build js && wasm

package main

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"

	. "github.com/monstercameron/GoWebComponents/v4/css/u"
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

// Typed-CSS palette for this demo. Tints/cyan aren't in the curated token set, so
// they're expressed with the typed Hex/RGBA value constructors.
var (
	cyan100   = Hex("#cffafe")
	white05   = RGBA(255, 255, 255, 0.05)
	white08   = RGBA(255, 255, 255, 0.08)
	white10   = RGBA(255, 255, 255, 0.10)
	cyanTint  = RGBA(34, 211, 238, 0.10)
	cyanEdge  = RGBA(34, 211, 238, 0.20)
	cyanField = RGBA(34, 211, 238, 0.15)
)

// buttonBase is the shared, hoisted button style (folded once).
var buttonBase = Rules(
	InlineFlex, ItemsCenter, JustifyCenter,
	H(Spacing(14)), W(Spacing(14)),
	Rounded(RadiusXl), FontSize(Rem(1.25)), FontSemibold,
	Cursor.Pointer,
	Transition(PropAll, Ms(200), Ease),
	Active(Transform(Scale(0.95))),
	Hover(Transform(TranslateY(RawLength("-2px")))),
)

// SQLitePersistenceExample is a counter whose value lives in a client-side
// SQLite database (db/sqlite) backed by IndexedDB. Reload the page and the
// count is still there: the DB image was snapshotted to IndexedDB on Flush and
// rehydrated on Open. No server, no cgo.
func SQLitePersistenceExample() ui.Node {
	parseCount := ui.UseState(0)
	parseReady := ui.UseState(false)
	parseDBRef := ui.UseRef[*sqlite.DB](nil)

	ui.UseEffect(func() func() {
		go func() {
			parseCtx := context.Background()
			parseDB, parseErr := sqlite.Open(parseCtx, sqlite.Options{
				Name:        "counter-demo",
				Persistence: sqlite.IndexedDB,
			})
			if parseErr != nil {
				return
			}
			if _, parseErr := parseDB.Exec(parseCtx, `CREATE TABLE IF NOT EXISTS counter(id INTEGER PRIMARY KEY CHECK (id = 1), n INTEGER NOT NULL)`); parseErr != nil {
				return
			}
			parseDB.Exec(parseCtx, `INSERT OR IGNORE INTO counter(id, n) VALUES(1, 0)`)

			var parseN int
			parseDB.QueryRow(parseCtx, `SELECT n FROM counter WHERE id = 1`).Scan(&parseN)

			parseDBRef.Set(parseDB)
			parseCount.Set(parseN)
			parseReady.Set(true)
		}()
		return func() {
			if parseDB := parseDBRef.Get(); parseDB != nil {
				parseDB.Close()
			}
		}
	})

	parsePersist := func(parseN int) {
		go func() {
			parseDB := parseDBRef.Get()
			if parseDB == nil {
				return
			}
			parseCtx := context.Background()
			parseDB.Exec(parseCtx, `UPDATE counter SET n = ? WHERE id = 1`, parseN)
			parseDB.Flush(parseCtx)
		}()
	}

	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev int) int {
			parseNext := parsePrev + 1
			parsePersist(parseNext)
			return parseNext
		})
	})
	parseDecrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev int) int {
			parseNext := parsePrev - 1
			parsePersist(parseNext)
			return parseNext
		})
	})
	parseReset := ui.UseEvent(func() {
		parseCount.Set(0)
		parsePersist(0)
	})

	parseStatus := "Loading from SQLite…"
	if parseReady.Get() {
		parseStatus = "Persisted in client-side SQLite (IndexedDB) — reload to verify"
	}

	return Div(Class(MaxWidth(Px(448)), Raw("margin-left", "auto"), Raw("margin-right", "auto"), Pad(Spacing6)),
		Div(Class(
			Raw("border-radius", "28px"), Border(white10), Bg(RGBA(15, 23, 42, 0.60)),
			Pad(Spacing6), Shadow(Shadow2xl),
		),
			Div(Class(Raw("border-bottom", "1px solid "+string(white10)), Raw("padding-bottom", "1rem")),
				H2(Class(FontSize(Rem(1.875)), FontSemibold, Fg(White)), Text("SQLite Persistence")),
				P(Class(Raw("margin-top", "0.5rem"), MaxWidth(Px(448)), TextSize(TextSm), Fg(Slate300)),
					Text("This counter is stored in a pure-Go SQLite database running in the browser. No server, no cgo.")),
			),
			Div(Class(Raw("margin-top", "1.25rem")),
				Div(Class(Raw("border-radius", "22px"), Border(cyanEdge), Bg(cyanTint), Pad(Spacing5)),
					Div(Class(FontSize(Rem(3.75)), FontSemibold, Fg(White), Raw("font-family", "ui-monospace, monospace")),
						Textf("%d", parseCount.Get())),
					P(Class(Raw("margin-top", "0.5rem"), TextSize(TextXs), TextTransform.Uppercase, Tracking(Ems(0.16)), Fg(cyan100)),
						Text(parseStatus)),
					Div(Class(Raw("margin-top", "1.25rem"), Flex, Raw("flex-wrap", "wrap"), Gap(Spacing2)),
						Button(FromProps(Props{OnClick: parseDecrement}),
							Class(buttonBase, Border(white10), Bg(white05), Fg(Slate100), Hover(Bg(white08))), Text("-")),
						Button(FromProps(Props{OnClick: parseIncrement}),
							Class(buttonBase, Border(cyanField), Bg(cyanField), Fg(cyan100), Hover(Bg(cyanEdge))), Text("+")),
						Button(FromProps(Props{OnClick: parseReset}),
							Class(
								InlineFlex, ItemsCenter, JustifyCenter, H(Spacing(14)), PadX(Spacing5),
								Raw("border-radius", "16px"), TextSize(TextXs), FontSemibold,
								TextTransform.Uppercase, Tracking(Ems(0.18)),
								Border(white10), Bg(white05), Fg(Slate200), Cursor.Pointer,
								Transition(PropAll, Ms(200), Ease), Active(Transform(Scale(0.95))),
								Hover(Bg(white08)),
							), Text("Reset")),
					),
				),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(SQLitePersistenceExample))
	exampleboot.WaitExampleRuntime()
}
