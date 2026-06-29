package runtime

// AgentStateVersion returns the monotonic runtime version exposed by the live
// agent bridge. It advances on commits and on bridge-visible write commands so
// command acks can preserve read-your-writes ordering without inspecting DOM
// timing internals.
func (parseRt *Runtime) AgentStateVersion() uint64 {
	if parseRt == nil {
		return 0
	}
	return parseRt.agentStateVersion.Load()
}

// AdvanceAgentStateVersion increments and returns the bridge-visible runtime
// state version. Agent write commands call this after a successful mutation or
// side-effect dispatch; commitRoot also advances it after a DOM commit.
func (parseRt *Runtime) AdvanceAgentStateVersion() uint64 {
	if parseRt == nil {
		return 0
	}
	return parseRt.agentStateVersion.Add(1)
}
