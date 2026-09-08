//go:build !js || !wasm

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/monstercameron/GoWebComponents/v6/ui"
	"strings"
	"sync"
	"time"
)

type serverInteractiveHub struct {
	applyMutex   sync.Mutex
	storeState   serverInteractiveState
	storeClients map[chan []byte]struct{}
}

// buildServerInteractiveState builds the initial dashboard state for the experiment.
func buildServerInteractiveState() serverInteractiveState {
	return serverInteractiveState{
		Version:          1,
		ActiveUsers:      19,
		PendingApprovals: 7,
		IncidentsToday:   2,
		RecentEvents: []string{
			"Initial dashboard snapshot seeded by server.",
		},
		UpdatedAt: time.Now().UTC(),
	}
}

// buildServerInteractiveHub builds shared state and fan-out channels for all connected clients.
func buildServerInteractiveHub() *serverInteractiveHub {
	return &serverInteractiveHub{
		storeState:   buildServerInteractiveState(),
		storeClients: map[chan []byte]struct{}{},
	}
}

// renderServerInteractiveHTML renders the shared GWC view for an SSE compatibility snapshot.
func renderServerInteractiveHTML(parseState serverInteractiveState) (string, error) {
	return ui.RenderToString(renderServerInteractiveView(serverInteractiveView{State: parseState, IsBusy: true, Status: "Connecting to server stream..."}))
}

// buildServerInteractiveSnapshot builds one JSON payload for SSE delivery.
func (buildHub *serverInteractiveHub) buildServerInteractiveSnapshot() ([]byte, error) {
	buildHub.applyMutex.Lock()
	buildState := buildHub.storeState
	buildHub.applyMutex.Unlock()
	parseMarkup, parseErr := renderServerInteractiveHTML(buildState)
	if parseErr != nil {
		return nil, parseErr
	}
	buildPayload := serverInteractiveSnapshot{
		HTML:      parseMarkup,
		State:     buildState,
		Version:   buildState.Version,
		UpdatedAt: buildState.UpdatedAt.Format(time.RFC3339),
	}
	return json.Marshal(buildPayload)
}

// storeServerInteractiveClient registers one SSE client channel.
func (storeHub *serverInteractiveHub) storeServerInteractiveClient(storeClient chan []byte) {
	storeHub.applyMutex.Lock()
	storeHub.storeClients[storeClient] = struct{}{}
	storeHub.applyMutex.Unlock()
}

// clearServerInteractiveClient unregisters one SSE client channel.
func (clearHub *serverInteractiveHub) clearServerInteractiveClient(clearClient chan []byte) {
	clearHub.applyMutex.Lock()
	defer clearHub.applyMutex.Unlock()
	delete(clearHub.storeClients, clearClient)
	close(clearClient)
}

// applyServerInteractiveAction mutates server-owned state and returns the generated event line.
func (applyHub *serverInteractiveHub) applyServerInteractiveAction(applyAction serverInteractiveAction) string {
	applyHub.applyMutex.Lock()
	defer applyHub.applyMutex.Unlock()
	applyEvent := ""
	switch applyAction.Action {
	case "add-user":
		applyHub.storeState.ActiveUsers++
		applyEvent = "Operator added one active user."
	case "resolve-approval":
		if applyHub.storeState.PendingApprovals > 0 {
			applyHub.storeState.PendingApprovals--
		}
		applyEvent = "Operator resolved one pending approval."
	case "add-incident":
		applyHub.storeState.IncidentsToday++
		applyEvent = "Operator logged one additional incident."
	case "clear-incidents":
		applyHub.storeState.IncidentsToday = 0
		applyEvent = "Operator cleared the incident queue."
	default:
		applyEvent = "Operator sent an unknown action."
	}
	applyHub.storeState.Version++
	applyHub.storeState.UpdatedAt = time.Now().UTC()
	applyHub.storeState.RecentEvents = append([]string{applyEvent}, applyHub.storeState.RecentEvents...)
	if len(applyHub.storeState.RecentEvents) > 6 {
		applyHub.storeState.RecentEvents = append([]string{}, applyHub.storeState.RecentEvents[:6]...)
	}
	return applyEvent
}

