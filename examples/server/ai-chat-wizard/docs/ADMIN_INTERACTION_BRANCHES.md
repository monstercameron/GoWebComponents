# Admin Interaction Branches

This document tracks the admin-visible interaction tree for example 100.

Scope:
- Includes admin dashboard entry points, dashboard routes, shipped admin slice interactions, and backend-exposed admin or superuser operations that are relevant to operators.
- Excludes ordinary customer chat flows already covered in `docs/CUSTOMER_INTERACTION_BRANCHES.md`.

Workspace-admin note:
- A workspace-admin panel scaffold exists behind the `admin_workspaces` client build tag in `client/app/admin_workspaces.go`.
- The default shipped client build does not expose that panel as a route, so this document keeps it in the unshipped / backend-exposed portion of the map.

Status legend:
- `implemented`: admin can complete the action end to end from a shipped client surface.
- `client-only`: action is real, but handled entirely in client state with no dedicated server mutation.
- `backend-only`: RPC or store-backed capability exists, but no shipped admin client surface currently drives it.
- `partial`: some supporting code exists, but the visible admin workflow is incomplete or narrower than the backend capability.
- `not implemented`: visible or expected admin action has no safe end-to-end path yet.

## Route Inventory

| Surface | Route(s) | Primary client entry | Direct-load server handling | Status |
| --- | --- | --- | --- | --- |
| Dashboard home | `/app/dashboard` | `ParseRun`, `ParseApp`, `renderDashboardHome` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Business slice | `/app/dashboard/business` | `ParseRun`, `ParseApp`, `renderDashboardBusiness` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Customers slice | `/app/dashboard/customers` | `ParseRun`, `ParseApp`, `renderDashboardCustomersEnhanced` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Chats slice | `/app/dashboard/chats` | `ParseRun`, `ParseApp`, `renderDashboardChats` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Providers slice | `/app/dashboard/providers` | `ParseRun`, `ParseApp`, `renderDashboardProviders` | `shouldServeClientShell`, `parseServeChatShell` | implemented |
| Ops slice | `/app/dashboard/ops` | `ParseRun`, `ParseApp`, `renderDashboardOps` | `shouldServeClientShell`, `parseServeChatShell` | implemented |

## Cross-Cutting Admin Entry And Access Control

| Action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| Bootstrap admin capability on login or refresh | role summary says admin -> `CanAccessAdmin` set true; role summary says non-admin -> admin entry hidden and admin post-login intent collapses to `/app` | auth bootstrap in `ParseApp`, `parseCanAccessAdminFromRoleSummary`, `parseResolvePostLoginRoute` | `GetSession` | implemented | This is the main gate that decides whether the dashboard entry is exposed. |
| Open dashboard from workspace sidebar | admin -> navigate to `/app/dashboard`; non-admin -> button hidden | `parseSidebar`, `parseOpenAdminDashboard` | none | client-only | Main desktop entry point. |
| Open dashboard from chat toolbar | admin -> navigate to `/app/dashboard`; non-admin -> button hidden | `renderMobileControlBar`, `renderDesktopControlBar`, `parseOpenAdminDashboard` | none | client-only | Secondary entry point from the active chat surface. |
| Direct-load an admin route | route shell loads; visible data then depends on auth and RPC scope | `ParseRun`, `ParseApp`, `isDashboardRoute` | `shouldServeClientShell`, `parseServeChatShell` | partial | Server boot is correct, but route usefulness still depends on later authz and slice fetches. |
| Fetch the shared admin dashboard snapshot | authenticated admin -> snapshot loads; unauthenticated or denied -> safe dashboard error/denied state | `parseUseAdminDashboard` | `GetAdminDashboard` | implemented | This powers home, Business, Chats, Providers, and Ops. |

## Shipped Dashboard Navigation

| Surface | Admin action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Dashboard home tiles | open Business / Customers / Chats / Providers / Ops | tile click -> route navigate; current slice route renders matching body | `renderDashboardSliceTiles`, `renderDashboardSliceTile`, `renderDashboardBody` | none | client-only | Tile routing is local; slice data comes from later fetches. |
| Dashboard top bar on home | return to workspace | click -> route to `/app` | `renderDashboardTopBar` | none | client-only | No server mutation. |
| Dashboard top bar on slice routes | return to dashboard home | click -> route to `/app/dashboard` | `renderDashboardTopBar` | none | client-only | No server mutation. |
| Dashboard slice routes | render current slice | home -> overview cards; customers -> enhanced customer slice; others -> read-only analytics views | `renderDashboardBody` | `GetAdminDashboard` for shared data | implemented | Only Customers has interactive account controls today. |

