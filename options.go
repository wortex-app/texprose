// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package texprose

// CodeLanguage selects the input syntax handled by Annotate.
type CodeLanguage string

const (
	LaTeX  CodeLanguage = "latex"
	BibTeX CodeLanguage = "bibtex"
)

// Action describes how a command or environment is handled, using the same
// vocabulary as LTeX's ltex.latex.commands/environments settings.
type Action int

const (
	// ActionDefault: the command tokens are markup, arguments are parsed
	// normally (environment contents stay checked).
	ActionDefault Action = iota
	// ActionIgnore: the command including its declared arguments (or the
	// whole environment contents) is skipped.
	ActionIgnore
	// ActionDummy: the command including its declared arguments is replaced
	// by a dummy word. Not valid for environments.
	ActionDummy
	// ActionPluralDummy: like ActionDummy, but the dummy word is a plural
	// noun. Not valid for environments.
	ActionPluralDummy
	// ActionVowelDummy: like ActionDummy, but the dummy word starts with a
	// vowel (for "a" vs. "an" agreement). Not valid for environments.
	ActionVowelDummy
)

// Options configures Annotate.
type Options struct {
	// CodeLanguage of the source; default LaTeX.
	CodeLanguage CodeLanguage

	// Checking language (BCP-47 style, e.g. "en-US", "de-DE"). Affects dummy
	// generation and language-specific handling. Default "en-US".
	Language string

	// Commands adds to the built-in command signature table. Keys use LTeX's
	// signature syntax: the command followed by its argument shape, e.g.
	// `\todo{}`, `\cite[]{}`, `\pict()`. All of ActionDefault, ActionIgnore,
	// ActionDummy, ActionPluralDummy and ActionVowelDummy are valid.
	Commands map[string]Action

	// Environments adds to the built-in environment table. Keys are either a
	// bare environment name (e.g. `verbatim`) or a `\begin{...}` prototype
	// with argument shape (e.g. `\begin{frame}[]`). Only ActionDefault and
	// ActionIgnore are valid for environments.
	Environments map[string]Action

	// BibTeXFields overrides which BibTeX fields contain prose to be checked.
	// Maps field name to true (check) or false (skip). Merged over the
	// built-in table; unlisted fields are checked.
	BibTeXFields map[string]bool
}
