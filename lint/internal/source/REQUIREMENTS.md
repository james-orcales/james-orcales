
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

# Declaration Index

Every package-level declaration in the parsed set, and the imports each file
resolves a qualifier through. It answers what a name declares without a type
checker, because the doctrine leaves one name in one file with one meaning.

### Declarations

Declarations maps a package symbol — directory, package clause, and name — to
the declaration that carries it. The package clause is part of the key because an
external test package shares a directory with the package it tests.

### Imports

Imports maps an importing file's local qualifier to the package it names. An
import the workspace does not own is left out, so a lookup through it misses. Unnamed default-tier
import uses parent package name required by shared-component layout.

### Import Paths

Import Paths maps first-party import path in each file to imported package's declared name.

### File Package

File Package maps a file path to its own directory and package clause, so a bare
name resolves without re-reading the syntax tree.

### Declaration Kind

Kind is a func, a type, or a const. There is no fourth: a package-level var is
banned, so no other package-level name can exist.

### Ambiguity

A name declared more than once in one package is ambiguous, as the build-tag
variants of a package are. Resolve refuses it rather than pick a variant.
