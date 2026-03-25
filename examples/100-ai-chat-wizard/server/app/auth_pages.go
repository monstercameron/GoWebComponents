package app

import (
	"html/template"
	"net/http"
	"strings"
)

type authPageData struct {
	Title           string
	Heading         string
	Subheading      string
	Action          string
	SubmitLabel     string
	SwitchLabel     string
	SwitchHref      string
	Error           string
	Email           string
	Name            string
	ShowNameField   bool
	SessionDuration string
}

var authPageTemplate = template.Must(template.New("auth-page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #111413;
      --panel: rgba(21, 27, 25, 0.92);
      --panel-border: rgba(255,255,255,0.08);
      --text: rgba(255,255,255,0.94);
      --muted: rgba(255,255,255,0.58);
      --accent: #19c37d;
      --accent-2: #0ea47e;
      --danger: #fb7185;
      --field: rgba(255,255,255,0.06);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      font-family: Georgia, "Iowan Old Style", "Palatino Linotype", serif;
      background:
        radial-gradient(circle at top left, rgba(25,195,125,0.18), transparent 28%),
        radial-gradient(circle at bottom right, rgba(14,164,126,0.18), transparent 26%),
        linear-gradient(180deg, #0f1312 0%, #151b19 55%, #101413 100%);
      color: var(--text);
      display: grid;
      place-items: center;
      padding: 1.5rem;
    }
    .shell {
      width: min(100%, 62rem);
      display: grid;
      grid-template-columns: 1.1fr 0.9fr;
      border: 1px solid var(--panel-border);
      border-radius: 1.75rem;
      overflow: hidden;
      background: rgba(8,10,10,0.45);
      box-shadow: 0 28px 80px rgba(0,0,0,0.35);
      backdrop-filter: blur(18px);
    }
    .hero {
      padding: 2.6rem;
      background:
        linear-gradient(180deg, rgba(25,195,125,0.15), rgba(0,0,0,0)),
        linear-gradient(135deg, rgba(255,255,255,0.04), rgba(255,255,255,0.01));
      border-right: 1px solid var(--panel-border);
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      gap: 1.5rem;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 3rem;
      height: 3rem;
      border-radius: 999px;
      background: linear-gradient(135deg, var(--accent), var(--accent-2));
      font-weight: 700;
      letter-spacing: 0.05em;
      box-shadow: 0 12px 30px rgba(25,195,125,0.25);
    }
    .hero h1 {
      margin: 1rem 0 0.6rem;
      font-size: clamp(2rem, 5vw, 3.2rem);
      line-height: 0.95;
      letter-spacing: -0.04em;
    }
    .hero p {
      margin: 0;
      color: var(--muted);
      font-size: 1rem;
      line-height: 1.6;
      max-width: 32rem;
    }
    .hero-note {
      padding: 1rem 1.1rem;
      border-radius: 1rem;
      border: 1px solid rgba(255,255,255,0.08);
      background: rgba(255,255,255,0.03);
      color: rgba(255,255,255,0.72);
      font-size: 0.95rem;
      line-height: 1.5;
    }
    .panel {
      padding: 2rem;
      background: var(--panel);
      display: flex;
      align-items: center;
    }
    .card {
      width: 100%;
    }
    .eyebrow {
      margin: 0 0 0.5rem;
      color: rgba(255,255,255,0.45);
      font-size: 0.76rem;
      text-transform: uppercase;
      letter-spacing: 0.18em;
    }
    .card h2 {
      margin: 0;
      font-size: 1.8rem;
      letter-spacing: -0.03em;
    }
    .card .sub {
      margin: 0.7rem 0 1.4rem;
      color: var(--muted);
      line-height: 1.6;
    }
    form {
      display: flex;
      flex-direction: column;
      gap: 0.9rem;
    }
    label {
      display: flex;
      flex-direction: column;
      gap: 0.38rem;
      font-size: 0.9rem;
      color: rgba(255,255,255,0.78);
    }
    input {
      width: 100%;
      padding: 0.9rem 1rem;
      border-radius: 0.95rem;
      border: 1px solid rgba(255,255,255,0.08);
      background: var(--field);
      color: var(--text);
      font: inherit;
      outline: none;
    }
    input:focus {
      border-color: rgba(25,195,125,0.6);
      box-shadow: 0 0 0 3px rgba(25,195,125,0.16);
    }
    .error {
      padding: 0.85rem 0.95rem;
      border-radius: 0.95rem;
      border: 1px solid rgba(251,113,133,0.28);
      background: rgba(251,113,133,0.08);
      color: #ffd3db;
      font-size: 0.92rem;
    }
    button {
      margin-top: 0.35rem;
      padding: 0.95rem 1.1rem;
      border: 0;
      border-radius: 999px;
      background: linear-gradient(135deg, var(--accent), var(--accent-2));
      color: #052016;
      font: inherit;
      font-weight: 700;
      cursor: pointer;
      box-shadow: 0 16px 32px rgba(25,195,125,0.18);
    }
    .switch {
      margin-top: 1rem;
      color: var(--muted);
      font-size: 0.92rem;
    }
    .switch a {
      color: #b8ffe0;
      text-decoration: none;
    }
    .meta {
      margin-top: 1rem;
      color: rgba(255,255,255,0.38);
      font-size: 0.82rem;
      line-height: 1.5;
    }
    @media (max-width: 860px) {
      .shell {
        grid-template-columns: 1fr;
      }
      .hero {
        border-right: 0;
        border-bottom: 1px solid var(--panel-border);
      }
    }
  </style>
</head>
<body>
  <main class="shell">
    <section class="hero">
      <div>
        <div class="badge">Go</div>
        <h1>Private chat workspace for your Go WASM experiment.</h1>
        <p>Sign in to keep conversations, preferences, and account data scoped to a real user instead of a shared development database.</p>
      </div>
      <div class="hero-note">
        Sessions currently last {{.SessionDuration}} in development. The server uses a signed JWT in an <code>HttpOnly</code> cookie so the browser can join the gRPC WebSocket tunnel without exposing the token to WASM code.
      </div>
    </section>
    <section class="panel">
      <div class="card">
        <p class="eyebrow">Authentication</p>
        <h2>{{.Heading}}</h2>
        <p class="sub">{{.Subheading}}</p>
        {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
        <form method="post" action="{{.Action}}">
          {{if .ShowNameField}}
          <label>
            Display name
            <input type="text" name="name" value="{{.Name}}" autocomplete="nickname" maxlength="80" />
          </label>
          {{end}}
          <label>
            Email
            <input type="email" name="email" value="{{.Email}}" autocomplete="email" required />
          </label>
          <label>
            Password
            <input type="password" name="password" autocomplete="{{if .ShowNameField}}new-password{{else}}current-password{{end}}" required />
          </label>
          <button type="submit">{{.SubmitLabel}}</button>
        </form>
        <p class="switch">{{.SwitchLabel}} <a href="{{.SwitchHref}}">Continue here</a>.</p>
        <p class="meta">The development TTL is intentionally relaxed for local iteration. Production should use a shorter lifetime and rotate the signing secret.</p>
      </div>
    </section>
  </main>
</body>
</html>`))

func renderAuthPage(w http.ResponseWriter, statusCode int, data authPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := authPageTemplate.Execute(w, data); err != nil {
		http.Error(w, "render auth page", http.StatusInternalServerError)
	}
}

func loginPageData(email, errMsg string) authPageData {
	return authPageData{
		Title:           "Login - GoWebComponents Lab",
		Heading:         "Welcome back",
		Subheading:      "Sign in to continue into the chat workspace.",
		Action:          "/login",
		SubmitLabel:     "Log in",
		SwitchLabel:     "Need an account?",
		SwitchHref:      "/signup",
		Error:           strings.TrimSpace(errMsg),
		Email:           strings.TrimSpace(email),
		SessionDuration: "2 hours",
	}
}

func signupPageData(name, email, errMsg string) authPageData {
	return authPageData{
		Title:           "Sign Up - GoWebComponents Lab",
		Heading:         "Create your account",
		Subheading:      "Set up a user so chat history and preferences are isolated per account.",
		Action:          "/signup",
		SubmitLabel:     "Create account",
		SwitchLabel:     "Already have an account?",
		SwitchHref:      "/login",
		Error:           strings.TrimSpace(errMsg),
		Email:           strings.TrimSpace(email),
		Name:            strings.TrimSpace(name),
		ShowNameField:   true,
		SessionDuration: "2 hours",
	}
}
