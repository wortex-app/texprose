// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: parsing/RegexCodeFragmentizer.kt,
// parsing/latex/LatexFragmentizer.kt and settings/SettingsParser.kt.
//
// Deviation from upstream: upstream additionally emits
// *duplicate* fragments for content that should be re-checked under other
// settings (\footnote/\todo contents, babel inline commands and babel
// environments). A flat annotation cannot check the same source span twice,
// so this port only performs the splits that partition the document: magic
// comments, \usepackage[...]{babel} and \selectlanguage{...}.

package texprose

import (
	"regexp"
	"sort"
	"strings"
)

// settings carries the effective options for one fragment.
type settings struct {
	language     string
	enabled      bool
	commands     map[string]Action
	environments map[string]Action
	bibtexFields map[string]bool
	// tablesOverridden is set when magic comments changed commands or
	// environments, so the signature tables must be rebuilt per fragment.
	tablesOverridden bool
}

// fragment is a piece of the source annotated with the settings in effect.
// markupOnly fragments (magic-comment lines, disabled regions) are emitted
// as markup without being parsed.
type fragment struct {
	code       string
	fromPos    int
	settings   settings
	markupOnly bool
}

var (
	magicCommentRegexp = regexp.MustCompile(`(?m)^[ \t]*%[ \t]*(?i:ltex):(.*?)$`)
	keyValueRegexp     = regexp.MustCompile(`^[ \t\n\r]*([^ \t]+)=([^ \t\n\r]*)[ \t\n\r]*`)

	nullValueRegexp = regexp.MustCompile(`^(?i:null|nil|clear|#)$`)
	trueValueRegexp = regexp.MustCompile(`^(?i:true|1|yes)$`)

	latexCommandKeyRegexp     = regexp.MustCompile(`^(?i:(?:ltex\.)?latex\.commands\.)(.+)$`)
	latexEnvironmentKeyRegexp = regexp.MustCompile(`^(?i:(?:ltex\.)?latex\.environments\.)(.+)$`)
	bibtexFieldKeyRegexp      = regexp.MustCompile(`^(?i:(?:ltex\.)?bibtex\.fields\.)(.+)$`)
)

// settingsOverride tracks magic-comment overrides on top of the original
// settings; a null value resets the corresponding override.
type settingsOverride struct {
	original     settings
	language     *string
	enabled      *bool
	commands     map[string]Action
	environments map[string]Action
	bibtexFields map[string]bool
}

func (o *settingsOverride) toSettings() settings {
	result := o.original
	if o.language != nil {
		result.language = *o.language
	}
	if o.enabled != nil {
		result.enabled = *o.enabled
	}
	result.commands = mergeActionMaps(o.original.commands, o.commands)
	result.environments = mergeActionMaps(o.original.environments, o.environments)
	result.tablesOverridden = len(o.commands) > 0 || len(o.environments) > 0
	if len(o.bibtexFields) > 0 {
		merged := make(map[string]bool, len(o.original.bibtexFields)+len(o.bibtexFields))
		for key, value := range o.original.bibtexFields {
			merged[key] = value
		}
		for key, value := range o.bibtexFields {
			merged[key] = value
		}
		result.bibtexFields = merged
	}
	return result
}

func mergeActionMaps(original, overrides map[string]Action) map[string]Action {
	if len(overrides) == 0 {
		return original
	}
	merged := make(map[string]Action, len(original)+len(overrides))
	for key, value := range original {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}
	return merged
}

func parseActionString(value string) (Action, bool) {
	switch value {
	case "default":
		return ActionDefault, true
	case "ignore":
		return ActionIgnore, true
	case "dummy":
		return ActionDummy, true
	case "pluralDummy":
		return ActionPluralDummy, true
	case "vowelDummy":
		return ActionVowelDummy, true
	}
	return 0, false
}

// applySettingsLine parses one magic-comment settings line ("key=value ...")
// and applies the keys this library honors; unknown keys are ignored.
func (o *settingsOverride) applySettingsLine(line string) {
	remaining := line
	for {
		match := keyValueRegexp.FindStringSubmatch(remaining)
		if match == nil {
			break
		}
		remaining = remaining[len(match[0]):]
		o.applySetting(match[1], match[2])
	}
}

func (o *settingsOverride) applySetting(key, value string) {
	isNull := nullValueRegexp.MatchString(value)

	switch {
	case strings.EqualFold(key, "language"):
		if isNull {
			o.language = nil
		} else {
			language := value
			o.language = &language
		}

	case strings.EqualFold(key, "enabled"):
		if isNull || value == "" {
			o.enabled = nil
		} else {
			enabled := trueValueRegexp.MatchString(value)
			o.enabled = &enabled
		}

	case latexCommandKeyRegexp.MatchString(key):
		command := latexCommandKeyRegexp.FindStringSubmatch(key)[1]
		if isNull {
			delete(o.commands, command)
		} else if action, ok := parseActionString(value); ok {
			if o.commands == nil {
				o.commands = make(map[string]Action)
			}
			o.commands[command] = action
		}

	case latexEnvironmentKeyRegexp.MatchString(key):
		environment := latexEnvironmentKeyRegexp.FindStringSubmatch(key)[1]
		if isNull {
			delete(o.environments, environment)
		} else if action, ok := parseActionString(value); ok && action <= ActionIgnore {
			if o.environments == nil {
				o.environments = make(map[string]Action)
			}
			o.environments[environment] = action
		}

	case bibtexFieldKeyRegexp.MatchString(key):
		field := bibtexFieldKeyRegexp.FindStringSubmatch(key)[1]
		if isNull || value == "" {
			delete(o.bibtexFields, field)
		} else {
			if o.bibtexFields == nil {
				o.bibtexFields = make(map[string]bool)
			}
			o.bibtexFields[field] = trueValueRegexp.MatchString(value)
		}
	}
}

