# Customer Interaction Branches

This document tracks the customer-visible interaction tree for example 100.

Scope:
- Includes public marketing and info pages, auth entry, authenticated chat workspace, settings, billing summary, and canvas.
- Excludes admin-only and superuser-only dashboard/control-plane flows.

Status legend:
- `implemented`: customer can complete the action end to end.
- `client-only`: action is real, but handled entirely in client state with no dedicated server mutation.
- `partial`: some supporting code exists, but the customer-facing flow is incomplete or misleading.
- `not implemented`: visible or expected action has no safe end-to-end path yet.

## Route Inventory

| Surface | Route(s) | Primary client entry | Direct-load server handling | Status |
| --- | --- | --- | --- | --- |
| Auth landing | `/` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Marketing home | `/home` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Marketing capabilities | `/capabilities` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Pricing | `/pricing` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Pricing alias | `/plans` | wildcard route via `ParseRun` -> `parseChatWizardRoot` -> `ParseApp` | not covered by `shouldServeClientShell` | partial |
| Signup | `/signup` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Static info pages | `/about`, `/contact`, `/privacy`, `/terms`, `/security`, `/status` | wildcard route via `ParseRun` -> `parseChatWizardRoot` -> `ParseApp` | not covered by `shouldServeClientShell` | partial |
| App root / new chat | `/app` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Thread route | `/app/thread/:publicID` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Canvas route | `/app/thread/:publicID/canvas/:canvasID` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Settings route | `/app/settings?panel=...` | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | implemented |

## Cross-Cutting Runtime Actions

| Action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| Bootstrap localized copy on first load | boot fetch succeeds -> hydrated copy; namespace later changes -> lazy namespace fetch; fetch failure -> fallback locale path | `parseChatWizardRoot` | `GetCatalogBootstrap`, `GetCatalogNamespace` | implemented | Customer-visible on all routes. |
| Bootstrap client log correlation | cached client id exists -> reuse; no client id -> fetch server-issued id; relay unavailable -> local logging only | `parseEnsureClientIdentity`, `parseSendClientLog` | `GetClientIdentity`, `ReportClientLog` | implemented | Supports traceability for later user-visible errors. |
| Direct load or refresh of a customer route | `/`, `/home`, `/capabilities`, `/pricing`, `/signup`, `/app`, and `/app/...` direct loads work; `/plans` and static info pages are not covered by the server shell gate and depend on client-side wildcard routing only | `ParseRun`, `parseChatWizardRoot`, `ParseApp` | `shouldServeClientShell`, `parseServeChatShell` | partial | Direct refresh support is not universal across all customer routes yet. |

## Public Marketing And Info Surfaces

| Surface | User action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Header/footer nav on public pages | navigate to another public page | same-path click -> no-op; different page -> router navigate | `parseLandingNavigateHandler`, `shouldNavigateLandingRoute`, `renderNavLink`, `renderFooterLink` | none | client-only | Applies to `/home`, `/capabilities`, `/pricing`, `/signup`, and info pages. |
| Public pages | switch locale | locale changes -> runtime reloads localized strings; missing namespace -> fetch fallback/default | `renderLanguageSelector` | `GetCatalogBootstrap`, `GetCatalogNamespace` | implemented | No page reload required. |
| Pricing page | jump to plans / compare / faq anchors | direct hash load -> scroll sync; in-page anchor click -> fragment update; back nav -> fragment restore | `parseUsePricingFragmentScroll` plus anchor links in pricing shell | none | client-only | Fragment logic is real, but page-specific. |
| CTA from public pages | open auth landing | click on log-in CTA -> route to `/` | `parseLandingNavigateHandler` | none | client-only | Server only serves shell on refresh. |
| CTA from public pages | open signup | click on signup CTA -> route to `/signup` | `parseLandingNavigateHandler` | none | client-only | No billing mutation happens here. |
| CTA from public pages | open app | unauthenticated -> auth guard eventually redirects to `/`; authenticated -> app shell loads | `parseLandingNavigateHandler`, auth/session guards in `ParseApp` | `GetSession` during auth bootstrap | implemented | Customer sees app entry but auth gate decides final branch. |
| About / privacy / terms / security / status | read informational copy | read-only page body, no mutation | `renderInfoShell`, `renderInfoBody` and page-specific renderers | none | client-only | These surfaces are content-only. |
| Contact page | read support/sales contact details | read-only content cards; no form submit | `renderInfoContact`, `renderInfoContactCard` | none | client-only | There is no in-app contact submission handler. |

