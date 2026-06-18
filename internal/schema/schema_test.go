package schema

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestValidatorRejectsInjectedExtraKeys(t *testing.T) {
	schema := []byte(`{
	  "type": "object",
	  "required": ["schema_version", "items"],
	  "additionalProperties": false,
	  "properties": {
	    "schema_version": {"const": "qratum.test.v1"},
	    "items": {
	      "type": "array",
	      "items": {
	        "type": "object",
	        "required": ["name"],
	        "additionalProperties": false,
	        "properties": {
	          "name": {"type": "string"},
	          "count": {"type": "integer"},
	          "state": {"enum": ["GREEN", "KNOWN-RED", "BLOCKING-RED"]}
	        }
	      }
	    }
	  }
	}`)
	good := []byte(`{"schema_version":"qratum.test.v1","items":[{"name":"d3","count":1,"state":"GREEN"}]}`)
	if err := Validate(schema, good); err != nil {
		t.Fatalf("good instance rejected: %v", err)
	}

	extraTop := []byte(`{"schema_version":"qratum.test.v1","items":[{"name":"d3"}],"extra":true}`)
	if err := Validate(schema, extraTop); err == nil || !strings.Contains(err.Error(), `additional property "extra"`) {
		t.Fatalf("extra top-level key error = %v, want additional property rejection", err)
	}
	extraNested := []byte(`{"schema_version":"qratum.test.v1","items":[{"name":"d3","leak":"secret"}]}`)
	if err := Validate(schema, extraNested); err == nil || !strings.Contains(err.Error(), `additional property "leak"`) {
		t.Fatalf("extra nested key error = %v, want additional property rejection", err)
	}
}

func TestValidatorRejectsRequiredConstEnumAndTypeDrift(t *testing.T) {
	schema := []byte(`{
	  "type": "object",
	  "required": ["schema_version", "headline", "count"],
	  "additionalProperties": false,
	  "properties": {
	    "schema_version": {"const": "qratum.score.v1"},
	    "headline": {"enum": ["TRUSTED", "NOT-TRUSTED"]},
	    "count": {"type": "integer"}
	  }
	}`)
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "missing", data: `{"schema_version":"qratum.score.v1","headline":"TRUSTED"}`, want: `missing required property "count"`},
		{name: "const", data: `{"schema_version":"qratum.score.v2","headline":"TRUSTED","count":1}`, want: "does not equal const"},
		{name: "enum", data: `{"schema_version":"qratum.score.v1","headline":"MAYBE","count":1}`, want: "is not in enum"},
		{name: "type", data: `{"schema_version":"qratum.score.v1","headline":"TRUSTED","count":"1"}`, want: "is not type integer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(schema, []byte(tt.data))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestDataClassLatticeIsStrictAndStable(t *testing.T) {
	want := []string{"raw", "redacted", "review", "corpus", "published"}
	if !reflect.DeepEqual(DataClassLattice, want) {
		t.Fatalf("data class lattice = %v, want %v", DataClassLattice, want)
	}
	seen := map[string]struct{}{}
	for i, value := range DataClassLattice {
		if _, exists := seen[value]; exists {
			t.Fatalf("duplicate data class %q", value)
		}
		seen[value] = struct{}{}
		rank, ok := DataClassRank(value)
		if !ok || rank != i {
			t.Fatalf("DataClassRank(%q) = %d, %t; want %d, true", value, rank, ok, i)
		}
	}
	if _, ok := DataClassRank("secret"); ok {
		t.Fatal("DataClassRank accepted unknown class")
	}
}

