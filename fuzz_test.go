// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package texprose

import (
	"strings"
	"testing"
)

// FuzzAnnotate checks that Annotate never panics, never loops forever, and
// always satisfies the reconstruction invariant: the concatenation of Text
// and Markup values reproduces the input byte-for-byte.
func FuzzAnnotate(f *testing.F) {
	for _, testCase := range latexCorpus {
		f.Add(testCase.code, byte(0))
	}
	f.Add(bibtexTestSource, byte(1))
	f.Add("% ltex: language=de-DE\nText $x$\n% ltex: enabled=false\nmore", byte(0))
	f.Add("\\verb|unterminated\n\\text abc \\hspace", byte(0))
	f.Add("<<chunk>>=\ncode\n@\nText", byte(2))
	f.Add("\\usepackage[main=ngerman]{babel}\\selectlanguage{french}", byte(0))

	codeLanguages := []CodeLanguage{LaTeX, BibTeX, Rsweave}

	f.Fuzz(func(t *testing.T, source string, mode byte) {
		codeLanguage := codeLanguages[int(mode)%len(codeLanguages)]

		annotation, err := Annotate(source, Options{CodeLanguage: codeLanguage})
		if err != nil {
			t.Fatalf("Annotate returned error for valid options: %v", err)
		}

		var reconstructed strings.Builder
		for _, segment := range annotation.Segments {
			if segment.Text != "" && segment.Markup != "" {
				t.Fatalf("segment has both Text and Markup: %+v", segment)
			}
			reconstructed.WriteString(segment.Text)
			reconstructed.WriteString(segment.Markup)
		}
		if reconstructed.String() != source {
			t.Fatalf("reconstruction mismatch for %q (%s):\ngot %q",
				source, codeLanguage, reconstructed.String())
		}
	})
}
