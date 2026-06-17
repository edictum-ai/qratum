package capture

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCaptureImportSetMatchesGolden(t *testing.T) {
	got, err := captureImportsFromDir(".")
	if err != nil {
		t.Fatal(err)
	}
	wantData, err := os.ReadFile("imports.golden")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(wantData))
	if got != want {
		t.Fatalf("internal/capture import set drifted\n got:\n%s\nwant:\n%s\n", got, want)
	}
}

func TestCaptureImportSetNegativeCatchesNetworkImport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "capture.go"), []byte("package capture\n\nimport \"net\"\n\nvar _ = net.IP{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := captureImportsFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	wantData, err := os.ReadFile("imports.golden")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(wantData))
	if got == want || !strings.Contains(got, "net") {
		t.Fatalf("negative import pin did not catch net import\n got:\n%s\nwant golden:\n%s\n", got, want)
	}
}

func captureImportsFromDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	set := map[string]struct{}{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			return "", err
		}
		for _, spec := range file.Imports {
			path := strings.Trim(spec.Path.Value, "\"")
			set[path] = struct{}{}
		}
	}
	imports := make([]string, 0, len(set))
	for path := range set {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	var b bytes.Buffer
	for i, path := range imports {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(path)
	}
	return b.String(), nil
}