## Auth Entry And Session Lifecycle

| Surface | User action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Auth landing | switch login vs signup mode | login -> signup shell; signup -> login shell; reset/update modes collapse back to login on toggle | `HandleModeToggle`, `renderAuthShell` | none | client-only | Pure client mode switch. |
| Auth landing | open forgot-password shell | auth mode changes to reset shell | `HandleForgotPassword`, `renderAuthResetShell` | none | client-only | Shell exists; submit path does not. |
| Auth landing | open update-password shell | auth mode changes to update-password shell | `HandleUpdatePassword`, `renderAuthUpdatePasswordShell` | none | client-only | Shell exists; submit path does not. |
| Login form | submit login | grpc not ready -> client error; missing fields -> client validation error; RPC error -> customer-safe auth error; success -> token persisted, workspace reset, session bootstrap, post-login navigate | `HandleSubmit`, `parseSubmit`, `parseAuthErrorMessage` | `Login`, then `GetSession` | implemented | Real customer path. |
| Signup form | submit signup | short password -> client validation error; duplicate account -> auth error; success -> token persisted, workspace reset, session bootstrap, post-login navigate | `HandleSubmit`, `parseSubmit`, `parseAuthErrorMessage` | `Signup`, then `GetSession` | implemented | Real customer path. |
| Existing session on page load | bootstrap authenticated state | no token -> unauthenticated shell; token invalid -> reset to auth; token valid -> authenticated app and route resolution | `parseUseAuthSession`, `parseApplyUnauthenticatedSessionState` | `GetSession` | implemented | Runs on initial hydration. |
| Active session | silent refresh | token nearing expiry -> refresh; refresh unauthenticated -> reset to auth; refresh failure -> log warning and keep session until next check | `parseUseAuthSession`, `parseRefreshSession` | `RefreshSession` | implemented | Driven by timers and user activity. |
| Active session | periodic/session health check | missing token -> logout state; unauthenticated response -> logout state; authenticated response -> keep session | `parseUseAuthSession`, `parseCheckSession` | `GetSession`, sometimes `RefreshSession` | implemented | Covers expiry and revoked sessions. |
| Authenticated user | sign out | local token cleared; workspace state reset; optional server logout; callback returns to auth landing | `Logout`, `parseApplyUnauthenticatedSessionState` | `Logout` | implemented | Real end-to-end path. |
| Reset-password shell | submit reset request | current client auth submit path falls through to normal login branch; no public reset RPC or HTTP endpoint exists | `HandleSubmit` -> `parseSubmit` | none on public service surface | not implemented | Server has token helpers internally, but customer-facing submit flow does not exist. |
| Update-password shell | submit new password | current client auth submit path falls through to normal login branch; no public update-password RPC or HTTP endpoint exists | `HandleSubmit` -> `parseSubmit` | none on public service surface | not implemented | UI surface exists but end-to-end action does not. |
| New signup | verify email | token tables and auth-manager helpers exist internally, but there is no public callback route or customer verification action | no customer client handler | internal-only helpers in `auth_service.go` and `store_auth.go` | not implemented | Backend groundwork exists, customer path does not. |
| Auth entry | sign in with Google / workspace SSO | no customer button, callback route, or public handshake endpoint in the client surface | no customer client handler | backend groundwork exists in `auth_google_oidc.go` and related store/auth files | not implemented | Treat as backend groundwork only, not a shipped customer flow. |

## Chat Workspace And Thread Lifecycle