// handleServerInteractiveIndex serves the server-interactive HTML shell.
func (handleHub *serverInteractiveHub) handleServerInteractiveIndex(handleWriter http.ResponseWriter, handleRequest *http.Request) {
	if handleRequest.Method != http.MethodGet {
		handleWriter.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if handleRequest.URL.Path != "/" {
		http.NotFound(handleWriter, handleRequest)
		return
	}
	parseMarkup, parseErr := renderServerInteractiveHTML(serverInteractiveState{})
	if parseErr != nil {
		http.Error(handleWriter, parseErr.Error(), http.StatusInternalServerError)
		return
	}
	handleWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	handleWriter.WriteHeader(http.StatusOK)
	_, _ = handleWriter.Write([]byte(strings.Replace(serverInteractiveShellHTML, "<!--gwc-view-->", parseMarkup, 1)))
}

// handleServerInteractiveEvents serves SSE snapshots whenever server-owned state changes.
func (handleHub *serverInteractiveHub) handleServerInteractiveEvents(handleWriter http.ResponseWriter, handleRequest *http.Request) {
	if handleRequest.Method != http.MethodGet {
		handleWriter.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	handleFlusher, hasFlusher := handleWriter.(http.Flusher)
	if !hasFlusher {
		handleWriter.WriteHeader(http.StatusInternalServerError)
		return
	}
	handleWriter.Header().Set("Content-Type", "text/event-stream")
	handleWriter.Header().Set("Cache-Control", "no-store")
	handleWriter.Header().Set("Connection", "keep-alive")

	handleClient := make(chan []byte, 8)
	handleHub.storeServerInteractiveClient(handleClient)
	defer handleHub.clearServerInteractiveClient(handleClient)

	handleSnapshot, handleSnapshotErr := handleHub.buildServerInteractiveSnapshot()
	if handleSnapshotErr == nil {
		fmt.Fprintf(handleWriter, "data: %s\n\n", handleSnapshot)
		handleFlusher.Flush()
	}

	handleContext := handleRequest.Context()
	for {
		select {
		case <-handleContext.Done():
			return
		case handlePayload, hasPayload := <-handleClient:
			if !hasPayload {
				return
			}
			fmt.Fprintf(handleWriter, "data: %s\n\n", handlePayload)
			handleFlusher.Flush()
		}
	}
}

// handleServerInteractiveAction accepts operator actions and broadcasts the new snapshot.
func (handleHub *serverInteractiveHub) handleServerInteractiveAction(handleWriter http.ResponseWriter, handleRequest *http.Request) {
	if handleRequest.Method != http.MethodPost {
		handleWriter.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var handleAction serverInteractiveAction
	if handleDecodeErr := json.NewDecoder(handleRequest.Body).Decode(&handleAction); handleDecodeErr != nil {
		http.Error(handleWriter, "invalid action payload", http.StatusBadRequest)
		return
	}
	handleHub.applyServerInteractiveAction(handleAction)
	handleSnapshot, handleSnapshotErr := handleHub.buildServerInteractiveSnapshot()
	if handleSnapshotErr == nil {
		handleHub.broadcastServerInteractiveSnapshot(handleSnapshot)
	}
	handleWriter.Header().Set("Content-Type", "application/json")
	handleWriter.WriteHeader(http.StatusAccepted)
	_, _ = handleWriter.Write([]byte(`{"ok":true}`))
}

// broadcastServerInteractiveSnapshot fan-outs one snapshot to connected SSE clients.
func (parseBroadcastHub *serverInteractiveHub) broadcastServerInteractiveSnapshot(parseBroadcastPayload []byte) {
	parseBroadcastHub.applyMutex.Lock()
	defer parseBroadcastHub.applyMutex.Unlock()
	// Hold the registration lock through nonblocking sends so disconnect cannot
	// close a channel between selection and delivery.
	for parseBroadcastClient := range parseBroadcastHub.storeClients {
		select {
		case parseBroadcastClient <- parseBroadcastPayload:
		default:
		}
	}
}

// main serves the native state owner and the GWC Wasm client.
func main() {
	parseMainHub := buildServerInteractiveHub()
	http.HandleFunc("/", parseMainHub.handleServerInteractiveIndex)
	http.HandleFunc("/client.wasm", func(parseWriter http.ResponseWriter, parseRequest *http.Request) {
		parsePath := os.Getenv("GWC_INTERACTIVE_WASM")
		if parsePath == "" {
			parsePath = "examples/server/server-interactive-proof-of-concept/client.wasm"
		}
		parseWriter.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(parseWriter, parseRequest, parsePath)
	})
	http.HandleFunc("/wasm_exec.js", func(parseWriter http.ResponseWriter, parseRequest *http.Request) {
		http.ServeFile(parseWriter, parseRequest, filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js"))
	})
	http.HandleFunc("/events", parseMainHub.handleServerInteractiveEvents)
	http.HandleFunc("/action", parseMainHub.handleServerInteractiveAction)
	parseAddress := os.Getenv("GWC_INTERACTIVE_ADDRESS")
	if parseAddress == "" {
		parseAddress = "127.0.0.1:8180"
	}
	log.Printf("server-interactive POC: http://%s", parseAddress)
	if parseRunErr := http.ListenAndServe(parseAddress, nil); parseRunErr != nil {
		log.Fatal(parseRunErr)
	}
}

const serverInteractiveShellHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Server-Interactive POC</title>
  <style>
    :root {
      --bg: #081025;
      --panel: #0f1d3d;
      --ink: #f8fbff;
      --muted: #8ca2c9;
      --accent: #5dd6ff;
      --border: rgba(255, 255, 255, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Segoe UI", "Helvetica Neue", sans-serif;
      background: radial-gradient(circle at 20% 20%, #0e2d64 0%, var(--bg) 60%);
      color: var(--ink);
      min-height: 100vh;
      padding: 28px;
    }
    h1 {
      margin: 0 0 10px;
      font-size: clamp(1.9rem, 3vw, 2.5rem);
      letter-spacing: -0.03em;
    }
    .subtitle {
      margin: 0 0 24px;
      color: var(--muted);
      max-width: 70ch;
      line-height: 1.5;
    }
    .layout {
      display: grid;
      gap: 16px;
    }
    .panel-grid {
      display: grid;
      gap: 12px;
      grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    }
    .metric, .action-panel, .events-panel {
      border: 1px solid var(--border);
      border-radius: 14px;
      background: color-mix(in srgb, var(--panel) 84%, black 16%);
      padding: 14px 16px;
    }
    .metric-label {
      margin: 0;
      color: var(--muted);
      font-size: 0.78rem;
      text-transform: uppercase;
      letter-spacing: 0.2em;
    }
    .metric-value {
      margin: 10px 0 2px;
      font-size: clamp(1.8rem, 3vw, 2.3rem);
      font-weight: 700;
    }
    .actions {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
      gap: 10px;
      margin-top: 10px;
    }
    button {
      border: 1px solid color-mix(in srgb, var(--accent) 55%, white 45%);
      border-radius: 999px;
      background: rgba(93, 214, 255, 0.16);
      color: var(--ink);
      font: inherit;
      padding: 10px 14px;
      cursor: pointer;
      transition: transform 120ms ease, background 120ms ease;
    }
    button:hover {
      background: rgba(93, 214, 255, 0.24);
      transform: translateY(-1px);
    }
    ul {
      margin: 8px 0 0;
      padding-left: 20px;
      display: grid;
      gap: 8px;
    }
    .meta {
      margin-top: 10px;
      color: var(--muted);
      font-size: 0.85rem;
    }
    .status {
      margin-top: 12px;
      color: var(--muted);
      font-size: 0.85rem;
    }
  </style>
</head>
<body>
 <div id="app"><!--gwc-view--></div>
 <script src="/wasm_exec.js"></script>
 <script>
  const go = new Go();
  WebAssembly.instantiateStreaming(fetch('/client.wasm'), go.importObject)
    .then(result => go.run(result.instance)).catch(error => console.error('GWC startup failed', error));
 </script>
</body>
</html>`
