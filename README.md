# deptree

A simple command-line tool to visualize Go module dependency trees.

## Features

- Analyze dependencies of local Go projects
- Fetch and analyze remote Go packages by name
- Display dependencies in a clean tree structure
- Shows transitive dependencies
- Fetch and display GitHub repository descriptions
- Concurrent API requests for fast description fetching
- GitHub token authentication for higher rate limits

## Installation

```bash
make install
```

Or manually:

```bash
go build -o deptree
```

## Usage

### Analyze a local package

```bash
deptree -path /path/to/your/project
```

Or from the current directory (default):

```bash
deptree
```

### Fetch and analyze a remote package

```bash
deptree -package github.com/spf13/cobra
```

With a specific version:

```bash
deptree -package github.com/spf13/cobra@v1.8.0
```

### Export as flat list

```bash
deptree -package github.com/spf13/cobra -export
```

### Fetch GitHub repository descriptions

```bash
deptree -package github.com/spf13/cobra -desc
```

Combine with export mode:

```bash
deptree -package github.com/spf13/cobra -desc -export
```

### Using GitHub token for higher rate limits

Without authentication, GitHub API allows 60 requests/hour. With a token, this increases to 5000 requests/hour.

Using environment variable (recommended):

```bash
export GITHUB_TOKEN="your_token_here"
deptree -package github.com/spf13/cobra -desc
```

Or using the flag:

```bash
deptree -package github.com/spf13/cobra -desc -token "your_token_here"
```

## Flags

- `-path` - Path to the Go package (default: current directory)
- `-package` - Package name to fetch and analyze (e.g., github.com/spf13/cobra)
- `-export` - Export as flat list sorted by name with no duplicates
- `-desc` - Fetch and display GitHub repository descriptions
- `-token` - GitHub personal access token (or use GITHUB_TOKEN env var)

## Example Output

### Standard tree view

```
github.com/spf13/cobra@v1.8.0
├── github.com/cpuguy83/go-md2man/v2@v2.0.3
│   └── github.com/russross/blackfriday/v2@v2.1.0
├── github.com/inconshreveable/mousetrap@v1.1.0
├── github.com/spf13/pflag@v1.0.5
└── gopkg.in/yaml.v3@v3.0.1
    └── gopkg.in/check.v1@v0.0.0-20161208181325-20d25e280405
```

### With descriptions (`-desc` flag)

```
github.com/spf13/cobra@v1.10.1 - A Commander for modern Go CLI interactions
├── github.com/cpuguy83/go-md2man/v2@v2.0.6 - (no description set)
│   └── github.com/russross/blackfriday/v2@v2.1.0 - Blackfriday: a markdown processor for Go
├── github.com/inconshreveable/mousetrap@v1.1.0 - Detect starting from Windows explorer
├── github.com/spf13/pflag@v1.0.9 - Drop-in replacement for Go's flag package, implementing POSIX/GNU-style --flags.
└── gopkg.in/yaml.v3@v3.0.1 - (not a GitHub module)
    └── gopkg.in/check.v1@v0.0.0-20161208181325-20d25e280405 - (not a GitHub module)
```

### Export mode with descriptions

```
github.com/cpuguy83/go-md2man/v2@v2.0.6 - (no description set)
github.com/inconshreveable/mousetrap@v1.1.0 - Detect starting from Windows explorer
github.com/russross/blackfriday/v2@v2.1.0 - Blackfriday: a markdown processor for Go
github.com/spf13/cobra@v1.10.1 - A Commander for modern Go CLI interactions
github.com/spf13/pflag@v1.0.9 - Drop-in replacement for Go's flag package, implementing POSIX/GNU-style --flags.
gopkg.in/check.v1@v0.0.0-20161208181325-20d25e280405 - (not a GitHub module)
gopkg.in/yaml.v3@v3.0.1 - (not a GitHub module)
```

## Creating a GitHub Token

To avoid rate limits when fetching descriptions, create a GitHub personal access token:

1. Go to https://github.com/settings/tokens
2. Click "Generate new token" → "Generate new token (classic)"
3. Give it a name (e.g., "deptree")
4. **No scopes needed** - you can create it with no permissions (just for authentication)
5. Click "Generate token" and copy it
6. Set it as an environment variable:
   ```bash
   export GITHUB_TOKEN="your_token_here"
   ```

Note: Without authentication, you're limited to 60 requests/hour. With a token, this increases to 5000 requests/hour.

## Requirements

- Go 1.16 or higher

## License

MIT
