package check

import (
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// Intent 0029 T2 (spec S1): a repository_host adapter without the `protect
// branches` heading is listed as missing that operation, against the real
// contract text of the catalog.
func TestRepositoryHostAdapterWithoutProtectBranchesIsListedMissing(t *testing.T) {
	t.Parallel()
	const contractPath = "gabo-contract-repository-host/SKILL.md"
	contract := catalogFile(t, contractPath)
	if !slices.Contains(operations([]byte(contract)), "protect branches") {
		t.Fatal("the repository_host contract does not name the protect branches operation")
	}
	// A scratch adapter that covers every other operation of the contract.
	var adapter strings.Builder
	adapter.WriteString("## Operations\n\n")
	for _, op := range operations([]byte(contract)) {
		if op != "protect branches" {
			adapter.WriteString("### " + op + "\n\n")
		}
	}
	const cfg = validConfig + "slots:\n  repository_host:\n    adapter: docs/adapters/host.md\n"
	proj := with(project(), map[string]string{
		CatalogDir + "/" + contractPath: contract,
		"docs/adapters/host.md":         adapter.String(),
	})
	r := run(t, proj, configFiles{projectFile: cfg}, Settings{})
	wantFailed(t, r, "✗ adapter docs/adapters/host.md does not cover: protect branches")
}

// catalogFile returns a catalog skill file as the binary embeds it.
func catalogFile(t *testing.T, path string) string {
	t.Helper()
	files, err := skills.Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Path == path {
			return string(f.Data)
		}
	}
	t.Fatalf("catalog has no %s", path)
	return ""
}