| Surface | User action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| App root | start a new draft chat | streaming -> blocked; otherwise clear draft/thread state, persist scroll state, navigate to root draft | `parseNewChatHandler` | none | client-only | New chat becomes persisted only after the next successful `Send`. |
| Sidebar | toggle sidebar open/closed | flip stored open state | `parseToggleSidebarHandler` | none | client-only | Workspace chrome only. |
| Sidebar | load conversation list | unauthenticated -> auth failure handler; first page -> replace list; later pages -> merge unique summaries | `parseUseConversationList`, `Refresh`, `LoadMore` | `ListConversations` | implemented | Real pagination path. |
| Sidebar | open a conversation | invalid id -> no-op; valid id -> load messages, update active thread, restore route | `Load`, `LoadByID`, `parseLoadConversationByID` | `LoadConversation` | implemented | Used from sidebar click and route resolution. |
| Direct thread route | resolve `:publicID` to internal conversation | known in current list -> local open; unknown -> resolve route RPC; inaccessible -> navigate to root draft | `ResolveRoute` | `ResolveConversationRoute`, then `LoadConversation` | implemented | Covers deep links and reloads. |
| Sidebar | delete conversation | request delete -> open modal; cancel -> clear target; confirm -> delete RPC, clear active thread if needed, refresh list | `RequestDelete`, `CancelDelete`, `ConfirmDelete` | `DeleteConversation` | implemented | Real mutation path. |
| Composer | type into input | update draft input text | `HandleInput` | none | client-only | No server call until send. |
| Composer | send message | empty input -> no-op; streaming -> blocked; grpc unavailable -> reconnect warning; send error -> assistant error bubble; stream success -> assistant chunks, final stats, route normalization, conversation refresh | `Send`, `HandleKey`, `parseTriggerSend`, `parsePerformSend` | `Send` | implemented | Core product action. |
| Starter prompts | apply a suggested prompt | streaming -> blocked; prompt copied into composer, focus returned to input | `ApplyStarterPrompt` | none | client-only | Only stages the next send. |
| Message bubble | edit user message and resend | invalid edit state -> no-op; valid edit -> truncate history before edited message and call send again | `StartEdit`, `CancelEdit`, `HandleEditChange`, `SubmitEdit`, `HandleEditKey` | `Send` | implemented | Uses resend rather than an edit mutation API. |
| Message bubble | copy assistant message | copy succeeds or silently fails per browser clipboard availability | inline copy handler -> `parseCopyToClipboard` | none | client-only | No server mutation. |
| Message bubble | fork thread from a prior message | streaming -> blocked; invalid message index -> no-op; valid index -> truncate message list, clear active conversation id, return to draft branch | `Fork` | none until later send | client-only | Branch becomes persisted only if the user sends again. |
| Selected text anywhere outside composer | quote selected text into composer | no selection -> dismiss prompt; valid selection -> quote inserted into draft, focus input | `parseUseQuoteSelection`, `QuoteSelectedText` | none | client-only | Pure drafting affordance. |
| Thread route after first reply | normalize to canonical thread URL | unresolved public id -> wait; resolved id -> replace current route with canonical thread path | route-sync logic in `ParseApp` | `ResolveConversationRoute` and `ListConversations` support resolution | implemented | Critical for reload-safe threads. |
| Chat view | scroll to bottom | button only shown when away from bottom; click forces bottom scroll | `parseScrollToBottom`, `threadScrollMemory.ParseScrollToBottom` | none | client-only | Pure view state. |

## Message-Level Assistant Features

| Surface | User action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Assistant message | play / pause / resume TTS | unsupported model/provider -> customer warning or upgrade modal; cached clip -> play locally; no cache -> stream synth audio; playback failure -> local error state | `parseUseTTSAudio`, `ParseToggle`, `ParseStop` | `SynthesizeSpeech` | implemented | Real streamed synthesis path. |
| Assistant message | stop TTS playback | active or loading clip -> stop and clear playback state | `ParseStop`, `ParseStopCurrent` | none after stream ends | client-only | Local playback control. |
| Assistant message with code block | open canvas preview | artifact missing -> no-op; artifact found -> open or retarget canvas session | `OpenFromMessage` | none | client-only | Canvas is derived from assistant markdown, not stored server-side. |

## Model, Tone, And Preference Controls