func TestRegistryIsUniqueAndKnownVersionsResolve(t *testing.T) {
	seenVersion := map[string]string{}
	seenFile := map[string]string{}
	for _, entry := range Registry {
		if entry.Version == "" || entry.File == "" {
			t.Fatalf("empty registry entry: %#v", entry)
		}
		if previous, exists := seenVersion[entry.Version]; exists {
			t.Fatalf("duplicate version %q in %s and %s", entry.Version, previous, entry.File)
		}
		if previous, exists := seenFile[entry.File]; exists {
			t.Fatalf("duplicate schema file %q for %s and %s", entry.File, previous, entry.Version)
		}
		seenVersion[entry.Version] = entry.File
		seenFile[entry.File] = entry.Version
		file, ok := RegistryFile(entry.Version)
		if !ok || file != entry.File {
			t.Fatalf("RegistryFile(%q) = %q, %t; want %q, true", entry.Version, file, ok, entry.File)
		}
		if !IsKnownVersion(entry.Version) {
			t.Fatalf("IsKnownVersion(%q) = false", entry.Version)
		}
	}
	if IsKnownVersion("qratum.missing.v1") {
		t.Fatal("unknown schema version resolved")
	}
}

func TestEnumValues(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"data_class":{"enum":["raw","redacted"]}}}`)
	values, err := EnumValues(schema, "data_class")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(values, []string{"raw", "redacted"}) {
		t.Fatalf("enum values = %v", values)
	}
	_, err = EnumValues([]byte(`{"type":"object","properties":{}}`), "data_class")
	if err == nil {
		t.Fatal("EnumValues missing enum error = nil")
	}
}

func TestValidatorUsesJSONNumbersForConstEquality(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"n":{"const":1}}}`)
	data := []byte(`{"n":1}`)
	if err := Validate(schema, data); err != nil {
		t.Fatalf("numeric const rejected: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
}

func TestRegistrySchemaFilesExistAndDeclareVersion(t *testing.T) {
	root := repoRoot(t)
	for _, entry := range Registry {
		t.Run(entry.Version, func(t *testing.T) {
			data := readRepoFile(t, root, entry.File)
			var schema map[string]any
			decodeUseNumber(t, data, &schema)
			properties, ok := schema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("%s has no properties object", entry.File)
			}
			schemaVersion, ok := properties["schema_version"].(map[string]any)
			if !ok {
				t.Fatalf("%s has no schema_version property", entry.File)
			}
			if got := schemaVersion["const"]; got != entry.Version {
				t.Fatalf("%s schema_version const = %v, want %q", entry.File, got, entry.Version)
			}
			if _, ok := properties["data_class"].(map[string]any); !ok {
				t.Fatalf("%s has no data_class property", entry.File)
			}
		})
	}
}

func TestSchemasAreRecursiveStrictExceptDeclaredOpenMaps(t *testing.T) {
	root := repoRoot(t)
	for _, entry := range Registry {
		t.Run(entry.Version, func(t *testing.T) {
			var schema map[string]any
			decodeUseNumber(t, readRepoFile(t, root, entry.File), &schema)
			assertRecursiveStrict(t, "$", schema)
		})
	}
}

func TestSchemaDataClassEnumsMatchLattice(t *testing.T) {
	root := repoRoot(t)
	allowed := map[string]struct{}{}
	for _, value := range DataClassLattice {
		allowed[value] = struct{}{}
	}
	for _, entry := range Registry {
		t.Run(entry.Version, func(t *testing.T) {
			var schema map[string]any
			decodeUseNumber(t, readRepoFile(t, root, entry.File), &schema)
			properties := schema["properties"].(map[string]any)
			dataClass := properties["data_class"].(map[string]any)
			values := []string{}
			if value, ok := dataClass["const"].(string); ok {
				values = append(values, value)
			}
			if enum, ok := dataClass["enum"].([]any); ok {
				for _, item := range enum {
					value, ok := item.(string)
					if !ok {
						t.Fatalf("data_class enum contains non-string %#v", item)
					}
					values = append(values, value)
				}
			}
			if len(values) == 0 {
				t.Fatalf("data_class has no const or enum")
			}
			for _, value := range values {
				if _, ok := allowed[value]; !ok {
					t.Fatalf("data_class value %q not in lattice %v", value, DataClassLattice)
				}
			}
		})
	}
}

