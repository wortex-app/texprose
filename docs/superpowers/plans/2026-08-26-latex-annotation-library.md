# LaTeX → LanguageTool Annotated Text Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A pure-Go, stdlib-only library that converts LaTeX/BibTeX source into LanguageTool's annotated-text segment form, ported from ltex-ls-plus (Kotlin, MPL-2.0), with byte-exact source reconstruction.

**Architecture:** Single root package `texprose` at module `github.com/wortex-app/texprose`. A character-stream state machine (port of `LatexAnnotatedTextBuilder`) driven by declarative signature tables (generated from the Kotlin defaults), fronted by a fragmentizer that handles magic comments / babel language switches / BibTeX fields, producing an ordered `[]Segment` whose text+markup concatenation reproduces the input byte-for-byte.

**Tech Stack:** Go 1.26/1.27, stdlib only. Throwaway Python scripts (not committed... committed under `internal/gen` as plain scripts? No — committed under `tools/` as documentation of table provenance) generate the signature tables and the NFC accent table.

**Spec:** `latex-annotation-library-spec.md` (repo root). Upstream sources: `ltex-plus/ltex-ls-plus` @ develop, cloned in scratchpad.

## Global Constraints

- Zero third-party dependencies; standard library only.
- License MPL-2.0 (`LICENSE.md`), `NOTICE.md` crediting valentjn/ltex-ls and ltex-plus/ltex-ls-plus; MPL header on every ported file.
- Invariant (hard, fuzz-enforced): concatenation of Segment.Text + Segment.Markup values reproduces input exactly.
- Port behavior faithfully; every deviation documented in code comment + covered by a test.
- Public API exactly as spec §4 (plus the Action vocabulary and a BibTeXFields extension).
- `go test ./...` only; CI = test + vet + gofmt on Linux, Go 1.26 & 1.27.

## Documented deviations from upstream (each needs a test)

