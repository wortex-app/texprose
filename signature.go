// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: parsing/latex/LatexCommandSignature.kt and
// parsing/latex/LatexEnvironmentSignature.kt

package texprose

import (
	"fmt"
	"regexp"
	"strings"
)

type action int

const (
	actionDefault action = iota
	actionIgnore
	actionDummy
)

type sigSpec struct {
	prototype string
	action    action
	plural    bool
	vowel     bool
}

type argumentType int

const (
	argBrace argumentType = iota
	argBracket
	argParenthesis
)

type commandSignature struct {
	prototype     string
	action        action
	dummy         dummyGenerator
	prefix        string
	argumentTypes []argumentType
	// prefixRegexp is set instead of matching prefix literally when the
	// signature was constructed with escapeCommandPrefix=false upstream
	// (only the BibTeX entry signature "@[A-Za-z]+{}").
	prefixRegexp *regexp.Regexp

	// Environment signature fields (LatexEnvironmentSignature). Only
	// meaningful when the signature was built by newEnvironmentSignature.
	ignoreAllArguments bool
	environmentName    string
}

// splitPrototype ports GENERIC_COMMAND_REGEX = ^(.+?)(\{}|\[]|\(\))*$: the
// lazy prefix group means the prefix is the shortest non-empty prefix whose
// remainder is a sequence of {} [] () tokens.
func splitPrototype(prototype string) (prefix string, argumentTypes []argumentType, ok bool) {
	if prototype == "" {
		return "", nil, false
	}

	for p := 1; p <= len(prototype); p++ {
		types, valid := parseArgumentTokens(prototype[p:])
		if valid {
			return prototype[:p], types, true
		}
	}
	return "", nil, false
}

func parseArgumentTokens(s string) ([]argumentType, bool) {
	var types []argumentType
	for s != "" {
		switch {
		case strings.HasPrefix(s, "{}"):
			types = append(types, argBrace)
		case strings.HasPrefix(s, "[]"):
			types = append(types, argBracket)
		case strings.HasPrefix(s, "()"):
			types = append(types, argParenthesis)
		default:
			return nil, false
		}
		s = s[2:]
	}
	return types, true
}

func newCommandSignature(prototype string, act action, dummy dummyGenerator, escapePrefix bool) (*commandSignature, error) {
	prefix, argumentTypes, ok := splitPrototype(prototype)
	if !ok {
		return nil, fmt.Errorf("invalid command prototype %q", prototype)
	}

	signature := &commandSignature{
		prototype:     prototype,
		action:        act,
		dummy:         dummy,
		prefix:        prefix,
		argumentTypes: argumentTypes,
	}

	if !escapePrefix {
		prefixRegexp, err := regexp.Compile("^(?:" + prefix + ")")
		if err != nil {
			return nil, fmt.Errorf("invalid command prototype %q: %v", prototype, err)
		}
		signature.prefixRegexp = prefixRegexp
	}

	return signature, nil
}

var environmentPrefixRegexp = regexp.MustCompile(`^(?:\\begin\{([^}]+)}|\\start([A-Za-z]+))`)

func newEnvironmentSignature(prototype string, act action) (*commandSignature, error) {
	groups := environmentPrefixRegexp.FindStringSubmatch(prototype)

	fullPrototype := prototype
	if groups == nil {
		fullPrototype = "\\begin{" + prototype + "}"
	}

	signature, err := newCommandSignature(fullPrototype, act, dummyGenerator{}, true)
	if err != nil {
		return nil, err
	}

	environmentName := ""
	if groups != nil {
		environmentName = groups[1]
		if environmentName == "" {
			environmentName = groups[2]
		}
	}

	if environmentName != "" {
		signature.environmentName = environmentName
	} else {
		signature.ignoreAllArguments = true
		signature.environmentName = prototype
	}
	return signature, nil
}

func (s *commandSignature) matchPrefix(code string, fromPos int) string {
	if s.prefixRegexp != nil {
		return matchRegexp(s.prefixRegexp, code, fromPos)
	}
	if strings.HasPrefix(code[fromPos:], s.prefix) {
		return s.prefix
	}
	return ""
}

type argumentSpan struct{ from, to int }

func (s *commandSignature) match(code string, fromPos int, arguments *[]argumentSpan) int {
	pos := fromPos
	prefixMatch := s.matchPrefix(code, pos)
	if prefixMatch == "" {
		return -1
	}
	pos += len(prefixMatch)

	for _, argType := range s.argumentTypes {
		// A comment may separate the command from its argument; the
		// signature variant of the comment pattern has no \r support.
		pos += len(matchComment(code, pos, false))
		argument := matchArgumentFromPosition(code, pos, argType)
		if argument == "" {
			return -1
		}
		if arguments != nil {
			*arguments = append(*arguments, argumentSpan{pos, pos + len(argument)})
		}
		pos += len(argument)
	}

	return pos
}

// matchFromPosition returns the full source matched by the signature at
// fromPos, or "" if it does not match.
func (s *commandSignature) matchFromPosition(code string, fromPos int) string {
	toPos := s.match(code, fromPos, nil)
	if toPos < 0 {
		return ""
	}
	return code[fromPos:toPos]
}

func (s *commandSignature) matchArgumentsFromPosition(code string, fromPos int) ([]argumentSpan, bool) {
	var arguments []argumentSpan
	toPos := s.match(code, fromPos, &arguments)
	if toPos < 0 {
		return nil, false
	}
	return arguments, true
}

// matchArgumentFromPosition ports the balanced-bracket argument scanner,
// including its quirks: "(" never nests, and a mismatched closing bracket
// aborts the match.
func matchArgumentFromPosition(code string, fromPos int, argType argumentType) string {
	if fromPos >= len(code) {
		return ""
	}

	var openChar byte
	switch argType {
	case argBrace:
		openChar = '{'
	case argBracket:
		openChar = '['
	case argParenthesis:
		openChar = '('
	}

	if code[fromPos] != openChar {
		return ""
	}
	pos := fromPos + 1
	stack := []argumentType{argType}

	for pos < len(code) {
		switch code[pos] {
		case '\\':
			if pos+1 < len(code) {
				pos++
			}
		case '{':
			stack = append(stack, argBrace)
		case '[':
			stack = append(stack, argBracket)
		case '}', ']':
			current := argBrace
			if code[pos] == ']' {
				current = argBracket
			}
			switch {
			case len(stack) > 0 && stack[len(stack)-1] != current:
				return ""
			case len(stack) == 1:
				return code[fromPos : pos+1]
			default:
				stack = stack[:len(stack)-1]
			}
		case ')':
			if len(stack) == 1 && stack[0] == argParenthesis {
				return code[fromPos : pos+1]
			}
		}
		pos++
	}

	return ""
}
