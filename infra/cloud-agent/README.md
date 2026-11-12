# Cloud agent image (internal)

A container image that brings up a cloud agent ready to work on gabo-ai (T071). It replaces the hand setup on the persistent volume of the current Railway server, which was built from gbai-alejandria `infra/orca-server` and has no Engram in its image.

This image is **not a reviewer environment**: the reviewer gate keeps its own clone, home and scripts (TRD 8.3, D20). Agents in this image follow the gabo rules, so they merge only into `dev`, never `main` (D13).

## What the image holds

| item | how it is pinned |
|---|---|
| Base `ubuntu:24.04` | image index digest in `FROM` |
| Ubuntu packages (`ca-certificates curl git jq less openssh-client procps tini unzip xz-utils`) | dated snapshot of the archive (`UBUNTU_SNAPSHOT`); apt checks the signed index with the base image keyring and every package against its SHA-256 |
| Engram, gentle-ai, gh, Go, Node.js, Bun | version and SHA-256 build args; `bin/fetch-verified` fails the build on a mismatch. Node.js is the runtime only: its npm and corepack are removed |
| Claude Code, codegraph | `tools/bun.lock` (`bun install --frozen-lockfile` checks every package against its integrity hash); `tools/package.json` pins the exact versions |
| gabo-ai repository | a seed built from the build context's own git history in `/opt/gabo-agent/seed.git` (`bin/gabo-make-seed`) |

At build time Engram is registered as a user-scope Claude Code MCP server (`claude mcp add --transport stdio --scope user engram -- /usr/local/bin/engram mcp --tools=agent`).

Scripts in `bin/` (installed to `/usr/local/bin`):

- `gabo-agent-start`: the entrypoint. Started as root it fixes the ownership of the agent home (`/home/agent`, a fresh volume is owned by root) and continues as user `agent`. It registers the Engram MCP server again if the home does not have it, checks the repository out at `~/gabo-ai` on `dev` from the seed, first fetching `dev` from GitHub when the run-time git access allows it (or clones it when the image has no seed; without `dev` from either it logs a `WARNING` and leaves no checkout), runs one Engram Cloud sync in the background and then runs the container command (default `sleep infinity`).
- `gabo-agent-check`: the start checks (pinned `engram version`, `engram` in `claude mcp list`, tooling on `PATH`, repository checked out on `GABO_REPO_BRANCH`). Exits non-zero when one fails.
- `gabo-make-seed <git dir> <seed> <url> [branch]`: builds the seed at build time with only the target branch and `main`, each from the local branch or else the source's `origin` remote-tracking branch, so a fresh clone of the default branch still yields `dev` and no other local work of the builder ships in the image.
- `gabo-engram-sync [project]`: one Engram Cloud sync (export, then import) for `gabo-ai`. It skips with exit 0 when Engram reports that Engram Cloud is not configured.
- `fetch-verified <url> <sha256> <out>`: the checksum-verified download used by the build.

The image holds no secret, credential or token. GitHub access, the Claude login and the Engram Cloud configuration are supplied by the operator at run time through the hosting platform; this document deliberately prescribes no way of passing them.

## Build

From the root of a clean checkout of this repository, with BuildKit (the default builder since Docker 23):

```bash
docker build -f infra/cloud-agent/Dockerfile -t gabo-cloud-agent .
```

`Dockerfile.dockerignore` sends only `.git` and `infra/cloud-agent/` as the build context. The repository seed is built from that `.git` with the network disabled and no credential; the source's git config is not copied and `origin` points at `https://github.com/testeos-z/gabo-ai.git`. The Claude Code install script also runs with the network disabled. If the builder sends no `.git`, the build still succeeds and the entrypoint clones the repository at start, which then needs the operator's run-time GitHub access because the repository is private.

Only `linux/amd64` is pinned; other platforms fail the build.

## Run

```bash
docker run -d --name gabo-agent -v gabo-agent-home:/home/agent gabo-cloud-agent
docker exec -u agent gabo-agent gabo-agent-check
docker exec -it -u agent gabo-agent bash   # then: cd ~/gabo-ai && claude
```

Run `docker exec` as `-u agent`: the container starts as root only so the entrypoint can fix the home ownership. Mount a volume on `/home/agent` to keep the Claude login, the Engram store (`~/.engram`) and the checkout across restarts. A one-shot check of a fresh container:

