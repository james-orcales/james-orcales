
# Simulation

Every run goes through pkg.Main on a seeded simulated loop: usage, a failed walk, a clean package,
and a package holding each diagnostic but the unprintable one. Every assertion holds over the seed
sweep, and every Sometimes fires but three, each naming a value no run of this program states.
