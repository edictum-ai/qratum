package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/edictum-ai/qratum/internal/capture"
	qschema "github.com/edictum-ai/qratum/internal/schema"
	qsource "github.com/edictum-ai/qratum/internal/source"
	"github.com/edictum-ai/qratum/internal/trust"
	"github.com/edictum-ai/qratum/internal/vault"
)

func TestEmittedStructSchemasMatchJSONTags(t *testing.T) {
	tests := []struct {
		version string
		typ     reflect.Type
	}{
		{version: "1.1.0", typ: reflect.TypeOf(adpStrictTrajectory{})},
		{version: qsource.CaptureEventSchemaVersion, typ: reflect.TypeOf(qsource.CaptureEvent{})},
		{version: qsource.CaptureStateSchemaVersion, typ: reflect.TypeOf(qsource.CaptureState{})},
		{version: "qratum.event.v1", typ: reflect.TypeOf(capture.Event{})},
		{version: "qratum.evidence.v1", typ: reflect.TypeOf(evidenceBundle{})},
		{version: "qratum.raw_ref.v1", typ: reflect.TypeOf(vault.RawRef{})},
		{version: "qratum.raw_tombstone.v1", typ: reflect.TypeOf(vault.RawTombstone{})},
		{version: qsource.PriceCatalogManifestSchemaVersion, typ: reflect.TypeOf(qsource.PriceCatalogManifest{})},
		{version: "qratum.redaction_summary.v1", typ: reflect.TypeOf(qratumRedactionSummary{})},
		{version: "qratum.review_card.v1", typ: reflect.TypeOf(reviewCard{})},
		{version: qsource.SessionRevisionSchemaVersion, typ: reflect.TypeOf(qsource.SessionRevision{})},
		{version: qsource.SessionTombstoneSchemaVersion, typ: reflect.TypeOf(qsource.SessionTombstone{})},
		{version: "qratum.session.v1", typ: reflect.TypeOf(qratumSession{})},
		{version: "qratum.trust_scorecard.v1", typ: reflect.TypeOf(trust.Scorecard{})},
		{version: qsource.UsageRecordSchemaVersion, typ: reflect.TypeOf(qsource.UsageRecord{})},
		{version: "qratum.vault_state.v1", typ: reflect.TypeOf(vault.State{})},
		{version: "qratum.ui.artifact_link.v1", typ: reflect.TypeOf(uiArtifactLink{})},
		{version: "qratum.ui.evidence_finding.v1", typ: reflect.TypeOf(uiEvidenceFinding{})},
		{version: "qratum.ui.review_card.v1", typ: reflect.TypeOf(uiReviewCardDTO{})},
		{version: "qratum.ui.session_detail.v1", typ: reflect.TypeOf(uiSessionDetail{})},
		{version: "qratum.ui.session_list_item.v1", typ: reflect.TypeOf(uiSessionListItem{})},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			schema := readRegisteredSchema(t, tt.version)
			if err := schemaTypeMismatch("$", schema, tt.typ); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSchemaParityRejectsLeafTypeAndRequiredDrift(t *testing.T) {
	type parityFixture struct {
		Count   int64  `json:"count"`
		Enabled bool   `json:"enabled"`
		Note    string `json:"note,omitempty"`
	}
	valid := map[string]any{
		"type":     "object",
		"required": []any{"count", "enabled"},
		"properties": map[string]any{
			"count":   map[string]any{"type": "integer"},
			"enabled": map[string]any{"type": "boolean"},
			"note":    map[string]any{"type": "string"},
		},
	}

	t.Run("leaf primitive", func(t *testing.T) {
		bad := cloneSchema(t, valid)
		bad["properties"].(map[string]any)["count"] = map[string]any{"type": "string"}
		if err := schemaTypeMismatch("$", bad, reflect.TypeOf(parityFixture{})); err == nil || !strings.Contains(err.Error(), "want integer") {
			t.Fatalf("type drift error = %v", err)
		}
	})
	t.Run("required field", func(t *testing.T) {
		bad := cloneSchema(t, valid)
		bad["required"] = []any{"enabled"}
		if err := schemaTypeMismatch("$", bad, reflect.TypeOf(parityFixture{})); err == nil || !strings.Contains(err.Error(), "required/omitempty mismatch") {
			t.Fatalf("required drift error = %v", err)
		}
	})
	t.Run("omitempty field", func(t *testing.T) {
		bad := cloneSchema(t, valid)
		bad["required"] = []any{"count", "enabled", "note"}
		if err := schemaTypeMismatch("$", bad, reflect.TypeOf(parityFixture{})); err == nil || !strings.Contains(err.Error(), "required/omitempty mismatch") {
			t.Fatalf("omitempty drift error = %v", err)
		}
	})
}

func cloneSchema(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var clone map[string]any
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func readRegisteredSchema(t *testing.T, version string) map[string]any {
	t.Helper()
	rel, ok := qschema.RegistryFile(version)
	if !ok {
		t.Fatalf("schema version %q is not registered", version)
	}
	// #nosec G304 -- rel comes from the in-process Qratum schema registry.
	data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode %s: %v", rel, err)
	}
	return schema
}

func schemaTypeMismatch(path string, schema map[string]any, typ reflect.Type) error {
	typ = unwrapReflectedType(typ)
	if typ == nil {
		return nil
	}

	switch typ.Kind() {
	case reflect.Struct:
		fields := jsonFieldTypes(typ)
		if len(fields) == 0 {
			return nil
		}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s: schema is missing object properties", path)
		}
		if got, want := sortedMapKeys(fields), sortedMapKeys(properties); !reflect.DeepEqual(got, want) {
			return fmt.Errorf("%s: JSON tag/schema property mismatch\nstruct: %v\nschema: %v", path, got, want)
		}
		required, err := schemaRequired(schema)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		wantRequired := make([]string, 0, len(fields))
		for name, field := range fields {
			if !field.omitempty {
				wantRequired = append(wantRequired, name)
			}
		}
		sort.Strings(wantRequired)
		if !reflect.DeepEqual(required, wantRequired) {
			return fmt.Errorf("%s: schema required/omitempty mismatch\nstruct required: %v\nschema required: %v", path, wantRequired, required)
		}
		for name, field := range fields {
			child, ok := properties[name].(map[string]any)
			if !ok {
				return fmt.Errorf("%s.%s: schema property must be an object", path, name)
			}
			if err := schemaTypeMismatch(path+"."+name, child, field.typ); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return nil
		}
		if schemaType(schema) != "array" {
			return fmt.Errorf("%s: schema type = %q, want array", path, schemaType(schema))
		}
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s: array schema is missing item schema", path)
		}
		return schemaTypeMismatch(path+"[]", items, typ.Elem())
	case reflect.Map:
		if schemaType(schema) != "object" {
			return fmt.Errorf("%s: map field schema type = %q, want object", path, schemaType(schema))
		}
	case reflect.Interface:
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return assertPrimitiveSchemaType(path, schema, "integer")
	case reflect.String:
		return assertPrimitiveSchemaType(path, schema, "string")
	case reflect.Bool:
		return assertPrimitiveSchemaType(path, schema, "boolean")
	}
	return nil
}

