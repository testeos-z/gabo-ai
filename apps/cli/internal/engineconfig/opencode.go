package engineconfig

// OpencodeStageGate is the stage-gate plugin that gabo-runtime-opencode
// writes on every build (intent 0004). The OpenCode rule only reports
// whether it exists; gabo init writes nothing for OpenCode (S6).
const OpencodeStageGate = ".opencode/plugins/gabo-stage-gate.js"

// opencodeStep is the S9 step outside the project for OpenCode when the
// plugin is missing (S6).
const opencodeStep = "ask your agent to build the agents with gabo-runtime-opencode (it installs " + OpencodeStageGate + " and the project-level experimental.subagent_depth key)"

// opencodeRule reports whether the stage-gate plugin is in the project and,
// when it is missing, names gabo-runtime-opencode as the next step. It never
// writes a file (S6).
func opencodeRule(in Input) Result {
	if _, ok := in.Files[OpencodeStageGate]; ok {
		return Result{Section: Section{Lines: []string{"stage-gate plugin in place (built by gabo-runtime-opencode)"}}}
	}
	return Result{Section: Section{
		Lines: []string{"no " + OpencodeStageGate + " yet; nothing written"},
		Steps: []string{opencodeStep},
	}}
}
