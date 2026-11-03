// Manifest, tarball and bootstrap-skill rules of the gabo Pi package
// (intent 0003: S1, S3, D1, D5, D6; plan task T4).
import { afterAll, describe, expect, test } from "bun:test";
import { existsSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

const pkgDir = resolve(import.meta.dir, "..");
const repoRoot = resolve(pkgDir, "..", "..");
const pkg = JSON.parse(readFileSync(join(pkgDir, "package.json"), "utf8"));

const isHostPackage = (name: string) => name.startsWith("@earendil-works/") || name === "typebox";

const listDir = (dir: string) => (existsSync(dir) ? readdirSync(dir).sort() : []);

const shippedSkillDirs = listDir(join(pkgDir, "skills")).filter((d) =>
  existsSync(join(pkgDir, "skills", d, "SKILL.md")),
);
const shippedExtensions = listDir(join(pkgDir, "extensions")).filter((f) => f.endsWith(".ts"));

function frontmatterName(skillFile: string): string | undefined {
  const match = readFileSync(skillFile, "utf8").match(/^---\n[\s\S]*?^name:\s*["']?([^"'\n]+?)["']?\s*$/m);
  return match?.[1];
}

function filesUnder(dir: string, rel: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const abs = join(dir, entry);
    const relPath = `${rel}/${entry}`;
    return statSync(abs).isDirectory() ? filesUnder(abs, relPath) : [relPath];
  });
}

// A content line holds at least one letter or digit; blank lines, fences and
// rules are shared by every Markdown file and prove nothing about copying.
const contentLines = (text: string) =>
  text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => /[\p{L}\p{N}]/u.test(line));

describe("package.json manifest (S1, D5, D6)", () => {
  test("is published as @testeos-z/gabo-pi", () => {
    expect(pkg.name).toBe("@testeos-z/gabo-pi");
    expect(pkg.private).toBeUndefined();
  });

  test("has a release version x.y.z", () => {
    expect(pkg.version).toMatch(/^\d+\.\d+\.\d+$/);
  });

  test("declares the pi-package keyword", () => {
    expect(pkg.keywords).toContain("pi-package");
  });

  test("ships at least the gabo-pi skill and the gabo extension", () => {
    expect(shippedSkillDirs).toContain("gabo-pi");
    expect(shippedExtensions).toContain("gabo.ts");
  });

  test("pi.skills lists exactly the shipped skill directories", () => {
    expect([...(pkg.pi?.skills ?? [])].sort()).toEqual(shippedSkillDirs.map((d) => `./skills/${d}`));
  });

  test("pi.extensions lists exactly the shipped extension files", () => {
    expect([...(pkg.pi?.extensions ?? [])].sort()).toEqual(shippedExtensions.map((f) => `./extensions/${f}`));
  });

  test("Pi host packages appear only in peerDependencies, with range *", () => {
    for (const field of ["dependencies", "devDependencies", "optionalDependencies", "bundleDependencies"]) {
      const names = Object.keys(pkg[field] ?? {});
      expect({ field, hostPackages: names.filter(isHostPackage) }).toEqual({ field, hostPackages: [] });
    }
    const peers: Record<string, string> = pkg.peerDependencies ?? {};
    expect(Object.keys(peers)).toContain("@earendil-works/pi-coding-agent");
    for (const [name, range] of Object.entries(peers)) {
      expect({ name, host: isHostPackage(name), range }).toEqual({ name, host: true, range: "*" });
    }
  });

  test("the workspace test script runs bun test", () => {
    expect(pkg.scripts?.test).toBe("bun test");
  });
});

