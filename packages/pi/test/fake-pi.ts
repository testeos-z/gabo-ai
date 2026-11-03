// Fake Pi engine API for the extension tests (intent 0003, plan TRD 11).
// It mirrors the parts of Pi 1.4.0 dist/core/extensions/types.d.ts the gabo
// extension may use: ExtensionAPI.on and getAllTools (ToolInfo), and the
// ExtensionContext passed to handlers (cwd, mode, hasUI, ui.notify). Any
// other ExtensionAPI member throws, so an unexpected use fails the test.
import { chmodSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";

// Pi 1.4.0: ToolInfo = Pick<ToolDefinition, "name" | "description" |
// "parameters" | ...> & { exposure; sourceInfo; ... }.
export interface FakeToolInfo {
  name: string;
  description: string;
  parameters: unknown;
  exposure: "direct" | "model-only" | "codemode" | "deferred" | "hidden";
  sourceInfo: { path: string; source: string; scope: string; origin: string };
}

// Pi 1.4.0: ExtensionUIContext.notify(message, type?: "info" | "warning" | "error").
export interface Notified {
  message: string;
  type: string | undefined;
}

type Handler = (event: unknown, ctx: unknown) => unknown;

export function tool(name: string): FakeToolInfo {
  return {
    name,
    description: `${name} tool`,
    parameters: { type: "object", properties: {} },
    exposure: "direct",
    sourceInfo: { path: `<builtin:${name}>`, source: "builtin", scope: "user", origin: "builtin" },
  };
}

export function fakePi(tools: FakeToolInfo[]) {
  const handlers = new Map<string, Handler[]>();
  const used: string[] = [];
  const api = {
    on(event: string, handler: Handler): () => void {
      used.push(`on:${event}`);
      handlers.set(event, [...(handlers.get(event) ?? []), handler]);
      return () => undefined;
    },
    getAllTools(): FakeToolInfo[] {
      used.push("getAllTools");
      return tools;
    },
  };
  const pi = new Proxy(api, {
    get(target, key) {
      if (key in target) return target[key as keyof typeof target];
      if (key === "then") return undefined;
      throw new Error(`fake ExtensionAPI: ${String(key)} is not faked`);
    },
  });

  // Fires an event the way Pi does: every handler in order, awaited.
  async function emit(event: { type: string } & Record<string, unknown>, cwd: string): Promise<Notified[]> {
    const notified: Notified[] = [];
    const ctx = {
      cwd,
      mode: "tui",
      hasUI: true,
      ui: { notify: (message: string, type?: string) => notified.push({ message, type }) },
    };
    for (const handler of handlers.get(event.type) ?? []) await handler(event, ctx);
    return notified;
  }

  return { pi, handlers, used, emit };
}

// A fake `gabo` executable in its own bin dir. Each run records its argv,
// stdin and working directory in the bin dir (never in the project), prints
// the configured stdout and exits with the configured code.
export function fakeGabo(binDir: string) {
  mkdirSync(binDir, { recursive: true });
  const script = join(binDir, "gabo");
  writeFileSync(
    script,
    [
      "#!/bin/sh",
      "PATH=/usr/bin:/bin",
      'd="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"',
      'n=$(( $(cat "$d/calls" 2>/dev/null || echo 0) + 1 ))',
      'echo "$n" > "$d/calls"',
      'printf "%s\\n" "$@" > "$d/argv"',
      'cat > "$d/stdin"',
      'pwd > "$d/cwd"',
      'cat "$d/out" 2>/dev/null',
      'exit "$(cat "$d/code" 2>/dev/null || echo 0)"',
      "",
    ].join("\n"),
  );
  chmodSync(script, 0o755);
  return {
    respond(stdout: string, code = 0) {
      writeFileSync(join(binDir, "out"), stdout);
      writeFileSync(join(binDir, "code"), String(code));
    },
    file(name: "calls" | "argv" | "stdin" | "cwd"): string | undefined {
      try {
        return readFileSync(join(binDir, name), "utf8");
      } catch {
        return undefined;
      }
    },
  };
}
