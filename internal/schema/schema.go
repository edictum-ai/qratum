// Package schema validates Qratum JSON contracts without pulling in runtime dependencies.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Data class constants identify Qratum artifact sensitivity tiers.
const (
	DataClassRaw       = "raw"
	DataClassRedacted  = "redacted"
	DataClassReview    = "review"
	DataClassCorpus    = "corpus"
	DataClassPublished = "published"
)

// DataClassLattice orders Qratum data classes from most sensitive to least.
var DataClassLattice = []string{
	DataClassRaw,
	DataClassRedacted,
	DataClassReview,
	DataClassCorpus,
	DataClassPublished,
}

// RegistryEntry maps a schema_version literal to its committed schema file.
type RegistryEntry struct {
	Version string
	File    string
}

// Registry lists every schema_version Qratum treats as known.
var Registry = []RegistryEntry{
	{Version: "1.1.0", File: "schemas/qratum-adp-strict.v1.schema.json"},
	{Version: "qratum.config.v1", File: "schemas/qratum-config.v1.schema.json"},
	{Version: "qratum.capture_event.v2", File: "schemas/qratum-capture-event.v2.schema.json"},
	{Version: "qratum.capture_state.v1", File: "schemas/qratum-capture-state.v1.schema.json"},
	{Version: "qratum.event.v1", File: "schemas/qratum-event.v1.schema.json"},
	{Version: "qratum.evidence.v1", File: "schemas/qratum-evidence.v1.schema.json"},
	{Version: "qratum.memory_import_receipt.v1", File: "schemas/qratum-memory-import-receipt.v1.schema.json"},
	{Version: "qratum.provenance.v1", File: "schemas/qratum-provenance.v1.schema.json"},
	{Version: "qratum.price_catalog_manifest.v1", File: "schemas/qratum-price-catalog-manifest.v1.schema.json"},
	{Version: "qratum.raw_ref.v1", File: "schemas/qratum-raw-ref.v1.schema.json"},
	{Version: "qratum.raw_tombstone.v1", File: "schemas/qratum-raw-tombstone.v1.schema.json"},
	{Version: "qratum.redaction_summary.v1", File: "schemas/qratum-redaction-summary.v1.schema.json"},
	{Version: "qratum.review_card.v1", File: "schemas/qratum-review-card.v1.schema.json"},
	{Version: "qratum.session_revision.v1", File: "schemas/qratum-session-revision.v1.schema.json"},
	{Version: "qratum.session_tombstone.v1", File: "schemas/qratum-session-tombstone.v1.schema.json"},
	{Version: "qratum.session.v1", File: "schemas/qratum-session.v1.schema.json"},
	{Version: "qratum.trust_scorecard.v1", File: "schemas/qratum-trust-scorecard.v1.schema.json"},
	{Version: "qratum.usage_record.v1", File: "schemas/qratum-usage-record.v1.schema.json"},
	{Version: "qratum.vault_state.v1", File: "schemas/qratum-vault-state.v1.schema.json"},
	{Version: "qratum.ui.api_error.v1", File: "schemas/ui/api-error.v1.schema.json"},
	{Version: "qratum.ui.artifact_link.v1", File: "schemas/ui/artifact-link.v1.schema.json"},
	{Version: "qratum.ui.evidence_finding.v1", File: "schemas/ui/evidence-finding.v1.schema.json"},
	{Version: "qratum.ui.review_card.v1", File: "schemas/ui/review-card.v1.schema.json"},
	{Version: "qratum.ui.session_detail.v1", File: "schemas/ui/session-detail.v1.schema.json"},
	{Version: "qratum.ui.session_list_item.v1", File: "schemas/ui/session-list-item.v1.schema.json"},
}

// DataClassRank returns a data class's position in DataClassLattice.
func DataClassRank(value string) (int, bool) {
	for i, item := range DataClassLattice {
		if item == value {
			return i, true
		}
	}
	return 0, false
}

// RegistryFile returns the schema file for a schema_version.
func RegistryFile(version string) (string, bool) {
	for _, entry := range Registry {
		if entry.Version == version {
			return entry.File, true
		}
	}
	return "", false
}

// Validate checks an instance document against a strict subset of JSON Schema.
// Source parsers enforce negative token-count and illegal field-combination
// rejection. Supporting JSON Schema minimum and conditional keywords here is a
// separate ticket; callers must not treat this subset as that parser boundary.
func Validate(schemaData []byte, instanceData []byte) error {
	var schema any
	if err := decodeJSON(schemaData, &schema); err != nil {
		return fmt.Errorf("decode schema: %w", err)
	}
	var instance any
	if err := decodeJSON(instanceData, &instance); err != nil {
		return fmt.Errorf("decode instance: %w", err)
	}
	root, ok := schema.(map[string]any)
	if !ok {
		return fmt.Errorf("$: schema must be an object")
	}
	return validateValue("$", root, instance)
}

