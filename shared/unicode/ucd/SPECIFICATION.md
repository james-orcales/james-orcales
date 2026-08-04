
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

Named_Table returns an immutable copy from a table family and a name. The named Is forms query the
encoded category, script, property, and fold tables without an allocation.

# Unicode Data

VERSION identifies the Unicode edition. Category_Alias, Turkish_Case, and Azeri_Case supply the
remaining public data that has no direct classification function.

# Domain Errors

An invalid Case, range, collection size, or name size causes a panic. An unknown table name gives
a false found report.
