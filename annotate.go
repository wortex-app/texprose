// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package texprose

import (
	"fmt"
	"sort"
	"sync"
)

const defaultLanguage = "en-US"

// sortedBabelTags returns the language tags derived from the babel language
// map in deterministic order (upstream iterates hash-map order, which only
// affects internal list order, not behavior).
func sortedBabelKeys() []string {
	keys := make([]string, 0, len(babelLanguageMap))
	for key := range babelLanguageMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func convertBabelLanguageToTag(language string) string {
	tag := make([]byte, 0, len(language))
	for i := 0; i < len(language); i++ {
		if isASCIILetter(language[i]) {
			tag = append(tag, language[i])
		}
	}
	return string(tag)
}

func specToSignature(spec sigSpec) *commandSignature {
	signature, err := newCommandSignature(
		spec.prototype, spec.action, dummyGenerator{plural: spec.plural, vowel: spec.vowel}, true)
	if err != nil {
		panic(fmt.Sprintf("invalid built-in prototype %q: %v", spec.prototype, err))
	}
	return signature
}

var defaultCommandSignatures = sync.OnceValue(func() []*commandSignature {
	signatures := make([]*commandSignature, 0, len(defaultCommandSpecs)+len(babelLanguageMap))
	for _, spec := range defaultCommandSpecs {
		signatures = append(signatures, specToSignature(spec))
	}
	for _, language := range sortedBabelKeys() {
		signatures = append(signatures, specToSignature(sigSpec{
			prototype: "\\text" + convertBabelLanguageToTag(language) + "{}",
			action:    actionDummy,
		}))
	}
	return signatures
})

var defaultEnvironmentSignatures = sync.OnceValue(func() []*commandSignature {
	var signatures []*commandSignature
	appendEnvironment := func(prototype string, act action) {
		signature, err := newEnvironmentSignature(prototype, act)
		if err != nil {
			panic(fmt.Sprintf("invalid built-in environment prototype %q: %v", prototype, err))
		}
		signatures = append(signatures, signature)
	}

	for _, spec := range defaultEnvironmentSpecs {
		appendEnvironment(spec.prototype, spec.action)
	}
	for _, language := range sortedBabelKeys() {
		appendEnvironment(language, actionIgnore)
		if tag := convertBabelLanguageToTag(language); len(tag) != len(language) {
			appendEnvironment(tag, actionIgnore)
		}
	}
	return signatures
})

func actionToInternal(a Action) (action, dummyGenerator, error) {
	switch a {
	case ActionDefault:
		return actionDefault, dummyGenerator{}, nil
	case ActionIgnore:
		return actionIgnore, dummyGenerator{}, nil
	case ActionDummy:
		return actionDummy, dummyGenerator{}, nil
	case ActionPluralDummy:
		return actionDummy, dummyGenerator{plural: true}, nil
	case ActionVowelDummy:
		return actionDummy, dummyGenerator{vowel: true}, nil
	}
	return 0, dummyGenerator{}, fmt.Errorf("invalid action %d", a)
}

func sortedKeys(m map[string]Action) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// buildSignatureMaps merges user command/environment definitions over the
// built-in tables. With strict=true (API options), invalid definitions are
// errors; with strict=false (magic comments), they are ignored like upstream.
func buildSignatureMaps(commands, environments map[string]Action, strict bool) (commandMap, environmentMap map[string][]*commandSignature, err error) {
	commandSignatures := append([]*commandSignature(nil), defaultCommandSignatures()...)
	for _, prototype := range sortedKeys(commands) {
		act, dummy, err := actionToInternal(commands[prototype])
		if err != nil {
			if strict {
				return nil, nil, fmt.Errorf("command %q: %w", prototype, err)
			}
			continue
		}
		signature, err := newCommandSignature(prototype, act, dummy, true)
		if err != nil {
			if strict {
				return nil, nil, err
			}
			continue
		}
		commandSignatures = append(commandSignatures, signature)
	}

	environmentSignatures := append([]*commandSignature(nil), defaultEnvironmentSignatures()...)
	for _, prototype := range sortedKeys(environments) {
		act, _, err := actionToInternal(environments[prototype])
		if err == nil && act == actionDummy {
			err = fmt.Errorf("dummy actions are not valid for environments")
		}
		if err != nil {
			if strict {
				return nil, nil, fmt.Errorf("environment %q: %w", prototype, err)
			}
			continue
		}
		signature, err := newEnvironmentSignature(prototype, act)
		if err != nil {
			if strict {
				return nil, nil, err
			}
			continue
		}
		environmentSignatures = append(environmentSignatures, signature)
	}

	return createSignatureMap(commandSignatures), createSignatureMap(environmentSignatures), nil
}

// Annotate converts LaTeX or BibTeX source code into LanguageTool annotated
// text. The concatenation of Text and Markup values of the returned segments
// reproduces source exactly.
func Annotate(source string, opts Options) (Annotation, error) {
	if opts.CodeLanguage == "" {
		opts.CodeLanguage = LaTeX
	}
	switch opts.CodeLanguage {
	case LaTeX, BibTeX, Rsweave:
	default:
		return Annotation{}, fmt.Errorf("texprose: unsupported code language %q", opts.CodeLanguage)
	}

	language := opts.Language
	if language == "" {
		language = defaultLanguage
	}

	bibtexFields := make(map[string]bool, len(defaultBibtexFields)+len(opts.BibTeXFields))
	for field, checked := range defaultBibtexFields {
		bibtexFields[field] = checked
	}
	for field, checked := range opts.BibTeXFields {
		bibtexFields[field] = checked
	}

	baseSettings := settings{
		language:     language,
		enabled:      true,
		commands:     opts.Commands,
		environments: opts.Environments,
		bibtexFields: bibtexFields,
	}

	baseCommandMap, baseEnvironmentMap, err := buildSignatureMaps(opts.Commands, opts.Environments, true)
	if err != nil {
		return Annotation{}, fmt.Errorf("texprose: %w", err)
	}

	var fragments []fragment
	builderLanguageID := "latex"
	switch opts.CodeLanguage {
	case BibTeX:
		fragments = fragmentizeBibtex(source, baseSettings)
	case Rsweave:
		builderLanguageID = "rsweave"
		fragments = fragmentizeLatex(source, baseSettings)
	default:
		fragments = fragmentizeLatex(source, baseSettings)
	}

	var segments []Segment
	cursor := 0

	for _, frag := range fragments {
		if frag.fromPos < cursor {
			// Defensive: overlapping fragments cannot be represented.
			continue
		}
		if frag.fromPos > cursor {
			segments = append(segments, Segment{Markup: source[cursor:frag.fromPos]})
		}

		if frag.markupOnly || !frag.settings.enabled {
			segments = append(segments, Segment{Markup: frag.code})
		} else {
			commandMap, environmentMap := baseCommandMap, baseEnvironmentMap
			if frag.settings.tablesOverridden {
				commandMap, environmentMap, _ = buildSignatureMaps(
					frag.settings.commands, frag.settings.environments, false)
			}

			fragmentBuilder := newBuilder(builderLanguageID, frag.settings.language, commandMap, environmentMap)
			fragmentBuilder.addCode(frag.code)
			segments = append(segments, fragmentBuilder.build()...)
		}

		cursor = frag.fromPos + len(frag.code)
	}

	if cursor < len(source) {
		segments = append(segments, Segment{Markup: source[cursor:]})
	}

	return Annotation{Segments: normalizeSegments(segments)}, nil
}

// normalizeSegments drops empty segments and merges adjacent segments of the
// same kind, mirroring the part coalescing of the upstream builder across
// fragment boundaries.
func normalizeSegments(segments []Segment) []Segment {
	var normalized []Segment

	for _, segment := range segments {
		if segment.Text == "" && segment.Markup == "" && segment.InterpretAs == "" {
			continue
		}

		if len(normalized) > 0 {
			last := &normalized[len(normalized)-1]
			if segment.Text != "" && last.Text != "" {
				last.Text += segment.Text
				continue
			}
			if segment.Text == "" && last.Text == "" {
				last.Markup += segment.Markup
				last.InterpretAs += segment.InterpretAs
				continue
			}
		}
		normalized = append(normalized, segment)
	}

	return normalized
}
