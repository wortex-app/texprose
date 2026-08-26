// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

// Package texprose converts LaTeX and BibTeX source code into the annotated
// text form accepted by LanguageTool's /v2/check endpoint ("data" parameter).
//
// The conversion is a lossless lexical transformation: the concatenation of
// all Text and Markup values of the produced segments reproduces the input
// source byte-for-byte, so match offsets reported by LanguageTool point at
// the original LaTeX source.
//
// The parsing logic is a Go translation of the LaTeX/BibTeX parsing layer of
// ltex-plus/ltex-ls-plus (MPL-2.0); see NOTICE.md.
package texprose

import (
	"bytes"
	"encoding/json"
	"strings"
)

// A Segment is either text (checked by LanguageTool) or markup (skipped).
// Exactly one of Text or Markup is set. InterpretAs is only valid on markup
// segments: LanguageTool pretends the markup reads as that string (used for
// dummy words replacing math, and for newlines/paragraph breaks).
type Segment struct {
	Text        string
	Markup      string
	InterpretAs string
}

// Annotation is an ordered list of segments covering the whole input source.
type Annotation struct {
	Segments []Segment
}

// MarshalJSON emits LanguageTool's /v2/check "data" parameter format:
//
//	{"annotation": [{"text": "..."}, {"markup": "...", "interpretAs": "..."}, ...]}
func (a Annotation) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"annotation":[`)

	for i, segment := range a.Segments {
		if i > 0 {
			buf.WriteByte(',')
		}

		buf.WriteByte('{')
		if segment.Text != "" {
			buf.WriteString(`"text":`)
			if err := writeJSONString(&buf, segment.Text); err != nil {
				return nil, err
			}
		} else {
			buf.WriteString(`"markup":`)
			if err := writeJSONString(&buf, segment.Markup); err != nil {
				return nil, err
			}
			if segment.InterpretAs != "" {
				buf.WriteString(`,"interpretAs":`)
				if err := writeJSONString(&buf, segment.InterpretAs); err != nil {
					return nil, err
				}
			}
		}
		buf.WriteByte('}')
	}

	buf.WriteString(`]}`)
	return buf.Bytes(), nil
}

func writeJSONString(buf *bytes.Buffer, s string) error {
	encoded, err := json.Marshal(s)
	if err != nil {
		return err
	}
	buf.Write(encoded)
	return nil
}

// PlainText returns what LanguageTool will effectively check: the
// concatenation of text segments and InterpretAs values.
func (a Annotation) PlainText() string {
	var sb strings.Builder
	for _, segment := range a.Segments {
		if segment.Text != "" {
			sb.WriteString(segment.Text)
		} else {
			sb.WriteString(segment.InterpretAs)
		}
	}
	return sb.String()
}
