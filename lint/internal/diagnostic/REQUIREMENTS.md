
# Diagnostic

The Diagnostic type carries one rule violation from a check to the printer.

### Position

Position is the offending source location, printed as the clickable
file:line:col prefix. A non-file tier leaves it zero.

### Name

Name is the machine-readable rule identity, stable for tooling that groups or
suppresses by rule.

### Want

Want is the suggested fix, phrased as an imperative sentence.

### Message

Message is the human-readable line printed to stdout.

### Tier

Tier gates printing: 1 always prints and suppresses tier 2 when present, 2
prints only when no tier 1 fired, and a non-file tier leaves it 0.

# Reporting

The scope filter, tier gate, and line format the printer applies to diagnostics.

### Within Scope

A diagnostic is in scope when its file matches the scope prefix; an empty scope
admits all, and a synthetic git filename is always admitted.

### Reportable

Reportable drops out-of-scope diagnostics and, when any tier one fired, every
tier two, since tier two may rely on tier one contracts.

### Format

Format renders a diagnostic as its clickable position and message line.
