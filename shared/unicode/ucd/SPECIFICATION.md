
# Range Tables

Range_Table holds sorted, non-overlapping Unicode ranges. Is, Is_One_Of, and In report
membership without mutable package data.

# Classification

The classification functions report the Unicode control, digit, graphic, letter, case, mark,
number, print, punctuation, space, and symbol properties.

# Case Conversion

To and the named case functions apply simple Unicode case conversion. Special_Case applies its
language rules first. Simple_Fold advances through one simple-fold orbit.

# Named Tables

Named_Table decodes one table into caller-owned 16-bit and 32-bit range storage. The named Is forms
query encoded category, script, property, and fold tables directly.

# Unicode Data

VERSION identifies the Unicode edition. Category_Alias, Turkish_Case, and Azeri_Case supply the
remaining public data that has no direct classification function. Language case functions write
their rules into caller-owned storage.

# Allocation

Every public operation performs zero heap allocation.

# Domain Errors

An invalid Case, range, collection size, name size, or undersized Named_Table or language-case
storage causes a panic. An unknown table name gives a false found report.
