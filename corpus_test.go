// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Test corpus ported from ltex-ls-plus:
// src/test/kotlin/org/bsplines/ltexls/parsing/latex/LatexAnnotatedTextBuilderTest.kt

package texprose

import (
	"testing"
	"unicode/utf8"
)

type corpusCase struct {
	name string
	code string
	want string
	opts Options
}

func german() Options { return Options{Language: "de-DE"} }
func lang(l string) Options {
	return Options{Language: l}
}

var latexCorpus = []corpusCase{
	{
		name: "itemize",
		code: "We can do\n\\begin{itemize}[first-{test}]{[second]-test}\n  \\item this or\n  \\item that.\n\\end{itemize}\n",
		want: "We can do this or that. ",
	},
	{
		name: "frames",
		code: "This is a test.\n\\begin{frame}{Test 1}{Test 2}\n  Frame 1.\n\\end{frame}\n\\begin{frame}[noframenumbering]{Test 3}{Test 4}\n  Frame 2.\n\\end{frame}\nFinal sentence.\n",
		want: "This is a test. Test 1 Test 2 Frame 1. Test 3 Test 4 Frame 2. Final sentence. ",
	},
	{
		name: "dots",
		code: "This is good\\dots No, it isn't.\n",
		want: "This is good\u2026 No, it isn't. ",
	},
	{
		name: "line breaks",
		code: "This is a test of\\\\line breaks.\n",
		want: "This is a test of line breaks. ",
	},
	{
		name: "comment paragraph break",
		code: "This is a sentence.%\n\nThis is another sentence.\n",
		want: "This is a sentence.\n\nThis is another sentence. ",
	},
	{
		name: "textcolor",
		code: "This is a \\textcolor{mittelblau}{test}.\n",
		want: "This is a test. ",
	},
	{
		name: "raisebox",
		code: "This is a \\raisebox{-0.5\\height-0.5mm}{test}.\n",
		want: "This is a test. ",
	},
	{
		name: "ampersand",
		code: "This is a &test.\n",
		want: "This is a test. ",
	},
	{
		name: "hyperref",
		code: "You can see this in \\hyperref[alg:abc]{Sec.\\ \\ref*{alg:abc}}.\n",
		want: "You can see this in Sec. Dummy0. ",
	},
	{
		name: "soft hyphens and german quotes",
		code: "This is a te\\-st. Another te\"-st. Donau\"=Dampf\"\"schiff\"~Fahrt.\n",
		want: "This is a test. Another test. Donau-Dampfschiff-Fahrt. ",
	},
	{
		name: "sharp s",
		code: "Ich hei\\ss{}e anders. Das Wasser ist hei\\ss.\n",
		want: "Ich hei\u00dfe anders. Das Wasser ist hei\u00df. ",
	},
	{
		name: "euro and spaces",
		code: "Das macht dann 10 \\euro. Oder z.\\,B. vielleicht doch 12~\\euro{}?\n",
		want: "Das macht dann 10 \u20ac. Oder z.\u202fB. vielleicht doch 12\u00a0\u20ac? ",
	},
	{
		name: "umlauts",
		code: "\\\"E\\\"in T\\\"ext m\\\"{i}t v\\\"i\\\"{e}l\\\"en \\\"{U}ml\\\"a\\\"{u}t\\\"en.\n",
		want: "\u00cb\u00efn T\u00ebxt m\u00eft v\u00ef\u00ebl\u00ebn \u00dcml\u00e4\u00fct\u00ebn. ",
	},
	{
		name: "special letters uppercase",
		code: "\\AA\\L\\O\\SS",
		want: "\u00c5\u0141\u00d8\u1e9e",
	},
	{
		name: "special letters lowercase",
		code: "\\aa\\i\\j\\l\\o\\ss",
		want: "\u00e5\u0131\u0237\u0142\u00f8\u00df",
	},
	{
		name: "accents uppercase",
		code: "\\`A\\'A\\^A\\~A\\\"A\\=A\\.A\\H{O}\\b{B}\\c{C}\\d{A}\\k{A}\\r{A}\\u{A}\\v{C}",
		want: "\u00c0\u00c1\u00c2\u00c3\u00c4\u0100\u0226\u0150\u1e06\u00c7\u1ea0\u0104\u00c5\u0102\u010c",
	},
	{
		name: "accents lowercase",
		code: "\\`a\\'a\\^a\\~a\\\"a\\=a\\.a\\H{o}\\b{b}\\c{c}\\d{a}\\k{a}\\r{a}\\u{a}\\v{c}",
		want: "\u00e0\u00e1\u00e2\u00e3\u00e4\u0101\u0227\u0151\u1e07\u00e7\u1ea1\u0105\u00e5\u0103\u010d",
	},
	{
		name: "accents mixed",
		code: "Nih\\'o\u014b\\=go \\u{y} {\\v Z}{\\'a}k",
		want: "Nih\u00f3\u014b\u1e21o y\u0306 \u017d\u00e1k",
	},
	{
		name: "dots in list",
		code: "This is a test: a, b, \\dots, c.\n",
		want: "This is a test: a, b, \u2026, c. ",
	},
	{
		name: "dashes",
		code: "This is a test -- this is another test --- this is the final test.\n",
		want: "This is a test \u2013 this is another test \u2014 this is the final test. ",
	},
	{
		name: "smart quotes",
		code: "This ``is'' a \"`test.\"'\n",
		want: "This \u201cis\u201d a \u201etest.\u201c ",
	},
	{
		name: "headings",
		code: "\\section{Heading}\nThis is a test.\n\\subsection[abc]{This is another heading.}\nThis is another test.\n",
		want: "Heading. This is a test. This is another heading. This is another test. ",
	},
	{
		name: "titleformat",
		code: "\\titleformat{\\chapter}{\\huge\\normalfont\\bfseries}{\\thechapter}{1em}{}\n\\section{Heading}\nTest text.\n",
		want: " Heading. Test text. ",
	},
	{
		name: "cites",
		code: "This is a test: \\cite{test1}, \\cite[a]{test2}, \\cite[a][b]{test3}.\n\\textcites{test1}{test2}{test3} shows that this should be plural.\n\\textcites(a)(b)[c][]{test1}[][d]{test2}[e][f]{test3} proves another error.\n",
		want: "This is a test: Dummy0, Dummy1, Dummy2. Dummies shows that this should be plural. Dummies proves another error. ",
	},
	{name: "cites en", code: "\\cites{test}", want: "Dummies"},
	{name: "cites de", code: "\\cites{test}", want: "Dummys", opts: german()},
	{name: "cites sv", code: "\\cites{test}", want: "Dumma-0", opts: lang("sv")},
	{name: "cites es", code: "\\cites{test}", want: "Maniqu\u00edes-0", opts: lang("es-AR")},
	{name: "cites nl", code: "\\cites{test}", want: "Dummy's+0", opts: lang("nl")},
	{
		name: "eg and ie",
		code: "This is a test, \\egc an actual test \\eg{} test.\nThis is a test, \\iec an actual test \\ie{} test.\n",
		want: "This is a test, e.g., an actual test e.g. test. This is a test, i.e., an actual test i.e. test. ",
	},
	{
		name: "textblock",
		code: "This is a test.\n\\begin{textblock*}{1mm}[2mm,3mm](4mm,5mm)\n  abc\\end{textblock*}\nThis is another test.\n",
		want: "This is a test. abc This is another test. ",
	},
	{
		name: "setcounter mismatched signature",
		code: "\\setcounter{a}[b]{c} This is an test.\n",
		want: "a[b]c This is an test. ",
	},
	{
		name: "colorbox",
		code: "This is a test: \\colorbox{abc}{def}.\n",
		want: "This is a test: def. ",
	},
	{
		name: "colorbox user dummy",
		code: "This is a test: \\colorbox{abc}{def}.\n",
		want: "This is a test: Dummy0def. ",
		opts: Options{Commands: map[string]Action{"\\colorbox{}": ActionDummy}},
	},
	{
		name: "unknown command",
		code: "This is a test: \\foobar{abc}{def}.\n",
		want: "This is a test: abc def. ",
	},
	{
		name: "user command default",
		code: "This is a test: \\foobar{abc}{def}.\n",
		want: "This is a test: abc def. ",
		opts: Options{Commands: map[string]Action{"\\foobar{}{}": ActionDefault}},
	},
	{
		name: "user command ignore",
		code: "This is a test: \\foobar{abc}{def}.\n",
		want: "This is a test: . ",
		opts: Options{Commands: map[string]Action{"\\foobar{}{}": ActionIgnore}},
	},
	{
		name: "user command dummy",
		code: "This is a test: \\foobar{abc}{def}.\n",
		want: "This is a test: Dummy0. ",
		opts: Options{Commands: map[string]Action{"\\foobar{}{}": ActionDummy}},
	},
	{
		name: "user command plural dummy",
		code: "This is a test: \\foobar{abc}{def}.\n",
		want: "This is a test: Dummies. ",
		opts: Options{Commands: map[string]Action{"\\foobar{}{}": ActionPluralDummy}},
	},
	{
		name: "user command vowel dummy",
		code: "This is a test: \\foobar{abc}{def}.\n",
		want: "This is a test: Ina0. ",
		opts: Options{Commands: map[string]Action{"\\foobar{}{}": ActionVowelDummy}},
	},
	{
		name: "unknown environment",
		code: "This is a test: \\begin{foobar}{abc}def\\end{foobar}.\n",
		want: "This is a test: def. ",
	},
	{
		name: "user environment default bare",
		code: "This is a test: \\begin{foobar}{abc}def\\end{foobar}.\n",
		want: "This is a test: def. ",
		opts: Options{Environments: map[string]Action{"foobar": ActionDefault}},
	},
	{
		name: "user environment default prototype without args",
		code: "This is a test: \\begin{foobar}{abc}def\\end{foobar}.\n",
		want: "This is a test: abcdef. ",
		opts: Options{Environments: map[string]Action{"\\begin{foobar}": ActionDefault}},
	},
	{
		name: "user environment default prototype with args",
		code: "This is a test: \\begin{foobar}{abc}def\\end{foobar}.\n",
		want: "This is a test: def. ",
		opts: Options{Environments: map[string]Action{"\\begin{foobar}{}": ActionDefault}},
	},
	{
		name: "user environment ignore",
		code: "This is a test: \\begin{foobar}{abc}def\\end{foobar}.\n",
		want: "This is a test: . ",
		opts: Options{Environments: map[string]Action{"foobar": ActionIgnore}},
	},

	// testTikzMode
	{
		name: "tikzset",
		code: "This is a \\tikzset{bla}test.\n",
		want: "This is a test. ",
	},
	{
		name: "tikzpicture",
		code: "This is a test.\n\\begin{tikzpicture}\n  \\node[color=mittelblau] at (42mm,0mm) {qwerty};\n\\end{tikzpicture}\nThis is another sentence.\n",
		want: "This is a test. This is another sentence. ",
	},
	{
		name: "tikzpicture with math",
		code: "This is a test:\n\\begin{tikzpicture}\n  \\node {$\\dots$};\n  \\node {$a$};\n\\end{tikzpicture}\n",
		want: "This is a test: ",
	},

	// testMathMode
	{
		name: "cases",
		code: "Recall that\n\\begin{equation*}\n  \\begin{cases}\n    a&\\text{if $b$,}\\\\\n    c&\\text{otherwise.}\n  \\end{cases}\n\\end{equation*}\nNow we argue.\n",
		want: "Recall that Dummy0 if Dummy1, Dummy2 otherwise. Now we argue. ",
	},
	{
		name: "inline math vowel",
		code: "This equals $a^{b}$.\n",
		want: "This equals Ina0. ",
	},
	{
		name: "equation with hspace qed",
		code: "This is the proof:\n\\begin{equation}\n    a^2 + b^2 = c^2\\hspace*{10mm}.\\quad\\qed\n\\end{equation}\n",
		want: "This is the proof: Dummy0. ",
	},
	{
		name: "equation with line break notag",
		code: "This is another proof:\n\\begin{equation}\n    a^2 + b^2 = c^2.\\\\[-6.4em]\\qquad\\notag\n\\end{equation}\n",
		want: "This is another proof: Dummy0. ",
	},
	{
		name: "equation with split",
		code: "This equals\n\\begin{equation}\n  \\begin{split}\n    abcdef.\n  \\end{split}\n\\end{equation}\nThis is the next sentence.\n",
		want: "This equals Dummy0. This is the next sentence. ",
	},
	{
		name: "equation with trailing text",
		code: "This is an equation:\n\\begin{equation}\n    a^2 + b^2 = c^2,\\qquad\\text{which proves the theorem.}\\end{equation}%\nThis is a sentence.\n",
		want: "This is an equation: Dummy0, which proves the theorem. This is a sentence. ",
	},
	{
		name: "math text with tie",
		code: "This is a test:\n\\begin{equation*}\n  a \\text{,~and} b.\n\\end{equation*}\n",
		want: "This is a test: Dummy0,\u00a0and Dummy1. ",
	},
	{
		name: "math with sharp s",
		code: "This is a test:\n\\begin{equation}\n    Gau\\ss{}: \\O(n^2).\n\\end{equation}\nThis is another test: $Gau\\ss{}: \\O(n^2)$.\n",
		want: "This is a test: Dummy0. This is another test: Dummy1. ",
	},
	{
		name: "display and inline math brackets",
		code: "This is a test:\n\\[\n  E = mc^2.\n\\]\nAnd this is another one: \\(c^2\\).\n",
		want: "This is a test: Dummy0. And this is another one: Dummy1. ",
	},
	{
		name: "math with footnote",
		code: "This is a test: $a = b \\footnote{This is another test: $c$.}$.\nThis is the next sentence: $E = mc^2$.\n",
		want: "This is a test: Ina0. This is the next sentence: Ina1. ",
	},
	{
		name: "math dots",
		code: "This is a test: $a, b, \\dots, c$.\nSecond sentence: a, b, $\\dots$, c.\n",
		want: "This is a test: Ina0. Second sentence: a, b, Dummy1, c. ",
	},
	{name: "math fr", code: "C'est un test: $E = mc^2$.\n", want: "C'est un test: Jimmy-0. ", opts: lang("fr")},
	{name: "math sv", code: "Detta \u00e4r ett test: $E = mc^2$.\n", want: "Detta \u00e4r ett test: Dummy-0. ", opts: lang("sv")},
	{name: "math es", code: "Esto es una prueba: $E = mc^2$.\n", want: "Esto es una prueba: Maniqu\u00ed-0. ", opts: lang("es")},
	{name: "math nl", code: "Dit is een test: $E = mc^2$.\n", want: "Dit is een test: Dummy+0. ", opts: lang("nl")},
	{name: "math pl", code: "To jest test: $E = mc^2$.\n", want: "To jest test: Atrapa-0. ", opts: lang("pl")},
	{
		name: "math vowel states",
		code: "This is an $A$, $e$, $F$, $h$, $I$, $l$, $M$,\n$n$, $O$, $r$, $S$, $X$, $\\ell$, $\\mathcal{r}$.\nThis is not a $b$, $C$, $\\ella$, $\\test a$, $\\mathcal{b}$.\n",
		want: "This is an Ina0, Ina1, Ina2, Ina3, Ina4, Ina5, Ina6, Ina7, Ina8, Ina9, Ina10, Ina11, Ina12, Ina13. This is not a Dummy14, Dummy15, Dummy16, Dummy17, Dummy18. ",
	},

	// testRsweave
	{
		name: "rsweave",
		code: "\\SweaveOpts{prefix.string=figures}\nThis is a first sentence.\n\n<<import-packages, echo=false>>=\nlibrary(tidyverse, quietly = T)\n@\n\nThis is a second sentence.\n<<mca-graph, fig=true, echo=false>>=\nplot(1:1000, rnorm(1000))\n@\n",
		want: " This is a first sentence.\n\n\n\nThis is a second sentence. ",
		opts: Options{CodeLanguage: Rsweave},
	},
	{
		name: "rsweave chunk only",
		code: "<<import-packages>>=\nlibrary(tidyverse)\n@\n",
		want: " ",
		opts: Options{CodeLanguage: Rsweave},
	},
	{
		name: "rsweave syntax as latex",
		code: "<<import-packages>>=\nlibrary(tidyverse)\n@\n",
		want: "<<import-packages>>= library(tidyverse) @ ",
	},

	// testContext (\start.../\stop... commands; behavior is independent of the
	// code language id upstream, so checked under plain LaTeX)
	{
		name: "context formula",
		code: "This is a first sentence.\n\n\\startformula\nE = mc^2\n\\stopformula\n\nThis is a second sentence.\n",
		want: "This is a first sentence.\n\nDummy0 \n\nThis is a second sentence. ",
	},
}

