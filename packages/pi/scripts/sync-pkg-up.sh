#!/usr/bin/env bash
# Merges origin/main into pi/pkg-up and pushes it, keeping pi/pkg-up's own
# .github/workflows/ (called by .github/workflows/sync-pi-pkg-up.yml).
#
# The push uses GITHUB_TOKEN, and GitHub refuses a GITHUB_TOKEN push that
# adds or changes anything under .github/workflows/. So workflow changes on
# main never reach pi/pkg-up through this sync:
#   - main touches no workflow since pi/pkg-up: plain merge (fast-forward
#     when possible);
#   - otherwise: a real merge commit (--no-ff) whose .github/workflows/ is
#     pi/pkg-up's pre-merge tree; a conflict only in workflow files is
#     resolved the same way;
#   - any other conflict fails and pushes nothing.
# The owner updates workflows on pi/pkg-up by hand (packages/pi/README.md).
# Tested by packages/pi/test/sync-pkg-up.test.ts against a bare origin.
set -euo pipefail

# The body is one function so bash parses it whole before the checkout below
# replaces this file in the working tree.
main() {
  local WF=.github/workflows base

  git fetch origin main pi/pkg-up
  git checkout -B pi/pkg-up origin/pi/pkg-up
  base=$(git rev-parse HEAD)

  if git diff --quiet "$base" origin/main -- "$WF"; then
    if ! git merge --no-edit origin/main; then
      echo "::error::Merging main into pi/pkg-up conflicts; resolve it by hand."
      git merge --abort || true
      exit 1
    fi
  else
    if ! git merge --no-ff --no-commit origin/main; then
      if git diff --name-only --diff-filter=U | grep -qv "^$WF/"; then
        echo "::error::Merging main into pi/pkg-up conflicts; resolve it by hand."
        git merge --abort || true
        exit 1
      fi
    fi
    # Drop every workflow path main added, then restore pi/pkg-up's own.
    git rm -r -q --cached --ignore-unmatch -- "$WF" >/dev/null
    rm -rf "$WF"
    if git cat-file -e "$base:$WF" 2>/dev/null; then
      git checkout "$base" -- "$WF"
    fi
    git commit -q --no-edit
    echo "Merged main into pi/pkg-up and kept pi/pkg-up's $WF/ (workflow changes on main are not synced)."
  fi

  if ! git diff --quiet "$base" HEAD -- "$WF"; then
    echo "::error::The sync would change $WF/; refusing to push."
    exit 1
  fi

  git push origin pi/pkg-up
}

main "$@"