```bash
docker run --rm gabo-cloud-agent gabo-agent-check
```

On Railway, build the service from this repository with `infra/cloud-agent/Dockerfile`, attach the volume at `/home/agent` and supply credentials at run time as the operator decides. The container keeps running with `sleep infinity`; agents run in sessions opened in it.

Non-secret settings read by `gabo-agent-start` (defaults in parentheses): `GABO_REPO_DIR` (`~/gabo-ai`), `GABO_REPO_URL` (the GitHub URL above), `GABO_REPO_BRANCH` (`dev`), `GABO_ENGRAM_SYNC_ON_START` (`1`; `0` skips the start sync) and `GABO_ENGRAM_PROJECT` for `gabo-engram-sync` (`gabo-ai`).

## Engram Cloud backend note (plan change #107)

The sync only works once the Engram Cloud backend lets this agent's principal into project `gabo-ai`:

- Managed Engram Cloud principals are deny-by-default: an operator must grant the principal access to the `gabo-ai` project explicitly on the server side. Until then `engram sync --cloud --project gabo-ai` fails (for example `blocked_unenrolled`) and `gabo-engram-sync` exits non-zero; the start sync only logs it.
- `ENGRAM_CLOUD_ALLOWED_PROJECTS` is a server setting that governs only legacy server credentials; it does not grant a managed principal access.

The grant is an operator task outside this image.

## Updating pins

Change one pin at a time, rebuild, and run the start checks.

| pin | where to read the new value |
|---|---|
| `FROM` digest | `docker buildx imagetools inspect ubuntu:24.04` (the index `Digest`) |
| `UBUNTU_SNAPSHOT` | a timestamp `YYYYMMDDTHHMMSSZ` served by `https://snapshot.ubuntu.com/ubuntu/<timestamp>/dists/noble/InRelease` |
| `ENGRAM_VERSION`, `ENGRAM_SHA256` | `checksums.txt` of the release in `Gentleman-Programming/engram` (`engram_<v>_linux_amd64.tar.gz`) |
| `GENTLE_AI_VERSION`, `GENTLE_AI_SHA256` | `checksums.txt` of the release in `Gentleman-Programming/gentle-ai` (`gentle-ai_<v>_linux_amd64.tar.gz`) |
| `GH_VERSION`, `GH_SHA256` | `gh_<v>_checksums.txt` of the release in `cli/cli` (`gh_<v>_linux_amd64.tar.gz`) |
| `GO_VERSION`, `GO_SHA256` | `https://go.dev/dl/?mode=json&include=all` (`go<v>.linux-amd64.tar.gz`); keep it at or above the `go` line of `apps/cli/go.mod`, since `GOTOOLCHAIN=local` forbids toolchain downloads |
| `NODE_VERSION`, `NODE_SHA256` | `https://nodejs.org/dist/v<v>/SHASUMS256.txt` (`node-v<v>-linux-x64.tar.xz`) |
| `BUN_VERSION`, `BUN_SHA256` | `SHASUMS256.txt` of the `bun-v<v>` release in `oven-sh/bun` (`bun-linux-x64.zip`); match `packageManager` in the root `package.json` |
| Claude Code, codegraph | edit the exact versions in `tools/package.json`, then `bun install --lockfile-only --ignore-scripts` in `infra/cloud-agent/tools` with the `BUN_VERSION` above and commit both files |

Download the file and compare `sha256sum` with the published value before committing a new checksum. Claude Code's self-updater is off (`DISABLE_AUTOUPDATER=1`), so the version changes only through the lockfile.

## Verify

- Script tests (stubbed `claude`, `engram` and `curl`; no Docker, network or credential): `infra/cloud-agent/tests/run.sh`. No project check runs them yet; run them by hand after changing a script.
- Build from a clean checkout, then `docker run --rm gabo-cloud-agent gabo-agent-check`: it must show `engram 3.6.0`, `engram` in `claude mcp list`, the tooling on `PATH` and the checkout.
- Secret scan over `infra/cloud-agent/` and the built image (for example its exported filesystem and `docker history`) with the team's scanner; it must find nothing.

## Review policy

The image runs agents unattended and handles a run-time credential, so this directory should be a human-only path. That is a separate `gabo.yaml` change (add `infra/cloud-agent/` to `review.human_only_paths`) that the owner merges; T071 does not edit `gabo.yaml`.
