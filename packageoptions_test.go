// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: LatexPackageOptionsParserTest.kt and
// LatexPackageOptionTest.kt

package texprose

import "testing"

func TestParsePackageOptions(t *testing.T) {
	options := parsePackageOptions(
		"  option1\\,=value1,\noption2 = \\value2 ,% option3 = value3,\noption4 =% value4,\n{This is\\} a \\textbf{test}, option5 = value5.},\n")

	if len(options) != 4 {
		t.Fatalf("got %d options, want 4: %+v", len(options), options)
	}

	assertOption := func(index int, key, keyPlain string, valueFromPos int, value, valuePlain string) {
		t.Helper()
		option := options[index]
		if option.key != key {
			t.Errorf("option %d key = %q, want %q", index, option.key, key)
		}
		if option.keyInfo.plainText != keyPlain {
			t.Errorf("option %d key plain = %q, want %q", index, option.keyInfo.plainText, keyPlain)
		}
		if option.valueInfo.fromPos != valueFromPos {
			t.Errorf("option %d value fromPos = %d, want %d", index, option.valueInfo.fromPos, valueFromPos)
		}
		if option.value != value {
			t.Errorf("option %d value = %q, want %q", index, option.value, value)
		}
		if option.valueInfo.plainText != valuePlain {
			t.Errorf("option %d value plain = %q, want %q", index, option.valueInfo.plainText, valuePlain)
		}
	}

	assertOption(0, "  option1\\,", "option1\\,", 12, "value1", "value1")
	assertOption(1, "\noption2 ", "option2", 29, " \\value2 ", "\\value2")
	assertOption(2, "% option3 = value3,\noption4 ", "option4", 68,
		"% value4,\n{This is\\} a \\textbf{test}, option5 = value5.}",
		"This is\\} a \\textbftest, option5 = value5.")
	assertOption(3, "\n", "", -1, "", "")
}

func TestPackageOptionProperties(t *testing.T) {
	option := newPackageOption(
		"\\usepackage[foo=bar]{foobar}",
		keyValueInfo{12, 15, "abc"},
		keyValueInfo{16, 19, "def"},
	)

	if option.key != "foo" || option.value != "bar" {
		t.Errorf("got key %q value %q, want foo/bar", option.key, option.value)
	}
	if option.keyInfo.plainText != "abc" || option.valueInfo.plainText != "def" {
		t.Errorf("plain texts not preserved: %+v", option)
	}
}
