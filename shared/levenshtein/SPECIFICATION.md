
# Distance

Distance returns the Levenshtein edit distance between two strings: the fewest
single-rune insertions, deletions, or substitutions that turn From into To.

### Cases

Equal strings are zero apart and an empty string is the other's length away; a lone
insertion, deletion, or substitution each costs one; and the distance is symmetric.

# Closest

Closest returns the candidate nearest the target by edit distance, when one is near
enough to be a likely typo rather than an unrelated string.

### Cases

A near-miss returns its candidate and true; a wild miss or an empty candidate set
returns the empty string and false; tied candidates keep the earliest.
