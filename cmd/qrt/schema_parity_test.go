package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/edictum-ai/qratum/internal/capture"
	qschema "github.com/edictum-ai/qratum/internal/schema"
	"github.com/edictum-ai/qratum/internal/trust"
	"github.com/edictum-ai/qratum/internal/vault"
)

func TestEmittedStructSchemasMatchJSONTags(t *testing.T) {
	tests := []struct {
		version string
		typ     reflect.Type
	}{
		{version: "1.1.0", typ: reflect.TypeOf(adpStrictTrajectory{})},
		{version: "qratum.event.v1", typ: reflect.TypeOf(capture.Event{})},
		{version: "qratum.evidence.v1", typ: reflect.TypeOf(evidenceBundle{})},
		{version: "qratum.raw_ref.v1", typ: reflect.TypeOf(vault.RawRef{})},
		{version: "qratum.raw_tombstone.v1", typ: reflect.TypeOf(vault.RawTombstone{})},
		{version: "qratum.redaction_summary.v1", typ: reflect.TypeOf(qratumRedactionSummary{})},
		{version: "qratum.review_card.v1", typ: reflect.TypeOf(reviewCard{})},
		{version: "qratum.session.v1", typ: reflect.TypeOf(qratumSession{})},
		{version: "qratum.trust_scorecard.v1", typ: reflect.TypeOf(trust.Scorecard{})},
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
			assertSchemaMatchesType(t, "$", schema, tt.typ)
		})
	}
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

func assertSchemaMatchesType(t *testing.T, path string, schema map[string]any, typ reflect.Type) {
	t.Helper()
	typ = unwrapReflectedType(typ)
	if typ == nil {
		return
	}

	switch typ.Kind() {
	case reflect.Struct:
		fields := jsonFieldTypes(typ)
		if len(fields) == 0 {
			return
		}
		properties := schemaProperties(t, path, schema)
		assertSameStringSet(t, path, sortedMapKeys(fields), sortedMapKeys(properties))
		for name, fieldType := range fields {
			child, ok := properties[name].(map[string]any)
			if !ok {
				t.Fatalf("%s.%s: schema property must be an object", path, name)
			}
			assertSchemaMatchesType(t, path+"."+name, child, fieldType)
		}
	case reflect.Slice, reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return
		}
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return
		}
		assertSchemaMatchesType(t, path+"[]", items, typ.Elem())
	case reflect.Map:
		if schemaType(schema) != "object" {
			t.Fatalf("%s: map field schema type = %q, want object", path, schemaType(schema))
		}
	case reflect.Interface:
		return
	}
}

func unwrapReflectedType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

func jsonFieldTypes(typ reflect.Type) map[string]reflect.Type {
	fields := map[string]reflect.Type{}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := jsonFieldName(field)
		if name == "" {
			continue
		}
		fields[name] = field.Type
	}
	return fields
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

func schemaProperties(t *testing.T, path string, schema map[string]any) map[string]any {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s: schema is missing object properties", path)
	}
	return properties
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

func assertSameStringSet(t *testing.T, path string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: JSON tag/schema property mismatch\nstruct: %v\nschema: %v", path, got, want)
	}
}
