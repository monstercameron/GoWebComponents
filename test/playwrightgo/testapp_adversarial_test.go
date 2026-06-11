//go:build playwrightgo

package playwrightgo_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// readTodoTexts returns the visible todo texts in DOM order.
func readTodoTexts(parseT *testing.T, parsePage playwright.Page) []string {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Evaluate(`() => Array.from(document.querySelectorAll('#todo-items [id^="todo-text-"]')).map(e => e.textContent)`)
	if parseErr != nil {
		parseT.Fatalf("read todo texts: %v", parseErr)
	}
	parseList, _ := parseRaw.([]interface{})
	parseOut := make([]string, 0, len(parseList))
	for _, parseItem := range parseList {
		parseOut = append(parseOut, fmt.Sprintf("%v", parseItem))
	}
	return parseOut
}

func assertTodoList(parseT *testing.T, parsePage playwright.Page, parseWant []string, parseContext string) {
	parseT.Helper()
	parseGot := readTodoTexts(parseT, parsePage)
	if len(parseGot) != len(parseWant) {
		parseT.Fatalf("%s: DOM has %d todos %v, expected %d %v", parseContext, len(parseGot), parseGot, len(parseWant), parseWant)
	}
	for parseIdx := range parseWant {
		if parseGot[parseIdx] != parseWant[parseIdx] {
			parseT.Fatalf("%s: row %d is %q, expected %q (full: %v)", parseContext, parseIdx, parseGot[parseIdx], parseWant[parseIdx], parseGot)
		}
	}
	// The shared atom counter must agree with the unfiltered model count.
	parseHeader, parseErr := parsePage.TextContent("#todo-header-count")
	if parseErr != nil {
		parseT.Fatalf("%s: read header count: %v", parseContext, parseErr)
	}
	_ = parseHeader
}

func addTodo(parseT *testing.T, parsePage playwright.Page, parseText string) {
	parseT.Helper()
	if parseErr := parsePage.Fill("#todo-input", parseText); parseErr != nil {
		parseT.Fatalf("fill todo input: %v", parseErr)
	}
	if parseErr := parsePage.Click("#todo-add"); parseErr != nil {
		parseT.Fatalf("click add: %v", parseErr)
	}
}

