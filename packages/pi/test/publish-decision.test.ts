// Publish decision of .github/workflows/publish-pi.yml: the workflow calls
// scripts/publish-decision.sh, which asks `bun info` whether the version is
// on npm and writes published=true|false to $GITHUB_OUTPUT. These tests run
// it against a fake `bun` on PATH. The 404 fixtures are verbatim stderr of
// bun 1.4.2 (the version pinned by package.json packageManager), captured
// against registry.npmjs.org on 2026-11-12:
//   bun info @gabo-ai/pi@9.9.9 version
//     error: No version of "@gabo-ai/pi" satisfying "9.9.9" found
//     Recent versions: ...
//   bun info @gabo-ai/nope-zz-probe 1.0.0 version
//     404 Not Found: https://registry.npmjs.org/@gabo-ai%2fnope-zz-probe
//      - '@gabo-ai/nope-zz-probe@latest' does not exist in this registry
import { afterAll, describe, expect, test } from "bun:test";
import { chmodSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

const script = resolve(import.meta.dir, "..", "scripts", "publish-decision.sh");
const scratch = mkdtempSync(join(tmpdir(), "gabo-pi-publish-decision-"));
afterAll(() => rmSync(scratch, { recursive: true, force: true }));

let caseNo = 0;

// fakeBun is the body of a bash `bun` that may print and exit; every call
// appends its arguments to calls.log.
function run(fakeBun: string, args = ["@gabo-ai/pi", "1.2.3"]) {
  const dir = join(scratch, `case-${++caseNo}`);
  const bin = join(dir, "bin");
  Bun.spawnSync(["mkdir", "-p", bin]);
  const calls = join(dir, "calls.log");
  const output = join(dir, "github-output");
  writeFileSync(output, "");
  const bun = join(bin, "bun");
  writeFileSync(bun, `#!/usr/bin/env bash\necho "$*" >> "${calls}"\n${fakeBun}\n`);
  chmodSync(bun, 0o755);
  const proc = Bun.spawnSync(["bash", script, ...args], {
    env: { PATH: `${bin}:${process.env.PATH}`, GITHUB_OUTPUT: output },
  });
  return {
    code: proc.exitCode,
    stdout: proc.stdout.toString(),
    stderr: proc.stderr.toString(),
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
    expect(r.calls).toEqual(["info @gabo-ai/pi@1.2.3 version"]);
  });

  test("a version missing from the registry takes the publish path", () => {
    const r = run(
      'echo "error: No version of \\"@gabo-ai/pi\\" satisfying \\"1.2.3\\" found" >&2; echo "" >&2; echo "Recent versions:" >&2; echo "- 0.2.6" >&2; exit 1',
    );
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("not on npm yet (404)");
    expect(r.output).toBe("published=false\n");
    expect(r.calls).toEqual(["info @gabo-ai/pi@1.2.3 version"]);
  });

  test("a package missing from the registry takes the publish path", () => {
    const r = run(
      'echo "" >&2; echo "404 Not Found: https://registry.npmjs.org/@gabo-ai%2fnope-zz-probe" >&2; echo "" >&2; echo " - \'@gabo-ai/nope-zz-probe@latest\' does not exist in this registry" >&2; exit 1',
    );
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("not on npm yet (404)");
    expect(r.output).toBe("published=false\n");
  });

  test("an empty bun info result takes the publish path", () => {
    const r = run("exit 0");
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("not on npm yet");
    expect(r.output).toBe("published=false\n");
  });

  test("any other bun info error fails without a decision", () => {
    const r = run('echo "error: ETIMEDOUT" >&2; exit 1');
    expect(r.code).not.toBe(0);
    expect(r.stdout).toContain("::error::bun info @gabo-ai/pi@1.2.3 failed:");
    expect(r.stdout).toContain("ETIMEDOUT");
    expect(r.output).toBe("");
  });

  test("a warning on stderr with empty stdout takes the publish path", () => {
    const r = run('echo "warn: some deprecated setting" >&2; exit 0');
    expect(r.code).toBe(0);
    expect(r.stdout).toContain("not on npm yet");
    expect(r.output).toBe("published=false\n");
  });

  test("a different version on stdout takes the publish path", () => {
    const r = run('echo "1.2.2"');
    expect(r.code).toBe(0);
    expect(r.output).toBe("published=false\n");
  });

  test("missing arguments exit 2 with the usage line and never call bun", () => {
    const r = run('echo "1.2.3"', []);
    expect(r.code).toBe(2);
    expect(r.stderr).toContain("usage: publish-decision.sh <name> <version>");
    expect(r.calls).toEqual([]);
    expect(r.output).toBe("");
  });
});