func unwrapReflectedType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

type jsonField struct {
	typ       reflect.Type
	omitempty bool
}

func jsonFieldTypes(typ reflect.Type) map[string]jsonField {
	fields := map[string]jsonField{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := jsonFieldName(field)
		if name == "" {
			continue
		}
		fields[name] = jsonField{typ: field.Type, omitempty: jsonTagHasOption(field, "omitempty")}
	}
	return fields
}

func jsonTagHasOption(field reflect.StructField, want string) bool {
	parts := strings.Split(field.Tag.Get("json"), ",")
	for _, option := range parts[1:] {
		if option == want {
			return true
		}
	}
	return false
}

func schemaRequired(schema map[string]any) ([]string, error) {
	raw, ok := schema["required"]
	if !ok {
		return []string{}, nil
	}
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("schema required must be an array")
	}
	required := make([]string, 0, len(values))
	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("schema required entries must be strings")
		}
		required = append(required, name)
	}
	sort.Strings(required)
	return required, nil
}

func assertPrimitiveSchemaType(path string, schema map[string]any, want string) error {
	got := schemaType(schema)
	if got == "" && schemaValuesHaveType(schema, "const", want) {
		got = want
	}
	if got == "" && schemaValuesHaveType(schema, "enum", want) {
		got = want
	}
	if got != want {
		return fmt.Errorf("%s: schema type = %q, want %s", path, got, want)
	}
	return nil
}

func schemaValuesHaveType(schema map[string]any, keyword string, want string) bool {
	value, ok := schema[keyword]
	if !ok {
		return false
	}
	values := []any{value}
	if keyword == "enum" {
		var arrayOK bool
		values, arrayOK = value.([]any)
		if !arrayOK || len(values) == 0 {
			return false
		}
	}
	for _, candidate := range values {
		candidateType := reflect.TypeOf(candidate)
		if candidateType == nil {
			return false
		}
		matches := (want == "string" && candidateType.Kind() == reflect.String) ||
			(want == "boolean" && candidateType.Kind() == reflect.Bool)
		if !matches {
			return false
		}
	}
	return true
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		name = field.Name
	}
	return name
}

func schemaType(schema map[string]any) string {
	value, _ := schema["type"].(string)
	return value
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
