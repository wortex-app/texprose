// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: parsing/latex/LatexPackageOption.kt and
// parsing/latex/LatexPackageOptionsParser.kt

package texprose

import "strings"

type keyValueInfo struct {
	fromPos   int
	toPos     int
	plainText string
}

type packageOption struct {
	key       string
	value     string
	keyInfo   keyValueInfo
	valueInfo keyValueInfo
}

func newPackageOption(code string, keyInfo, valueInfo keyValueInfo) packageOption {
	option := packageOption{keyInfo: keyInfo, valueInfo: valueInfo}
	option.key = code[keyInfo.fromPos:keyInfo.toPos]
	if valueInfo.fromPos != -1 && valueInfo.toPos != -1 {
		option.value = code[valueInfo.fromPos:valueInfo.toPos]
	}
	return option
}

func trimControlAndSpace(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r <= ' ' })
}

type optionsParseMode int

const (
	optionsParseKey optionsParseMode = iota
	optionsParseValue
)

func parsePackageOptions(optionsString string) []packageOption {
	var options []packageOption
	mode := optionsParseKey
	groupDepth := 0
	keyFromPos := 0
	keyToPos := -1
	var keyBuilder strings.Builder
	valueFromPos := -1
	valueToPos := -1
	var valueBuilder strings.Builder
	pos := 0

	appendToOption := func(s string) {
		if mode == optionsParseKey {
			keyBuilder.WriteString(s)
		} else {
			valueBuilder.WriteString(s)
		}
	}
	appendSpaceToOption := func() {
		builder := &keyBuilder
		if mode == optionsParseValue {
			builder = &valueBuilder
		}
		if s := builder.String(); len(s) == 0 || s[len(s)-1] != ' ' {
			builder.WriteByte(' ')
		}
	}

	for pos < len(optionsString) {
		oldPos := pos

		switch curChar := optionsString[pos]; curChar {
		case '{':
			groupDepth++
			pos++
		case '}':
			groupDepth--
			pos++
		case ' ', '\n', '\r', '\t':
			whitespace := matchWhitespace(optionsString, pos)
			appendSpaceToOption()
			pos += len(whitespace)
		default:
			addCurCharToOption := true

			if curChar == '%' {
				comment := matchComment(optionsString, pos, true)
				if comment != "" {
					pos += len(comment)
					addCurCharToOption = false
				}
			} else if curChar == ',' && groupDepth == 0 {
				if mode == optionsParseKey {
					keyToPos = pos
				} else {
					valueToPos = pos
				}

				options = append(options, newPackageOption(
					optionsString,
					keyValueInfo{keyFromPos, keyToPos, trimControlAndSpace(keyBuilder.String())},
					keyValueInfo{valueFromPos, valueToPos, trimControlAndSpace(valueBuilder.String())},
				))

				mode = optionsParseKey
				keyFromPos = pos + 1
				keyToPos = -1
				keyBuilder.Reset()
				valueFromPos = -1
				valueToPos = -1
				valueBuilder.Reset()
				addCurCharToOption = false
			} else if curChar == '\\' && pos < len(optionsString)-1 {
				switch nextChar := optionsString[pos+1]; nextChar {
				case '%', ',', '\\', '{', '}', ' ':
					appendToOption(optionsString[pos : pos+2])
					pos += 2
					addCurCharToOption = false
				}
			} else if curChar == '=' && mode == optionsParseKey {
				mode = optionsParseValue
				keyToPos = pos
				valueFromPos = pos + 1
				addCurCharToOption = false
			}

			if addCurCharToOption {
				appendToOption(optionsString[pos : pos+1])
				pos++
			}
		}

		if pos == oldPos {
			pos++
		}
	}

	if keyFromPos < len(optionsString) {
		if mode == optionsParseKey {
			keyToPos = len(optionsString)
		} else {
			valueToPos = len(optionsString)
		}

		options = append(options, newPackageOption(
			optionsString,
			keyValueInfo{keyFromPos, keyToPos, trimControlAndSpace(keyBuilder.String())},
			keyValueInfo{valueFromPos, valueToPos, trimControlAndSpace(valueBuilder.String())},
		))
	}

	return options
}