1. **Reconstruction over text-preference:** upstream `processWhitespace` emits `addText(" ")` for a single source `\n` (substituting the byte). We emit markup `"\n"` interpreted as `" "` via a raw append that replicates `addText`'s exact state side effects (`textAdded`, no dummy-state reset), preserving the invariant.
2. **Infinite-loop guard keeps bytes:** where upstream logs and skips a character (`pos++`) without emitting it (e.g. `\verb` with no closing delimiter, trailing lone `\`), we emit the skipped character as markup.
3. **`\text`/`\intertext` guard:** upstream blindly appends `command + "{"` as markup even when no `{` follows (breaking reconstruction); we take that branch only when the next char is `{`, else fall through to generic command handling.
4. **No overlapping re-check fragments:** upstream's fragmentizer emits *duplicate* fragments for `\footnote`/`\todo` contents and babel inline commands/environments so nested text is checked twice under different settings. A flat annotation cannot represent overlap; v1 keeps the outer (main-pass) behavior only. Magic comments, `\usepackage[...]{babel}` and `\selectlanguage{}` — which partition the document — are honored.
5. **Invalid user signature prototypes** return an error from `Annotate` instead of being silently skipped with a never-matching regex.

## File Structure

| File | Responsibility |
| --- | --- |
| `segment.go` | `Segment`, `Annotation`, `MarshalJSON`, `PlainText` |
| `options.go` | `CodeLanguage`, `Action`, `Options`, validation |
| `annotate.go` | `Annotate()`: fragmentize → per-fragment build → assemble |
| `builder.go` | port of `CodeAnnotatedTextBuilder`+`CharacterBased`+`LatexAnnotatedTextBuilder` |
| `scan.go` | hand-written token matchers with Java-regex semantics (command, comment, whitespace, verb, accent, lengths, dashes, quotes) |
| `signature.go` | port of `LatexCommandSignature`/`LatexEnvironmentSignature` (+prototype parsing, argument matching) |
| `matcher.go` | port of `LatexCommandSignatureMatcher`/`Match` (lookbehind replaced by scanner) |
| `dummy.go` | port of `DummyGenerator` |
| `fragmentizer.go` | magic comments (`% ltex:` lines), babel usepackage/selectlanguage partition, settings override parsing |
| `bibtex.go` | port of `BibtexFragmentizer` (+ defaults field table) |
| `packageoptions.go` | port of `LatexPackageOption(sParser)` |
| `defaults_gen.go` | generated: command/environment signature tables, babel language map |
| `accents_gen.go` | generated: NFC combining-mark composition table |
| `position.go` | `PositionMapper` (UTF-16 ⇄ byte ⇄ line/character) |
| `tools/gen_defaults.py`, `tools/gen_accents.py` | throwaway generators (committed for provenance) |
| `*_test.go` | ported corpus + property + fuzz tests |

## Tasks

### Task 1: Skeleton
- [ ] go.mod (`github.com/wortex-app/texprose`, `go 1.26`), LICENSE.md (full MPL-2.0), NOTICE.md, `.github/workflows/ci.yml` (test+vet+gofmt, go 1.26/1.27)
- [ ] `segment.go`, `options.go` with the spec §4 types compiling; `Annotate` stub returning empty annotation
- [ ] `go build ./...` passes; commit

### Task 2: Corpus (red)
- [ ] Port `LatexAnnotatedTextBuilderTest` → `builder_test.go` table tests (`assertPlainText(code, want, opts)`), including text/tikz/math/rsweave/context/merging/whitespace cases, minus dictionary-masking cases (feature out of scope — no dictionary in Options)
- [ ] Port position assertions using a test helper mapping plain-text offsets → source offsets over segments
- [ ] Port `BibtexFragmentizerTest`, `LatexPackageOption(sParser)Test`; adapt `LatexFragmentizerTest` per deviation 4 (magic comments partition, babel usepackage/selectlanguage cases)
- [ ] Property test: reconstruction invariant over every corpus input
- [ ] `go test ./...` compiles and fails; commit

### Task 3: Generated tables
- [ ] `tools/gen_defaults.py`: parse Kotlin `LatexAnnotatedTextBuilderDefaults.kt` + `BABEL_LANGUAGE_MAP` + `BibtexFragmentizerDefaults.kt` → `defaults_gen.go` (same order, same actions, plural-dummy flags, `\text<lang>{}` and babel environment derivations)
- [ ] `tools/gen_accents.py`: NFC table for base letters (A–Z, a–z, ı, ȷ) × 15 combining marks → `accents_gen.go`
- [ ] Spot-check counts against Kotlin source; commit

### Task 4: Primitives
- [ ] `dummy.go` (language-prefix tables, plural/vowel variants) with unit tests from the corpus expectations
- [ ] `scan.go` matchers, incl. Java-`$` comment semantics and backreference-free `\verb` matcher; unit tests
- [ ] `signature.go`: prototype parsing (trailing `{}`/`[]`/`()` stripping), `matchArgumentFromPosition` (balanced-brace scanner), `matchFromPosition` with inter-argument comments; unit tests
- [ ] `packageoptions.go` until its ported tests pass
- [ ] `matcher.go` with unescaped-`%`-in-line-prefix scanner replacing the Java lookbehind
- [ ] Commit

### Task 5: Builder state machine
- [ ] Port `LatexAnnotatedTextBuilder` into `builder.go`: part coalescing, mode stack, math dummies, accents, environments, headings, whitespace/quotes/dashes, deviations 1–3
- [ ] Iterate until the LaTeX builder corpus is green; commit

### Task 6: Fragmentizer, BibTeX, user extensions
- [ ] `fragmentizer.go`: `% ltex:` magic comments (enabled / language / latex.commands.* / latex.environments.* / bibtex.fields.*), babel usepackage + selectlanguage partition; disabled or nop fragments → markup
- [ ] `bibtex.go`: entry matcher (`@[A-Za-z]+{}` unescaped prefix), field filtering, gaps → markup
- [ ] Wire `Options.Commands`/`Environments`/`BibTeXFields` merging (sorted for determinism); `Annotate` orchestration in `annotate.go`
- [ ] Corpus fully green; commit

### Task 7: JSON + positions
- [ ] `MarshalJSON` (`{"annotation":[...]}`), `PlainText`; JSON golden test
- [ ] `position.go`: line-start index over the source; `Position`/`Offset` in UTF-16 code units (documented loudly — LanguageTool is Java), plus `ByteOffset`/`UTF16Offset` converters; tests with emoji + combining chars
- [ ] Commit

### Task 8: Fuzz + property
- [ ] `FuzzAnnotate`: never panics, reconstruction invariant, both code languages; corpus inputs as seeds
- [ ] Run `go test -fuzz=FuzzAnnotate -fuzztime=60s` (per code language); fix findings; commit

### Task 9: Release polish
- [ ] README per spec §6 (what/what-not, 20-line example ending at the POST instruction, provenance/license, scope statement)
- [ ] gofmt, go vet, full test run; final commit
