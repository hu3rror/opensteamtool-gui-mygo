package app_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var purePackages = map[string]bool{
	"errors":        true,
	"fmt":           true,
	"math":          true,
	"path":          true,
	"path/filepath": true,
	"regexp":        true,
	"slices":        true,
	"sort":          true,
	"strconv":       true,
	"strings":       true,
	"time":          true,
	"unicode/utf8":  true,
}

func TestTheDerivationImportsOnlyPurePackages(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(".", name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			path := strings.Trim(imported.Path.Value, `"`)
			if !purePackages[path] {
				t.Errorf("%s imports %q", name, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no source file was checked")
	}
}
