# buyer-flow/form-submit

Status: planning skeleton.

Routes:

- `/shop/frame-desk`
- `/warehouses/new-jersey-hub/availability/frame-desk`

Assertions:

- quote request preserves invalid input and shows field plus summary errors
- restock request handles alternate warehouse selection, validation, and success
- public comment shows optimistic pending state and moderation expectation messaging
- reload and direct-entry SSR preserve confirmation state where supported
