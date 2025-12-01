package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewNode(t *testing.T) {
	node := NewNode("test-package")

	if node.Name != "test-package" {
		t.Errorf("Expected node name to be 'test-package', got '%s'", node.Name)
	}

	if node.Children == nil {
		t.Error("Expected Children map to be initialized")
	}

	if len(node.Children) != 0 {
		t.Errorf("Expected empty Children map, got %d items", len(node.Children))
	}
}

func TestIsToolchainDep(t *testing.T) {
	tests := []struct {
		name     string
		dep      string
		expected bool
	}{
		{"go toolchain", "go@1.21.0", true},
		{"toolchain", "toolchain@go1.21.0", true},
		{"regular dependency", "github.com/spf13/cobra@v1.7.0", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolchainDep(tt.dep)
			if result != tt.expected {
				t.Errorf("isToolchainDep(%q) = %v, want %v", tt.dep, result, tt.expected)
			}
		})
	}
}

func TestGetModuleDependencies(t *testing.T) {
	// Create a temporary directory with a minimal Go module
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := []byte("module test\n\ngo 1.21\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), goModContent, 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a simple main.go
	mainGoContent := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), mainGoContent, 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	deps, err := getModuleDependencies(tmpDir)
	if err != nil {
		t.Fatalf("getModuleDependencies failed: %v", err)
	}

	// For a minimal module with no dependencies, we should get an empty or minimal result
	if deps == nil {
		t.Error("Expected non-nil deps map")
	}
}

func TestBuildTree(t *testing.T) {
	deps := map[string][]string{
		"root": {"dep1@v1.0.0", "dep2@v1.0.0"},
		"dep1@v1.0.0": {"dep3@v1.0.0"},
		"dep2@v1.0.0": {},
		"dep3@v1.0.0": {},
	}

	root := NewNode("root")
	visited := make(map[string]bool)
	buildTree(root, deps, visited)

	// Check that root has two children
	if len(root.Children) != 2 {
		t.Errorf("Expected root to have 2 children, got %d", len(root.Children))
	}

	// Check that dep1 exists and has dep3 as child
	dep1, exists := root.Children["dep1@v1.0.0"]
	if !exists {
		t.Error("Expected dep1@v1.0.0 to be in root's children")
	} else {
		if len(dep1.Children) != 1 {
			t.Errorf("Expected dep1 to have 1 child, got %d", len(dep1.Children))
		}
		if _, exists := dep1.Children["dep3@v1.0.0"]; !exists {
			t.Error("Expected dep3@v1.0.0 to be in dep1's children")
		}
	}

	// Check that dep2 exists
	_, exists = root.Children["dep2@v1.0.0"]
	if !exists {
		t.Error("Expected dep2@v1.0.0 to be in root's children")
	}

	// Check visited map
	if !visited["root"] {
		t.Error("Expected root to be marked as visited")
	}
}

func TestBuildDependencyTree(t *testing.T) {
	deps := map[string][]string{
		"mymodule": {"dep1@v1.0.0", "dep2@v1.0.0"},
		"dep1@v1.0.0": {"dep3@v1.0.0"},
		"dep2@v1.0.0": {},
		"dep3@v1.0.0": {},
	}

	tree := buildDependencyTree(deps, "")

	if tree.Name != "mymodule" {
		t.Errorf("Expected root name to be 'mymodule', got '%s'", tree.Name)
	}

	if len(tree.Children) != 2 {
		t.Errorf("Expected tree to have 2 children, got %d", len(tree.Children))
	}
}

func TestBuildDependencyTreeWithTemp(t *testing.T) {
	deps := map[string][]string{
		"temp": {"github.com/example/pkg@v1.0.0"},
		"github.com/example/pkg@v1.0.0": {"dep1@v1.0.0"},
		"dep1@v1.0.0": {},
	}

	tree := buildDependencyTree(deps, "github.com/example/pkg")

	// Should return the requested package as root, not "temp"
	if !strings.Contains(tree.Name, "github.com/example/pkg") {
		t.Errorf("Expected root to be the requested package, got '%s'", tree.Name)
	}
}

