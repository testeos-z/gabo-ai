// tool_call handler of the gabo extension (intent 0003: S5, S6, S8, D3, D4,
// D9; plan task T7). A fake ExtensionAPI (Pi 1.4.0 shape) and a fake `gabo`
// on PATH that records its argv, stdin and working directory. The block
// rules themselves are tested once, in Go; here gabo's answer is scripted.
import { afterAll, afterEach, beforeEach, describe, expect, test } from "bun:test";
import { mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fakeGabo, fakePi, tool } from "./fake-pi";

const pkgDir = resolve(import.meta.dir, "..");
const pkg = JSON.parse(readFileSync(join(pkgDir, "package.json"), "utf8"));
const extension = join(pkgDir, "extensions", "gabo.ts");

const scratch = mkdtempSync(join(tmpdir(), "gabo-pi-tool-call-"));
afterAll(() => rmSync(scratch, { recursive: true, force: true }));

const savedPath = process.env.PATH;
let caseDir = "";
let project = "";
let gabo: ReturnType<typeof fakeGabo>;
let caseNo = 0;

beforeEach(() => {
  caseDir = join(scratch, `case-${++caseNo}`);
  project = join(caseDir, "project");
  mkdirSync(project, { recursive: true });
  gabo = fakeGabo(join(caseDir, "bin"));
  process.env.PATH = join(caseDir, "bin");
});
afterEach(() => {
  process.env.PATH = savedPath;
});

const tools = [tool("read"), tool("bash"), tool("edit"), tool("write")];

// Pi 1.4.0 WriteToolCallEvent / EditToolCallEvent: input.path is the target.
function call(toolName: string, input: Record<string, unknown>) {
  return { type: "tool_call", toolCallId: "call-1", toolName, input };
}
const write = (path: string) => call("write", { path, content: "x\n" });
const edit = (path: string) => call("edit", { path, edits: [{ oldText: "a", newText: "b" }] });

async function callTool(event: ReturnType<typeof call>) {
  const mod = await import(extension);
  const fake = fakePi(tools);
  await mod.default(fake.pi);
  const { notified, results } = await fake.fire(event, project);
  expect(results.length).toBe(1);
  return { notified, result: results[0], fake };
}

function snapshot(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const abs = join(dir, entry);
    const st = statSync(abs);
    return st.isDirectory() ? [`${entry}/`, ...snapshot(abs).map((e) => `${entry}/${e}`)] : [`${entry} ${st.size}`];
  });
}

const gaboProject = () => writeFileSync(join(project, "gabo.yaml"), "repo:\n  name: x/y\n");
const isBlocked = (result: unknown) => (result as { block?: boolean } | undefined)?.block === true;

describe("tool_call on write or edit in a gabo project (S5, S6, D3, D4, D9)", () => {
  test("registers one tool_call handler", async () => {
    const mod = await import(extension);
    const fake = fakePi(tools);
    await mod.default(fake.pi);
    expect(fake.handlers.get("tool_call")?.length).toBe(1);
  });

  for (const [name, event] of [
    ["write", write(".pi/agents/gabo-coder.md")],
    ["edit", edit("/abs/docs/product/intents/0009-x/plan.md")],
  ] as const) {
    test(`${name} runs gabo check --pre-write once with the session facts, event tool_call and [input.path]`, async () => {
      gaboProject();
      gabo.respond('{"notify":[],"block":null}');
      await callTool(event);
      expect(gabo.file("calls")).toBe("1\n");
      expect(gabo.file("argv")).toBe("check\n--pre-write\n");
      expect(gabo.file("cwd")?.trim()).toBe(project);
      expect(JSON.parse(gabo.file("stdin") ?? "null")).toEqual({
        gabo_facts: 1,
        event: "tool_call",
        engine: "pi",
        tools: ["read", "bash", "edit", "write"],
        package_version: pkg.version,
        target_paths: [event.input.path],
      });
    });
  }

  test("gabo exit 1 with a block returns { block: true, reason } with gabo's reason unchanged", async () => {
    gaboProject();
    const reason = "Pi has no subagents: install one.\n✗ docs/product/intents/0009-x/plan.md: spec not accepted";
    gabo.respond(JSON.stringify({ notify: [], block: { reason } }), 1);
    const { result, notified } = await callTool(write(".pi/agents/gabo-coder.md"));
    expect(result).toEqual({ block: true, reason });
    expect(notified).toEqual([]);
  });

  test("an edit is blocked the same way", async () => {
    gaboProject();
    gabo.respond('{"notify":[],"block":{"reason":"✗ plan.md: blocked"}}', 1);
    const { result } = await callTool(edit("docs/product/intents/0009-x/plan.md"));
    expect(result).toEqual({ block: true, reason: "✗ plan.md: blocked" });
  });

  test("gabo exit 0 returns no block and shows nothing", async () => {
    gaboProject();
    gabo.respond('{"notify":[],"block":null}');
    const { result, notified } = await callTool(write("src/main.go"));
    expect(isBlocked(result)).toBe(false);
    expect(notified).toEqual([]);
  });

  test("notifications gabo returns are shown with text and level unchanged", async () => {
    gaboProject();
    const notify = [{ level: "info", text: "note from gabo" }];
    gabo.respond(JSON.stringify({ notify, block: null }));
    const { result, notified } = await callTool(write("src/main.go"));
    expect(isBlocked(result)).toBe(false);
    expect(notified).toEqual([{ message: "note from gabo", type: "info" }]);
  });
});

