// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Hand-written equivalents of the anchored java.util.regex patterns used by
// the upstream builder (LatexAnnotatedTextBuilder.kt companion object). They
// are hand-written where Go's RE2 cannot reproduce Java semantics: "$"
// matching just before a final line terminator, the backreference in the
// \verb pattern, and the lookbehind in the comment-prefix check.

package texprose

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// matchCommand ports COMMAND_REGEX = ^\\(([^A-Za-z@]|([A-Za-z@]+))\*?)
func matchCommand(code string, pos int) string {
	if pos >= len(code) || code[pos] != '\\' || pos+1 >= len(code) {
		return ""
	}

	i := pos + 1
	if isASCIILetter(code[i]) || code[i] == '@' {
		for i < len(code) && (isASCIILetter(code[i]) || code[i] == '@') {
			i++
		}
	} else {
		_, size := utf8.DecodeRuneInString(code[i:])
		i += size
	}

	if i < len(code) && code[i] == '*' {
		i++
	}
	return code[pos:i]
}

// matchBraceArgument ports ARGUMENT_REGEX = ^\{[^}]*?}
func matchBraceArgument(code string, pos int) string {
	if pos >= len(code) || code[pos] != '{' {
		return ""
	}
	end := strings.IndexByte(code[pos+1:], '}')
	if end < 0 {
		return ""
	}
	return code[pos : pos+1+end+1]
}

// javaDollar reports whether Java's "$" (without MULTILINE) matches at pos:
// at the end of the input, or just before a line terminator that ends the
// input.
func javaDollar(code string, pos int) bool {
	rest := code[pos:]
	return rest == "" || rest == "\n" || rest == "\r" || rest == "\r\n"
}

// matchComment ports COMMENT_REGEX = ^%.*?($|(\r?\n[ \n\r\t]*)). With
// allowCR false it instead ports the LatexCommandSignature variant
// ^%.*?($|(\n[ \n\r\t]*)). Returns the matched string ("" if no match).
func matchComment(code string, pos int, allowCR bool) string {
	if pos >= len(code) || code[pos] != '%' {
		return ""
	}

	for i := pos + 1; ; i++ {
		// Alternation order as in the Java pattern: "$" first.
		if javaDollar(code, i) {
			return code[pos:i]
		}
		j := i
		if allowCR && code[j] == '\r' && j+1 < len(code) && code[j+1] == '\n' {
			j++
		}
		if code[j] == '\n' {
			j++
			for j < len(code) && (code[j] == ' ' || code[j] == '\n' || code[j] == '\r' || code[j] == '\t') {
				j++
			}
			return code[pos:j]
		}
		if code[i] == '\n' || code[i] == '\r' {
			// "." cannot consume a line terminator; the pattern fails.
			return ""
		}
	}
}

// matchWhitespace ports WHITESPACE_REGEX = ^[ \n\r\t]+(%.*?($|(\r?\n[ \n\r\t]*)))?
func matchWhitespace(code string, pos int) string {
	i := pos
	for i < len(code) && (code[i] == ' ' || code[i] == '\n' || code[i] == '\r' || code[i] == '\t') {
		i++
	}
	if i == pos {
		return ""
	}
	if i < len(code) && code[i] == '%' {
		i += len(matchComment(code, i, true))
	}
	return code[pos:i]
}

// matchVerb ports VERB_COMMAND_REGEX = ^\\verb\*?(.).*?\1
func matchVerb(code string, pos int) string {
	i := pos + len("\\verb")
	if !strings.HasPrefix(code[pos:], "\\verb") {
		return ""
	}
	if i < len(code) && code[i] == '*' {
		i++
	}
	if i >= len(code) || code[i] == '\n' || code[i] == '\r' {
		return ""
	}
	delimiter, size := utf8.DecodeRuneInString(code[i:])
	i += size

	for i < len(code) {
		if code[i] == '\n' || code[i] == '\r' {
			return ""
		}
		r, size := utf8.DecodeRuneInString(code[i:])
		i += size
		if r == delimiter {
			return code[pos:i]
		}
	}
	return ""
}

const lengthPattern = `-?[0-9]*(\.[0-9]+)?(pt|mm|cm|ex|em|bp|dd|pc|in)`

var (
	lengthInBraceRegexp   = regexp.MustCompile(`^\{` + lengthPattern + `}`)
	lengthInBracketRegexp = regexp.MustCompile(`^\[` + lengthPattern + `]`)

	accentPattern = "(\\\\[`'^~\"=.Hbcdkruv])" +
		`(?: *([A-Za-z]|\\i|\\j)|\{([A-Za-z]|\\i|\\j)})`
	accentRegexp        = regexp.MustCompile(`^` + accentPattern)
	accentInBraceRegexp = regexp.MustCompile(`^\{` + accentPattern + `}`)

	rsweaveBeginRegexp = regexp.MustCompile(`^<<[^\n\r]*?>>=`)
)

func matchRegexp(re *regexp.Regexp, code string, pos int) string {
	if pos < 0 || pos > len(code) {
		return ""
	}
	return re.FindString(code[pos:])
}

// matchAccent returns the full match, the accent command (e.g. `\"`), and the
// letter (from either alternative) of ACCENT_REGEX or ACCENT_IN_BRACE_REGEX.
func matchAccent(re *regexp.Regexp, code string, pos int) (match, accentCommand, letter string) {
	groups := re.FindStringSubmatch(code[pos:])
	if groups == nil {
		return "", "", ""
	}
	letter = groups[2]
	if letter == "" {
		letter = groups[3]
	}
	return groups[0], groups[1], letter
}

// hasUnescapedPercent reports whether s contains a "%" preceded by an even
// number of backslashes, porting the lookbehind pattern
// (?<!\\)(?:\\\\)*% of LatexCommandSignatureMatcher.
func hasUnescapedPercent(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		backslashes := 0
		for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return true
		}
	}
	return false
}

func containsTwoEndsOfLine(text string) bool {
	return strings.Contains(text, "\n\n") || strings.Contains(text, "\r\n\r\n")
}