func TestCommittedFixturesValidateAgainstSchemas(t *testing.T) {
	root := repoRoot(t)
	validateFile(t, root, "qratum.session.v1", "fixtures/claude-code/transcript-basic.normalized.golden.json")
	validateFile(t, root, "qratum.session.v1", "fixtures/redaction/secret-session.redacted.golden.json")
	validateFile(t, root, "qratum.evidence.v1", "fixtures/evidence/verification-gap.evidence.golden.json")
	validateFile(t, root, "qratum.review_card.v1", "fixtures/review/verification-gap.review.golden.json")
	validateFile(t, root, "qratum.raw_ref.v1", "fixtures/vault/raw-ref.source-metadata.golden.json.tmpl")
	validateFile(t, root, "qratum.memory_import_receipt.v1", "fixtures/memory-import/synthetic-receipt.json")
	validateFile(t, root, "1.1.0", "fixtures/adp/session.adp-strict.golden.jsonl")
	validateFile(t, root, "qratum.ui.session_detail.v1", "fixtures/ui/session-detail.golden.json")
	validateFile(t, root, "qratum.ui.review_card.v1", "fixtures/ui/review.golden.json")
	validateArrayItems(t, root, "qratum.ui.session_list_item.v1", "fixtures/ui/sessions.golden.json")

	session := readJSONMap(t, root, "fixtures/redaction/secret-session.redacted.golden.json")
	validateBytes(t, root, "qratum.redaction_summary.v1", mustJSON(t, session["redaction"]))
	review := readJSONMap(t, root, "fixtures/ui/review.golden.json")
	findings := review["findings"].([]any)
	validateBytes(t, root, "qratum.ui.evidence_finding.v1", mustJSON(t, findings[0]))
	artifacts := review["artifacts"].([]any)
	validateBytes(t, root, "qratum.ui.artifact_link.v1", mustJSON(t, artifacts[0]))

	validateSample(t, root, "qratum.event.v1", sampleEvent())
	validateSample(t, root, "qratum.vault_state.v1", sampleVaultState())
	validateSample(t, root, "qratum.raw_tombstone.v1", sampleRawTombstone())
	validateSample(t, root, "qratum.provenance.v1", sampleProvenance())
	validateSample(t, root, "qratum.ui.api_error.v1", sampleAPIError())
	validateSample(t, root, "qratum.config.v1", sampleConfig())
	validateSample(t, root, "qratum.trust_scorecard.v1", sampleTrustScorecard())
	validateSample(t, root, "qratum.memory_import_receipt.v1", sampleMemoryImportReceipt())
}

func TestSchemasRejectInjectedFixtureKeys(t *testing.T) {
	root := repoRoot(t)
	session := readJSONMap(t, root, "fixtures/claude-code/transcript-basic.normalized.golden.json")
	session["extra"] = true
	if err := validateValueForVersion(root, "qratum.session.v1", session); err == nil || !strings.Contains(err.Error(), `additional property "extra"`) {
		t.Fatalf("top-level injection error = %v, want rejection", err)
	}
	session = readJSONMap(t, root, "fixtures/claude-code/transcript-basic.normalized.golden.json")
	turns := session["turns"].([]any)
	turn := turns[0].(map[string]any)
	turn["secret_path"] = "/tmp/should-not-pass"
	if err := validateValueForVersion(root, "qratum.session.v1", session); err == nil || !strings.Contains(err.Error(), `additional property "secret_path"`) {
		t.Fatalf("nested injection error = %v, want rejection", err)
	}

	evidence := readJSONMap(t, root, "fixtures/evidence/verification-gap.evidence.golden.json")
	findings := evidence["findings"].([]any)
	finding := findings[0].(map[string]any)
	facts := finding["evidence"].([]any)
	fact := facts[0].(map[string]any)
	fact["raw_transcript_path"] = "/tmp/leak.jsonl"
	if err := validateValueForVersion(root, "qratum.evidence.v1", evidence); err == nil || !strings.Contains(err.Error(), `additional property "raw_transcript_path"`) {
		t.Fatalf("evidence nested injection error = %v, want rejection", err)
	}
}

