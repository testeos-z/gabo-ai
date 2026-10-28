# Cloud agent image for working on gabo-ai (T071). Build from the repository
# root of a clean checkout (see infra/cloud-agent/README.md):
#
#   docker build -f infra/cloud-agent/Dockerfile -t gabo-cloud-agent .
#
# Every download is pinned: the base image by digest, Ubuntu packages through a
# dated snapshot of the signed archive, release files by SHA-256 (fetch-verified)
# and npm packages by the committed lockfile (npm ci). A mismatch fails the build.
# The image holds no secret; the operator supplies credentials at run time.
# This image is not a reviewer environment.

FROM ubuntu:24.04@sha256:24737cef17f4e22b1977df7d9a05e77c4e28b7da5b8911f8d8ccda136bf3573c

# --- pins (update together with README.md "Updating pins") -------------------
ARG UBUNTU_SNAPSHOT=20261027T000000Z
ARG ENGRAM_VERSION=3.6.0
ARG ENGRAM_SHA256=945fb471188826c251160947f8dda9fe148ba0dd095fa3453174fe7ab5c6479e
ARG GENTLE_AI_VERSION=4.3.1
ARG GENTLE_AI_SHA256=04ec685c56b42c29dff6db10fc7dc2c730217dc9c96736e53ad2acf3226dafb7
ARG GH_VERSION=2.102.0
ARG GH_SHA256=bb766f710eef8ede859c18578c72c327597cd4c8a85b06001b1f3843c6019386
ARG GO_VERSION=1.27.2
ARG GO_SHA256=ecbadb99091a3f46e31f5f934b068b1864eafa7995211b39eaddf76996045fe5
ARG NODE_VERSION=22.23.3
ARG NODE_SHA256=df450af89261115ef9f9e3830c3eeb2cc9213b63c720b1af623cb5dcbe2e02de
ARG BUN_VERSION=1.4.2
ARG BUN_SHA256=36368faef7527875d5ffa52e53cd48021741f2a83eb6208a8dd64068d422a913

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

# Only linux/amd64 checksums are pinned.
RUN [ "$(dpkg --print-architecture)" = amd64 ] || { echo "only linux/amd64 is supported" >&2; exit 1; }

