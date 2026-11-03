// gabo extension for Pi (intent 0003). A no-op stub for now: the
// session_start handler (plan task T6) and the tool_call handler (T7) are
// added here. The extension only relays session facts to the gabo CLI and
// applies what gabo returns; every rule and message lives in Go (D9).
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function gabo(_pi: ExtensionAPI): void {}
