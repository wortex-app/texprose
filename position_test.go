// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package texprose

import "testing"

func TestPositionMapperASCII(t *testing.T) {
	mapper := NewPositionMapper("abc\ndef\n\nx")

	cases := []struct {
		offset int
		want   Position
	}{
		{0, Position{0, 0}},
		{3, Position{0, 3}},
		{4, Position{1, 0}},
		{7, Position{1, 3}},
		{8, Position{2, 0}},
		{9, Position{3, 0}},
		{10, Position{3, 1}},
		{99, Position{3, 1}}, // clamped
	}
	for _, testCase := range cases {
		if got := mapper.Position(testCase.offset); got != testCase.want {
			t.Errorf("Position(%d) = %+v, want %+v", testCase.offset, got, testCase.want)
		}
		if testCase.offset <= 10 {
			if got := mapper.Offset(testCase.want); got != testCase.offset {
				t.Errorf("Offset(%+v) = %d, want %d", testCase.want, got, testCase.offset)
			}
		}
	}
}

// "😀" is U+1F600: 4 UTF-8 bytes, 2 UTF-16 code units (non-BMP).
// "ö" is U+00F6: 2 UTF-8 bytes, 1 UTF-16 code unit.
// "y̆" is "y" + combining breve U+0306: 3 UTF-8 bytes, 2 UTF-16 code units.
func TestPositionMapperNonASCII(t *testing.T) {
	source := "a😀ö\ny̆z"
	mapper := NewPositionMapper(source)

	// UTF-16:  a=0  😀=1,2  ö=3  \n=4  y=5  ̆=6  z=7
	// Bytes:   a=0  😀=1-4  ö=5-6  \n=7  y=8  ̆=9-10  z=11
	if got := mapper.Position(3); got != (Position{0, 3}) {
		t.Errorf("Position(3) = %+v, want {0 3}", got)
	}
	if got := mapper.Position(7); got != (Position{1, 2}) {
		t.Errorf("Position(7) = %+v, want {1 2}", got)
	}
	if got := mapper.Offset(Position{1, 2}); got != 7 {
		t.Errorf("Offset({1 2}) = %d, want 7", got)
	}

	byteCases := map[int]int{0: 0, 1: 1, 2: 1, 3: 5, 4: 7, 5: 8, 6: 9, 7: 11, 8: 12}
	for utf16Offset, wantByte := range byteCases {
		if got := mapper.ByteOffset(utf16Offset); got != wantByte {
			t.Errorf("ByteOffset(%d) = %d, want %d", utf16Offset, got, wantByte)
		}
	}

	utf16Cases := map[int]int{0: 0, 1: 1, 3: 1, 5: 3, 7: 4, 8: 5, 9: 6, 11: 7, 12: 8}
	for byteOffset, wantUTF16 := range utf16Cases {
		if got := mapper.UTF16Offset(byteOffset); got != wantUTF16 {
			t.Errorf("UTF16Offset(%d) = %d, want %d", byteOffset, got, wantUTF16)
		}
	}

	// Mid-character offsets map to the character start.
	if got := mapper.ByteOffset(2); got != 1 {
		t.Errorf("ByteOffset(2) [mid-surrogate] = %d, want 1", got)
	}
	if got := mapper.UTF16Offset(3); got != 1 {
		t.Errorf("UTF16Offset(3) [mid-rune] = %d, want 1", got)
	}
}

func TestPositionMapperEmpty(t *testing.T) {
	mapper := NewPositionMapper("")
	if got := mapper.Position(0); got != (Position{0, 0}) {
		t.Errorf("Position(0) = %+v, want {0 0}", got)
	}
	if got := mapper.Offset(Position{5, 5}); got != 0 {
		t.Errorf("Offset({5 5}) = %d, want 0", got)
	}
}
