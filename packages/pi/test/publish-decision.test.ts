// Publish decision of .github/workflows/publish-pi.yml: the workflow calls
// scripts/publish-decision.sh, which asks `npm view` whether the version is
// on npm and writes published=true|false to $GITHUB_OUTPUT. These tests run
// it against a fake `npm` on PATH.
import { afterAll, describe, expect, test } from "bun:test";
import { chmodSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

const script = resolve(import.meta.dir, "..", "scripts", "publish-decision.sh");
const scratch = mkdtempSync(join(tmpdir(), "gabo-pi-publish-decision-"));
afterAll(() => rmSync(scratch, { recursive: true, force: true }));

let caseNo = 0;

// fakeNpm is the body of a bash `npm` that may print and exit; every call
// appends its arguments to calls.log.
function run(fakeNpm: string) {
  const dir = join(scratch, `case-${++caseNo}`);
  const bin = join(dir, "bin");
  Bun.spawnSync(["mkdir", "-p", bin]);
  const calls = join(dir, "calls.log");
  const output = join(dir, "github-output");
  writeFileSync(output, "");
  const npm = join(bin, "npm");
  writeFileSync(npm, `#!/usr/bin/env bash\necho "$*" >> "${calls}"\n${fakeNpm}\n`);
  chmodSync(npm, 0o755);
  const proc = Bun.spawnSync(["bash", script, "@gabo-ai/pi", "1.2.3"], {
    env: { PATH: `${bin}:${process.env.PATH}`, GITHUB_OUTPUT: output },
  });
  return {
    code: proc.exitCode,
    stdout: proc.stdout.toString(),
    output: readFileSync(output, "utf8"),
    calls: existsSync(calls) ? readFileSync(calls, "utf8").trim().split("\n") : [],
  };
}

describe("publish decision", () => {
  test("an already published version exits 0, says so and does not publish", () => {
    const r = run('echo "1.2.3"');
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("@gabo-ai/pi@1.2.3 already published");
    expect(r.output).toBe("published=true\n");
    expect(r.calls).toEqual(["view @gabo-ai/pi@1.2.3 version"]);
  });

  test("an E404 from npm view takes the publish path", () => {
    const r = run('echo "npm error code E404" >&2; echo "npm error 404 Not Found" >&2; exit 1');
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("not on npm yet (E404)");
    expect(r.output).toBe("published=false\n");
    expect(r.calls).toEqual(["view @gabo-ai/pi@1.2.3 version"]);
  });

  test("an empty npm view result takes the publish path", () => {
    const r = run("exit 0");
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("not on npm yet");
    expect(r.output).toBe("published=false\n");
  });

  test("any other npm view error fails without a decision", () => {
    const r = run('echo "npm error code ETIMEDOUT" >&2; exit 1');
    expect(r.code).not.toBe(0);
    expect(r.stdout).toContain("::error::npm view @gabo-ai/pi@1.2.3 failed:");
    expect(r.stdout).toContain("ETIMEDOUT");
    expect(r.output).toBe("");
  });

  test("missing arguments fail without calling npm", () => {
    const proc = Bun.spawnSync(["bash", script], { env: { PATH: process.env.PATH, GITHUB_OUTPUT: "/dev/null" } });
    expect(proc.exitCode).not.toBe(0);
  });
});
