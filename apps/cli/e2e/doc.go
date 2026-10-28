// Package e2e holds the end-to-end mini wave (plan task T034). Its tests carry
// the build tag e2e, so the default `go test ./...` skips them:
//
//	go -C apps/cli test -tags e2e ./e2e/
//
// The test builds the gabo binary, runs `gabo init` and `gabo check` on a
// scratch sample repository with two dependent tasks, and plays one wave the
// way the gabo-cycle and gabo-coder skills describe it, against a fake
// tracker, a fake memory and a local bare repository standing in for GitHub.
// The installed user-level reviewer scripts review every PR head with fake
// gh and claude. Nothing touches a remote, a real tracker or a real memory.
// demo-mini-wave.sh covers the parts that need real Orca and agents.
package e2e
