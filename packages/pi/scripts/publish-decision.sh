#!/usr/bin/env bash
# Decides whether publish-pi.yml publishes <name>@<version> to npm.
# Writes published=true|false to $GITHUB_OUTPUT:
#   bun info prints the version on stdout -> already published, nothing to do
#   bun info prints anything else         -> not on npm yet, publish
#   bun info reports a 404 (the package, or this version, is not in the registry)
#                                         -> not on npm yet, publish
#   any other bun info failure            -> exit 1, no decision
# Tested by packages/pi/test/publish-decision.test.ts with a fake bun.
set -uo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: publish-decision.sh <name> <version>" >&2
  exit 2
fi
NAME="$1"
VERSION="$2"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT must be set}"

err=$(mktemp)
trap 'rm -f "$err"' EXIT

# bun info reads the package.json of its directory, so it runs in this package.
# stdout and stderr are read apart: bun may warn on stderr even on success.
if out=$(cd "$(dirname "$0")/.." && bun info "${NAME}@${VERSION}" version 2>"$err"); then
  if [ "$(printf '%s' "$out" | tr -d '[:space:]')" = "$VERSION" ]; then
    echo "${NAME}@${VERSION} already published; nothing to do."
    echo "published=true" >> "$GITHUB_OUTPUT"
  else
    echo "${NAME}@${VERSION} not on npm yet."
    echo "published=false" >> "$GITHUB_OUTPUT"
  fi
else
  text="$out
$(cat "$err")"
  case "$text" in
    *"404 Not Found"* | *"does not exist in this registry"* | *"No version of \"${NAME}\" satisfying \"${VERSION}\" found"*)
      echo "${NAME}@${VERSION} not on npm yet (404)."
      echo "published=false" >> "$GITHUB_OUTPUT"
      ;;
    *)
      echo "::error::bun info ${NAME}@${VERSION} failed:"
      printf '%s\n' "$out"
      cat "$err"
      exit 1
      ;;
  esac
fi
