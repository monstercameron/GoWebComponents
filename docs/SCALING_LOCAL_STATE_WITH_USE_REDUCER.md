# Scaling Local State With UseReducer

This guide covers the point where one local feature grows past a couple of `ui.UseState(...)` calls and starts needing a more deliberate transition model.

## Use This Guide When

- one screen or one component subtree owns the state
- several user actions need to update multiple fields together
- the render function is starting to accumulate scattered `Set` and `Update` calls
- you want a feature-specific hook that hides reducer plumbing from the renderer

If the same source of truth needs to drive distant consumers across the app, a local reducer is usually the wrong ownership model. Use context for subtree-scoped sharing, and use atoms or other shared state when unrelated branches need the same state source.

## The Scaling Problem

`ui.UseState` stays ideal while the local state is small and the updates are obvious.

Typical good `UseState` cases:

- one counter or toggle
- one input draft
- one selected tab or one disclosure state
- a small number of independent values

The cost starts to rise when a single user action needs to move several fields together. At that point the renderer often turns into a list of unrelated mutations:

```go
selectedThread.Set(thread)
reviewRequired.Set(true)
queueStatus.Set("Waiting for review")
lastAction.Set("Escalated policy exception")
reviewerCount.Update(func(count int) int { return max(1, count) })
```

That shape works, but it spreads one workflow transition across the component. The risk is not performance first. The risk is readability, missed field updates, and feature-specific rules leaking into multiple handlers.

## The Reducer Pattern

Use `ui.UseReducer` when the state behaves like a local workflow or state machine.

The reducer should own:

- the local state struct
- the list of legal actions
- the transition rules that derive the next state

The component should mostly own:

- rendering
- layout
- button placement
- attaching semantic handlers

The most maintainable shape in this repo is usually not calling `Dispatch(...)` all over the renderer. The better pattern is wrapping the reducer in an app-specific hook.

## Wrap The Primitive In A Feature Hook

Start from the primitive:

```go
workflow := ui.UseReducer(reduceReplyState, initialReplyState())
```

Then turn it into a feature-local API:

```go
type supportReplyWorkflow struct {
    State         replyState
    RequestReview ui.Handler
    ApproveReply  ui.Handler
    QueueReply    ui.Handler
}

func useSupportReplyWorkflow() supportReplyWorkflow {
    workflow := ui.UseReducer(reduceReplyState, initialReplyState())
    return supportReplyWorkflow{
        State: workflow.Get(),
        RequestReview: ui.UseEvent(func() {
            workflow.Dispatch(replyAction{Type: actionRequestReview})
        }),
        ApproveReply: ui.UseEvent(func() {
            workflow.Dispatch(replyAction{Type: actionApprove})
        }),
        QueueReply: ui.UseEvent(func() {
            workflow.Dispatch(replyAction{Type: actionQueue})
        }),
    }
}
```

That gives the renderer a surface that reads in product terms instead of reducer mechanics.

## What To Keep In Reducer State

Keep values in reducer state when they:

- change together as part of one local transition
- must stay mutually consistent
- drive visible workflow status or gating rules
- are easier to reason about as named actions than as ad hoc field updates

Avoid storing values in reducer state when they are:

- purely derived from other state and cheap to compute during render
- shared across unrelated branches of the app
- better modeled as form field lifecycle through `ui.UseForm[T]`
- long-lived shared data that should survive outside one feature subtree

## Boundary With Other State Tools

Choose the smallest ownership model that matches the problem:

- `ui.UseState`: one or a few independent local values
- `ui.UseReducer`: local workflow state with named transitions
- `ui.UseForm[T]`: field state, validation, and submit lifecycle
- `ui.CreateContext` and `ui.UseContext`: one subtree needs the same value without prop threading
- `state.UseAtom`: multiple components outside one local subtree need the same source of truth

Two boundaries matter in practice:

1. A reducer can stay local even when its API is rich.
2. A rich local reducer is still not a replacement for shared state.

If you need both, compose them explicitly. A common pattern is keeping field values in `ui.UseForm[T]`, workflow status in `ui.UseReducer`, and app-wide preferences or cached data in atoms.

## Keep Reducers Boring

Good reducers in this codebase should be:

- pure
- deterministic
- small enough to read quickly
- centered on named actions rather than generic patch maps

Avoid putting I/O, timers, or browser interop into the reducer. Effects and async work should stay outside and dispatch actions back into the reducer when the result arrives.

## Example

See the dedicated example at `examples/28-scaling-local-state-with-use-reducer`.

That page demonstrates:

- a reducer-owned reply workflow
- one feature-specific hook built on top of `ui.UseReducer`
- a renderer that only consumes `State` plus semantic handlers
- the point where local state is still local, but no longer simple enough for scattered `UseState` calls