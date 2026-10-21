# Pull request benchmark drift

The `Bench Drift` workflow compares a pull request's base and head revisions on one dedicated
GitHub runner. The suite tool is built from the head revision because the v6 base predates
`gwc bench suite`. Before measuring, the workflow applies the repository's
`normalize-v6-benchmarks.py` driver conversion to the base checkout and compile-checks its eight
representative production packages. It then runs the bounded native suite against both roots in
sequence. Each side uses five samples, one warm-up, a fixed shuffle seed, and production tags.

The gate runs `gwc bench suite compare -max-regress-pct 15`. Suite comparison uses medians and
MAD-derived significance thresholds, then fails only when a significant entry exceeds 15%. This
is a fast regression screen; the full fair v6/v7 benchmark remains the release acceptance gate.

The workflow uploads the two suite reports and comparison output as an artifact named
`bench-drift-<run id>`. Reviewers can use those reports to inspect the machine block, matched
metrics, and top regressions. The tracked
files in this directory (`latest.json`, `reference.json`, and `DRIFT_NOTE.md`) document historical
runs and do not serve as a timing baseline for a pull request.

Pushes to `main` compare the previous commit. Manual runs accept an optional `base_ref` input and
default to `main`. A missing or invalid base revision fails checkout instead of silently comparing
the head against itself.