func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func validateValue(path string, schema map[string]any, value any) error {
	if expected, ok := schema["const"]; ok && !jsonEqual(expected, value) {
		return fmt.Errorf("%s: value %s does not equal const %s", path, describeJSON(value), describeJSON(expected))
	}
	if enumRaw, ok := schema["enum"]; ok {
		enumValues, ok := enumRaw.([]any)
		if !ok {
			return fmt.Errorf("%s: schema enum must be an array", path)
		}
		matches := false
		for _, candidate := range enumValues {
			if jsonEqual(candidate, value) {
				matches = true
				break
			}
		}
		if !matches {
			return fmt.Errorf("%s: value %s is not in enum", path, describeJSON(value))
		}
	}
	if typeRaw, ok := schema["type"]; ok {
		typeName, ok := typeRaw.(string)
		if !ok {
			return fmt.Errorf("%s: schema type must be a string", path)
		}
		if err := validateType(path, typeName, value); err != nil {
			return err
		}
	}

	if patternRaw, ok := schema["pattern"]; ok {
		pattern, ok := patternRaw.(string)
		if !ok {
			return fmt.Errorf("%s: schema pattern must be a string", path)
		}
		// JSON Schema pattern is an UNANCHORED substring match (RE2 here): it
		// succeeds when the regex matches anywhere in the string. Schemas that
		// need a full-string match anchor with ^...$ themselves; we never anchor.
		// A pattern that is not a string or does not compile fails closed above /
		// here rather than being skipped.
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("%s: schema pattern %q is not a valid regexp: %w", path, pattern, err)
		}
		// pattern applies to string instance values only: a non-string value is
		// ignored by pattern and, if illegal, is rejected by the type keyword.
		if str, isString := value.(string); isString && !re.MatchString(str) {
			return fmt.Errorf("%s: value %s does not match pattern %q", path, describeJSON(value), pattern)
		}
	}

	if propertiesRaw, hasProperties := schema["properties"]; hasProperties {
		object, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: properties require object value", path)
		}
		properties, ok := propertiesRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("%s: schema properties must be an object", path)
		}
		if err := validateRequired(path, schema, object); err != nil {
			return err
		}
		for key, child := range properties {
			childSchema, ok := child.(map[string]any)
			if !ok {
				return fmt.Errorf("%s.properties.%s: child schema must be an object", path, key)
			}
			childValue, exists := object[key]
			if !exists {
				continue
			}
			if err := validateValue(path+"."+key, childSchema, childValue); err != nil {
				return err
			}
		}
		if additional, ok := schema["additionalProperties"].(bool); ok && !additional {
			allowed := map[string]struct{}{}
			for key := range properties {
				allowed[key] = struct{}{}
			}
			for key := range object {
				if _, ok := allowed[key]; !ok {
					return fmt.Errorf("%s: additional property %q is not allowed", path, key)
				}
			}
		}
	}

	if itemsRaw, ok := schema["items"]; ok {
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("%s: items require array value", path)
		}
		itemSchema, ok := itemsRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("%s.items: schema must be an object", path)
		}
		for i, item := range array {
			if err := validateValue(fmt.Sprintf("%s[%d]", path, i), itemSchema, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRequired(path string, schema map[string]any, object map[string]any) error {
	requiredRaw, ok := schema["required"]
	if !ok {
		return nil
	}
	required, ok := requiredRaw.([]any)
	if !ok {
		return fmt.Errorf("%s: schema required must be an array", path)
	}
	for _, item := range required {
		key, ok := item.(string)
		if !ok {
			return fmt.Errorf("%s: required entry must be a string", path)
		}
		if _, exists := object[key]; !exists {
			return fmt.Errorf("%s: missing required property %q", path, key)
		}
	}
	return nil
}

func validateType(path string, typeName string, value any) error {
	ok := false
	switch typeName {
	case "object":
		_, ok = value.(map[string]any)
	case "array":
		_, ok = value.([]any)
	case "string":
		_, ok = value.(string)
	case "integer":
		number, numberOK := value.(json.Number)
		if numberOK {
			if _, err := number.Int64(); err == nil {
				ok = true
			} else if f, err := number.Float64(); err == nil {
				ok = math.Trunc(f) == f
			}
		}
	case "number":
		_, ok = value.(json.Number)
	case "boolean":
		_, ok = value.(bool)
	case "null":
		ok = value == nil
	default:
		return fmt.Errorf("%s: unsupported schema type %q", path, typeName)
	}
	if !ok {
		return fmt.Errorf("%s: value %s is not type %s", path, describeJSON(value), typeName)
	}
	return nil
}

func jsonEqual(left any, right any) bool {
	left = normalizeNumber(left)
	right = normalizeNumber(right)
	return reflect.DeepEqual(left, right)
}

func normalizeNumber(value any) any {
	if v, ok := value.(json.Number); ok {
		if i, err := v.Int64(); err == nil {
			return i
		}
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return value
}

func describeJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%T", value)
	}
	return string(data)
}

// SortedRegistryVersions returns registered schema versions in lexical order.
func SortedRegistryVersions() []string {
	versions := make([]string, 0, len(Registry))
	for _, entry := range Registry {
		versions = append(versions, entry.Version)
	}
	sort.Strings(versions)
	return versions
}

// EnumValues returns string enum values for a top-level schema property.
func EnumValues(schemaData []byte, property string) ([]string, error) {
	var root map[string]any
	if err := decodeJSON(schemaData, &root); err != nil {
		return nil, err
	}
	properties, _ := root["properties"].(map[string]any)
	prop, _ := properties[property].(map[string]any)
	enumRaw, ok := prop["enum"].([]any)
	if !ok {
		return nil, fmt.Errorf("property %q has no enum", property)
	}
	values := make([]string, 0, len(enumRaw))
	for _, item := range enumRaw {
		value, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("property %q enum contains non-string", property)
		}
		values = append(values, value)
	}
	return values, nil
}

// IsKnownVersion reports whether a schema_version literal is registered.
func IsKnownVersion(version string) bool {
	_, ok := RegistryFile(strings.TrimSpace(version))
	return ok
}
