
# Commits

These rules govern the commit history on a branch.

### Subject Size

A commit subject runs to at most 100 characters.

### Conventional Subjects

A commit subject is a lowercase type, an optional (scope), an optional ! breaking
marker, then a colon, a space, and a non-empty description; a GitHub
synthetic-merge subject is exempt.

### Fixup Commits

A subject is a fixup when it opens with fixup! or squash!, or reads as a review
follow-up; autosquash such commits into their target.

### Merge Commits

A branch other than main or master carries no merge commit; a git-subtree merge
is the sole exception, required when vendoring another repo's history.
