// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: parsing/DummyGenerator.kt

package texprose

import (
	"fmt"
	"strings"
)

type dummyGenerator struct {
	plural bool
	vowel  bool
}

type languageDummy struct {
	languagePrefix string
	format         string
	numbered       bool
}

var pluralDummies = []languageDummy{
	{"sv", "Dumma-%d", true},
	{"es", "Maniquíes-%d", true},
	{"nl", "Dummy's+%d", true},
	{"de", "Dummys", false},
}

var singularDummies = []languageDummy{
	{"fr", "Jimmy-%d", true},
	{"sv", "Dummy-%d", true},
	{"es", "Maniquí-%d", true},
	{"nl", "Dummy+%d", true},
	{"pl", "Atrapa-%d", true},
}

func (g dummyGenerator) generate(language string, number int, vowel bool) string {
	lang := strings.ToLower(language)

	if g.plural {
		for _, dummy := range pluralDummies {
			if strings.HasPrefix(lang, dummy.languagePrefix) {
				if dummy.numbered {
					return fmt.Sprintf(dummy.format, number)
				}
				return dummy.format
			}
		}
		return "Dummies"
	}

	for _, dummy := range singularDummies {
		if strings.HasPrefix(lang, dummy.languagePrefix) {
			return fmt.Sprintf(dummy.format, number)
		}
	}

	if vowel || g.vowel {
		return fmt.Sprintf("Ina%d", number)
	}
	return fmt.Sprintf("Dummy%d", number)
}
