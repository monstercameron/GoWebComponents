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