func annotateOrFail(t *testing.T, code string, opts Options) Annotation {
	t.Helper()
	annotation, err := Annotate(code, opts)
	if err != nil {
		t.Fatalf("Annotate(%q) failed: %v", code, err)
	}
	return annotation
}

func assertReconstruction(t *testing.T, code string, annotation Annotation) {
	t.Helper()
	var reconstructed string
	for _, segment := range annotation.Segments {
		if segment.Text != "" && segment.Markup != "" {
			t.Errorf("segment has both Text %q and Markup %q", segment.Text, segment.Markup)
		}
		if segment.Text != "" && segment.InterpretAs != "" {
			t.Errorf("text segment %q has InterpretAs %q", segment.Text, segment.InterpretAs)
		}
		reconstructed += segment.Text + segment.Markup
	}
	if reconstructed != code {
		t.Errorf("segments do not reconstruct source:\ngot  %q\nwant %q", reconstructed, code)
	}
}

func TestLatexCorpus(t *testing.T) {
	for _, testCase := range latexCorpus {
		t.Run(testCase.name, func(t *testing.T) {
			annotation := annotateOrFail(t, testCase.code, testCase.opts)
			if got := annotation.PlainText(); got != testCase.want {
				t.Errorf("plain text mismatch:\ncode %q\ngot  %q\nwant %q", testCase.code, got, testCase.want)
			}
			assertReconstruction(t, testCase.code, annotation)
		})
	}
}

