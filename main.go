package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Node struct {
	Name        string
	Description string
	Children    map[string]*Node
}

func NewNode(name string) *Node {
	return &Node{
		Name:     name,
		Children: make(map[string]*Node),
	}
}

func main() {
	var packagePath string
	var packageName string
	var exportMode bool
	var fetchDesc bool
	var githubToken string
	flag.StringVar(&packagePath, "path", ".", "Path to the Go package (default: current directory)")
	flag.StringVar(&packageName, "package", "", "Package name to fetch and analyze (e.g., github.com/spf13/cobra)")
	flag.BoolVar(&exportMode, "export", false, "Export as flat list sorted by name with no duplicates")
	flag.BoolVar(&fetchDesc, "desc", false, "Fetch and display GitHub repository descriptions")
	flag.StringVar(&githubToken, "token", "", "GitHub personal access token (or use GITHUB_TOKEN env var)")
	flag.Parse()

	// Use environment variable if token not provided via flag
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	if err := run(packagePath, packageName, exportMode, fetchDesc, githubToken); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(packagePath, packageName string, exportMode, fetchDesc bool, githubToken string) error {
	var workDir string
	var cleanup bool

	if packageName != "" {
		tmpDir, err := os.MkdirTemp("", "deptree-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer func() {
			if cleanup {
				os.RemoveAll(tmpDir)
			}
		}()

		if err := setupPackage(tmpDir, packageName); err != nil {
			cleanup = true
			return fmt.Errorf("failed to setup package: %w", err)
		}

		workDir = tmpDir
		cleanup = true
	} else {
		workDir = packagePath
	}

	deps, err := getModuleDependencies(workDir)
	if err != nil {
		return fmt.Errorf("failed to get dependencies: %w", err)
	}

	if len(deps) == 0 {
		fmt.Println("No dependencies found")
		return nil
	}

	tree := buildDependencyTree(deps, packageName)

	if fetchDesc {
		fetchDescriptions(tree, githubToken)
	}

	if exportMode {
		printExport(deps, fetchDesc, githubToken)
	} else {
		printTree(tree, fetchDesc)
	}

	return nil
}

func setupPackage(tmpDir, packageName string) error {
	modInit := exec.Command("go", "mod", "init", "temp")
	modInit.Dir = tmpDir
	if err := modInit.Run(); err != nil {
		return fmt.Errorf("failed to run 'go mod init': %w", err)
	}

	goGet := exec.Command("go", "get", packageName)
	goGet.Dir = tmpDir
	output, err := goGet.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run 'go get %s': %w\nOutput: %s", packageName, err, string(output))
	}

	mainGo := filepath.Join(tmpDir, "main.go")
	content := fmt.Sprintf("package main\n\nimport _ \"%s\"\n\nfunc main() {}\n", packageName)
	if err := os.WriteFile(mainGo, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	return nil
}

func getModuleDependencies(packagePath string) (map[string][]string, error) {
	cmd := exec.Command("go", "mod", "graph")
	cmd.Dir = packagePath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run 'go mod graph': %w", err)
	}

	deps := make(map[string][]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 {
			from := parts[0]
			to := parts[1]
			deps[from] = append(deps[from], to)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading output: %w", err)
	}

	return deps, nil
}

type GitHubRepo struct {
	Description string `json:"description"`
}

func extractGitHubRepo(modulePath string) (owner, repo string, ok bool) {
	// Remove version suffix if present
	parts := strings.Split(modulePath, "@")
	path := parts[0]

	// Check if it's a GitHub module
	if !strings.HasPrefix(path, "github.com/") {
		return "", "", false
	}

	// Extract owner and repo (handle subpackages)
	pathParts := strings.Split(strings.TrimPrefix(path, "github.com/"), "/")
	if len(pathParts) < 2 {
		return "", "", false
	}

	return pathParts[0], pathParts[1], true
}

func fetchGitHubDescription(modulePath, token string) (string, error) {
	owner, repo, ok := extractGitHubRepo(modulePath)
	if !ok {
		return "", fmt.Errorf("not a GitHub module")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to avoid GitHub API rate limiting issues
	req.Header.Set("User-Agent", "deptree-cli")

	// Add authentication if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var ghRepo GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&ghRepo); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if ghRepo.Description == "" {
		return "", fmt.Errorf("no description set")
	}

	return ghRepo.Description, nil
}

func fetchDescriptions(root *Node, token string) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect all unique modules
	modules := make(map[string]*Node)
	var collectModules func(*Node)
	collectModules = func(node *Node) {
		mu.Lock()
		modules[node.Name] = node
		mu.Unlock()

		for _, child := range node.Children {
			collectModules(child)
		}
	}
	collectModules(root)

	// Fetch descriptions concurrently
	for _, node := range modules {
		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			desc, err := fetchGitHubDescription(n.Name, token)
			mu.Lock()
			if err != nil {
				// Store error message as description for display
				n.Description = fmt.Sprintf("(%s)", err.Error())
			} else {
				n.Description = desc
			}
			mu.Unlock()
		}(node)
	}

	wg.Wait()
}

