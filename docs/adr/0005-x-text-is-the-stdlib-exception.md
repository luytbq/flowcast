# ADR-0005: golang.org/x/text is the exception to the stdlib-only core rule

- Status: accepted
- Date: 2026-09-21

## Context

The core uses only the standard library. That rule was chosen to keep the binary small,
keep a future WASM build light, and avoid having to track a supply chain.

But reading tables requires Unicode normalization to NFC, and Go's standard library
does not have it.

Normalization is not cosmetic. The same text seen on screen can be stored as several
different code sequences: "Xử lý" in precomposed form is 5 Unicode characters, in
decomposed form it is 8. The tool measures text by summing the width of each
character, so the two forms give 30.75px and 42.43px, a 38% difference. Text width
determines box width, box width determines lane width, and lanes determine the
coordinates of everything.

Vietnamese is hit harder than most languages because many letters stack two marks: a
letter-forming mark as in ư, ơ, ă, â, ê, ô, đ, plus a tone mark.

The decomposed form is not hypothetical. macOS stores file names in that form, some
editors and some Vietnamese input methods produce it, and so does copy-pasting between
some applications. Users do not know which form their file is in.

The two remaining options considered:

- **Generate a composition table from Python and commit it**, the same way the font
  measurement table was made. This keeps the rule, but requires writing the ordering of
  combining marks too, which means rewriting part of the Unicode normalization
  algorithm, and the table is only correct within the range of characters generated.
- **Do not normalize, only check and report an error.** Doable with the stdlib, but
  files copied from macOS get rejected instead of working, and the behavior diverges
  sharply from the reference implementation.

## Decision

The core uses `golang.org/x/text/unicode/norm`. This is the only exception granted to
the stdlib-only rule, and it applies only to Unicode normalization.

The version is pinned at v0.30.0, because newer versions require Go 1.26 while the
project's compatibility baseline is Go 1.25.

## Consequences

- The binary grows by about 219 KB, from 2331 to 2551 KB. Acceptable for the CLI and
  the server; needs remeasuring if WASM comes into scope.
- A single dependency, maintained by the Go team itself.
- **This exception sets no precedent.** Adding any other dependency to the core still
  requires its own ADR and must prove that the stdlib offers no way, not that the
  external library is more convenient. The reason for accepting it here is that
  rewriting Unicode normalization correctly is a minefield, not that it saves effort.
- `conformance/cases/30-input-nfd.md` keeps this branch alive. Before that case every
  table in the conformance suite was already precomposed, so a port that forgot to
  normalize would still pass every gate.