// originalPosition maps an offset in the plain text (in runes) to a byte
// offset in the source, mimicking LanguageTool's
// AnnotatedText.getOriginalTextPositionFor for the ported position tests.
func originalPosition(annotation Annotation, plainPos int, isEnd bool) int {
	plain := 0
	orig := 0

	for _, segment := range annotation.Segments {
		if segment.Text != "" {
			n := utf8.RuneCountInString(segment.Text)
			if plainPos < plain+n || (isEnd && plainPos == plain+n) {
				byteOffset := 0
				for i := 0; i < plainPos-plain; i++ {
					_, size := utf8.DecodeRuneInString(segment.Text[byteOffset:])
					byteOffset += size
				}
				return orig + byteOffset
			}
			plain += n
			orig += len(segment.Text)
		} else {
			n := utf8.RuneCountInString(segment.InterpretAs)
			if n > 0 && plainPos < plain+n {
				if isEnd {
					return orig + len(segment.Markup)
				}
				return orig
			}
			plain += n
			orig += len(segment.Markup)
		}
	}

	return orig
}

// plainPosition is the inverse: maps a source byte offset to a plain-text
// rune offset.
func plainPosition(annotation Annotation, origPos int, isEnd bool) int {
	plain := 0
	orig := 0

	for _, segment := range annotation.Segments {
		if segment.Text != "" {
			if origPos < orig+len(segment.Text) || (isEnd && origPos == orig+len(segment.Text)) {
				return plain + utf8.RuneCountInString(segment.Text[:origPos-orig])
			}
			plain += utf8.RuneCountInString(segment.Text)
			orig += len(segment.Text)
		} else {
			if origPos < orig+len(segment.Markup) {
				if isEnd {
					return plain + utf8.RuneCountInString(segment.InterpretAs)
				}
				return plain
			}
			plain += utf8.RuneCountInString(segment.InterpretAs)
			orig += len(segment.Markup)
		}
	}

	return plain
}