# Ubuntu packages from a dated snapshot of the archive. apt verifies the
# InRelease signature with the keyring of the pinned base image and every
# index and package against the SHA-256 the signed index lists. The base image
# has no CA bundle, so ca-certificates is installed first with TLS peer checks
# off (integrity still rests on those signatures); everything after it uses TLS.
RUN rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*; \
    printf '%s\n' \
      'Types: deb' \
      "URIs: https://snapshot.ubuntu.com/ubuntu/${UBUNTU_SNAPSHOT}/" \
      'Suites: noble noble-updates noble-security' \
      'Components: main universe' \
      'Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg' \
      >/etc/apt/sources.list.d/ubuntu-snapshot.sources; \
    no_tls=(-o Acquire::https::Verify-Peer=false -o Acquire::https::Verify-Host=false); \
    apt-get "${no_tls[@]}" update; \
    DEBIAN_FRONTEND=noninteractive apt-get "${no_tls[@]}" install -y --no-install-recommends ca-certificates; \
    apt-get update; \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      curl git jq less openssh-client procps tini unzip xz-utils; \
    rm -rf /var/lib/apt/lists/*

COPY infra/cloud-agent/bin/ /usr/local/bin/

# Release binaries, each verified against its pinned SHA-256.
RUN tmp="$(mktemp -d)"; cd "$tmp"; \
    fetch-verified "https://github.com/Gentleman-Programming/engram/releases/download/v${ENGRAM_VERSION}/engram_${ENGRAM_VERSION}_linux_amd64.tar.gz" "$ENGRAM_SHA256" engram.tar.gz; \
    tar --no-same-owner -xzf engram.tar.gz engram; install -m 0755 engram /usr/local/bin/engram; \
    fetch-verified "https://github.com/Gentleman-Programming/gentle-ai/releases/download/v${GENTLE_AI_VERSION}/gentle-ai_${GENTLE_AI_VERSION}_linux_amd64.tar.gz" "$GENTLE_AI_SHA256" gentle-ai.tar.gz; \
    tar --no-same-owner -xzf gentle-ai.tar.gz gentle-ai; install -m 0755 gentle-ai /usr/local/bin/gentle-ai; \
    fetch-verified "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" "$GH_SHA256" gh.tar.gz; \
    tar --no-same-owner -xzf gh.tar.gz "gh_${GH_VERSION}_linux_amd64/bin/gh"; install -m 0755 "gh_${GH_VERSION}_linux_amd64/bin/gh" /usr/local/bin/gh; \
    fetch-verified "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" "$GO_SHA256" go.tar.gz; \
    tar -C /usr/local --no-same-owner -xzf go.tar.gz; \
    fetch-verified "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" "$NODE_SHA256" node.tar.xz; \
    tar -C /usr/local --strip-components=1 --no-same-owner -xJf node.tar.xz \
      "node-v${NODE_VERSION}-linux-x64/bin" "node-v${NODE_VERSION}-linux-x64/lib" "node-v${NODE_VERSION}-linux-x64/include"; \
    fetch-verified "https://github.com/oven-sh/bun/releases/download/bun-v${BUN_VERSION}/bun-linux-x64.zip" "$BUN_SHA256" bun.zip; \
    unzip -q -j bun.zip bun-linux-x64/bun; install -m 0755 bun /usr/local/bin/bun; \
    cd /; rm -rf "$tmp"

# Claude Code and codegraph from the committed lockfile: npm ci refuses a
# lockfile that does not match package.json and checks every package against
# its lockfile integrity hash. Install scripts are off; the one Claude Code
# needs (it links the native binary from the already verified platform
# package) runs explicitly and downloads nothing.
COPY infra/cloud-agent/npm/package.json infra/cloud-agent/npm/package-lock.json /opt/gabo-agent/npm/
RUN cd /opt/gabo-agent/npm; \
    npm ci --ignore-scripts --no-audit --no-fund --update-notifier=false; \
    (cd node_modules/@anthropic-ai/claude-code && node install.cjs); \
    ln -s /opt/gabo-agent/npm/node_modules/.bin/claude /usr/local/bin/claude; \
    ln -s /opt/gabo-agent/npm/node_modules/.bin/codegraph /usr/local/bin/codegraph; \
    npm cache clean --force

# Repository seed: a bare clone of the build context's own git history (no
# network, no credential; the source's git config is not copied). The
# entrypoint checks it out under the agent home and points origin at GitHub.
ARG GABO_REPO_URL=https://github.com/testeos-z/gabo-ai.git
RUN --mount=type=bind,source=.,target=/tmp/context \
    if [ -d /tmp/context/.git ]; then \
      git -c safe.directory='*' clone --quiet --bare --no-local /tmp/context/.git /opt/gabo-agent/seed.git; \
      git -C /opt/gabo-agent/seed.git remote set-url origin "$GABO_REPO_URL"; \
      chmod -R a+rX /opt/gabo-agent/seed.git; \
    else \
      echo "no .git in the build context: the repository is cloned at start instead" >&2; \
    fi

RUN useradd --create-home --shell /bin/bash --uid 10001 agent

ENV GOTOOLCHAIN=local \
    PATH=/usr/local/go/bin:/home/agent/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    DISABLE_AUTOUPDATER=1 \
    GABO_ENGRAM_VERSION=${ENGRAM_VERSION} \
    GABO_REPO_URL=${GABO_REPO_URL}

# Build-time registration for the image home; gabo-agent-start repeats it when
# a volume mounted on the home hides this file.
USER agent
RUN claude mcp add --transport stdio --scope user engram -- /usr/local/bin/engram mcp --tools=agent

# The entrypoint starts as root only to fix the ownership of a mounted home,
# then drops to the agent user.
USER root
WORKDIR /home/agent
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/gabo-agent-start"]
CMD ["sleep", "infinity"]