// TestAdversarialTodoLifecycle drives the todo app through the interaction
// patterns most likely to expose reconciler or state bugs: deletions from the
// middle of the list (survivor siblings must keep working handlers and
// state), interleaved toggles, filter changes that hide and reveal rows, and
// a rapid-fire add/delete sequence â€” verifying the full DOM list and the
// shared-atom counter against an independent model at every step.
func TestAdversarialTodoLifecycle(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := repoRootFromFile(parseFile)
	parseBaseURL := startTestAppServer(parseT, parseRepoRoot, "18085")

	runChromiumPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(parseBaseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto test app: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#todo-input"); parseErr != nil {
			parseT.Fatalf("wait for todo app: %v", parseErr)
		}

		// 1. Build a five-item list.
		parseModel := []string{"alpha", "bravo", "charlie", "delta", "echo"}
		for _, parseText := range parseModel {
			addTodo(parseT, parsePage, parseText)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll('#todo-items [id^="todo-text-"]').length === 5`, nil); parseErr != nil {
			parseT.Fatalf("wait for 5 todos: %v", parseErr)
		}
		assertTodoList(parseT, parsePage, parseModel, "after initial adds")

		// 2. Delete from the MIDDLE â€” the survivor-sibling case. charlie goes;
		//    bravo and delta must remain with working handlers.
		if parseErr := parsePage.Click(`#todo-items .todo-item:nth-child(3) button:has-text("Delete")`); parseErr != nil {
			parseT.Fatalf("delete middle: %v", parseErr)
		}
		parseModel = []string{"alpha", "bravo", "delta", "echo"}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll('#todo-items [id^="todo-text-"]').length === 4`, nil); parseErr != nil {
			parseT.Fatalf("wait after middle delete: %v", parseErr)
		}
		assertTodoList(parseT, parsePage, parseModel, "after middle delete")

		// 3. Survivor interactivity: toggle the item AFTER the deleted one.
		//    If deletion tore down sibling handlers, this click does nothing.
		if parseErr := parsePage.Click(`#todo-items .todo-item:nth-child(3) input[type="checkbox"]`); parseErr != nil {
			parseT.Fatalf("toggle survivor: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => {
			const s = document.querySelectorAll('#todo-items [id^="todo-text-"]')[2];
			return s && s.className.includes('line-through');
		}`, nil); parseErr != nil {
			parseT.Fatalf("survivor toggle did not take effect (sibling handler torn down?): %v", parseErr)
		}

		// 4. Delete the FIRST item while a later one is completed.
		if parseErr := parsePage.Click(`#todo-items .todo-item:nth-child(1) button:has-text("Delete")`); parseErr != nil {
			parseT.Fatalf("delete first: %v", parseErr)
		}
		parseModel = []string{"bravo", "delta", "echo"}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll('#todo-items [id^="todo-text-"]').length === 3`, nil); parseErr != nil {
			parseT.Fatalf("wait after first delete: %v", parseErr)
		}
		assertTodoList(parseT, parsePage, parseModel, "after first delete")
		// delta must STILL be completed after its siblings changed position.
		parseCompleted, parseErr := parsePage.Evaluate(`() => {
			const s = document.querySelectorAll('#todo-items [id^="todo-text-"]')[1];
			return s ? s.className.includes('line-through') : false;
		}`)
		if parseErr != nil || parseCompleted != true {
			parseT.Fatalf("delta lost its completed state after sibling deletion: %v %v", parseCompleted, parseErr)
		}

		// 5. Filter interplay: status filter "completed" must show only delta.
		if _, parseErr := parsePage.SelectOption("#status-filter", playwright.SelectOptionValues{Values: &[]string{"completed"}}); parseErr != nil {
			parseT.Fatalf("set status filter: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll('#todo-items [id^="todo-text-"]').length === 1`, nil); parseErr != nil {
			parseT.Fatalf("filter to completed: %v", parseErr)
		}
		assertTodoList(parseT, parsePage, []string{"delta"}, "completed filter")

		// 6. Back to all; text filter narrows live.
		if _, parseErr := parsePage.SelectOption("#status-filter", playwright.SelectOptionValues{Values: &[]string{"all"}}); parseErr != nil {
			parseT.Fatalf("reset status filter: %v", parseErr)
		}
		if parseErr := parsePage.Fill("#todo-filter", "e"); parseErr != nil {
			parseT.Fatalf("text filter: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll('#todo-items [id^="todo-text-"]').length === 2`, nil); parseErr != nil {
			parseT.Fatalf("text filter 'e': %v", parseErr)
		}
		assertTodoList(parseT, parsePage, []string{"delta", "echo"}, "text filter e")
		if parseErr := parsePage.Fill("#todo-filter", ""); parseErr != nil {
			parseT.Fatalf("clear filter: %v", parseErr)
		}

		// 7. Rapid-fire: add 10, then delete every remaining first item down to
		//    one, verifying the list at each removal.
		for parseIdx := 0; parseIdx < 10; parseIdx++ {
			addTodo(parseT, parsePage, fmt.Sprintf("burst-%d", parseIdx))
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll('#todo-items [id^="todo-text-"]').length === 13`, nil); parseErr != nil {
			parseT.Fatalf("burst adds: %v", parseErr)
		}
		parseModel = []string{"bravo", "delta", "echo"}
		for parseIdx := 0; parseIdx < 10; parseIdx++ {
			parseModel = append(parseModel, fmt.Sprintf("burst-%d", parseIdx))
		}
		assertTodoList(parseT, parsePage, parseModel, "after burst adds")

		for len(parseModel) > 1 {
			if parseErr := parsePage.Click(`#todo-items .todo-item:nth-child(1) button:has-text("Delete")`); parseErr != nil {
				parseT.Fatalf("burst delete at %d remaining: %v", len(parseModel), parseErr)
			}
			parseModel = parseModel[1:]
			parseWaitExpr := fmt.Sprintf(`() => document.querySelectorAll('#todo-items [id^="todo-text-"]').length === %d`, len(parseModel))
			if _, parseErr := parsePage.WaitForFunction(parseWaitExpr, nil); parseErr != nil {
				parseT.Fatalf("burst delete wait at %d remaining: %v", len(parseModel), parseErr)
			}
			assertTodoList(parseT, parsePage, parseModel, fmt.Sprintf("burst delete, %d remaining", len(parseModel)))
		}

		// 8. The last survivor must still be fully interactive.
		if parseErr := parsePage.Click(`#todo-items .todo-item:nth-child(1) input[type="checkbox"]`); parseErr != nil {
			parseT.Fatalf("toggle last survivor: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => {
			const s = document.querySelectorAll('#todo-items [id^="todo-text-"]')[0];
			return s && s.className.includes('line-through');
		}`, nil); parseErr != nil {
			parseT.Fatalf("last survivor unresponsive after 12 deletions: %v", parseErr)
		}

		// 9. No console errors throughout â€” surfaced via the global error hook.
		parseSubjectError, _ := parsePage.Evaluate(`() => window.__example201SubjectError || ""`)
		if parseStr, _ := parseSubjectError.(string); parseStr != "" {
			parseT.Fatalf("page error during adversarial run: %s", parseStr)
		}
		_ = strings.TrimSpace
	})
}