describe("npm pack --dry-run (S1, D1)", () => {
  const cache = mkdtempSync(join(tmpdir(), "gabo-pi-npm-cache-"));
  afterAll(() => rmSync(cache, { recursive: true, force: true }));

  const packed = (): string[] => {
    const run = Bun.spawnSync(["npm", "pack", "--dry-run", "--json"], {
      cwd: pkgDir,
      env: { ...process.env, npm_config_cache: cache, npm_config_update_notifier: "false" },
    });
    expect({ exitCode: run.exitCode, stderr: run.exitCode === 0 ? "" : run.stderr.toString() }).toEqual({
      exitCode: 0,
      stderr: "",
    });
    const [result] = JSON.parse(run.stdout.toString());
    return result.files.map((f: { path: string }) => f.path).sort();
  };

  test("holds package.json, the listed skills and extensions, and nothing else", () => {
    const files = packed();
    const listed = [
      ...(pkg.pi?.skills ?? []).flatMap((s: string) => filesUnder(join(pkgDir, s), s.replace(/^\.\//, ""))),
      ...(pkg.pi?.extensions ?? []).map((e: string) => e.replace(/^\.\//, "")),
    ];
    expect(files).toContain("package.json");
    for (const file of listed) expect(files).toContain(file);
    const unexpected = files.filter((f) => !["package.json", "README.md", "LICENSE"].includes(f) && !listed.includes(f));
    expect(unexpected).toEqual([]);
    expect(files.filter((f) => f.includes("gabo-reviewer/") || f.endsWith("gabo-reviewer"))).toEqual([]);
  });
});

describe("gabo-pi bootstrap skill (S3)", () => {
  const skillFile = join(pkgDir, "skills", "gabo-pi", "SKILL.md");
  const body = () => readFileSync(skillFile, "utf8");

  test("its frontmatter name equals its directory name", () => {
    for (const dir of shippedSkillDirs) expect(frontmatterName(join(pkgDir, "skills", dir, "SKILL.md"))).toBe(dir);
  });

  test("shares no name with a catalog skill, an agents.yaml skill or a gabo-<slot> link", () => {
    const catalogDir = join(repoRoot, "apps", "cli", "internal", "skills");
    const catalog = listDir(catalogDir).filter((d) => existsSync(join(catalogDir, d, "SKILL.md")));
    const catalogNames = catalog.flatMap((d) => [d, frontmatterName(join(catalogDir, d, "SKILL.md")) ?? d]);

    const agentsYaml = readFileSync(join(catalogDir, "gabo-agent-templates", "agents.yaml"), "utf8");
    const specNames = [...agentsYaml.matchAll(/^\s*skills:\s*\[([^\]]*)\]/gm)].flatMap((m) =>
      m[1].split(",").map((s) => s.trim()).filter(Boolean),
    );

    const projectKeys = readFileSync(join(repoRoot, "apps", "cli", "internal", "config", "projectkeys.go"), "utf8");
    const slots = [...projectKeys.matchAll(/^\s*Slot\w+\s*=\s*"([a-z_]+)"/gm)].map((m) => m[1]);
    const slotLinks = slots.map((s) => `gabo-${s.replaceAll("_", "-")}`);

    expect(catalog.length).toBeGreaterThan(0);
    expect(specNames.length).toBeGreaterThan(0);
    expect(slotLinks).toContain("gabo-methodology");

    const taken = new Set([...catalogNames, ...specNames, ...slotLinks]);
    for (const dir of shippedSkillDirs) {
      const names = [dir, frontmatterName(join(pkgDir, "skills", dir, "SKILL.md")) ?? dir];
      expect(names.filter((n) => taken.has(n))).toEqual([]);
    }
  });

  test("names the CLI install step, gabo init and gabo prompt", () => {
    const text = body();
    expect(text).toContain("go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@latest");
    expect(text).toContain("gabo init");
    expect(text).toContain("gabo prompt");
  });

  test("holds no line of docs/setup-prompt.md", () => {
    const prompt = new Set(contentLines(readFileSync(join(repoRoot, "docs", "setup-prompt.md"), "utf8")));
    expect(contentLines(body()).filter((line) => prompt.has(line))).toEqual([]);
  });
});

describe("extensions/gabo.ts stub", () => {
  test("is a no-op: it touches nothing on the extension API", async () => {
    const mod = await import(join(pkgDir, "extensions", "gabo.ts"));
    expect(typeof mod.default).toBe("function");
    const touched: PropertyKey[] = [];
    const pi = new Proxy({}, { get: (_t, key) => (touched.push(key), () => undefined) });
    expect(await mod.default(pi)).toBeUndefined();
    expect(touched).toEqual([]);
  });
});
