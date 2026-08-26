# Project Specification: Go Library for LaTeX → LanguageTool Annotated Text

This document is the complete, self-contained brief for a new open-source Go
library. An implementing agent should be able to start from this file alone.

## 1. What this library is

[LanguageTool](https://languagetool.org) is an open-source grammar and spell
checker (Java, LGPL-2.1). It is typically run as a local HTTP server
(`languagetool-server.jar`) exposing `POST /v2/check`. LanguageTool checks
*plain text* — it does not understand LaTeX. Feeding it raw LaTeX source
produces garbage diagnostics on every command and math formula.

LanguageTool's check endpoint solves this generically through its `data`
parameter: instead of plain `text`, the caller submits **annotated text** — an
ordered list of segments, each either *text* (checked) or *markup* (skipped, or
"interpreted as" a replacement string such as a newline or a dummy word). The
server checks only the text segments and **maps all match offsets back to the
original source**, markup included.

This library does exactly one thing: **convert LaTeX (and BibTeX) source code
into that annotated-segment form, in pure Go.** It is a lossless lexical
transformation with position fidelity, so that a match reported by LanguageTool
points at the right characters of the original LaTeX source.

Explicit non-goals — the library must NOT contain:

- Any HTTP client, or any code that calls LanguageTool. Consumers do that
  themselves. The library only *produces the payload format*.
- Any LSP (Language Server Protocol) code.
- Any grammar/spell-checking logic, rule handling, or dictionary handling.
- Any service wiring, goroutine pools, caching, or configuration loading.

Zero third-party dependencies. Standard library only.

## 2. Provenance: this is a port, not a from-scratch design

The conversion logic already exists, battle-tested over many years, inside
**LTeX**, an editor language server that fronts LanguageTool for markup
documents. The original project `valentjn/ltex-ls` was archived (read-only) in
April 2026; the maintained fork is:

- **https://github.com/ltex-plus/ltex-ls-plus** (Kotlin, MPL-2.0)

The library is a **Kotlin → Go port of the LaTeX/BibTeX parsing layer of
ltex-ls-plus**. Port behavior faithfully first; refactor into idiomatic Go
second. Do not "improve" parsing behavior during the port — every deviation
from upstream behavior must be justified and covered by a test.

### 2.1 Source files to port

Expected locations in the ltex-ls-plus repository (verify against the current
`develop` branch before starting; the structure was inherited from the original
project and may have drifted):

Core (under `src/main/kotlin/org/bsplines/ltexls/parsing/`):

| File | Role |
| --- | --- |
| `latex/LatexAnnotatedTextBuilder.kt` | The main character-stream state machine: walks LaTeX source, emits text/markup segments, tracks math mode, handles accents, punctuation, sentence breaks. **The heart of the port.** |
| `latex/LatexAnnotatedTextBuilderDefaults.kt` | Large declarative tables: default command signatures and environment names with their handling actions. Years of accumulated edge cases live here — port these tables completely and mechanically. |
| `latex/LatexCommandSignature.kt` | Model of a command signature (name + argument spec like `{}`, `[]` and the action to take). |
| `latex/LatexCommandSignatureMatch.kt`, `latex/LatexCommandSignatureMatcher.kt` | Matching commands in the source against signatures. |
| `latex/LatexEnvironmentSignature.kt` | Same idea for environments (`\begin{...}` / `\end{...}`). |
| `latex/LatexFragmentizer.kt` | Splits a document into fragments (also parses in-source settings comments; see §5.6). |
| `latex/LatexPackageOption.kt`, `latex/LatexPackageOptionsParser.kt` | Parsing of package/command options. |
| `CodeAnnotatedTextBuilder.kt` (parent package) | Abstract base for builders — fold into the Go design rather than mirroring the class hierarchy. |
| `DummyGenerator.kt`, `SentenceEnd`-related helpers (parent package) | Generation of dummy placeholder words that stand in for math/unshowable content (see §5.3). |
| `bibtex/BibtexAnnotatedTextBuilder.kt` + `bibtex/BibtexFragmentizer.kt` | BibTeX mode: checks free-text fields (e.g. `title`), skips structural fields. Reuses the LaTeX builder internally. |

Also relevant: upstream's settings model for user-supplied extra commands and
environments (in `settings/`), which defines the accepted action vocabulary
(e.g. ignore, dummy, vowel-dummy) — needed to shape this library's `Options`.

Do NOT port: everything else — the LSP server layer, LanguageTool invocation
(`languagetool/` package), code actions, diagnostics, settings files, Markdown /
Org / reStructuredText / HTML builders (out of scope for v1; the architecture
may leave room for them later).

### 2.2 Tests to port

Upstream's builder tests are (LaTeX input → expected plaintext) pairs — they
are the de-facto specification of the state machine. Expected location:
`src/test/kotlin/org/bsplines/ltexls/parsing/latex/LatexAnnotatedTextBuilderTest.kt`
(plus the bibtex equivalent). **Port the full corpus into table-driven Go tests
before porting the builder**, then implement until the corpus passes. Where
upstream asserts only plaintext output, add assertions on the segment structure
and position mapping as the port stabilizes.

## 3. Licensing (requirement, not a suggestion)

- The port is a **derivative work of MPL-2.0 files**. The repository is
  licensed **MPL-2.0** as a whole. Include the full license text as
  `LICENSE.md`.
- Every ported source file keeps an MPL-2.0 header (the standard
  `This Source Code Form is subject to the terms of the Mozilla Public License,
  v. 2.0. ...` exhibit), plus upstream copyright lines.
- Add a `NOTICE.md` crediting: the original `valentjn/ltex-ls` project and the
  `ltex-plus/ltex-ls-plus` fork, with repository URLs, stating that the
  LaTeX/BibTeX parsing logic is a Go translation of their code.
- MPL-2.0 is file-scoped weak copyleft: it imposes nothing on programs that
  import this module.

## 4. Public API

Small on purpose. One root package (module path and package name to be chosen
at repo creation; the name should be generic and searchable, not branded).

```go
// A Segment is either text (checked by LanguageTool) or markup (skipped).
// Exactly one of Text or Markup is set. InterpretAs is only valid on markup
// segments: LanguageTool pretends the markup reads as that string (used for
// dummy words replacing math, and for newlines/paragraph breaks).
type Segment struct {
    Text        string
    Markup      string
    InterpretAs string
}

type Annotation struct {
    Segments []Segment
}

// MarshalJSON emits LanguageTool's /v2/check "data" parameter format (§5.1).
func (a Annotation) MarshalJSON() ([]byte, error)

// PlainText returns what LanguageTool will effectively check: the
// concatenation of text segments and InterpretAs values.
func (a Annotation) PlainText() string

type CodeLanguage string

const (
    LaTeX  CodeLanguage = "latex"
    BibTeX CodeLanguage = "bibtex"
)

type Options struct {
    CodeLanguage CodeLanguage // default LaTeX
    // Checking language (BCP-47 style, e.g. "en-US", "de-DE"). Affects dummy
    // generation (§5.3) and language-specific quotation/punctuation handling.
    Language string
    // User-supplied additions to the built-in tables, using the same action
    // vocabulary as the defaults (ignore / dummy / vowel-dummy / ...), e.g.
    // {`\todo{}`: ActionIgnore, `\person{}`: ActionDummy}.
    Commands     map[string]Action
    Environments map[string]Action
}

func Annotate(source string, opts Options) (Annotation, error)
```

Position utilities (needed because LanguageTool reports flat offsets while most
consumers address documents by line/column):

```go
// Position is 0-based line and 0-based column.
type Position struct{ Line, Character int }

// PositionMapper converts between representations of a location in source.
// Construct once per document; must be cheap to query.
func NewPositionMapper(source string) *PositionMapper
func (m *PositionMapper) Position(offset int) Position
func (m *PositionMapper) Offset(p Position) int
```

⚠️ **Offset units must be pinned down early in implementation.** LanguageTool
is Java; its reported `offset`/`length` count what Java counts — verify against
a running LanguageTool instance whether that is UTF-16 code units or Unicode
code points, using inputs with non-BMP characters (e.g. emoji) and combining
characters. The mapper must support whichever unit LanguageTool uses *and*
document it loudly; Go-native byte offsets should also be obtainable. Getting
this wrong produces diagnostics that drift on any document containing non-ASCII
text, which for a LaTeX audience is most documents.

## 5. Behavior the port must preserve

The ported test corpus is the authoritative spec; this section is the map of
what those behaviors are, so the implementer knows what they're looking at.

### 5.1 Output format contract

LanguageTool's `data` parameter (see the API references in §8) is:

```json
{"annotation": [
  {"text": "A sentence about "},
  {"markup": "\\LaTeX{}", "interpretAs": "LaTeX"},
  {"text": ", with math "},
  {"markup": "$x^2$", "interpretAs": "Dummy0"},
  {"text": "."}
]}
```

Invariant: **the concatenation of `text` + `markup` values, in order, must
reproduce the input source byte-for-byte.** This is what makes LanguageTool's
returned offsets land on the original source. Make this invariant a property
test that runs against the whole corpus and the fuzzer (§6).

### 5.2 Command and environment handling

Driven by the signature tables. Categories of behavior (all present in the
upstream defaults; the tables define which command gets which):

- Commands whose argument is prose and stays checked: `\textbf{...}`,
  `\emph{...}`, `\section{...}` (markup: the command tokens; text: the
  argument content).
- Commands dropped entirely, including arguments: `\label{...}`, `\cite{...}`
  (variants may interpret as a dummy so sentence grammar survives),
  `\usepackage[...]{...}`, `\ref{...}`, etc.
- Commands interpreted as a fixed string: `\LaTeX` → "LaTeX", `\&` → "&",
  `\dots` → "…", tilde `~` → non-breaking space, `--`/`---` → en/em dash.
- Accent commands folded to precomposed Unicode: `\"o` → ö, `\'e` → é, `\ss` →
  ß, etc.
- Environments: `\begin{...}`/`\end{...}` markers are markup; content handling
  depends on the environment — prose environments stay checked, verbatim-like
  environments (`verbatim`, `lstlisting`, `minted`, ...) are skipped entirely,
  math environments follow §5.3.
- Comments (`%` to end of line) are markup; comment-only lines must not break
  sentence continuity incorrectly.
- Paragraph breaks (blank lines) are markup interpreted as `\n\n`; single
  newlines within a paragraph as a space-equivalent, matching upstream.

### 5.3 Math mode → dummy words

The signature feature, and the hardest part. Inline math (`$...$`, `\(...\)`)
and display math (`$$...$$`, `\[...\]`, `equation`, `align`, etc.) cannot be
checked, but simply deleting a formula destroys the sentence around it
("… where $E=mc^2$ holds" must remain a grammatical sentence). Upstream
replaces each math island with a generated **dummy word** (e.g. `Dummy0`,
`Dummy1`, …) via `interpretAs`, with variants chosen so the surrounding grammar
still parses — including language-aware behavior (e.g. dummies acting as nouns;
a "vowel dummy" variant where "a" vs "an" agreement matters; German plural
handling). Trailing punctuation inside or after math must end up in the right
place in the plain text. Port `DummyGenerator` and the math-mode states of the
builder faithfully; this area has the densest test coverage upstream, which is
not an accident.

### 5.4 BibTeX mode

Checks only human-prose field values (`title`, `note`, ...), skips entry
structure, keys, and data fields. Field values themselves may contain LaTeX,
so this mode wraps the LaTeX builder. Upstream has a default table of which
fields are prose.

### 5.5 User extensions

`Options.Commands` / `Options.Environments` merge over the built-in tables
using the same signature syntax upstream uses in its settings (e.g.
`"\\todo{}"` — trailing `{}`/`[]` marking the argument shape) and the same
action vocabulary. This is required day-one: every real document uses custom
macros, and per-document extension is how downstream tooling deals with them.

### 5.6 In-source settings comments

Upstream recognizes magic comments in the source (e.g.
`% LTeX: enabled=false`, `language=de-DE`) via the fragmentizer, toggling
checking or switching language mid-document. Port the fragmentizer's mechanism;
whether the library honors all keys or exposes fragments to the caller can be
decided during implementation — but mid-document language switching affects
dummy generation, so the hook must exist.

## 6. Repository conventions

- Pure Go, stdlib only. Support the two most recent Go releases.
- Plain `go test ./...` — no build system beyond the Go toolchain. CI via
  GitHub Actions: test + vet + gofmt check, on Linux at minimum.
- Table-driven tests from the ported corpus (§2.2). Additionally:
  - A **property test**: for arbitrary corpus inputs, concatenated segment
    sources reproduce the input exactly (§5.1 invariant).
  - **Native Go fuzzing** (`go test -fuzz`) on `Annotate`: must never panic,
    never loop forever, and always satisfy the reconstruction invariant, for
    arbitrary byte sequences. A lexer state machine is the ideal fuzz target;
    wire the corpus in as the seed corpus.
- Semantic versioning; tag `v0.x` until the API settles.
- README: what it does, what it deliberately doesn't do (no HTTP, no checking),
  a 20-line usage example ending at "POST this JSON as the `data` parameter of
  LanguageTool's `/v2/check` yourself", provenance and license notice, and a
  scope statement ("narrow by design; parsing-fidelity issues with a minimal
  `.tex` reproduction are the most welcome contribution").

## 7. Suggested milestones

1. **Skeleton** — module, LICENSE.md, NOTICE.md, CI, empty API with types
   compiling.
2. **Corpus** — port the upstream LaTeX + BibTeX test corpus to Go tables;
   all red.
3. **Tables** — port `LatexAnnotatedTextBuilderDefaults` signature tables
   (mechanical, large; consider generating the Go from the Kotlin source with
   a throwaway script, then committing the generated Go).
4. **State machine** — port the builder + dummy generator until the corpus is
   green. This is the bulk of the work.
5. **Fragmentizer + BibTeX + user extensions** (§5.4–5.6).
6. **Positions + fuzz** — PositionMapper with the offset-unit question (§4)
   resolved empirically against a real LanguageTool server; fuzz targets;
   property test.
7. **v0.1.0** — README, examples, tag.

Milestones 3–5 should each land with their slice of the corpus passing; don't
port the whole state machine in one opaque step.

## 8. External references

- Port source: https://github.com/ltex-plus/ltex-ls-plus (branch `develop`);
  original archived project: https://github.com/valentjn/ltex-ls
- LanguageTool HTTP API (OpenAPI spec, documents the `data` annotated-text
  parameter and the match/offset response schema):
  https://languagetool.org/http-api/languagetool-swagger.json
  (browsable: https://languagetool.org/http-api/swagger-ui/)
- Running a local LanguageTool server:
  https://dev.languagetool.org/http-server
- MPL-2.0 text: https://www.mozilla.org/en-US/MPL/2.0/
- LTeX documentation of command/environment settings and magic comments
  (shapes `Options` and §5.6): https://ltex-plus.github.io/ltex-plus/
