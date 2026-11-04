#!/usr/bin/env bash
# Decides whether publish-pi.yml publishes <name>@<version> to npm.
# Writes published=true|false to $GITHUB_OUTPUT:
#   npm view prints the version on stdout -> already published, nothing to do
#   npm view prints anything else         -> not on npm yet, publish
#   npm view fails with E404    -> not on npm yet, publish
#   any other npm view failure  -> exit 1, no decision
# Tested by packages/pi/test/publish-decision.test.ts with a fake npm.
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

# stdout and stderr are read apart: npm may warn on stderr even on success.
if out=$(npm view "${NAME}@${VERSION}" version 2>"$err"); then
  if [ "$(printf '%s' "$out" | tr -d '[:space:]')" = "$VERSION" ]; then
    echo "${NAME}@${VERSION} already published; nothing to do."
    echo "published=true" >> "$GITHUB_OUTPUT"
  else
    echo "${NAME}@${VERSION} not on npm yet."
    echo "published=false" >> "$GITHUB_OUTPUT"
  fi
elif grep -q 'E404' "$err" || printf '%s\n' "$out" | grep -q 'E404'; then
  echo "${NAME}@${VERSION} not on npm yet (E404)."
  echo "published=false" >> "$GITHUB_OUTPUT"
else
  echo "::error::npm view ${NAME}@${VERSION} failed:"
  printf '%s\n' "$out"
  cat "$err"
  exit 1
fi