## Shipped Analytics Slices

| Surface | Admin action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Dashboard home | view role banner, slice tiles, personal account summary | read-only render from app state and shared dashboard snapshot | `renderDashboardHomeBody`, `renderDashboardRoleBanner`, `renderDashboardAccountSummary` | `GetAdminDashboard` | implemented | Summary block is mostly presentation over already-fetched state. |
| Business | review KPIs, top accounts, recent users, recent conversations, daily usage | read-only analytics view | `renderDashboardBusiness` | `GetAdminDashboard` | partial | Slice is useful for viewing, but no business drill-down or corrective action UI is wired yet. |
| Chats | review conversation, message, usage, and onboarding metrics | read-only analytics view | `renderDashboardChats` | `GetAdminDashboard` | partial | Chat investigation is overview-only today. |
| Providers | review provider health, last error, latency, request counts | read-only analytics view | `renderDashboardProviders`, `renderDashboardProvidersTable` | `GetAdminDashboard` | partial | No provider-routing or guardrail mutation UI is wired. |
| Ops | review incidents, support count, experiment count, failure totals, usage trend | read-only analytics view | `renderDashboardOps` | `GetAdminDashboard` | partial | No incident, support, or runtime control workflow is exposed in current client. |

## Shipped Customers Slice

| Surface | Admin action | Branches | Client handler(s) | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Customers slice | view default recent-user list | empty search -> mirror `RecentUsers` from the shared dashboard snapshot | `parseUseAdminCustomers`, `renderDashboardCustomersEnhanced` | `GetAdminDashboard` | implemented | Baseline customer queue on entry. |
| Customers slice | search users | empty query -> stop search and show recent users; query -> debounced search; RPC error -> warning log and prior UI remains | `parseUseAdminCustomers`, `HandleSearch`, `renderAdminCustomersSearchBar` | `SearchAdminUsers` | implemented | Search is the main discovery action. |
| Customers slice | move between result pages | previous/next page clamps within local result set | `HandleNextPage`, `HandlePrevPage`, `renderAdminCustomersPagination` | none | client-only | Pagination is local over the current result set. |
| Customers slice | select one user row | same selected row -> close detail panel; different row -> open detail panel and fetch detail | `HandleSelectUser`, `renderAdminUserListTable`, `renderAdminUserDetailPanel` | `GetAdminUserDetail` | implemented | Selection drives the detail panel. |
| Customers detail panel | review sessions, usage rows, and audit history | read-only detail render | `renderAdminUserDetailPanel`, `renderAdminSessionsTable`, `renderAdminUsageEventsTable`, `renderAdminAuditTable` | `GetAdminUserDetail` | implemented | The only shipped customer drill-down today. |
| Customers detail panel | open disable confirmation | action button click -> modal opens with `disable` action | `HandleConfirmStart`, `renderAdminUserActionBand`, `renderAdminConfirmModal` | none | client-only | Mutation does not happen until submit. |
| Customers detail panel | open restore confirmation | action button click -> modal opens with `restore` action | `HandleConfirmStart`, `renderAdminUserActionBand`, `renderAdminConfirmModal` | none | client-only | Mutation does not happen until submit. |
| Customers confirmation modal | edit confirmation reason | local text state updates | `HandleConfirmReason`, `renderAdminConfirmModal` | none | client-only | Reason is later sent with the mutation. |
| Customers confirmation modal | cancel mutation | close modal and clear staged reason | `HandleConfirmCancel`, `renderAdminConfirmModal` | none | client-only | No server call. |
| Customers confirmation modal | submit disable | invalid state -> no-op; valid request -> disable RPC, success banner, detail refresh | `HandleConfirmSubmit` | `DisableAdminUser` | implemented | The only shipped destructive admin action today. |
| Customers confirmation modal | submit restore | invalid state -> no-op; valid request -> restore RPC, success banner, detail refresh | `HandleConfirmSubmit` | `RestoreAdminUser` | implemented | The only shipped restorative admin action today. |

## Backend-Exposed Admin Capabilities Without Shipped Client Wiring

These actions have real RPC handlers and store-backed logic, but no current admin route or client workflow drives them.

