// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: parsing/latex/LatexCommandSignatureMatcher.kt
// and parsing/latex/LatexCommandSignatureMatch.kt

package texprose

import (
	"regexp"
	"strings"
)

type signatureMatch struct {
	signature *commandSignature
	code      string
	fromPos   int
	toPos     int
	arguments []argumentSpan
}

func newSignatureMatch(signature *commandSignature, code string, fromPos int, arguments []argumentSpan) signatureMatch {
	toPos := fromPos + len(signature.prefix)
	if len(arguments) > 0 {
		toPos = arguments[len(arguments)-1].to
	}
	return signatureMatch{signature, code, fromPos, toPos, arguments}
}

func (m signatureMatch) argumentContents(index int) string {
	span := m.arguments[index]
	return m.code[span.from+1 : span.to-1]
}

func (m signatureMatch) argumentContentsFromPos(index int) int {
	return m.arguments[index].from + 1
}

type signatureMatcher struct {
	signatures  []*commandSignature
	prefixRegex *regexp.Regexp
}

func newSignatureMatcher(signatures []*commandSignature, escapePrefixes bool) *signatureMatcher {
	patterns := make([]string, len(signatures))
	for i, signature := range signatures {
		if escapePrefixes {
			patterns[i] = regexp.QuoteMeta(signature.prefix)
		} else {
			patterns[i] = signature.prefix
		}
	}
	return &signatureMatcher{
		signatures:  signatures,
		prefixRegex: regexp.MustCompile(strings.Join(patterns, "|")),
	}
}

// findAllMatches returns all signature matches in code, in source order,
// skipping occurrences on commented-out line parts and signatures whose
// prototype is in ignorePrototypes.
func (m *signatureMatcher) findAllMatches(code string, ignorePrototypes map[string]bool) []signatureMatch {
	var matches []signatureMatch

	for _, location := range m.prefixRegex.FindAllStringIndex(code, -1) {
		fromPos := location[0]

		lineStart := strings.LastIndexByte(code[:fromPos], '\n') + 1
		if hasUnescapedPercent(code[lineStart:fromPos]) {
			continue
		}

		var best *signatureMatch
		for _, signature := range m.signatures {
			if ignorePrototypes[signature.prototype] {
				continue
			}
			arguments, ok := signature.matchArgumentsFromPosition(code, fromPos)
			if !ok {
				continue
			}
			match := newSignatureMatch(signature, code, fromPos, arguments)
			if best == nil || match.toPos > best.toPos {
				best = &match
			}
		}
		if best != nil {
			matches = append(matches, *best)
		}
	}

	return matches
}