describe("tool_call without a usable gabo (S6, intent 0001 S7)", () => {
  test("gabo missing from PATH: no block and one warning", async () => {
    gaboProject();
    process.env.PATH = join(caseDir, "empty");
    const { result, notified } = await callTool(write(".pi/agents/gabo-coder.md"));
    expect(isBlocked(result)).toBe(false);
    expect(notified.length).toBe(1);
    expect(notified[0].type).toBe("warning");
  });

  test("gabo exiting 2: no block and one warning", async () => {
    gaboProject();
    gabo.respond('{"notify":[],"block":{"reason":"ignored"}}', 2);
    const { result, notified } = await callTool(write(".pi/agents/gabo-coder.md"));
    expect(gabo.file("calls")).toBe("1\n");
    expect(isBlocked(result)).toBe(false);
    expect(notified.length).toBe(1);
    expect(notified[0].type).toBe("warning");
    expect(notified[0].message).not.toContain("ignored");
  });

  test("gabo exit 1 without a block reason: no block and one warning", async () => {
    gaboProject();
    gabo.respond('{"notify":[],"block":null}', 1);
    const { result, notified } = await callTool(write("src/main.go"));
    expect(isBlocked(result)).toBe(false);
    expect(notified.length).toBe(1);
    expect(notified[0].type).toBe("warning");
  });
});

describe("tool_call where gabo is not run (S5, S6)", () => {
  for (const event of [
    call("read", { path: ".pi/agents/gabo-coder.md" }),
    call("bash", { command: "echo x > .pi/agents/gabo-coder.md" }),
    call("my_write", { path: ".pi/agents/gabo-coder.md" }),
  ]) {
    test(`${event.toolName} in a gabo project does not run gabo`, async () => {
      gaboProject();
      gabo.respond('{"notify":[],"block":{"reason":"x"}}', 1);
      const { result, notified } = await callTool(event);
      expect(gabo.file("calls")).toBeUndefined();
      expect(isBlocked(result)).toBe(false);
      expect(notified).toEqual([]);
    });
  }

  test("write outside a gabo project does not run gabo", async () => {
    gabo.respond('{"notify":[],"block":{"reason":"x"}}', 1);
    const { result, notified } = await callTool(write(".pi/agents/gabo-coder.md"));
    expect(gabo.file("calls")).toBeUndefined();
    expect(isBlocked(result)).toBe(false);
    expect(notified).toEqual([]);
  });
});

describe("tool_call writes nothing and holds no rule (S5, S6, S8, D4)", () => {
  test("the handler writes no file in the project", async () => {
    gaboProject();
    const before = snapshot(project);
    gabo.respond('{"notify":[],"block":{"reason":"x"}}', 1);
    await callTool(write(".pi/agents/gabo-coder.md"));
    gabo.respond('{"notify":[],"block":null}');
    await callTool(edit("README.md"));
    gabo.respond("", 2);
    await callTool(write("README.md"));
    expect(snapshot(project)).toEqual(before);
    expect(before).toEqual([`gabo.yaml ${statSync(join(project, "gabo.yaml")).size}`]);
  });

  const sourceFiles = ["extensions", "src"].flatMap((dir) => {
    const walk = (d: string): string[] =>
      readdirSync(d).flatMap((e) => (statSync(join(d, e)).isDirectory() ? walk(join(d, e)) : [join(d, e)]));
    try {
      return walk(join(pkgDir, dir));
    } catch {
      return [];
    }
  });
  const sources = sourceFiles.map((f) => ({ f, text: readFileSync(f, "utf8") }));

  test("the source exists", () => {
    expect(sources.map((s) => s.f)).toContain(extension);
  });

  test("tests no .pi/agents/ path", () => {
    for (const { f, text } of sources) {
      expect({ f, piAgents: /\.pi[\\/]+agents|["'`]agents["'`]/.test(text) }).toEqual({ f, piAgents: false });
    }
  });

  test("parses no spec.md or Acceptance table", () => {
    for (const { f, text } of sources) {
      expect({ f, spec: /spec\.md|plan\.md|acceptance|intents/i.test(text) }).toEqual({ f, spec: false });
    }
  });
});