| Surface family | Admin action | Closest client surface today | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| Workspace controls | inspect one workspace in detail | none | `GetAdminWorkspaceDetail` | backend-only | No current dashboard route opens workspace detail. |
| Workspace controls | suspend one workspace | none | `SuspendAdminWorkspace` | backend-only | No shipped mutation UI. |
| Workspace controls | restore one workspace | none | `RestoreAdminWorkspace` | backend-only | No shipped mutation UI. |
| Workspace controls | revoke a workspace API key | none | `RevokeWorkspaceAPIKey` | backend-only | No API key management surface in current dashboard. |
| Workspace controls | pause one webhook endpoint | none | `PauseWorkspaceWebhookEndpoint` | backend-only | No webhook control surface in current dashboard. |
| Billing controls | list billing access overrides | none | `ListAdminBillingAccessOverrides` | backend-only | Customer slice mentions billing state, but this drill-down is not wired. |
| Billing controls | list billing events | none | `ListAdminBillingEvents` | backend-only | No invoice or event timeline UI. |
| Billing controls | list dunning events | none | `ListAdminBillingDunningEvents` | backend-only | No failed-payment review surface. |
| Billing controls | set one billing access override | none | `SetAdminBillingAccessOverride` | backend-only | No shipped override workflow. |
| Billing controls | set one billing quota override | none | `SetAdminBillingQuotaOverride` | backend-only | No shipped override workflow. |
| Billing controls | resolve one failed payment | none | `ResolveAdminBillingFailedPayment` | backend-only | No shipped recovery workflow. |
| Business drill-down | inspect business metrics in more detail | Business slice overview only | `GetAdminBusinessDrilldown` | backend-only | Business slice is currently summary-only. |
| Support queue | list support tickets | Ops slice summary only | `ListAdminSupportTickets` | backend-only | No support queue table is rendered yet. |
| Support queue | inspect support ticket detail | none | `GetAdminSupportTicketDetail` | backend-only | No current route or panel opens ticket detail. |
| Support queue | add internal support note | none | `AddAdminSupportInternalNote` | backend-only | Requires explicit confirmation server-side. |
| Support queue | assign support ticket | none | `AssignAdminSupportTicket` | backend-only | No shipped assignment UI. |
| Support queue | escalate support ticket | none | `EscalateAdminSupportTicket` | backend-only | No shipped escalation UI. |
| Control plane | toggle feature flag | none | `SetAdminFeatureFlag` | backend-only | No feature-flag operator UI yet. |
| Control plane | rollback experiment | none | `RollbackAdminExperiment` | backend-only | No experiment control UI yet. |
| Control plane | update incident status and publish incident note | none | `UpdateAdminIncident` | backend-only | Ops slice is read-only today. |
| Control plane | inspect incident blast radius | none | `GetAdminIncidentBlastRadius` | backend-only | No blast-radius viewer in current client. |

## Superuser And Workspace-Admin Capability Surface

These are operator-relevant surfaces in the service, but they are not yet represented by a shipped admin dashboard client.

| Surface family | Operator action | Visible client surface today | Server / RPC handler(s) | Status | Notes |
| --- | --- | --- | --- | --- | --- |
| Workspace-admin slices | inspect workspace detail, membership, API keys, webhooks, and audit logs from the build-tagged workspace-admin panel scaffold | `client/app/admin_workspaces.go` behind the `admin_workspaces` build tag | `GetWorkspaceAdminSlices`, `GetWorkspaceAdminDetail`, `SuspendAdminWorkspace`, `RestoreAdminWorkspace`, `RevokeWorkspaceAPIKey`, `PauseWorkspaceWebhookEndpoint` | partial | The scaffold exists, but the default shipped client build does not expose it as a route. |
| Superuser control plane | fetch global control-plane snapshot | none | `GetSuperuserControlPlane` | backend-only | No current `/app/dashboard/...` route renders this snapshot. |
| Superuser slices | fetch superuser slice bundle | none | `GetSuperuserSlices` | backend-only | No current client integration. |
| Reliability operations | set SLOs, incidents, provider routing, quota policies, etc. | none | `SetSuperuserServiceLevelObjective`, `SetSuperuserIncident`, `SetSuperuserProviderPolicy`, `SetSuperuserWorkspaceModelRoutingPolicy`, `SetSuperuserWorkspaceCostGuardrail`, related superuser RPCs | backend-only | Broad superuser surface exists in proto/server but not in shipped UI. |
| Pricing operations | mutate plans, quota policies, overages, upgrade triggers, and dunning policy | none | `SetSuperuserBillingPlan`, `SetSuperuserBillingQuotaPolicy`, `SetSuperuserBillingPlanOverage`, `SetSuperuserBillingUpgradeTrigger`, `SetSuperuserBillingDunningPolicy`, related superuser RPCs | backend-only | Pricing control plane is backend-capable but not client-exposed. |
| Runtime diagnostics | tail logs | none | `GetLogTail` | backend-only | No log-view route in current dashboard. |
| Runtime diagnostics | inspect server tool policy | none | `GetServerToolPolicy` | backend-only | No current operator surface. |
| Runtime diagnostics | mutate server tool policy | none | `SetServerToolPolicy` | backend-only | No current operator surface. |
| Runtime diagnostics | execute one server tool run | none | `RunServerTool` | backend-only | Streaming RPC exists, but there is no shipped client flow. |