func TestOriginalTextPositions(t *testing.T) {
	assertOriginalTextPositions := func(code string, plainStart, plainEnd int) {
		t.Helper()
		annotation := annotateOrFail(t, code, Options{})
		start := originalPosition(annotation, plainStart, false)
		end := originalPosition(annotation, plainEnd, true)
		if start >= end {
			t.Errorf("code %q: original start %d not smaller than end %d", code, start, end)
		}
	}

	assertOriginalTextPositions("\\cite{Kubota}*{Theorem 3.7}\n", 5, 8)
	assertOriginalTextPositions(
		"This is a test:\n\\begin{equation}\n  \\scalebox{0.92}{$a$}.\n\\end{equation}\nThis is a sentence.\n",
		29, 31)
	assertOriginalTextPositions(
		"This is a test:\n\\begin{equation*}\n  a \\text{,~and} b.\n\\end{equation*}\n",
		22, 24)
	assertOriginalTextPositions(
		"abc. Let $$\\footnote{$a$.}$$\n\nabc\n",
		16, 18)
}

func TestAccentPositionMapping(t *testing.T) {
	annotation := annotateOrFail(t, "\\v{S}ekki\n", Options{})

	for plainPos, wantOrig := range map[int]int{0: 0, 1: 5, 2: 6, 3: 7, 4: 8} {
		if got := originalPosition(annotation, plainPos, false); got != wantOrig {
			t.Errorf("originalPosition(%d) = %d, want %d", plainPos, got, wantOrig)
		}
	}
}

