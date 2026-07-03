
# Parsed File

A tracked Go file, parsed once and reused by every AST-tier rule.

### Path

Path is the repo-relative path of the file.

### File Set

File Set is the token.FileSet that resolves the AST's positions to file:line:col.

### File

File is the parsed syntax tree.

### Source

Source is the raw bytes the file was parsed from.

# Component

One component: a top-level directory of Go source in the single-module repo.

### Root

Root is the component's top-level directory, such as shared/cli or lint.

### Import Path

Import Path is the component's full import path.

### Shared Library

Is Shared Library marks a shared component; false marks a binary component.

### Directory Package

Directory Package maps each directory to the package name declared there.

# Component Index

The workspace's component graph, built once after parsing and threaded through
the directory-level checks.

### Components

Components is every component, sorted longest-Root first.

### File To Component

File To Component maps a file path to its component's index in Components, or -1
when the file belongs to no discovered component.
