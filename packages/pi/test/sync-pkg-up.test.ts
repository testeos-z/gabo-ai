// Sync of pi/pkg-up from main (.github/workflows/sync-pi-pkg-up.yml calls
// scripts/sync-pkg-up.sh). The push uses GITHUB_TOKEN, which GitHub refuses
// for any change under .github/workflows/, so the sync must keep pi/pkg-up's
// own workflow files. These tests run the script against a bare origin.
import { afterAll, describe, expect, test } from "bun:test";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";

const script = resolve(import.meta.dir, "..", "scripts", "sync-pkg-up.sh");
const scratch = mkdtempSync(join(tmpdir(), "gabo-pi-sync-pkg-up-"));
afterAll(() => rmSync(scratch, { recursive: true, force: true }));

const env = {
  PATH: process.env.PATH ?? "",
  HOME: scratch,
  GIT_CONFIG_GLOBAL: "/dev/null",
  GIT_CONFIG_NOSYSTEM: "1",
  GIT_AUTHOR_NAME: "test",
  GIT_AUTHOR_EMAIL: "test@example.com",
  GIT_COMMITTER_NAME: "test",
  GIT_COMMITTER_EMAIL: "test@example.com",
};

function git(cwd: string, ...args: string[]): string {
  const p = Bun.spawnSync(["git", ...args], { cwd, env });
  if (p.exitCode !== 0) throw new Error(`git ${args.join(" ")}: ${p.stderr.toString()}`);
  return p.stdout.toString().trim();
}

function write(repo: string, rel: string, text: string) {
  mkdirSync(dirname(join(repo, rel)), { recursive: true });
  writeFileSync(join(repo, rel), text);
}

function commit(repo: string, files: Record<string, string>, msg: string) {
  for (const [rel, text] of Object.entries(files)) write(repo, rel, text);
  git(repo, "add", "-A");
  git(repo, "commit", "-q", "-m", msg);
}

let caseNo = 0;

// A bare origin with main and pi/pkg-up at the same commit, plus a dev clone
// on main for adding commits.
function fixture() {
  const dir = join(scratch, `case-${++caseNo}`);
  const origin = join(dir, "origin.git");
  const dev = join(dir, "dev");
  mkdirSync(dir, { recursive: true });
  git(dir, "init", "-q", "--bare", "-b", "main", origin);
  git(dir, "clone", "-q", origin, dev);
  git(dev, "checkout", "-q", "-B", "main");
  commit(dev, { ".github/workflows/publish-pi.yml": "v1\n", "packages/pi/a.txt": "a1\n" }, "init");
  git(dev, "push", "-q", "origin", "main", "main:pi/pkg-up");
  return { dir, origin, dev };
}

// Runs the script in a fresh clone on main, like the workflow's checkout;
// inRepo runs the copy committed on main instead of this package's file.
function sync(f: { dir: string; origin: string }, inRepo?: string) {
  const runner = join(f.dir, `runner-${Math.random().toString(36).slice(2)}`);
  git(f.dir, "clone", "-q", f.origin, runner);
  const p = Bun.spawnSync(["bash", inRepo ?? script], { cwd: runner, env });
  return { code: p.exitCode, stdout: p.stdout.toString(), stderr: p.stderr.toString() };
}

const show = (repo: string, ref: string, path: string) => git(repo, "show", `${ref}:${path}`);
const changed = (repo: string, from: string, to: string) =>
  git(repo, "diff", "--name-only", from, to).split("\n").filter(Boolean);

