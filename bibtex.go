// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: parsing/bibtex/BibtexFragmentizer.kt

package texprose

var (
	bibtexEntrySignature = func() *commandSignature {
		signature, err := newCommandSignature("@[A-Za-z]+{}", actionIgnore, dummyGenerator{}, false)
		if err != nil {
			panic(err)
		}
		return signature
	}()
	bibtexEntryMatcher = newSignatureMatcher([]*commandSignature{bibtexEntrySignature}, false)
)

// fragmentizeBibtex extracts prose field values from BibTeX entries as
// checkable fragments. Everything else (entry structure, keys, data fields)
// is left uncovered and becomes markup during assembly.
func fragmentizeBibtex(code string, originalSettings settings) []fragment {
	var newFragments []fragment

	for _, oldFragment := range fragmentizeLatex(code, originalSettings) {
		if oldFragment.markupOnly {
			newFragments = append(newFragments, oldFragment)
			continue
		}

		for _, match := range bibtexEntryMatcher.findAllMatches(oldFragment.code, nil) {
			lastArgument := len(match.arguments) - 1
			argumentContents := match.argumentContents(lastArgument)
			argumentContentsFromPos := match.argumentContentsFromPos(lastArgument)

			for _, keyValuePair := range parsePackageOptions(argumentContents) {
				fieldName := keyValuePair.keyInfo.plainText
				if keyValuePair.valueInfo.fromPos == -1 {
					continue
				}
				if checked, known := oldFragment.settings.bibtexFields[fieldName]; known && !checked {
					continue
				}

				newFragments = append(newFragments, fragment{
					keyValuePair.value,
					oldFragment.fromPos + argumentContentsFromPos + keyValuePair.valueInfo.fromPos,
					oldFragment.settings,
					false,
				})
			}
		}
	}

	return dropEmptyAndSort(newFragments)
}