func buildDependencyTree(deps map[string][]string, requestedPackage string) *Node {
	var rootModule string

	// First, find the actual root in the dependency graph (usually the local module or "temp")
	for from := range deps {
		if !strings.Contains(from, "@") {
			rootModule = from
			break
		}
	}

	if rootModule == "" {
		for from := range deps {
			rootModule = from
			break
		}
	}

	root := NewNode(rootModule)
	visited := make(map[string]bool)
	buildTree(root, deps, visited)

	// If we have a temp module and a requested package, find the requested package and use it as root
	if rootModule == "temp" && requestedPackage != "" {
		packageBase := strings.Split(requestedPackage, "@")[0]

		// Look for the requested package in temp's children
		// The requested package might include a subpath (e.g., github.com/a-h/templ/cmd/templ)
		// but the module name is just the base (e.g., github.com/a-h/templ@v0.3.960)
		for childName, childNode := range root.Children {
			childBase := strings.Split(childName, "@")[0]
			// Check if the requested package path starts with this module's base path
			if strings.HasPrefix(packageBase, childBase) || strings.HasPrefix(childBase, packageBase) {
				return childNode
			}
		}
	}

	return root
}

func buildTree(node *Node, deps map[string][]string, visited map[string]bool) {
	if visited[node.Name] {
		return
	}
	visited[node.Name] = true

	children := deps[node.Name]
	for _, child := range children {
		if _, exists := node.Children[child]; !exists {
			childNode := NewNode(child)
			node.Children[child] = childNode
			buildTree(childNode, deps, visited)
		}
	}
}

func printTree(node *Node, showDesc bool) {
	if showDesc && node.Description != "" {
		fmt.Printf("%s - %s\n", node.Name, node.Description)
	} else {
		fmt.Println(node.Name)
	}
	printNode(node, "", showDesc)
}

func printNode(node *Node, prefix string, showDesc bool) {
	childCount := len(node.Children)

	var childNames []string
	for name := range node.Children {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)

	for i, name := range childNames {
		child := node.Children[name]
		isLast := i == childCount-1

		var connector, childPrefix string
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		} else {
			connector = "├── "
			childPrefix = prefix + "│   "
		}

		if showDesc && child.Description != "" {
			fmt.Printf("%s%s%s - %s\n", prefix, connector, child.Name, child.Description)
		} else {
			fmt.Printf("%s%s%s\n", prefix, connector, child.Name)
		}
		printNode(child, childPrefix, showDesc)
	}
}

func printExport(deps map[string][]string, showDesc bool, token string) {
	uniqueDeps := make(map[string]bool)

	for from, tos := range deps {
		// Include the "from" module unless it's "temp"
		if from != "temp" && !isToolchainDep(from) {
			uniqueDeps[from] = true
		}
		// Include all "to" modules
		for _, to := range tos {
			if !isToolchainDep(to) {
				uniqueDeps[to] = true
			}
		}
	}

	var depList []string
	for dep := range uniqueDeps {
		depList = append(depList, dep)
	}

	sort.Strings(depList)

	if showDesc {
		// Fetch descriptions concurrently for export mode
		descriptions := make(map[string]string)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, dep := range depList {
			wg.Add(1)
			go func(d string) {
				defer wg.Done()
				desc, err := fetchGitHubDescription(d, token)
				mu.Lock()
				if err != nil {
					descriptions[d] = fmt.Sprintf("(%s)", err.Error())
				} else {
					descriptions[d] = desc
				}
				mu.Unlock()
			}(dep)
		}
		wg.Wait()

		for _, dep := range depList {
			if desc, ok := descriptions[dep]; ok {
				fmt.Printf("%s - %s\n", dep, desc)
			} else {
				fmt.Println(dep)
			}
		}
	} else {
		for _, dep := range depList {
			fmt.Println(dep)
		}
	}
}

func isToolchainDep(dep string) bool {
	return strings.HasPrefix(dep, "go@") || strings.HasPrefix(dep, "toolchain@")
}
