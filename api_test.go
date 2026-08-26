// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package texprose

import (
	"encoding/json"
	"testing"
)

const bibtexTestSource = "@article{some-label,\n" +
	"  name = {Some Name},\n" +
	"  description = {This is a test.}\n" +
	"}\n" +
	"\n" +
	"@entry{some-label2,\n" +
	"  name = {Some Other Name},\n" +
	"  description = {This is another\n" +
	"  test.},\n" +
	"}\n" +
	"\n" +
	"@abbreviation{some-label3,\n" +
	"  short =shortform,\n" +
	"  see   = {abc},\n" +
	"  long  = longform,\n" +
	" }\n"

func TestBibtex(t *testing.T) {
	annotation := annotateOrFail(t, bibtexTestSource, Options{CodeLanguage: BibTeX})
	assertReconstruction(t, bibtexTestSource, annotation)

	want := " Some Name This is a test.  Some Other Name This is another test.shortform longform"
	if got := annotation.PlainText(); got != want {
		t.Errorf("plain text mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestBibtexFieldOverride(t *testing.T) {
	source := "@article{key,\n  title = {A Title},\n  description = {A description.},\n}\n"

	annotation := annotateOrFail(t, source, Options{
		CodeLanguage: BibTeX,
		BibTeXFields: map[string]bool{"description": false},
	})
	assertReconstruction(t, source, annotation)

	want := " A Title"
	if got := annotation.PlainText(); got != want {
		t.Errorf("plain text mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestMagicCommentLanguage(t *testing.T) {
	code := "% ltex: language=fr\n$x$\n"
	annotation := annotateOrFail(t, code, Options{})
	assertReconstruction(t, code, annotation)

	want := " Jimmy-0 "
	if got := annotation.PlainText(); got != want {
		t.Errorf("plain text mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestMagicCommentEnabled(t *testing.T) {
	code := "A sentence.\n% ltex: enabled=false\nBad sentence.\n"
	annotation := annotateOrFail(t, code, Options{})
	assertReconstruction(t, code, annotation)

	want := "A sentence. "
	if got := annotation.PlainText(); got != want {
		t.Errorf("plain text mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestBabelUsePackage(t *testing.T) {
	code := "$y$\n\\usepackage[french]{babel}\n$z$\n"
	annotation := annotateOrFail(t, code, Options{})
	assertReconstruction(t, code, annotation)

	want := "Dummy0  Jimmy-0 "
	if got := annotation.PlainText(); got != want {
		t.Errorf("plain text mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestBabelSelectLanguage(t *testing.T) {
	code := "$y$ \\selectlanguage{french} $z$\n"
	annotation := annotateOrFail(t, code, Options{})
	assertReconstruction(t, code, annotation)

	want := "Dummy0  Jimmy-0 "
	if got := annotation.PlainText(); got != want {
		t.Errorf("plain text mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestMarshalJSON(t *testing.T) {
	annotation := Annotation{Segments: []Segment{
		{Text: "A sentence about "},
		{Markup: "\\LaTeX{}", InterpretAs: "LaTeX"},
		{Text: ", with math "},
		{Markup: "$x^2$", InterpretAs: "Dummy0"},
		{Text: "."},
		{Markup: "\n\n", InterpretAs: "\n\n"},
		{Markup: "% comment"},
	}}

	got, err := json.Marshal(annotation)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"annotation":[` +
		`{"text":"A sentence about "},` +
		`{"markup":"\\LaTeX{}","interpretAs":"LaTeX"},` +
		`{"text":", with math "},` +
		`{"markup":"$x^2$","interpretAs":"Dummy0"},` +
		`{"text":"."},` +
		`{"markup":"\n\n","interpretAs":"\n\n"},` +
		`{"markup":"% comment"}` +
		`]}`
	if string(got) != want {
		t.Errorf("JSON mismatch:\ngot  %s\nwant %s", got, want)
	}
}

func TestInvalidOptions(t *testing.T) {
	if _, err := Annotate("test", Options{CodeLanguage: "markdown"}); err == nil {
		t.Error("expected error for unsupported code language")
	}
	if _, err := Annotate("test", Options{Commands: map[string]Action{"\\foo{}": Action(99)}}); err == nil {
		t.Error("expected error for invalid action")
	}
	if _, err := Annotate("test", Options{Environments: map[string]Action{"foo": ActionDummy}}); err == nil {
		t.Error("expected error for dummy action on environment")
	}
}
