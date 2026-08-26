// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package texprose

import (
	"sort"
	"unicode/utf8"
)

// Position is a 0-based line and 0-based character offset within that line.
// Character counts UTF-16 code units, matching both LanguageTool and the
// Language Server Protocol's default position encoding.
type Position struct{ Line, Character int }

// PositionMapper converts between representations of a location in source.
//
// ⚠️ Offset units: LanguageTool is a Java program, so the offset/length
// values in its /v2/check responses count UTF-16 code units (Java string
// indices), NOT bytes and NOT Unicode code points. Position, Offset and the
// argument of ByteOffset therefore work in UTF-16 code units. Use ByteOffset
// and UTF16Offset to convert to and from Go-native byte offsets. On pure
// ASCII sources all three units coincide; on any source containing non-ASCII
// text they do not.
//
// Lines are separated by "\n"; a "\r\n" sequence counts as ending the line
// after the "\n" (the "\r" belongs to the preceding line).
//
// Construct once per document; queries are O(log lines + line length).
type PositionMapper struct {
	source string
	// Byte and UTF-16 offsets of each line start (line i starts at
	// lineByteStarts[i]); index 0 is always 0.
	lineByteStarts  []int
	lineUTF16Starts []int
	totalUTF16      int
}

func utf16Len(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}

// NewPositionMapper indexes source for position conversions.
func NewPositionMapper(source string) *PositionMapper {
	mapper := &PositionMapper{
		source:          source,
		lineByteStarts:  []int{0},
		lineUTF16Starts: []int{0},
	}

	byteOffset := 0
	utf16Offset := 0
	for byteOffset < len(source) {
		r, size := utf8.DecodeRuneInString(source[byteOffset:])
		byteOffset += size
		utf16Offset += utf16Len(r)
		if r == '\n' {
			mapper.lineByteStarts = append(mapper.lineByteStarts, byteOffset)
			mapper.lineUTF16Starts = append(mapper.lineUTF16Starts, utf16Offset)
		}
	}
	mapper.totalUTF16 = utf16Offset
	return mapper
}

func (m *PositionMapper) clampUTF16(offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > m.totalUTF16 {
		return m.totalUTF16
	}
	return offset
}

func (m *PositionMapper) lineOfUTF16(offset int) int {
	return sort.SearchInts(m.lineUTF16Starts, offset+1) - 1
}

// Position converts a UTF-16 code unit offset (as reported by LanguageTool)
// to a line/character position. Out-of-range offsets are clamped.
func (m *PositionMapper) Position(offset int) Position {
	offset = m.clampUTF16(offset)
	line := m.lineOfUTF16(offset)
	return Position{Line: line, Character: offset - m.lineUTF16Starts[line]}
}

// Offset converts a line/character position to a UTF-16 code unit offset.
// Out-of-range positions are clamped.
func (m *PositionMapper) Offset(p Position) int {
	if p.Line < 0 {
		return 0
	}
	if p.Line >= len(m.lineUTF16Starts) {
		return m.totalUTF16
	}

	lineStart := m.lineUTF16Starts[p.Line]
	lineEnd := m.totalUTF16
	if p.Line+1 < len(m.lineUTF16Starts) {
		lineEnd = m.lineUTF16Starts[p.Line+1]
	}

	if p.Character < 0 {
		return lineStart
	}
	if lineStart+p.Character > lineEnd {
		return lineEnd
	}
	return lineStart + p.Character
}

// ByteOffset converts a UTF-16 code unit offset to a Go byte offset into the
// source. An offset inside a surrogate pair maps to the start of that
// character.
func (m *PositionMapper) ByteOffset(utf16Offset int) int {
	utf16Offset = m.clampUTF16(utf16Offset)
	line := m.lineOfUTF16(utf16Offset)

	byteOffset := m.lineByteStarts[line]
	current := m.lineUTF16Starts[line]
	for current < utf16Offset {
		r, size := utf8.DecodeRuneInString(m.source[byteOffset:])
		if current+utf16Len(r) > utf16Offset {
			break
		}
		byteOffset += size
		current += utf16Len(r)
	}
	return byteOffset
}

// UTF16Offset converts a Go byte offset into the source to a UTF-16 code
// unit offset. An offset inside a multi-byte character maps to the start of
// that character. Out-of-range offsets are clamped.
func (m *PositionMapper) UTF16Offset(byteOffset int) int {
	if byteOffset < 0 {
		return 0
	}
	if byteOffset > len(m.source) {
		byteOffset = len(m.source)
	}

	line := sort.SearchInts(m.lineByteStarts, byteOffset+1) - 1
	current := m.lineByteStarts[line]
	utf16Offset := m.lineUTF16Starts[line]
	for current < byteOffset {
		r, size := utf8.DecodeRuneInString(m.source[current:])
		if current+size > byteOffset {
			break
		}
		current += size
		utf16Offset += utf16Len(r)
	}
	return utf16Offset
}