| Surface | User action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Workspace toolbar / settings | load model catalog and stored preferences | grpc unavailable -> cached/default values; authenticated -> fetch model catalog + selected model/tone/thinking prefs | `parseUseModelPreferences`, `RefreshCatalog` | `ListModelOptions`, `GetSelectedModel`, `GetSelectedTone`, `GetSelectedThinkingEnabled`, `GetSelectedThinkingEffort` | implemented | Bootstraps most chat controls. |
| Workspace toolbar | switch provider | provider changes -> visible model list filters; no matching model -> default provider model chosen | `SetProvider` inside model preference flow | `SetSelectedModel` indirectly after resolved model choice | implemented | Provider choice is a client-derived filter over model catalog. |
| Workspace toolbar / settings | switch model | invalid/blank model -> no-op; valid model -> selected model updated and persisted | `SetModel`, `applyModelSelection`, `parsePersistSelectedModel` | `SetSelectedModel` | implemented | Appends a local switch marker when a thread already has messages. |
| Workspace toolbar / settings | switch reasoning mode | off -> disable thinking; on -> persist enabled + effort | `SetThinkingMode`, `parsePersistThinkingPreferences` | `SetSelectedThinkingEnabled`, `SetSelectedThinkingEffort` | implemented | Requires a thinking-capable model to matter. |
| Settings | change tone | local input changes immediately; persisted on save | `HandleToneChange`, `Save` | `SetSelectedTone` | implemented | Real preference mutation. |

## Settings, Profile, Memories, And Billing Summary

| Surface | User action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Sidebar / app shell | open settings modal/route | from non-settings route -> remember return path and navigate to `/app/settings`; already on settings -> replace with default panel | `Open`, `buildSettingsRoute` | none | client-only | Opens route-backed settings. |
| Settings | navigate between sections | update active section and replace query string | `NavigateSection`, `buildSettingsRoute` | none | client-only | Route-backed panel navigation. |
| Settings load | fetch current profile + memories + prompt | unauthenticated -> auth failure handler; store data returned -> seed modal inputs | `Refresh` in `parseUseProfileSettings` | `GetUserName`, `ListUserMemories`, `GetCustomSystemPrompt` | implemented | Load path is real. |
| Settings save | save display name | blank name -> skipped; non-blank -> optimistic local update then persisted | `Save` | `SetUserName` | implemented | Real mutation. |
| Settings save | save system prompt | optimistic local update then persisted; RPC failure invalidates cache | `Save` | `SetCustomSystemPrompt` | implemented | Real mutation. |
| Settings save | save locale | immediate runtime locale change; persisted by i18n runtime | `Save`, `parseIntl.SetLocale` | `GetCatalogBootstrap`, `GetCatalogNamespace` on demand | implemented | No dedicated profile RPC. |
| Settings save | save remembered memories | blank summary -> skipped; deleted keys -> delete RPC; remaining memories -> upsert RPC | `AddMemory`, `HandleMemoryChange`, `DeleteMemory`, `Save` | `UpsertUserMemory`, `DeleteUserMemory` | implemented | Real persistence path. |
| Settings save | save TTS provider | local state + persisted app state only | `HandleTTSProvider`, `Save` | none | client-only | There is no dedicated TTS provider RPC yet. |
| Billing panel in settings | view account cost summary | unauthenticated -> empty summary; authenticated -> derive totals by loading conversation histories and pricing metadata | `parseUseAccountCostSummary` | `LoadConversation`, `ListModelOptions` | partial | Read-only and derived; there is no customer billing endpoint or invoice mutation here. |

## Canvas Workspace