func TestRuntimeSchemaVersionLiteralsAreRegistered(t *testing.T) {
	root := repoRoot(t)
	versionRE := regexp.MustCompile(`"((?:qratum\.[a-z0-9_\.]+\.v1)|(?:1\.1\.0))"`)
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if filepath.Base(path) == "schema" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			// #nosec G304,G122 -- test walks repo-owned Go source under fixed cmd/internal roots.
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, match := range versionRE.FindAllSubmatch(data, -1) {
				version := string(match[1])
				if !IsKnownVersion(version) {
					t.Fatalf("%s contains unregistered schema version %q", relPath(root, path), version)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readRepoFile(t *testing.T, root string, rel string) []byte {
	t.Helper()
	// #nosec G304 -- tests read committed fixtures relative to the resolved repo root.
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return data
}

func readJSONMap(t *testing.T, root string, rel string) map[string]any {
	t.Helper()
	var value map[string]any
	decodeUseNumber(t, readRepoFile(t, root, rel), &value)
	return value
}

func decodeUseNumber(t *testing.T, data []byte, target any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, string(data))
	}
}

func validateFile(t *testing.T, root string, version string, rel string) {
	t.Helper()
	validateBytes(t, root, version, readRepoFile(t, root, rel))
}

func validateArrayItems(t *testing.T, root string, version string, rel string) {
	t.Helper()
	var values []any
	decodeUseNumber(t, readRepoFile(t, root, rel), &values)
	for i, value := range values {
		t.Run(rel+"#"+strconv.Itoa(i), func(t *testing.T) {
			validateBytes(t, root, version, mustJSON(t, value))
		})
	}
}

func validateSample(t *testing.T, root string, version string, value map[string]any) {
	t.Helper()
	validateBytes(t, root, version, mustJSON(t, value))
}

func validateBytes(t *testing.T, root string, version string, data []byte) {
	t.Helper()
	schemaPath, ok := RegistryFile(version)
	if !ok {
		t.Fatalf("missing registry entry for %s", version)
	}
	if err := Validate(readRepoFile(t, root, schemaPath), data); err != nil {
		t.Fatalf("%s failed schema %s: %v\n%s", version, schemaPath, err, string(data))
	}
}

func validateValueForVersion(root string, version string, value any) error {
	schemaPath, ok := RegistryFile(version)
	if !ok {
		return nil
	}
	return Validate(mustRead(filepath.Join(root, schemaPath)), mustMarshal(value))
}

func assertRecursiveStrict(t *testing.T, path string, schema map[string]any) {
	t.Helper()
	if schema["type"] == "object" {
		if schema["x-qratum-open-map"] == true {
			if schema["additionalProperties"] != true {
				t.Fatalf("%s open map must set additionalProperties true", path)
			}
		} else if schema["additionalProperties"] != false {
			t.Fatalf("%s object must set additionalProperties false or x-qratum-open-map", path)
		}
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		for key, child := range properties {
			childSchema, ok := child.(map[string]any)
			if !ok {
				t.Fatalf("%s.properties.%s is not an object schema", path, key)
			}
			assertRecursiveStrict(t, path+"."+key, childSchema)
		}
	}
	if items, ok := schema["items"].(map[string]any); ok {
		assertRecursiveStrict(t, path+"[]", items)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mustRead(path string) []byte {
	// #nosec G304 -- tests read committed schema files resolved from the local registry.
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

func mustMarshal(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func relPath(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func sampleEvent() map[string]any {
	return map[string]any{
		"schema_version":   "qratum.event.v1",
		"data_class":       "raw",
		"event_id":         "evt_schema_sample",
		"source":           "claude-code",
		"event_type":       "session_end",
		"timestamp":        "2026-06-17T00:00:00Z",
		"timestamp_source": "capture_time",
		"session_ref": map[string]any{
			"session_id":      "ses_schema_sample",
			"transcript_path": "/tmp/transcript.jsonl",
		},
		"workspace": map[string]any{"cwd": "/tmp/qratum"},
		"raw": map[string]any{
			"copy_status": "copied",
			"raw_ref_id":  "raw_abc",
			"digest":      "sha256:abc",
			"kind":        "main_transcript",
			"size_bytes":  12,
		},
	}
}

func sampleVaultState() map[string]any {
	return map[string]any{
		"schema_version":     "qratum.vault_state.v1",
		"data_class":         "raw",
		"last_capture_at":    "2026-06-17T00:00:00Z",
		"copy_failure_count": 0,
		"raw_missing_count":  0,
	}
}

func sampleRawTombstone() map[string]any {
	return map[string]any{
		"schema_version": "qratum.raw_tombstone.v1",
		"data_class":     "raw",
		"raw_ref_id":     "raw_abc",
		"digest":         "sha256:abc",
		"reason":         "test erasure",
		"erased_at":      "2026-06-17T00:00:00Z",
		"blob_removed":   true,
	}
}

func sampleProvenance() map[string]any {
	return map[string]any{
		"schema_version":   "qratum.provenance.v1",
		"data_class":       "published",
		"canonicalization": "json-canonical-v1",
		"digests": map[string]any{
			"schema": "sha256:abc",
		},
		"local_only": map[string]any{
			"raw_transcript": false,
			"vault_blob":     false,
		},
	}
}

func sampleAPIError() map[string]any {
	return map[string]any{
		"schema_version": "qratum.ui.api_error.v1",
		"data_class":     "published",
		"error": map[string]any{
			"code":    "daemon.run_once_failed",
			"message": "sample",
		},
	}
}

func sampleConfig() map[string]any {
	return map[string]any{
		"schema_version": "qratum.config.v1",
		"data_class":     "raw",
		"raw":            map[string]any{"archive": true, "retention": "forever"},
		"sources":        map[string]any{"claude_code": true},
		"ai": map[string]any{
			"local":        true,
			"external":     "ask",
			"local_config": map[string]any{"endpoint": "http://localhost:1234/v1", "model": ""},
			"openrouter":   map[string]any{"enabled": false, "model": "", "api_key_env": "OPENROUTER_API_KEY"},
		},
		"backend":       map[string]any{"mode": "none"},
		"app":           map[string]any{"host": "127.0.0.1", "port": 9218, "idle_timeout": "30m", "raw_routes": false},
		"observability": map[string]any{"enabled": true, "exporter": "local", "otlp": map[string]any{"enabled": false, "endpoint": ""}},
		"worker":        map[string]any{"max_jobs": 4, "max_ai_jobs": 1, "disk_free_min_gb": 10},
		"publish":       map[string]any{"mode": "manual", "local_folder": map[string]any{"path": "~/QratumPublished"}},
	}
}

func sampleTrustScorecard() map[string]any {
	return map[string]any{
		"schema_version": "qratum.trust_scorecard.v1",
		"data_class":     "published",
		"headline":       "TRUSTED-WITH-NAMED-GAPS",
		"dimensions": []any{
			map[string]any{"id": "D9", "state": "GREEN", "summary": "schemas validate", "evidence": []any{"go test ./internal/schema"}},
			map[string]any{"id": "D6a", "state": "KNOWN-RED", "summary": "recoverability remains explicit"},
		},
		"gap_count": 1,
		"known_red": []any{
			map[string]any{"id": "D10", "note": "gateway producer gated", "owner": "maintainer", "deadline": "2026-12-31"},
		},
		"extended_recall": map[string]any{
			"class_count":              8,
			"baseline_class_count":     8,
			"recall_percent":           100,
			"baseline_recall_percent":  100,
			"covered_corpus_leak_free": true,
			"extended_corpus_monotone": true,
		},
		"honest_residual": []any{"cloud-only transcripts are outside local proof"},
		"provenance": map[string]any{
			"build_commit":  "abc123",
			"corpus_digest": "sha256:abc",
			"schema_digest": "sha256:def",
			"timestamp":     "2026-06-17T00:00:00Z",
		},
	}
}

func sampleMemoryImportReceipt() map[string]any {
	return map[string]any{
		"schema_version":        "qratum.memory_import_receipt.v1",
		"data_class":            "redacted",
		"tool":                  "memory_store",
		"decision":              "allowed",
		"outcome":               "created",
		"namespace":             "project:qratum",
		"content_class":         "coding_context",
		"memory_ids":            []any{"mem_123"},
		"supersedes":            []any{},
		"superseded_memory_ids": []any{},
		"embedding_provider":    "bedrock",
		"embedding_model_id":    "amazon.titan-embed-text-v2:0",
		"embedding_dimensions":  1024,
		"received_at":           "2026-06-17T00:00:00Z",
	}
}
