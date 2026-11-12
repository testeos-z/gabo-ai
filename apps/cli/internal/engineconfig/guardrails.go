package engineconfig

import "strings"

// GuardrailsFile is the project file that turns on gentle-pi's autonomous
// mode (intent 0023 S5, D4). It is project config: committed, so it applies
// to every developer's Pi in the repository.
const GuardrailsFile = ".pi/gentle-ai/runtime-guardrails.json"

// ModeAuto is the gabo.yaml mode in which no person answers questions (PRD
// R15). Only this mode writes the guardrails file (D4).
const ModeAuto = "auto"

// guardKey is the one key gabo writes into the guardrails file.
const guardKey = "autonomousMode"

// guardWrote is the report line of a written guardrails file. It names the
// file as project config and the environment variable as the per-session
// alternative (S5, D4).
const guardWrote = "wrote " + GuardrailsFile + ": autonomousMode true (mode auto; gentle-pi will not ask before git push; the file is project config, commit it or set GENTLE_PI_AUTONOMOUS_MODE=1 for the session instead)"

// Guardrails is the S5 rule (PD8). In mode auto it makes GuardrailsFile hold
// autonomousMode true: a missing file becomes exactly that one key, and an
// existing file keeps every other byte and key. An existing autonomousMode of
// either value is kept as the owner left it. An existing file that is not a
// JSON object is left unchanged and gets one failure line. Any other mode
// writes nothing and prints nothing. The rule is pure: the caller reads the
// file into Input.Files and writes the result. It never depends on
// Input.Version, so a development build writes the file too.
func Guardrails(in Input) Result {
	if in.Mode != ModeAuto {
		return Result{}
	}
	data, ok := in.Files[GuardrailsFile]
	if !ok {
		return guardWriteResult([]byte("{\"" + guardKey + "\": true}\n"))
	}
	if err := checkJSON(data); err != nil {
		return guardFail("invalid JSON (" + err.Error() + ")")
	}
	obj, ok := scanTopObject(data)
	if !ok {
		return guardFail("not a JSON object")
	}
	m, found := obj.member(guardKey)
	if !found {
		return guardWriteResult(obj.insert(data, "\""+guardKey+"\": true"))
	}
	value := strings.TrimSpace(string(data[m.start:m.end]))
	if value == "true" {
		return Result{Section: Section{Lines: []string{"unchanged " + GuardrailsFile + ": " + guardKey + " true already set"}}}
	}
	return Result{Section: Section{Lines: []string{"kept " + GuardrailsFile + ": " + guardKey + " " + value + " by the user's choice"}}}
}

// guardWriteResult is the result of writing the guardrails file b.
func guardWriteResult(b []byte) Result {
	return Result{
		Section: Section{Lines: []string{guardWrote}},
		Writes:  map[string][]byte{GuardrailsFile: b},
	}
}

// guardFail is the failure line of the guardrails file, in the style of the
// .pi/settings.json failure (S5).
func guardFail(reason string) Result {
	return Result{Section: Section{Lines: []string{FailurePrefix + " " + GuardrailsFile + ": " + reason + "; fix it and run gabo init again"}}}
}