| Surface | User action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Canvas session | open from assistant artifact | artifact not found -> no-op; no active session -> open; active session -> retarget artifact | `OpenFromMessage` | none | client-only | Derived from message markdown only. |
| Canvas session | close canvas | on canvas route -> navigate back to thread/root; always closes local canvas session | `Close` | none | client-only | No server state. |
| Canvas session | open canvas-only route | active session + thread public id -> navigate to `/app/thread/:publicID/canvas/:canvasID` | `OpenCanvasOnly` | none | client-only | Route is reload-safe because artifact id is in URL. |
| Canvas session | back to thread | thread route exists -> navigate there; otherwise navigate to app root | `BackToThread` | none | client-only | Pure navigation. |
| Canvas session | reload / refresh preview | active session required; refresh clears stale/runtime state and rerenders iframe document | `Reload`, `RefreshPreview` | none | client-only | No server render API. |
| Canvas session | toggle overlay vs split | active session required; layout mode flips between overlay and split | `ToggleOverlay` | none | client-only | Layout state only. |
| Canvas session | drag/keyboard resize split | ratio clamps to `20%` and `80%`, persisted in local storage | `StartSplitDrag`, `HandleSplitKey` | none | client-only | State is local only. |
| Canvas session | open/clear console | toggles or clears iframe console log buffer | `ToggleConsole`, `ClearConsole` | none | client-only | Console entries come from iframe bridge messages. |
| Canvas session | change focus region | invalid focus index -> no-op; valid focus -> focused snippet updates | `ChangeFocus` | none | client-only | Works over derived code regions. |
| Canvas session | edit focused code and apply patch | inactive session -> blocked; valid draft -> updates current source, patch history, dirty state | `HandleFocusDraft`, `ApplyFocusDraft` | none | client-only | No server patch/save path exists. |
| Canvas session | reset original source | active session required; current source reset to original artifact | `ResetOriginal` | none | client-only | Local only. |
| Canvas session | revert last patch | no patches -> no-op; otherwise pop patch history and restore prior source | `RevertLastPatch` | none | client-only | Local only. |
| Canvas session | copy current code | current source copied to clipboard | `CopyCurrentCode` | none | client-only | No export endpoint. |

## Explicit Customer-Facing Gaps

These are interactions that a customer could reasonably expect from the product language, the route set, or the current UI shells, but which do not currently have a safe end-to-end implementation.

| Expected action | Current surface hint | Closest existing code | Status | Why it is marked missing |
| --- | --- | --- | --- | --- |
| Request password reset by email | Reset-password auth shell | `renderAuthResetShell`, internal `parseBeginPasswordResetToken` helper | not implemented | No public RPC or HTTP endpoint is wired to the reset shell submit action. |
| Complete password update from tokenized recovery flow | Update-password auth shell | `renderAuthUpdatePasswordShell`, internal `parseCompletePasswordResetWithToken` helper | not implemented | No public callback route or submit handler is wired. |
| Verify email after signup | Signup lifecycle implies verification tables | internal verification token helpers in `auth_service.go` and `store_auth.go` | not implemented | No public route, RPC, or UI action completes verification. |
| Sign in with Google | Auth/product roadmap and backend groundwork | `auth_google_oidc.go`, external identity/store helpers | not implemented | No customer-facing button, route, or public service endpoint is wired into the client. |
| Sign in with workspace SSO | Workspace SSO config exists in backend | workspace SSO config store and external-auth groundwork | not implemented | Config exists, but customer-facing SSO start/callback flow is absent. |
| Move thread into a folder | Product expectation and backlog | none in customer client/server surface | not implemented | No folder model, no client handler, no RPC. |
| Create thread folders | Product expectation and backlog | none in customer client/server surface | not implemented | No folder model, no client handler, no RPC. |
| Reorder threads or folders | Product expectation and backlog | none in customer client/server surface | not implemented | No ordering model or mutation path. |
| Rename a conversation | Common chat affordance | none in customer client/server surface | not implemented | Threads only expose load and delete. |
| Share a chat thread | Product expectation and backlog | none in customer client/server surface | not implemented | No share route, token, or permission model in customer flow. |
| Comment on a chat thread | Product expectation and backlog | none in customer client/server surface | not implemented | No comment UI, data model, or RPC in customer flow. |

## Verification Notes

Triple-check sources used for this map:
- route registration in `client/app/app.go` and `client/app/routes.go`
- public/auth render surfaces in `client/app/marketing_shared.go`, `client/app/landing_info.go`, and `client/app/auth_shell.go`
- customer interaction controllers in `client/app/auth.go`, `client/app/conversations.go`, `client/app/stream.go`, `client/app/profile.go`, `client/app/model_preferences.go`, `client/app/tts.go`, `client/app/quote_selection.go`, `client/app/canvas_workspace.go`, and `client/app/app_shell.go`
- customer RPC surface in `proto/chat.proto`
- customer RPC handlers in `server/app/server.go`, `server/app/catalog_ops.go`, and `server/app/client_logging.go`
- direct-load shell handling in `server/app/server.go`

If this document says `implemented`, there is a visible customer path and a matching client or server handler today.
If this document says `partial` or `not implemented`, the codebase may still contain groundwork, tests, storage, or internal helpers, but the customer-facing flow is not complete yet.