func TestPlainTextPositions(t *testing.T) {
	code := "\\begin{equation}\n  X\n\\end{equation}\n$a$ $b$ $c$ $d$ $e$ $f$ $g$\n\\ref{foo} \\ref{bar} 5$\\times$5\n"
	annotation := annotateOrFail(t, code, Options{Language: "fr"})
	start := plainPosition(annotation, 84, false)
	end := plainPosition(annotation, 90, true)
	if start >= end {
		t.Errorf("plain start %d not smaller than end %d", start, end)
	}
}

func TestSegmentMerging(t *testing.T) {
	annotation := annotateOrFail(t, "\\hskip\\textbf{SomeBoldText}% followed by a comment", Options{})

	want := []Segment{
		{Markup: "\\hskip\\textbf{"},
		{Text: "SomeBoldText"},
		{Markup: "}% followed by a comment"},
	}
	if len(annotation.Segments) != len(want) {
		t.Fatalf("got %d segments %v, want %d", len(annotation.Segments), annotation.Segments, len(want))
	}
	for i, segment := range annotation.Segments {
		if segment.Text != want[i].Text || segment.Markup != want[i].Markup {
			t.Errorf("segment %d = %+v, want %+v", i, segment, want[i])
		}
	}
}

// Upstream merges simple whitespace into a single text part by substituting
// " " for a lone "\n" inside a text run. This port instead emits the newline
// as markup interpreted as " " so that segments always reconstruct the source
// byte-for-byte (deviation 1 in the plan).
func TestSimpleWhitespace(t *testing.T) {
	annotation := annotateOrFail(t, "This is a test\nOver multiple\n\nLines.", Options{})

	want := []Segment{
		{Text: "This is a test"},
		{Markup: "\n", InterpretAs: " "},
		{Text: "Over multiple\n\nLines."},
	}
	if len(annotation.Segments) != len(want) {
		t.Fatalf("got %d segments %v, want %d", len(annotation.Segments), annotation.Segments, len(want))
	}
	for i, segment := range annotation.Segments {
		if segment != want[i] {
			t.Errorf("segment %d = %+v, want %+v", i, segment, want[i])
		}
	}
}
