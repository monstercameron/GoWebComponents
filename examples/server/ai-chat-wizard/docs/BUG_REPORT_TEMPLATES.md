# Example 100 Bug Report Templates

Use these templates when filing regressions so triage starts with reproducible evidence.

## 1) Route Bug Template

### Summary

- Expected behavior:
- Actual behavior:
- Failing route(s):

### Reproduction

1.
2.
3.

### Environment

- Commit/branch:
- Browser + version:
- Auth state (unauth/authenticated):
- Server start mode (`gwc examples ... start` / other):

### Captured evidence

- Final URL:
- HTTP status (if known):
- Boot shell state (shown/hidden/stuck):
- Router state (path/query/hash):
- Console/page errors:
- Relevant server log lines:
- Screenshot/video:

### Suspected failure seam

- [ ] shell delivery
- [ ] client route classification
- [ ] auth gate/redirect
- [ ] hydration/runtime boot
- [ ] unknown

## 2) First-Chat Bug Template

### Summary

- Expected first-chat behavior:
- Actual first-chat behavior:

### Reproduction

1.
2.
3.

### Auth/bootstrap state

- Login/signup path used:
- `GetSession` outcome:
- Auth token persisted (yes/no):
- Model catalog readiness:
- Conversation list readiness:

### Send/stream state

- Composer input submitted:
- `Send` request context (conv id/model/tone/thinking):
- Stream outcome (delta/done/error):
- Route after reply:

### Token/cost evidence

- `provider_id`:
- `usage_event_id`:
- prompt/completion tokens:
- `total_cost_usd`:

### Diagnostics

- Console/page errors:
- Relevant server log lines:
- Screenshot/video:

## 3) Admin-Dashboard Bug Template

### Summary

- Expected admin behavior:
- Actual admin behavior:
- Affected slice/action:

### Reproduction

1.
2.
3.

### Role and scope context

- Actor account:
- Role grant context (normal/workspace-admin/superuser):
- Expected access scope:
- Actual access scope:

### Slice/mutation state

- Entry route and resulting route:
- RPC status/error (if known):
- Filter/search/pagination state:
- Mutation payload/confirmation state (if applicable):
- Post-action refresh/rollback behavior:

### Diagnostics

- Console/page errors:
- Relevant server log lines:
- Audit-event evidence (if mutation succeeded):
- Screenshot/video:
