# texprose

[![CI](https://github.com/wortex-app/texprose/actions/workflows/ci.yml/badge.svg)](https://github.com/wortex-app/texprose/actions/workflows/ci.yml)

Convert LaTeX and BibTeX source code into [LanguageTool](https://languagetool.org)'s
annotated-text format, in pure Go.

LanguageTool checks *plain text*; feeding it raw LaTeX produces garbage
diagnostics on every command and formula. Its `/v2/check` endpoint solves this
with the `data` parameter: an ordered list of segments, each either *text*
(checked) or *markup* (skipped, or interpreted as a replacement string such as
a dummy word for a math formula). This library performs exactly that
conversion — a lossless lexical transformation with position fidelity, so that
every match LanguageTool reports points at the right characters of the
original LaTeX source. The concatenation of the text and markup segments
reproduces the input byte-for-byte; this invariant is enforced by tests and a
fuzzer.

## What it deliberately does not do

- **No HTTP.** It never talks to LanguageTool; it only produces the payload
  format. You POST it yourself.
- **No checking.** No grammar or spelling logic, rules, or dictionaries.
- **No LSP.** No editor integration of any kind.
- **No dependencies.** Standard library only.

## Usage

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/wortex-app/texprose"
)

func main() {
	source := "We show that \\emph{clearly} $E = mc^2$ holds.\\cite{einstein}\n"

	annotation, err := texprose.Annotate(source, texprose.Options{
		Language: "en-US",
		Commands: map[string]texprose.Action{
			"\\todo{}": texprose.ActionIgnore, // your own macros
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(annotation.PlainText()) // what LanguageTool will check

	payload, _ := json.Marshal(annotation)
	fmt.Println(string(payload))
}
```

Now POST this JSON as the `data` parameter of LanguageTool's `/v2/check`
yourself (along with `language`), e.g. against a local
`languagetool-server.jar`.

Also included:

- **BibTeX mode** (`Options{CodeLanguage: texprose.BibTeX}`): checks prose
  field values (`title`, ...), skips entry structure and data fields.
- **Magic comments**: `% ltex: language=de-DE`, `% ltex: enabled=false` etc.
  switch settings mid-document, as in LTeX.
- **`PositionMapper`**: converts LanguageTool's reported offsets — which are
  **UTF-16 code units**, because LanguageTool is a Java program — to 0-based
  line/character positions and Go byte offsets. Do not index Go strings with
  raw LanguageTool offsets on non-ASCII documents.

## Provenance and license

The parsing logic is a Go translation of the LaTeX/BibTeX parsing layer of
[ltex-plus/ltex-ls-plus](https://github.com/ltex-plus/ltex-ls-plus) (the
maintained fork of [valentjn/ltex-ls](https://github.com/valentjn/ltex-ls)),
battle-tested there over many years — including its full test corpus. See
`NOTICE.md`. Licensed under the [Mozilla Public License 2.0](LICENSE.md);
MPL-2.0 is file-scoped weak copyleft and imposes nothing on programs that
import this module.

## Scope

Narrow by design: this library converts LaTeX to annotated text, and that's
it. Parsing-fidelity issues that come with a minimal `.tex` reproduction are
the most welcome contribution.
