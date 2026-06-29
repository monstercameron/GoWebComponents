// Package render is the public, supported facade over the internal testkit/render harness
// for component render tests. Import this package (not testkit/render directly): it re-exports
// the stable harness types and helpers so the underlying implementation can evolve without
// breaking test code.
package render

import (
	stdtesting "testing"

	base "github.com/monstercameron/GoWebComponents/testkit/render"
)

type Option = base.Option
type Event = base.Event
type Fixture = base.Fixture
type QueryNode = base.QueryNode

func WithQueuedScheduler() Option {
	return base.WithQueuedScheduler()
}

func New(parseTb stdtesting.TB, parseOptions ...Option) *Fixture {
	return base.New(parseTb, parseOptions...)
}

func ParallelSafetyContract() string {
	return base.ParallelSafetyContract()
}