## Explicit Admin-Facing Gaps

These are actions that an operator could reasonably expect from the current dashboard labels, role descriptions, or proto surface, but which do not currently have a shipped client workflow.

| Expected admin action | Current surface hint | Closest existing code | Status | Why it is marked missing |
| --- | --- | --- | --- | --- |
| Drill into Business slice and resolve failed-payment or churn issues from there | Business tile and role banner promise business operations | `renderDashboardBusiness`, `GetAdminBusinessDrilldown`, billing RPCs | partial | Slice ships as read-only analytics, with no drill-down or mutation path. |
| Drill into Chat failures and inspect one bad conversation from Chats slice | Chats tile promises reply-quality and failure review | `renderDashboardChats`, admin conversation and read-only report RPCs | partial | Current UI stays at aggregate analytics only. |
| Adjust provider routing or guardrails from Providers slice | Providers tile promises routing and guardrail controls | `renderDashboardProviders`, superuser provider policy RPCs | partial | Current UI is read-only health and cost data. |
| Manage incidents, support queue, and job failures from Ops slice | Ops tile promises incident log and SLO tracking | `renderDashboardOps`, support and incident RPCs | partial | Current UI shows counts and summaries only. |
| Search workspaces and suspend or restore them | Customers subtitle mentions workspace controls | workspace admin RPCs in `admin_workspace_controls.go` | not implemented | No workspace list or detail client flow exists today. |
| Inspect or mutate customer billing state from Customers slice | Customers subtitle mentions billing state | billing list and mutation RPCs in `admin_billing_ops.go` | not implemented | Current shipped detail panel stops at sessions, usage, and audit. |
| Review support tickets, assign them, and add internal notes | Ops/support concepts appear in metrics and proto | support RPCs in `admin_support_ops.go` | not implemented | No current client queue or detail view. |
| Toggle feature flags, rollback experiments, or publish incident updates | admin-control RPC surface exists | `admin_control_ops.go` | not implemented | No client control-plane route or modal exists. |
| Use log tail or server tools from the dashboard | server tool and log RPCs exist | `GetLogTail`, `GetServerToolPolicy`, `SetServerToolPolicy`, `RunServerTool` | not implemented | No shipped diagnostics or tools UI. |

## Verification Notes

Triple-check sources used for this map:
- route registration and admin entry handlers in `client/app/app.go`, `client/app/routes.go`, `client/app/sidebar.go`, `client/app/panel.go`, and `client/app/route_sync.go`
- shipped dashboard render surfaces in `client/app/dashboard.go`, `client/app/admin_data.go`, and `client/app/admin_customers.go`
- build-tagged workspace-admin scaffold in `client/app/admin_workspaces.go`
- admin and superuser RPC surface in `proto/chat.proto`
- workspace-admin and shipped admin RPC handlers in `server/app/workspace_admin_ops.go`, `server/app/admin_dashboard.go`, `server/app/admin_workspace_controls.go`, `server/app/admin_billing_ops.go`, `server/app/admin_support_ops.go`, `server/app/admin_control_ops.go`, `server/app/admin_business_ops.go`, `server/app/superuser_control.go`, `server/app/log_tail.go`, and `server/app/server_tool_stub.go`

If this document says `implemented`, there is a visible shipped admin path and a matching client or server handler today.
If this document says `backend-only`, the service has real operator capability but the current admin client does not expose it.
If this document says `partial` or `not implemented`, the codebase may still contain groundwork, store support, tests, or proto definitions, but the operator-facing workflow is not complete yet.
