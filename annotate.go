// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package texprose

import "fmt"

// Annotate converts LaTeX or BibTeX source code into LanguageTool annotated
// text. The concatenation of Text and Markup values of the returned segments
// reproduces source exactly.
func Annotate(source string, opts Options) (Annotation, error) {
	if opts.CodeLanguage == "" {
		opts.CodeLanguage = LaTeX
	}
	if opts.CodeLanguage != LaTeX && opts.CodeLanguage != BibTeX {
		return Annotation{}, fmt.Errorf("texprose: unsupported code language %q", opts.CodeLanguage)
	}
	return Annotation{}, fmt.Errorf("texprose: not implemented")
}
