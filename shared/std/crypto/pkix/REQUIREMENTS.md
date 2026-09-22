
# Algorithm Identifier

Parse_Algorithm_Identifier borrows one canonical DER sequence and identifies supported OIDs.
Parameters remain borrowed encoded bytes for schema-specific validation.

# Name

Parse_Name validates a DER RDN sequence and borrows the first common-name value when present.

# Bounds

Every encoded input and borrowed field follows the repository DER bound. Oversized input panics.

# Allocation

Every exported runtime operation performs zero heap allocation.
