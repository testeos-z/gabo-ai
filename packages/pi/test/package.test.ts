// Manifest, tarball and skill rules of the gabo Pi package
// (intent 0003: S1, S3, S10, D1, D5, D6; plan tasks T4 and T5).
import { describe, expect, test } from "bun:test";
import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { join, resolve } from "node:path";

const pkgDir = resolve(import.meta.dir, "..");
const repoRoot = resolve(pkgDir, "..", "..");
const pkg = JSON.parse(readFileSync(join(pkgDir, "package.json"), "utf8"));

const isHostPackage = (name: string) => name.startsWith("@earendil-works/") || name === "typebox";

const listDir = (dir: string) => (existsSync(dir) ? readdirSync(dir).sort() : []);

const shippedSkillDirs = listDir(join(pkgDir, "skills")).filter((d) =>
  existsSync(join(pkgDir, "skills", d, "SKILL.md")),
);
const catalogDir = join(repoRoot, "apps", "cli", "internal", "skills");
const reviewerConfig = "gabo-reviewer-config";
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

// A shipped skill directory that equals its catalog folder file by file and
// byte for byte (S10). Only such a copy may carry a catalog skill name.
function isCatalogCopy(dir: string): boolean {
  const shipped = join(pkgDir, "skills", dir);
  const source = join(catalogDir, dir);
  if (!existsSync(join(source, "SKILL.md"))) return false;
  const files = filesUnder(shipped, ".").sort();
  if (JSON.stringify(files) !== JSON.stringify(filesUnder(source, ".").sort())) return false;
  return files.every((f) => readFileSync(join(shipped, f)).equals(readFileSync(join(source, f))));
}

// A content line holds at least one letter or digit; blank lines, fences and
// rules are shared by every Markdown file and prove nothing about copying.
const contentLines = (text: string) =>
  text
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => /[\p{L}\p{N}]/u.test(line));

// `bun pm pack --dry-run` prints one "packed <size> <path>" line per file.
const packed = (): string[] => {
  const run = Bun.spawnSync(["bun", "pm", "pack", "--dry-run"], { cwd: pkgDir });
  expect({ exitCode: run.exitCode, stderr: run.exitCode === 0 ? "" : run.stderr.toString() }).toEqual({
    exitCode: 0,
    stderr: "",
  });
  const files = [...run.stdout.toString().matchAll(/^packed \S+ (.+)$/gm)].map((m) => m[1]).sort();
  // An unparsed listing must fail here, not pass the exclusion checks below vacuously.
  expect(files).toContain("package.json");
  return files;
};

describe("package.json manifest (S1, D5, D6)", () => {
  test("is published as @gabo-ai/pi", () => {
    expect(pkg.name).toBe("@gabo-ai/pi");
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

describe("bun pm pack --dry-run (S1, D1)", () => {
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

    const fixed = new Set([...specNames, ...slotLinks]);
    const catalogTaken = new Set(catalogNames);
    for (const dir of shippedSkillDirs) {
      const names = [dir, frontmatterName(join(pkgDir, "skills", dir, "SKILL.md")) ?? dir];
      const taken = (n: string) => fixed.has(n) || (catalogTaken.has(n) && !isCatalogCopy(dir));
      expect({ dir, shared: names.filter(taken) }).toEqual({ dir, shared: [] });
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

describe("extensions/gabo.ts at load", () => {
  test("only registers event handlers through pi.on", async () => {
    const mod = await import(join(pkgDir, "extensions", "gabo.ts"));
    expect(typeof mod.default).toBe("function");
    const touched: PropertyKey[] = [];
    const pi = new Proxy({}, { get: (_t, key) => (touched.push(key), () => undefined) });
    expect(await mod.default(pi)).toBeUndefined();
    expect(touched.length).toBeGreaterThan(0);
    expect(touched.filter((key) => key !== "on")).toEqual([]);
  });
});

describe("gabo-reviewer-config skill (S10, S1, S3)", () => {
  const shipped = join(pkgDir, "skills", reviewerConfig);
  const source = join(catalogDir, reviewerConfig);

  test("is byte-identical, file by file, to its catalog folder", () => {
    expect(existsSync(shipped)).toBe(true);
    const sourceFiles = filesUnder(source, ".").sort();
    expect(sourceFiles).toContain("./scripts/isolation-test.sh");
    expect(filesUnder(shipped, ".").sort()).toEqual(sourceFiles);
    for (const file of sourceFiles) {
      const same = readFileSync(join(shipped, file)).equals(readFileSync(join(source, file)));
      const sameMode = statSync(join(shipped, file)).mode === statSync(join(source, file)).mode;
      expect({ file, same, sameMode }).toEqual({ file, same: true, sameMode: true });
    }
  });

  test("is listed in pi.skills", () => {
    expect(pkg.pi?.skills ?? []).toContain(`./skills/${reviewerConfig}`);
  });

  test("bun pm pack holds every file of its catalog folder", () => {
    const files = packed();
    for (const file of filesUnder(source, `skills/${reviewerConfig}`)) expect(files).toContain(file);
    expect(files.filter((f) => f.split("/").includes("gabo-reviewer"))).toEqual([]);
  });

  test("no shipped skill invokes the reviewer scripts", () => {
    for (const dir of shippedSkillDirs) {
      for (const file of filesUnder(join(pkgDir, "skills", dir), dir)) {
        const text = readFileSync(join(pkgDir, "skills", file), "utf8");
        expect({ file, invokesReviewer: /reviewer\.sh/.test(text) }).toEqual({ file, invokesReviewer: false });
      }
    }
  });
});
