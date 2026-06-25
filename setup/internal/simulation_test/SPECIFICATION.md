
# Mirror

setup.Main mirrors the seed-generated source tree into the destination; over a seed sweep it
converges and stays idempotent, prunes the ignored subtree, and rewrites only the files whose
destination differs.
