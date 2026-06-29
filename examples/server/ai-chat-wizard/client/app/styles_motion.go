//go:build js && wasm

package app

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/anim"
)

// chatWizardMotionStyles is generated once at init by sampling the anim
// package's physical spring integrator into CSS keyframes: the entry and
// press micro-interactions carry real spring character (overshoot, settle)
// without running a per-frame loop inside wasm. The keyframes bake the curve,
// so the animations use linear timing.
var chatWizardMotionStyles = buildSpringMotionStyles()

// buildSpringKeyframes samples one 0->1 spring into a named @keyframes block.
// parseFrame maps the spring position at each sample to a CSS declaration.
func buildSpringKeyframes(parseName string, parseConfig anim.SpringConfig, parseDurationSeconds float64, parseFrame func(parsePosition float64) string) string {
	parseSpring := anim.NewSpring(parseConfig, 0)
	parseSpring.SetTarget(1)
	const parseSamples = 16
	parseDt := parseDurationSeconds / float64(parseSamples)
	parseBuilder := strings.Builder{}
	fmt.Fprintf(&parseBuilder, "@keyframes %s {\n", parseName)
	for parseI := 0; parseI <= parseSamples; parseI++ {
		parsePosition := parseSpring.Position()
		if parseI == parseSamples {
			// Land exactly on the target so the final frame never drifts.
			parsePosition = 1
		}
		fmt.Fprintf(&parseBuilder, "  %.2f%% { %s }\n", float64(parseI)/float64(parseSamples)*100, parseFrame(parsePosition))
		parseSpring.Step(parseDt)
	}
	parseBuilder.WriteString("}\n")
	return parseBuilder.String()
}

func clampUnit(parseValue float64) float64 {
	if parseValue < 0 {
		return 0
	}
	if parseValue > 1 {
		return 1
	}
	return parseValue
}

func buildSpringMotionStyles() string {
	parseBuilder := strings.Builder{}

	// Message entry: gentle spring with a soft rise; opacity leads position.
	parseBuilder.WriteString(buildSpringKeyframes("spring-bubble-in", anim.GentleSpring(), 0.34, func(parsePosition float64) string {
		return fmt.Sprintf("opacity: %.3f; transform: translateY(%.2fpx) scale(%.4f);",
			clampUnit(parsePosition*1.35), (1-parsePosition)*10, 0.97+0.03*parsePosition)
	}))

	// Press feedback: wobbly spring so the send button visibly rings once.
	parseBuilder.WriteString(buildSpringKeyframes("spring-pop", anim.WobblySpring(), 0.5, func(parsePosition float64) string {
		return fmt.Sprintf("transform: scale(%.4f);", 0.9+0.1*parsePosition)
	}))

	// Conversation rows slide in on a gentle spring.
	parseBuilder.WriteString(buildSpringKeyframes("spring-row-in", anim.GentleSpring(), 0.3, func(parsePosition float64) string {
		return fmt.Sprintf("opacity: %.3f; transform: translateX(%.2fpx);",
			clampUnit(parsePosition*1.35), (parsePosition-1)*12)
	}))

	parseBuilder.WriteString(`
/* The composer wrap draws its own focus-within ring; the inner textarea must
   not double up with the global :focus-visible outline. */
#chat-input:focus, #chat-input:focus-visible { outline: none; }

.msg-bubble { animation: spring-bubble-in 340ms linear both; }
.conv-row { animation: spring-row-in 300ms linear both; }
#send-btn:not(:disabled):active { animation: spring-pop 500ms linear; }

/* Persisted UI density (ui.UsePersistedState): compact tightens the reading
   surfaces without touching chrome. */
.density-compact .msg-bubble-assistant,
.density-compact .msg-bubble-user {
  font-size: 0.9375rem;
  line-height: 1.6;
  padding: 0.625rem 1rem;
}
.density-compact #chat-input { font-size: 0.9375rem; }
`)
	return parseBuilder.String()
}
