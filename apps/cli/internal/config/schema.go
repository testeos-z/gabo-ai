package config

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

// The schemas at the repository root (schemas/) are the one source; this
// package embeds a byte-identical copy so validation works offline, and a
// test fails when the copy drifts.
//
//go:generate cp ../../../../schemas/gbai.user.schema.json ../../../../schemas/gbai.project.schema.json schemas/
//go:embed schemas/*.schema.json
var schemaFS embed.FS

// Schema file names, as published under schemas/.
const (
	UserSchemaFile    = "gbai.user.schema.json"
	ProjectSchemaFile = "gbai.project.schema.json"
)

// SchemaFileNames lists the embedded schema files.
var SchemaFileNames = []string{UserSchemaFile, ProjectSchemaFile}

// schemaBaseURL is the $id prefix both schema files declare.
const schemaBaseURL = "https://raw.githubusercontent.com/testeos-z/gbai-team/main/schemas/"

// Schema selects what a document is validated as.
type Schema string

// Schemas.
const (
	// SchemaUser is the global user file: personal keys plus any project
	// key as a fallback, none required.
	SchemaUser Schema = UserSchemaFile
	// SchemaProject is a complete gbai.yaml as editors check it: project
	// keys only, required keys set.
	SchemaProject Schema = ProjectSchemaFile
	// SchemaLayer is one file as the loader reads it: every key class, none
	// required (required keys are checked on the merged result).
	SchemaLayer Schema = ProjectSchemaFile + "#/$defs/layer"
	// SchemaFile is a complete file with every key class, the schema twin
	// of Parse.
	SchemaFile Schema = ProjectSchemaFile + "#/$defs/file"
)

var (
	compiled       = sync.OnceValues(compileSchemas)
	englishPrinter = message.NewPrinter(language.English)
)

func compileSchemas() (map[Schema]*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	for _, name := range SchemaFileNames {
		b, err := schemaFS.ReadFile("schemas/" + name)
		if err != nil {
			return nil, err
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("schema %s: %w", name, err)
		}
		if err := c.AddResource(schemaBaseURL+name, doc); err != nil {
			return nil, fmt.Errorf("schema %s: %w", name, err)
		}
	}
	out := map[Schema]*jsonschema.Schema{}
	for _, s := range []Schema{SchemaUser, SchemaProject, SchemaLayer, SchemaFile} {
		sch, err := c.Compile(schemaBaseURL + string(s))
		if err != nil {
			return nil, fmt.Errorf("compile schema %s: %w", s, err)
		}
		out[s] = sch
	}
	return out, nil
}

// ValidateSchema validates a YAML document against s. A failure lists one
// "<key path>: <reason>" line per failing value.
func ValidateSchema(s Schema, b []byte) error {
	var doc any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("decode yaml: %w", err)
	}
	if doc == nil {
		// An empty document is an empty mapping, as the loader reads it.
		doc = map[string]any{}
	}
	return validateValue(s, doc)
}

// validateValue validates a decoded YAML value against s. The value goes
// through JSON so it has the types the validator expects.
func validateValue(s Schema, v any) error {
	schemas, err := compiled()
	if err != nil {
		return err
	}
	sch, ok := schemas[s]
	if !ok {
		return fmt.Errorf("unknown schema %q", s)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("convert yaml to json: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("convert yaml to json: %w", err)
	}
	err = sch.Validate(inst)
	var verr *jsonschema.ValidationError
	if !errors.As(err, &verr) {
		return err
	}
	return schemaErrors(verr)
}

// schemaErrors flattens a validation error into one "<key path>: <reason>"
// line per failing value, sorted and without duplicates. Only the leaves of
// the error tree are kept: combinators just repeat their causes.
func schemaErrors(verr *jsonschema.ValidationError) error {
	var lines []string
	var walk func(e *jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			lines = append(lines, keyPath(e.InstanceLocation)+": "+e.ErrorKind.LocalizedString(englishPrinter))
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(verr)
	slices.Sort(lines)
	lines = slices.Compact(lines)
	errs := make([]error, 0, len(lines))
	for _, l := range lines {
		errs = append(errs, errors.New(l))
	}
	return errors.Join(errs...)
}

// keyPath joins an instance location into a dotted key path; the root is
// "(root)".
func keyPath(loc []string) string {
	if len(loc) == 0 {
		return "(root)"
	}
	return strings.Join(loc, ".")
}
