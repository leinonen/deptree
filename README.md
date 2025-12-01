# deptree

A simple command-line tool to visualize Go module dependency trees.

## Features

- Analyze dependencies of local Go projects
- Fetch and analyze remote Go packages by name
- Display dependencies in a clean tree structure
- Shows transitive dependencies

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

## Flags

- `-path` - Path to the Go package (default: current directory)
- `-package` - Package name to fetch and analyze (e.g., github.com/spf13/cobra)
- `-export` - Export as flat list sorted by name with no duplicates

## Example Output

```
github.com/spf13/cobra@v1.8.0
├── github.com/cpuguy83/go-md2man/v2@v2.0.3
│   └── github.com/russross/blackfriday/v2@v2.1.0
├── github.com/inconshreveable/mousetrap@v1.1.0
├── github.com/spf13/pflag@v1.0.5
└── gopkg.in/yaml.v3@v3.0.1
    └── gopkg.in/check.v1@v0.0.0-20161208181325-20d25e280405
```

## Requirements

- Go 1.16 or higher

## License

MIT
