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

func New(tb stdtesting.TB, options ...Option) *Fixture {
	return base.New(tb, options...)
}

func ParallelSafetyContract() string {
	return base.ParallelSafetyContract()
}
