// Package configcmd shows which config files gbai reads and the resolved
// values (`gbai config`). It is read-only.
package configcmd

import (
	"context"
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
)

// Resolve returns the layers looked up and the resolved config; on error the
// layers are still filled in.
type Resolve func(context.Context) (config.Resolution, error)

// Report is what `gbai config` prints.
type Report struct {
	Layers []config.Layer
	// YAML is the resolved config, defaults included, with credential-looking
	// values redacted. Empty when resolution failed.
	YAML []byte
}

// UseCase resolves and renders the config.
type UseCase struct{ resolve Resolve }

// NewUseCase returns the use case over resolve.
func NewUseCase(resolve Resolve) *UseCase { return &UseCase{resolve: resolve} }

// Show resolves the config. The report carries the layers even on error.
func (u *UseCase) Show(ctx context.Context) (Report, error) {
	res, err := u.resolve(ctx)
	rep := Report{Layers: res.Layers}
	if err != nil {
		return rep, err
	}
	b, err := res.Config.Marshal()
	if err != nil {
		return rep, err
	}
	if rep.YAML, err = Redact(b); err != nil {
		return rep, err
	}
	return rep, nil
}

// secretKey matches mapping keys whose value would be a credential. Keys that
// only name where a credential lives (pi_credential_env) do not match.
var secretKey = regexp.MustCompile(`(?i)(^|_)(token|secret|secrets|password|passwd|passphrase|api_?key|private_key|credential|credentials)$`)

// Redacted replaces every redacted value.
const Redacted = "[redacted]"

// Redact replaces the value of every credential-looking key, at any depth,
// with Redacted. The config holds no secret today; this keeps a future field
// from leaking through `gbai config`.
func Redact(doc []byte) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(doc, &root); err != nil {
		return nil, fmt.Errorf("redact config: %w", err)
	}
	redactNode(&root)
	b, err := yaml.Marshal(&root)
	if err != nil {
		return nil, fmt.Errorf("redact config: %w", err)
	}
	return b, nil
}

func redactNode(n *yaml.Node) {
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if secretKey.MatchString(n.Content[i].Value) {
				n.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: Redacted}
				continue
			}
			redactNode(n.Content[i+1])
		}
		return
	}
	for _, c := range n.Content {
		redactNode(c)
	}
}
