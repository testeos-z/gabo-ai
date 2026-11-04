// session_start handler of the gabo extension (intent 0003: S4, S7, S8, D2,
// D9; plan task T6). A fake ExtensionAPI (Pi 1.4.0 shape) and a fake `gabo`
// on PATH that records its argv, stdin and working directory.
import { afterAll, afterEach, beforeEach, describe, expect, test } from "bun:test";
import { mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fakeGabo, fakePi, tool } from "./fake-pi";

const pkgDir = resolve(import.meta.dir, "..");
const repoRoot = resolve(pkgDir, "..", "..");
const pkg = JSON.parse(readFileSync(join(pkgDir, "package.json"), "utf8"));
const extension = join(pkgDir, "extensions", "gabo.ts");

const scratch = mkdtempSync(join(tmpdir(), "gabo-pi-session-start-"));
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

async function startSession() {
  const mod = await import(extension);
  const fake = fakePi(tools);
  await mod.default(fake.pi);
  const notified = await fake.emit({ type: "session_start", reason: "startup" }, project);
  return { notified, fake };
}

function snapshot(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const abs = join(dir, entry);
    const st = statSync(abs);
    return st.isDirectory() ? [`${entry}/`, ...snapshot(abs).map((e) => `${entry}/${e}`)] : [`${entry} ${st.size}`];
  });
}

const gaboProject = () => writeFileSync(join(project, "gabo.yaml"), "repo:\n  name: x/y\n");

describe("session_start in a gabo project (S4, S7, D2, D9)", () => {
  test("registers a session_start handler", async () => {
    const { fake } = await startSession();
    expect(fake.handlers.get("session_start")?.length).toBe(1);
  });

  test("runs gabo check --pre-write once in the working directory with the session facts on stdin", async () => {
    gaboProject();
    gabo.respond('{"notify":[],"block":null}\n');
    await startSession();
    expect(gabo.file("calls")).toBe("1\n");
    expect(gabo.file("argv")).toBe("check\n--pre-write\n");
    expect(gabo.file("cwd")?.trim()).toBe(project);
    expect(JSON.parse(gabo.file("stdin") ?? "null")).toEqual({
      gabo_facts: 1,
      event: "session_start",
      engine: "pi",
      tools: ["read", "bash", "edit", "write"],
      package_version: pkg.version,
    });
  });

  test("shows each notification once, with gabo's text and level unchanged", async () => {
    gaboProject();
    const notify = [
      { level: "warning", text: "Pi has no subagents: `gentle-ai install --agent pi`." },
      { level: "info", text: "package 0.5.0 ≠ .gabo/VERSION 0.4.0; run gabo init" },
    ];
    gabo.respond(JSON.stringify({ notify, block: null }));
    const { notified } = await startSession();
    expect(notified).toEqual(notify.map((n) => ({ message: n.text, type: n.level })));
  });

  test("an empty notify shows nothing", async () => {
    gaboProject();
    gabo.respond('{"notify":[],"block":null}');
    const { notified } = await startSession();
    expect(notified).toEqual([]);
  });
});

describe("session_start outside a gabo project or without a usable gabo (S4, S6, S8)", () => {
  test("without gabo.yaml gabo is not run and nothing is shown", async () => {
    gabo.respond('{"notify":[{"level":"warning","text":"x"}],"block":null}');
    const { notified } = await startSession();
    expect(gabo.file("calls")).toBeUndefined();
    expect(notified).toEqual([]);
  });

  test("gabo missing from PATH shows one warning and nothing else", async () => {
    gaboProject();
    process.env.PATH = join(caseDir, "empty");
    const { notified } = await startSession();
    expect(gabo.file("calls")).toBeUndefined();
    expect(notified.length).toBe(1);
    expect(notified[0].type).toBe("warning");
  });

  test("gabo exiting 2 shows one warning and nothing else", async () => {
    gaboProject();
    gabo.respond('{"notify":[{"level":"info","text":"ignored"}],"block":null}', 2);
    const { notified } = await startSession();
    expect(gabo.file("calls")).toBe("1\n");
    expect(notified.length).toBe(1);
    expect(notified[0].type).toBe("warning");
    expect(notified[0].message).not.toContain("ignored");
  });

  test("the handler writes no file in the project", async () => {
    gaboProject();
    const before = snapshot(project);
    gabo.respond('{"notify":[{"level":"info","text":"hello"}],"block":null}');
    await startSession();
    expect(snapshot(project)).toEqual(before);
    gabo.respond("", 2);
    await startSession();
    expect(snapshot(project)).toEqual(before);
    process.env.PATH = join(caseDir, "empty");
    await startSession();
    expect(snapshot(project)).toEqual(before);
    expect(before).toEqual([`gabo.yaml ${statSync(join(project, "gabo.yaml")).size}`]);
  });
});

describe("extension source holds no rule (S4, S7, D7, D9)", () => {
  const sourceFiles = ["extensions", "src"].flatMap((dir) => {
    const abs = join(pkgDir, dir);
    const walk = (d: string): string[] =>
      readdirSync(d).flatMap((e) => (statSync(join(d, e)).isDirectory() ? walk(join(d, e)) : [join(d, e)]));
    try {
      return walk(abs);
    } catch {
      return [];
    }
  });
  const sources = sourceFiles.map((f) => ({ f, text: readFileSync(f, "utf8") }));

  const piGo = readFileSync(join(repoRoot, "apps", "cli", "internal", "engineconfig", "pi.go"), "utf8");
  const refusal = JSON.parse(piGo.match(/const SubagentRefusal = ("(?:[^"\\]|\\.)*")/)?.[1] ?? '""') as string;
  const refusalSentences = refusal
    .split(/(?<=\.)\s+/)
    .map((s) => s.trim())
    .filter(Boolean);

  test("the source and the refusal text exist", () => {
    expect(sources.map((s) => s.f)).toContain(extension);
    expect(refusal.length).toBeGreaterThan(0);
    expect(refusalSentences.length).toBeGreaterThan(1);
  });

  test("names no subagent tool", () => {
    expect(sources.filter((s) => s.text.includes("subagent_run")).map((s) => s.f)).toEqual([]);
  });

  test("holds no line or sentence of the refusal text", () => {
    for (const { f, text } of sources) {
      expect({ f, hits: [refusal, ...refusalSentences].filter((line) => text.includes(line)) }).toEqual({ f, hits: [] });
    }
  });

  test("reads no .gabo/VERSION and compares no versions", () => {
    for (const { f, text } of sources) {
      expect({ f, gaboVersion: /\.gabo|VERSION/.test(text) }).toEqual({ f, gaboVersion: false });
      expect({ f, semver: /semver|localeCompare|compareVersions/i.test(text) }).toEqual({ f, semver: false });
      expect({ f, compared: /version\w*\s*(===?|!==?|<=?|>=?)|(===?|!==?|<=?|>=?)\s*[\w.]*version/i.test(text) }).toEqual({
        f,
        compared: false,
      });
    }
  });
});