describe("sync pi/pkg-up", () => {
  test("a normal change on main is merged into pi/pkg-up", () => {
    const f = fixture();
    commit(f.dev, { "packages/pi/a.txt": "a2\n" }, "change a");
    git(f.dev, "push", "-q", "origin", "main");
    const r = sync(f);
    expect(r.code).toBe(0);
    git(f.dev, "fetch", "-q", "origin");
    expect(show(f.dev, "origin/pi/pkg-up", "packages/pi/a.txt")).toBe("a2");
    expect(git(f.dev, "rev-parse", "origin/pi/pkg-up")).toBe(git(f.dev, "rev-parse", "origin/main"));
  });

  test("a workflow change on main keeps pi/pkg-up's workflows and merges the rest", () => {
    const f = fixture();
    const before = git(f.dev, "rev-parse", "origin/pi/pkg-up");
    commit(
      f.dev,
      { ".github/workflows/publish-pi.yml": "v2\n", ".github/workflows/new.yml": "n\n", "packages/pi/a.txt": "a2\n" },
      "change workflow and a",
    );
    git(f.dev, "push", "-q", "origin", "main");
    const r = sync(f);
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("kept pi/pkg-up's .github/workflows/");
    git(f.dev, "fetch", "-q", "origin");
    expect(show(f.dev, "origin/pi/pkg-up", "packages/pi/a.txt")).toBe("a2");
    expect(show(f.dev, "origin/pi/pkg-up", ".github/workflows/publish-pi.yml")).toBe("v1");
    expect(changed(f.dev, before, "origin/pi/pkg-up")).toEqual(["packages/pi/a.txt"]);
    // main is an ancestor, so the next sync starts from it.
    git(f.dev, "merge-base", "--is-ancestor", "origin/main", "origin/pi/pkg-up");
  });

  test("a later workflow change that conflicts still keeps pi/pkg-up's workflows", () => {
    const f = fixture();
    const before = git(f.dev, "rev-parse", "origin/pi/pkg-up");
    commit(f.dev, { ".github/workflows/publish-pi.yml": "v2\n" }, "workflow v2");
    git(f.dev, "push", "-q", "origin", "main");
    expect(sync(f).code).toBe(0);
    commit(f.dev, { ".github/workflows/publish-pi.yml": "v3\n", "packages/pi/a.txt": "a3\n" }, "workflow v3");
    git(f.dev, "push", "-q", "origin", "main");
    const r = sync(f);
    expect(r.code).toBe(0);
    git(f.dev, "fetch", "-q", "origin");
    expect(show(f.dev, "origin/pi/pkg-up", ".github/workflows/publish-pi.yml")).toBe("v1");
    expect(show(f.dev, "origin/pi/pkg-up", "packages/pi/a.txt")).toBe("a3");
    expect(changed(f.dev, before, "origin/pi/pkg-up")).toEqual(["packages/pi/a.txt"]);
  });

  test("a conflict outside the workflows fails and pushes nothing", () => {
    const f = fixture();
    git(f.dev, "checkout", "-q", "-b", "pkg", "origin/pi/pkg-up");
    commit(f.dev, { "packages/pi/a.txt": "pkg\n" }, "pkg only");
    git(f.dev, "push", "-q", "origin", "pkg:pi/pkg-up");
    const before = git(f.dev, "rev-parse", "pkg");
    git(f.dev, "checkout", "-q", "main");
    commit(f.dev, { "packages/pi/a.txt": "main\n" }, "main only");
    git(f.dev, "push", "-q", "origin", "main");
    const r = sync(f);
    expect(r.code).not.toBe(0);
    expect(r.stdout).toContain("::error::Merging main into pi/pkg-up conflicts");
    git(f.dev, "fetch", "-q", "origin");
    expect(git(f.dev, "rev-parse", "origin/pi/pkg-up")).toBe(before);
  });

  test("runs from main's checkout even though the checkout switches to pi/pkg-up", () => {
    const f = fixture();
    const rel = "packages/pi/scripts/sync-pkg-up.sh";
    commit(f.dev, { [rel]: readFileSync(script, "utf8"), ".github/workflows/publish-pi.yml": "v2\n" }, "add script");
    git(f.dev, "push", "-q", "origin", "main");
    const r = sync(f, rel);
    expect(r.code).toBe(0);
    git(f.dev, "fetch", "-q", "origin");
    expect(show(f.dev, "origin/pi/pkg-up", rel)).toBe(readFileSync(script, "utf8").trim());
    expect(show(f.dev, "origin/pi/pkg-up", ".github/workflows/publish-pi.yml")).toBe("v1");
  });
});
