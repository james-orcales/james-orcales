
# Commits

These rules govern the commit history on a branch.

### Subject Size

A commit subject runs to at most 100 characters.

### Conventional Subjects

A subject is a lowercase type, optional (scope), optional !, colon, space, and a nonempty
description. GitHub synthetic merges and Git's exact nonempty Revert "..." form are exempt; Revert
is also size-exempt. Malformed Revert prefixes are not exempt.

### Fixup Commits

A subject is a fixup when it opens with fixup! or squash!, or reads as a review
follow-up; autosquash such commits into their target.

### Merge Commits

A branch other than main or master carries no merge commit; a git-subtree merge
is the sole exception, required when vendoring another repo's history.
