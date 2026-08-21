
# Simulation

Every run goes through pkg.Main on a seeded simulated loop: usage, a failed walk, a clean package,
and a package holding each diagnostic. Across the seed sweep every assertion in the pkg tree holds
and every Sometimes fires.
