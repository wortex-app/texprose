// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: LatexFragmentizerTest.kt and
// BibtexFragmentizerTest.kt, adapted for deviation 4: upstream additionally
// emits overlapping re-check fragments (\footnote/\todo contents, babel
// inline commands/environments), which this port does not produce.

package texprose

import "testing"

func defaultTestSettings() settings {
	return settings{language: "en-US", enabled: true, bibtexFields: defaultBibtexFields}
}

type fragmentExpectation struct {
	code       string
	fromPos    int
	language   string
	markupOnly bool
}

func assertFragments(t *testing.T, fragments []fragment, want []fragmentExpectation) {
	t.Helper()
	if len(fragments) != len(want) {
		t.Fatalf("got %d fragments, want %d: %+v", len(fragments), len(want), fragments)
	}
	for i, fragment := range fragments {
		if fragment.code != want[i].code {
			t.Errorf("fragment %d code = %q, want %q", i, fragment.code, want[i].code)
		}
		if fragment.fromPos != want[i].fromPos {
			t.Errorf("fragment %d fromPos = %d, want %d", i, fragment.fromPos, want[i].fromPos)
		}
		if fragment.settings.language != want[i].language {
			t.Errorf("fragment %d language = %q, want %q", i, fragment.settings.language, want[i].language)
		}
		if fragment.markupOnly != want[i].markupOnly {
			t.Errorf("fragment %d markupOnly = %v, want %v", i, fragment.markupOnly, want[i].markupOnly)
		}
	}
}

func TestFragmentizerMagicComments(t *testing.T) {
	code := "Sentence\\footnote[abc]{Footnote} 1\n" +
		"\t\t  %\t ltex: language=de-DE\n" +
		"Sentence 2\\todo{Todo note}\n" +
		"%ltex:\tlanguage=en-US\n" +
		"\nSentence 3\n"

	fragments := fragmentizeLatex(code, defaultTestSettings())

	assertFragments(t, fragments, []fragmentExpectation{
		{"Sentence\\footnote[abc]{Footnote} 1\n", 0, "en-US", false},
		{"\t\t  %\t ltex: language=de-DE", 35, "de-DE", true},
		{"\nSentence 2\\todo{Todo note}\n", 62, "de-DE", false},
		{"%ltex:\tlanguage=en-US", 90, "en-US", true},
		{"\n\nSentence 3\n", 111, "en-US", false},
	})
}

func TestFragmentizerBabelUsePackage(t *testing.T) {
	code := "This is a test.\n" +
		"\\usepackage[\n" +
		"  american,  % American English\n" +
		"  ngerman,   % German\n" +
		"  dummy={abc,def}\n" +
		"]{babel}\n" +
		"Dies ist ein Test.\n"

	fragments := fragmentizeLatex(code, defaultTestSettings())
	if len(fragments) != 2 {
		t.Fatalf("got %d fragments, want 2: %+v", len(fragments), fragments)
	}
	if len(fragments[0].code) != 16 || len(fragments[1].code) != 113 {
		t.Errorf("got code lengths %d/%d, want 16/113",
			len(fragments[0].code), len(fragments[1].code))
	}
	if fragments[0].settings.language != "en-US" || fragments[1].settings.language != "de-DE" {
		t.Errorf("got languages %q/%q, want en-US/de-DE",
			fragments[0].settings.language, fragments[1].settings.language)
	}
}

func TestFragmentizerBabelUsePackageMain(t *testing.T) {
	code := "This is a test.\n" +
		"\\usepackage[\n" +
		"  main=ngerman,  % German\n" +
		"  american,      % American English\n" +
		"  dummy={abc,def}\n" +
		"]{babel}\n" +
		"Dies ist ein Test.\n"

	fragments := fragmentizeLatex(code, defaultTestSettings())
	if len(fragments) != 2 {
		t.Fatalf("got %d fragments, want 2: %+v", len(fragments), fragments)
	}
	if len(fragments[0].code) != 16 || len(fragments[1].code) != 121 {
		t.Errorf("got code lengths %d/%d, want 16/121",
			len(fragments[0].code), len(fragments[1].code))
	}
	if fragments[1].settings.language != "de-DE" {
		t.Errorf("got language %q, want de-DE (main= wins)", fragments[1].settings.language)
	}
}

func TestFragmentizerCommentedUsePackage(t *testing.T) {
	code := "This is a test.\n  % \\usepackage[ngerman]{babel}\nThis is another test."
	fragments := fragmentizeLatex(code, defaultTestSettings())
	if len(fragments) != 1 {
		t.Fatalf("got %d fragments, want 1 (commented-out usepackage): %+v", len(fragments), fragments)
	}

	code = "This is a test.\n  \\usepackage[ngerman]{babel}\nThis is another test."
	fragments = fragmentizeLatex(code, defaultTestSettings())
	if len(fragments) != 2 {
		t.Fatalf("got %d fragments, want 2: %+v", len(fragments), fragments)
	}
}

func TestFragmentizerBibtex(t *testing.T) {
	fragments := fragmentizeBibtex(bibtexTestSource, defaultTestSettings())

	assertFragments(t, fragments, []fragmentExpectation{
		{" {Some Name}", 29, "en-US", false},
		{" {This is a test.}\n", 58, "en-US", false},
		{" {Some Other Name}", 108, "en-US", false},
		{" {This is another\n  test.}", 143, "en-US", false},
		{"shortform", 210, "en-US", false},
		{" longform", 247, "en-US", false},
	})
}
