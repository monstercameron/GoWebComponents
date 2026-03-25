package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type serverInteractiveState struct {
	Version          int
	ActiveUsers      int
	PendingApprovals int
	IncidentsToday   int
	RecentEvents     []string
	UpdatedAt        time.Time
}

type serverInteractiveAction struct {
	Action string `json:"action"`
}

type serverInteractiveSnapshot struct {
	HTML      string `json:"html"`
	Version   int    `json:"version"`
	UpdatedAt string `json:"updatedAt"`
}

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

// renderServerInteractiveHTML renders the server-owned dashboard surface for one state snapshot.
func renderServerInteractiveHTML(renderState serverInteractiveState) string {
	var renderBuilder strings.Builder
	renderBuilder.WriteString("<section class=\"panel-grid\">")
	renderBuilder.WriteString(renderServerInteractiveMetric("Active Users", fmt.Sprintf("%d", renderState.ActiveUsers)))
	renderBuilder.WriteString(renderServerInteractiveMetric("Pending Approvals", fmt.Sprintf("%d", renderState.PendingApprovals)))
	renderBuilder.WriteString(renderServerInteractiveMetric("Incidents Today", fmt.Sprintf("%d", renderState.IncidentsToday)))
	renderBuilder.WriteString("</section>")
	renderBuilder.WriteString("<section class=\"action-panel\">")
	renderBuilder.WriteString("<h2>Operator Actions</h2>")
	renderBuilder.WriteString("<div class=\"actions\">")
	renderBuilder.WriteString("<button data-action=\"add-user\">Add User</button>")
	renderBuilder.WriteString("<button data-action=\"resolve-approval\">Resolve Approval</button>")
	renderBuilder.WriteString("<button data-action=\"add-incident\">Add Incident</button>")
	renderBuilder.WriteString("<button data-action=\"clear-incidents\">Clear Incidents</button>")
	renderBuilder.WriteString("</div>")
	renderBuilder.WriteString("</section>")
	renderBuilder.WriteString("<section class=\"events-panel\">")
	renderBuilder.WriteString("<h2>Recent Server Events</h2>")
	renderBuilder.WriteString("<ul>")
	for _, renderEvent := range renderState.RecentEvents {
		renderBuilder.WriteString("<li>")
		renderBuilder.WriteString(renderEvent)
		renderBuilder.WriteString("</li>")
	}
	renderBuilder.WriteString("</ul>")
	renderBuilder.WriteString(fmt.Sprintf("<p class=\"meta\">Version %d · Updated %s UTC</p>", renderState.Version, renderState.UpdatedAt.Format("15:04:05")))
	renderBuilder.WriteString("</section>")
	return renderBuilder.String()
}

// renderServerInteractiveMetric renders one metric tile.
func renderServerInteractiveMetric(renderLabel string, renderValue string) string {
	return fmt.Sprintf(
		"<article class=\"metric\"><p class=\"metric-label\">%s</p><p class=\"metric-value\">%s</p></article>",
		renderLabel,
		renderValue,
	)
}

// buildServerInteractiveSnapshot builds one JSON payload for SSE delivery.
func (buildHub *serverInteractiveHub) buildServerInteractiveSnapshot() ([]byte, error) {
	buildHub.applyMutex.Lock()
	buildState := buildHub.storeState
	buildHub.applyMutex.Unlock()
	buildPayload := serverInteractiveSnapshot{
		HTML:      renderServerInteractiveHTML(buildState),
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
	delete(clearHub.storeClients, clearClient)
	clearHub.applyMutex.Unlock()
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
	handleWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	handleWriter.WriteHeader(http.StatusOK)
	_, _ = handleWriter.Write([]byte(serverInteractiveShellHTML))
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
	parseBroadcastClients := make([]chan []byte, 0, len(parseBroadcastHub.storeClients))
	for parseBroadcastClient := range parseBroadcastHub.storeClients {
		parseBroadcastClients = append(parseBroadcastClients, parseBroadcastClient)
	}
	parseBroadcastHub.applyMutex.Unlock()
	for _, parseBroadcastClient2 := range parseBroadcastClients {
		select {
		case parseBroadcastClient2 <- parseBroadcastPayload:
		default:
		}
	}
}

func main() {
	parseMainHub := buildServerInteractiveHub()
	http.HandleFunc("/", parseMainHub.handleServerInteractiveIndex)
	http.HandleFunc("/events", parseMainHub.handleServerInteractiveEvents)
	http.HandleFunc("/action", parseMainHub.handleServerInteractiveAction)
	log.Println("server-interactive POC: http://127.0.0.1:8180")
	if parseRunErr := http.ListenAndServe("127.0.0.1:8180", nil); parseRunErr != nil {
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
  <h1>Server-Interactive Dashboard POC</h1>
  <p class="subtitle">
    This narrow experiment keeps state and rendering authority on the server.
    Browser clients only send actions and apply streamed server snapshots.
  </p>
  <div id="app" class="layout"></div>
  <p id="status" class="status">Connecting to server stream...</p>

  <script>
    const appEl = document.getElementById('app');
    const statusEl = document.getElementById('status');

    function bindActions() {
      const buttons = appEl.querySelectorAll('button[data-action]');
      buttons.forEach((button) => {
        button.addEventListener('click', async () => {
          const action = button.getAttribute('data-action');
          await fetch('/action', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ action }),
          });
        });
      });
    }

    const stream = new EventSource('/events');
    stream.onopen = () => {
      statusEl.textContent = 'Connected. Waiting for server-driven snapshots.';
    };
    stream.onmessage = (event) => {
      const payload = JSON.parse(event.data);
      appEl.innerHTML = payload.html;
      bindActions();
      statusEl.textContent = 'Snapshot v' + payload.version + ' applied at ' + payload.updatedAt + '.';
    };
    stream.onerror = () => {
      statusEl.textContent = 'Connection lost. Browser will retry automatically.';
    };
  </script>
</body>
</html>`