func TestPrintExport(t *testing.T) {
	deps := map[string][]string{
		"mymodule": {"dep1@v1.0.0", "dep2@v2.0.0"},
		"dep1@v1.0.0": {"dep3@v1.5.0"},
		"dep2@v2.0.0": {},
		"dep3@v1.5.0": {},
		"temp": {"go@1.21.0"},
		"go@1.21.0": {},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printExport(deps)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that output contains expected dependencies
	if !strings.Contains(output, "dep1@v1.0.0") {
		t.Error("Expected output to contain dep1@v1.0.0")
	}
	if !strings.Contains(output, "dep2@v2.0.0") {
		t.Error("Expected output to contain dep2@v2.0.0")
	}
	if !strings.Contains(output, "dep3@v1.5.0") {
		t.Error("Expected output to contain dep3@v1.5.0")
	}
	if !strings.Contains(output, "mymodule") {
		t.Error("Expected output to contain mymodule")
	}

	// Check that temp and toolchain deps are filtered out
	if strings.Contains(output, "temp") {
		t.Error("Expected output to not contain 'temp'")
	}
	if strings.Contains(output, "go@1.21.0") {
		t.Error("Expected output to not contain toolchain dependency go@1.21.0")
	}

	// Check that output is sorted (each line should be >= previous)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := 1; i < len(lines); i++ {
		if lines[i] < lines[i-1] {
			t.Errorf("Output is not sorted: %s comes before %s", lines[i-1], lines[i])
		}
	}
}

func TestPrintTree(t *testing.T) {
	root := NewNode("root@v1.0.0")
	child1 := NewNode("child1@v1.0.0")
	child2 := NewNode("child2@v2.0.0")
	root.Children["child1@v1.0.0"] = child1
	root.Children["child2@v2.0.0"] = child2

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTree(root)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that output contains the root and children
	if !strings.Contains(output, "root@v1.0.0") {
		t.Error("Expected output to contain root@v1.0.0")
	}
	if !strings.Contains(output, "child1@v1.0.0") {
		t.Error("Expected output to contain child1@v1.0.0")
	}
	if !strings.Contains(output, "child2@v2.0.0") {
		t.Error("Expected output to contain child2@v2.0.0")
	}

	// Check for tree characters
	if !strings.Contains(output, "├──") && !strings.Contains(output, "└──") {
		t.Error("Expected output to contain tree drawing characters")
	}
}

func TestRun_NoPackageNameOrPath(t *testing.T) {
	// Create a temporary directory with a minimal Go module
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := []byte("module test\n\ngo 1.21\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), goModContent, 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a simple main.go
	mainGoContent := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), mainGoContent, 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	err := run(tmpDir, "", false)
	if err != nil {
		t.Errorf("run() failed: %v", err)
	}
}

func TestRun_ExportMode(t *testing.T) {
	// Create a temporary directory with a minimal Go module
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := []byte("module test\n\ngo 1.21\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), goModContent, 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a simple main.go
	mainGoContent := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), mainGoContent, 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// Capture stdout to avoid polluting test output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := run(tmpDir, "", true)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("run() in export mode failed: %v", err)
	}
}

func TestBuildTreeCyclicDependency(t *testing.T) {
	// Test that buildTree handles cyclic dependencies gracefully
	deps := map[string][]string{
		"root": {"dep1@v1.0.0"},
		"dep1@v1.0.0": {"dep2@v1.0.0"},
		"dep2@v1.0.0": {"dep1@v1.0.0"}, // Cycle back to dep1
	}

	root := NewNode("root")
	visited := make(map[string]bool)

	// This should not cause infinite recursion
	buildTree(root, deps, visited)

	// Verify the tree was built
	if len(root.Children) != 1 {
		t.Errorf("Expected root to have 1 child, got %d", len(root.Children))
	}

	// Verify visited tracking prevents infinite loops
	if !visited["root"] || !visited["dep1@v1.0.0"] || !visited["dep2@v1.0.0"] {
		t.Error("Expected all nodes to be marked as visited")
	}
}