// fragmentizeMagicComments splits code at "% ltex:" magic comments,
// threading the cumulative settings overrides through the fragments
// (RegexCodeFragmentizer.buildFragments). The comment lines themselves
// become markup-only fragments (upstream: "nop" fragments).
func fragmentizeMagicComments(code string, originalSettings settings) []fragment {
	var fragments []fragment
	override := settingsOverride{original: originalSettings}
	curSettings := override.toSettings()
	curPos := 0

	for _, location := range magicCommentRegexp.FindAllStringSubmatchIndex(code, -1) {
		start, end := location[0], location[1]
		fragments = append(fragments, fragment{code[curPos:start], curPos, curSettings, false})

		settingsLine := code[location[2]:location[3]]
		override.applySettingsLine(settingsLine)
		curSettings = override.toSettings()

		fragments = append(fragments, fragment{code[start:end], start, curSettings, true})
		curPos = end
	}

	fragments = append(fragments, fragment{code[curPos:], curPos, curSettings, false})
	return fragments
}

var (
	usePackageSignature     = mustSignature("\\usepackage[]{}")
	usePackageMatcher       = newSignatureMatcher([]*commandSignature{usePackageSignature}, true)
	selectLanguageSignature = mustSignature("\\selectlanguage{}")
	selectLanguageMatcher   = newSignatureMatcher([]*commandSignature{selectLanguageSignature}, true)
)

func mustSignature(prototype string) *commandSignature {
	signature, err := newCommandSignature(prototype, actionIgnore, dummyGenerator{}, true)
	if err != nil {
		panic(err)
	}
	return signature
}

func ignorePrototypes(fragmentSettings settings) map[string]bool {
	var prototypes map[string]bool
	for prototype, action := range fragmentSettings.commands {
		if action == ActionIgnore {
			if prototypes == nil {
				prototypes = make(map[string]bool)
			}
			prototypes[prototype] = true
		}
	}
	return prototypes
}

// fragmentizeBabelUsePackage splits fragments at \usepackage[...]{babel}
// commands that determine a language.
func fragmentizeBabelUsePackage(fragments []fragment) []fragment {
	var newFragments []fragment

	for _, oldFragment := range fragments {
		if oldFragment.markupOnly {
			newFragments = append(newFragments, oldFragment)
			continue
		}

		prevFromPos := 0
		prevSettings := oldFragment.settings

		for _, match := range usePackageMatcher.findAllMatches(oldFragment.code, ignorePrototypes(oldFragment.settings)) {
			if match.argumentContents(1) != "babel" {
				continue
			}

			babelLanguage := ""
			for _, option := range parsePackageOptions(match.argumentContents(0)) {
				if _, ok := babelLanguageMap[option.keyInfo.plainText]; ok {
					babelLanguage = option.keyInfo.plainText
				} else if option.keyInfo.plainText == "main" {
					if _, ok := babelLanguageMap[option.valueInfo.plainText]; ok {
						babelLanguage = option.valueInfo.plainText
						break
					}
				}
			}
			if babelLanguage == "" {
				continue
			}

			nextSettings := prevSettings
			nextSettings.language = babelLanguageMap[babelLanguage]

			newFragments = append(newFragments, fragment{
				oldFragment.code[prevFromPos:match.fromPos],
				oldFragment.fromPos + prevFromPos,
				prevSettings,
				false,
			})
			prevFromPos = match.fromPos
			prevSettings = nextSettings
		}

		newFragments = append(newFragments, fragment{
			oldFragment.code[prevFromPos:],
			oldFragment.fromPos + prevFromPos,
			prevSettings,
			false,
		})
	}

	return newFragments
}

// fragmentizeBabelSwitch splits fragments at \selectlanguage commands.
func fragmentizeBabelSwitch(fragments []fragment) []fragment {
	var newFragments []fragment

	for _, oldFragment := range fragments {
		if oldFragment.markupOnly {
			newFragments = append(newFragments, oldFragment)
			continue
		}

		prevFromPos := 0
		prevSettings := oldFragment.settings

		for _, match := range selectLanguageMatcher.findAllMatches(oldFragment.code, ignorePrototypes(oldFragment.settings)) {
			languageShortCode, ok := babelLanguageMap[match.argumentContents(0)]
			if !ok {
				continue
			}

			nextSettings := prevSettings
			nextSettings.language = languageShortCode

			newFragments = append(newFragments, fragment{
				oldFragment.code[prevFromPos:match.fromPos],
				oldFragment.fromPos + prevFromPos,
				prevSettings,
				false,
			})
			prevFromPos = match.fromPos
			prevSettings = nextSettings
		}

		newFragments = append(newFragments, fragment{
			oldFragment.code[prevFromPos:],
			oldFragment.fromPos + prevFromPos,
			prevSettings,
			false,
		})
	}

	return newFragments
}

func fragmentizeLatex(code string, originalSettings settings) []fragment {
	fragments := fragmentizeMagicComments(code, originalSettings)
	fragments = fragmentizeBabelUsePackage(fragments)
	fragments = fragmentizeBabelSwitch(fragments)
	return dropEmptyAndSort(fragments)
}

func dropEmptyAndSort(fragments []fragment) []fragment {
	filtered := fragments[:0]
	for _, f := range fragments {
		if f.code != "" {
			filtered = append(filtered, f)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].fromPos < filtered[j].fromPos
	})
	return filtered
}
