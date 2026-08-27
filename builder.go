// Copyright (C) 2019-2025
// Julian Valentin, Daniel Spitzer, LTeX+ Development Community
// Copyright (C) 2026 texprose contributors
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Ported from ltex-ls-plus: parsing/CodeAnnotatedTextBuilder.kt,
// parsing/CharacterBasedCodeAnnotatedTextBuilder.kt and
// parsing/latex/LatexAnnotatedTextBuilder.kt

package texprose

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type builderMode int

const (
	modeParagraphText builderMode = iota
	modeInlineText
	modeHeading
	modeInlineMath
	modeDisplayMath
	modeIgnoreEnvironment
	modeRsweave
)

func isMathMode(mode builderMode) bool {
	return mode == modeInlineMath || mode == modeDisplayMath
}

func isIgnoreEnvironmentMode(mode builderMode) bool { return mode == modeIgnoreEnvironment }

func isTextMode(mode builderMode) bool {
	return !isMathMode(mode) && !isIgnoreEnvironmentMode(mode)
}

type mathVowelState int

const (
	mathVowelUndecided mathVowelState = iota
	mathStartsWithVowel
	mathStartsWithConsonant
)

type segmentKind int

const (
	kindNone segmentKind = iota
	kindText
	kindMarkup
)

// ignoreEnvironmentEnd describes what terminates the current ignored
// environment: \end{name} for \begin environments, \stop<name> (not followed
// by a letter) for ConTeXt-style ones. It replaces the dynamically built
// regexes of the upstream builder (Go's RE2 has no lookahead).
type ignoreEnvironmentEnd struct {
	name    string
	isBegin bool
}

func (e *ignoreEnvironmentEnd) match(code string, pos int) string {
	if e.isBegin {
		end := "\\end{" + e.name + "}"
		if strings.HasPrefix(code[pos:], end) {
			return end
		}
		return ""
	}

	end := "\\stop" + e.name
	if !strings.HasPrefix(code[pos:], end) {
		return ""
	}
	if next := pos + len(end); next < len(code) && isASCIILetter(code[next]) {
		return ""
	}
	return end
}

var mathEnvironments = map[string]bool{
	"align": true, "align*": true, "alignat": true, "alignat*": true,
	"displaymath": true, "eqnarray": true, "eqnarray*": true,
	"equation": true, "equation*": true, "flalign": true, "flalign*": true,
	"formula": true, "gather": true, "gather*": true, "math": true,
	"multline": true, "multline*": true,
}

// Math commands whose appearance leaves the vowel state undecided.
var mathFormattingCommands = map[string]bool{
	"\\bm": true, "\\boldsymbol": true, "\\hat": true, "\\mathbb": true,
	"\\mathbf": true, "\\mathcal": true, "\\mathfrak": true, "\\mathit": true,
	"\\mathnormal": true, "\\mathsf": true, "\\mathtt": true, "\\mathop": true,
	"\\operatorname": true, "\\overbrace": true, "\\overleftarrow": true,
	"\\overleftrightarrow": true, "\\overline": true, "\\overrightarrow": true,
	"\\tilde": true, "\\underbrace": true, "\\underline": true, "\\vec": true,
	"\\widetilde": true, "\\widehat": true,
}

var mathVowelCommands = map[string]bool{
	"\\alpha": true, "\\ell": true, "\\epsilon": true, "\\eta": true,
	"\\iota": true, "\\Omega": true, "\\omega": true, "\\varepsilon": true,
}

func isPunctuationChar(r rune) bool {
	switch r {
	case '.', ',', ':', ';', '…':
		return true
	}
	return false
}

func isVowelChar(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'f', 'h', 'i', 'l', 'm', 'n', 'o', 'r', 's', 'x':
		return true
	}
	return false
}

type builder struct {
	codeLanguageID string
	language       string
	dummyGen       dummyGenerator
	dummyCounter   int

	commandSignatureMap     map[string][]*commandSignature
	environmentSignatureMap map[string][]*commandSignature

	// Segment assembly (CodeAnnotatedTextBuilder)
	segments       []Segment
	curText        strings.Builder
	curMarkup      strings.Builder
	curInterpretAs strings.Builder
	curKind        segmentKind

	// Character stream (CharacterBasedCodeAnnotatedTextBuilder)
	code      string
	pos       int
	curChar   rune
	curString string

	// LaTeX state machine (LatexAnnotatedTextBuilder)
	lastSpace                 string
	lastPunctuation           string
	dummyLastSpace            string
	dummyLastPunctuation      string
	isMathEmpty               bool
	mathVowel                 mathVowelState
	preserveDummyLast         bool
	canInsertSpaceBeforeDummy bool
	isMathCharTrivial         bool
	ignoreEnvironmentEnd      *ignoreEnvironmentEnd
	modeStack                 []builderMode
	curMode                   builderMode
}

func newBuilder(codeLanguageID, language string, commandMap, environmentMap map[string][]*commandSignature) *builder {
	return &builder{
		codeLanguageID:          codeLanguageID,
		language:                language,
		commandSignatureMap:     commandMap,
		environmentSignatureMap: environmentMap,
		modeStack:               []builderMode{modeParagraphText},
		curMode:                 modeParagraphText,
	}
}

func createSignatureMap(signatures []*commandSignature) map[string][]*commandSignature {
	byPrefix := make(map[string][]*commandSignature)
	for _, signature := range signatures {
		byPrefix[signature.prefix] = append(byPrefix[signature.prefix], signature)
	}
	return byPrefix
}

// --- segment assembly ---------------------------------------------------

func (b *builder) finalizePart() {
	switch b.curKind {
	case kindMarkup:
		b.segments = append(b.segments, Segment{
			Markup:      b.curMarkup.String(),
			InterpretAs: b.curInterpretAs.String(),
		})
		b.curMarkup.Reset()
		b.curInterpretAs.Reset()
	case kindText:
		b.segments = append(b.segments, Segment{Text: b.curText.String()})
		b.curText.Reset()
	}
	b.curKind = kindNone
}

func (b *builder) build() []Segment {
	b.finalizePart()
	return b.segments
}

func (b *builder) addText(text string) {
	if text == "" {
		return
	}
	if b.curKind == kindMarkup {
		b.finalizePart()
	}
	b.curKind = kindText
	b.curText.WriteString(text)
	b.pos += len(text)
	b.textAdded(text)
}

func (b *builder) addMarkup(markup string) {
	if markup == "" {
		return
	}
	if b.curKind == kindText {
		b.finalizePart()
	}
	b.curKind = kindMarkup
	b.curMarkup.WriteString(markup)
	b.pos += len(markup)

	if b.preserveDummyLast {
		b.preserveDummyLast = false
	} else {
		b.dummyLastSpace = ""
		b.dummyLastPunctuation = ""
	}
}

func (b *builder) addMarkupIA(markup, interpretAs string) {
	if interpretAs == "" {
		b.addMarkup(markup)
		return
	}
	if b.curKind == kindText {
		b.finalizePart()
	}
	b.curKind = kindMarkup
	b.curMarkup.WriteString(markup)
	b.curInterpretAs.WriteString(interpretAs)
	b.pos += len(markup)

	b.preserveDummyLast = false
	b.textAdded(interpretAs)
}

// addTextAsMarkup emits source as markup interpreted as shown while applying
// exactly the state effects of addText(shown). Used for the lone-newline
// substitution (deviation 1): upstream adds the text " " for a source "\n",
// which would break byte-exact source reconstruction.
func (b *builder) addTextAsMarkup(source, shown string) {
	if b.curKind == kindText {
		b.finalizePart()
	}
	b.curKind = kindMarkup
	b.curMarkup.WriteString(source)
	b.curInterpretAs.WriteString(shown)
	b.pos += len(source)
	b.textAdded(shown)
}

func (b *builder) textAdded(text string) {
	if text == "" {
		return
	}
	lastChar, _ := utf8.DecodeLastRuneInString(text)
	switch lastChar {
	case ' ', '\n', '\r':
		b.lastSpace = " "
	default:
		b.lastSpace = ""
	}
	if isPunctuationChar(lastChar) {
		b.lastPunctuation = " "
	} else {
		b.lastPunctuation = ""
	}
}

// --- mode stack ---------------------------------------------------------

func (b *builder) topMode() (builderMode, bool) {
	if len(b.modeStack) == 0 {
		return 0, false
	}
	return b.modeStack[len(b.modeStack)-1], true
}

func (b *builder) topModeIs(mode builderMode) bool {
	top, ok := b.topMode()
	return ok && top == mode
}

func (b *builder) pushMode(mode builderMode) {
	b.modeStack = append(b.modeStack, mode)
}

func (b *builder) popMode() {
	if len(b.modeStack) > 0 {
		b.modeStack = b.modeStack[:len(b.modeStack)-1]
	}
	if len(b.modeStack) == 0 {
		b.modeStack = append(b.modeStack, modeParagraphText)
	}
}

func (b *builder) enterDisplayMath() {
	b.pushMode(modeDisplayMath)
	b.isMathEmpty = true
	b.mathVowel = mathVowelUndecided
	b.canInsertSpaceBeforeDummy = true
}

func (b *builder) enterInlineMath() {
	b.pushMode(modeInlineMath)
	b.isMathEmpty = true
	b.mathVowel = mathVowelUndecided
	b.canInsertSpaceBeforeDummy = true
	b.isMathCharTrivial = true
}

// --- dummies ------------------------------------------------------------

func (b *builder) generateDummy() string {
	return b.generateDummyWith(b.dummyGen)
}

func (b *builder) generateDummyWith(gen dummyGenerator) string {
	startsWithVowel := b.mathVowel == mathStartsWithVowel
	var dummy string

	if isTextMode(b.curMode) {
		dummy = gen.generate(b.language, b.dummyCounter, startsWithVowel)
		b.dummyCounter++
	} else if b.isMathEmpty {
		if b.curMode == modeDisplayMath {
			if b.lastSpace == "" {
				dummy = " "
			}
		}
	} else if b.curMode == modeDisplayMath {
		leadingSpace := ""
		if b.lastSpace == "" {
			leadingSpace = " "
		}
		trailingSpace := " "
		if b.topModeIs(modeInlineText) {
			trailingSpace = b.dummyLastSpace
		}
		dummy = leadingSpace + gen.generate(b.language, b.dummyCounter, false) +
			b.dummyLastPunctuation + trailingSpace
		b.dummyCounter++
	} else {
		dummy = gen.generate(b.language, b.dummyCounter, startsWithVowel) +
			b.dummyLastPunctuation + b.dummyLastSpace
		b.dummyCounter++
	}

	b.dummyLastSpace = ""
	b.dummyLastPunctuation = ""
	b.mathVowel = mathVowelUndecided
	return dummy
}

// --- character loop -----------------------------------------------------

func (b *builder) addCode(code string) {
	b.pos = len(b.code)
	b.code += code

	for b.pos < len(b.code) {
		lastPos := b.pos
		r, size := utf8.DecodeRuneInString(b.code[b.pos:])
		b.curChar = r
		b.curString = b.code[b.pos : b.pos+size]
		b.processCharacter()

		if b.pos <= lastPos {
			// Upstream logs and skips the character, losing it from the
			// output. Emit it as markup instead so that segments always
			// reconstruct the source (deviation 2).
			b.pos = lastPos
			b.addMarkup(b.code[lastPos : lastPos+size])
		}
	}
}

func (b *builder) processCharacter() {
	b.curMode, _ = b.topMode()
	b.isMathCharTrivial = false

	if isIgnoreEnvironmentMode(b.curMode) {
		b.processIgnoredEnvironmentContents()
	} else if b.codeLanguageID == "rsweave" && b.curMode == modeRsweave {
		if b.curChar == '@' {
			b.popMode()
		}
		b.addMarkup(b.curString)
	} else {
		switch b.curChar {
		case '\\':
			b.processBackslash()
		case '{':
			b.processOpeningBrace()
		case '}':
			b.processClosingBrace()
		case '$':
			b.processDollar()
		case '%':
			b.processPercentage()
		case ' ', '&', '~', '\n', '\r', '\t':
			b.processWhitespace()
		case '`', '\'', '"':
			b.processQuotationMark()
		default:
			b.processDefaultCharacter()
		}
	}

	if !b.isMathCharTrivial {
		b.canInsertSpaceBeforeDummy = false
		b.isMathEmpty = false
	}
}

func (b *builder) processIgnoredEnvironmentContents() {
	if b.ignoreEnvironmentEnd != nil {
		if end := b.ignoreEnvironmentEnd.match(b.code, b.pos); end != "" {
			b.popMode()
			b.addMarkup(end)
			return
		}
		b.addMarkup(b.curString)
		return
	}
	b.popMode()
}

func (b *builder) processBackslash() {
	command := matchCommand(b.code, b.pos)

	switch {
	case command == "\\begin" || command == "\\end" ||
		strings.HasPrefix(command, "\\start") || strings.HasPrefix(command, "\\stop"):
		b.processEnvironmentCommand(command)

	case command == "\\$" || command == "\\%" || command == "\\&":
		b.addMarkupIA(command, command[1:])

	case command == "\\[":
		b.enterDisplayMath()
		b.addMarkup(command)

	case command == "\\(":
		b.enterInlineMath()
		b.addMarkup(command)

	case command == "\\]" || command == "\\)":
		b.popMode()
		b.addMarkupIA(command, b.generateDummy())

	case command == "\\AA":
		b.addSpecialLetter(command, "Å")
	case command == "\\L":
		b.addSpecialLetter(command, "Ł")
	case command == "\\O":
		b.addSpecialLetter(command, "Ø")
	case command == "\\SS":
		b.addSpecialLetter(command, "ẞ")
	case command == "\\aa":
		b.addSpecialLetter(command, "å")
	case command == "\\i":
		b.addSpecialLetter(command, "ı")
	case command == "\\j":
		b.addSpecialLetter(command, "ȷ")
	case command == "\\l":
		b.addSpecialLetter(command, "ł")
	case command == "\\o":
		b.addSpecialLetter(command, "ø")
	case command == "\\ss":
		b.addSpecialLetter(command, "ß")

	case command == "\\`" || command == "\\'" || command == "\\^" || command == "\\~" ||
		command == "\\\"" || command == "\\=" || command == "\\." || command == "\\H" ||
		command == "\\b" || command == "\\c" || command == "\\d" || command == "\\k" ||
		command == "\\r" || command == "\\u" || command == "\\v":
		if !isMathMode(b.curMode) {
			if match, accentCommand, letter := matchAccent(accentRegexp, b.code, b.pos); match != "" {
				b.addMarkupIA(match, convertAccentToUnicode(accentCommand, letter))
			} else {
				b.addMarkup(command)
			}
		} else {
			b.addMarkup(command)
		}

	case command == "\\-":
		b.addMarkup(command)

	case command == "\\ " || command == "\\," || command == "\\;" || command == "\\\\" ||
		command == "\\hfill" || command == "\\hspace" || command == "\\hspace*" ||
		command == "\\quad" || command == "\\qquad" || command == "\\newline":
		b.processSpacingCommand(command)

	case command == "\\dots" || command == "\\eg" || command == "\\egc" ||
		command == "\\euro" || command == "\\ie" || command == "\\iec":
		interpretAs := ""
		if !isMathMode(b.curMode) {
			switch command {
			case "\\dots":
				interpretAs = "…"
			case "\\eg":
				interpretAs = "e.g."
			case "\\egc":
				interpretAs = "e.g.,"
			case "\\euro":
				interpretAs = "€"
			case "\\ie":
				interpretAs = "i.e."
			case "\\iec":
				interpretAs = "i.e.,"
			}
		}
		b.addMarkupIA(command, interpretAs)

	case command == "\\notag" || command == "\\qed":
		b.preserveDummyLast = true
		b.addMarkup(command)

	case isHeadingCommand(command):
		b.addMarkup(command)
		if headingArgument := matchArgumentFromPosition(b.code, b.pos, argBracket); headingArgument != "" {
			b.addMarkup(headingArgument)
		}
		if b.pos < len(b.code) && b.code[b.pos] == '{' {
			b.pushMode(modeHeading)
			b.addMarkup("{")
		}

	case (command == "\\text" || command == "\\intertext") &&
		b.pos+len(command) < len(b.code) && b.code[b.pos+len(command)] == '{':
		// Upstream appends command+"{" unconditionally; requiring the brace
		// here preserves source reconstruction (deviation 3). Without a
		// brace the command falls through to the generic branch below.
		b.pushMode(modeInlineText)
		interpretAs := ""
		if isMathMode(b.curMode) {
			interpretAs = b.generateDummy()
		}
		b.addMarkupIA(command+"{", interpretAs)

	case command == "\\verb":
		if verbCommand := matchVerb(b.code, b.pos); verbCommand != "" {
			b.addMarkupIA(verbCommand, b.generateDummy())
		}
		// An unterminated \verb leaves pos unchanged; the loop guard in
		// addCode consumes the backslash as markup (deviation 2).

	default:
		b.processGenericCommand(command)
	}
}

func (b *builder) addSpecialLetter(command, letter string) {
	if isMathMode(b.curMode) {
		b.addMarkup(command)
		return
	}
	b.addMarkupIA(command, letter)
}

func isHeadingCommand(command string) bool {
	command = strings.TrimSuffix(command, "*")
	switch command {
	case "\\part", "\\chapter", "\\section", "\\subsection", "\\subsubsection",
		"\\paragraph", "\\subparagraph":
		return true
	}
	return false
}

func (b *builder) processEnvironmentCommand(command string) {
	b.preserveDummyLast = true
	isBeginEnvironment := command == "\\begin" || strings.HasPrefix(command, "\\start")

	var argument, environmentName string
	if command == "\\begin" || command == "\\end" {
		argument = matchBraceArgument(b.code, b.pos+len(command))
		if len(argument) >= 2 {
			environmentName = argument[1 : len(argument)-1]
		}
	} else if isBeginEnvironment {
		environmentName = command[len("\\start"):]
	} else {
		environmentName = command[len("\\stop"):]
	}

	argumentsProcessed := false
	interpretAs := ""

	if mathEnvironments[environmentName] {
		b.addMarkup(command)
		if isBeginEnvironment {
			if environmentName == "math" {
				b.enterInlineMath()
			} else {
				b.enterDisplayMath()
			}
		} else {
			b.popMode()
			interpretAs = b.generateDummy()
		}
	} else if isBeginEnvironment {
		var match string
		var matchingSignature *commandSignature

		for _, signature := range b.environmentSignatureMap[command+argument] {
			curMatch := signature.matchFromPosition(b.code, b.pos)
			if curMatch != "" && (len(curMatch) >= len(match) || signature.ignoreAllArguments) {
				match = curMatch
				matchingSignature = signature
			}
		}

		if matchingSignature != nil {
			if matchingSignature.action == actionIgnore {
				b.pushMode(modeIgnoreEnvironment)
				b.ignoreEnvironmentEnd = &ignoreEnvironmentEnd{
					name:    environmentName,
					isBegin: command == "\\begin",
				}
			}

			if matchingSignature.ignoreAllArguments {
				b.addMarkup(command)
			} else {
				b.addMarkup(match)
				argumentsProcessed = true
			}
		} else {
			b.addMarkup(command)
			b.pushMode(b.curMode)
		}
	} else {
		b.addMarkup(command)
		b.popMode()
	}

	if top, ok := b.topMode(); !ok || !isIgnoreEnvironmentMode(top) {
		b.isMathCharTrivial = true
		b.preserveDummyLast = true

		if !argumentsProcessed {
			b.addMarkupIA(argument, interpretAs)
			if isBeginEnvironment {
				b.processEnvironmentArguments()
			}
		}
	}
}

func (b *builder) processSpacingCommand(command string) {
	if command == "\\hspace" || command == "\\hspace*" {
		command += matchBraceArgument(b.code, b.pos+len(command))
	}

	if isMathMode(b.curMode) && b.lastSpace == "" && b.canInsertSpaceBeforeDummy {
		b.addMarkupIA(command, " ")
		return
	}

	b.preserveDummyLast = true

	if isMathMode(b.curMode) {
		b.addMarkup(command)
		b.dummyLastSpace = " "
	} else {
		space := " "
		if b.lastSpace != "" {
			space = ""
		} else if command == "\\," {
			space = " "
		}
		b.addMarkupIA(command, space)
	}
}

func (b *builder) processGenericCommand(command string) {
	var match string
	var matchingSignature *commandSignature

	for _, signature := range b.commandSignatureMap[command] {
		curMatch := signature.matchFromPosition(b.code, b.pos)
		if curMatch != "" && len(curMatch) >= len(match) {
			match = curMatch
			matchingSignature = signature
		}
	}

	if matchingSignature != nil && matchingSignature.action != actionDefault {
		switch matchingSignature.action {
		case actionIgnore:
			b.addMarkup(match)
		case actionDummy:
			b.addMarkupIA(match, b.generateDummyWith(matchingSignature.dummy))
		}
		return
	}

	if isMathMode(b.curMode) && b.mathVowel == mathVowelUndecided {
		switch {
		case mathFormattingCommands[command]:
			// state stays undecided
		case mathVowelCommands[command]:
			b.mathVowel = mathStartsWithVowel
		default:
			b.mathVowel = mathStartsWithConsonant
		}
	}
	b.addMarkup(command)
}

func (b *builder) processOpeningBrace() {
	if length := matchRegexp(lengthInBraceRegexp, b.code, b.pos); length != "" {
		b.addMarkup(length)
		return
	}

	if match, accentCommand, letter := matchAccent(accentInBraceRegexp, b.code, b.pos); match != "" {
		b.addMarkupIA(match, convertAccentToUnicode(accentCommand, letter))
		return
	}

	b.pushMode(b.curMode)
	b.addMarkup(b.curString)
}

func (b *builder) processClosingBrace() {
	interpretAs := ""
	if b.curMode == modeHeading && b.lastPunctuation == "" {
		interpretAs = "."
	} else if isTextMode(b.curMode) && b.pos+1 < len(b.code) && b.code[b.pos+1] == '{' {
		interpretAs = " "
	}

	b.popMode()
	b.addMarkupIA(b.curString, interpretAs)
	b.canInsertSpaceBeforeDummy = true

	if top, ok := b.topMode(); isTextMode(b.curMode) && ok && isMathMode(top) {
		b.isMathEmpty = true
	}

	b.isMathCharTrivial = true
}

func (b *builder) processDollar() {
	if strings.HasPrefix(b.code[b.pos:], "$$") {
		if b.curMode == modeDisplayMath {
			b.popMode()
			b.addMarkupIA("$$", b.generateDummy())
		} else {
			b.enterDisplayMath()
			b.addMarkup("$$")
		}
		return
	}

	if b.curMode == modeInlineMath {
		b.popMode()
		b.addMarkupIA(b.curString, b.generateDummy())
	} else {
		b.enterInlineMath()
		b.addMarkup(b.curString)
	}
}

func (b *builder) processPercentage() {
	comment := matchComment(b.code, b.pos, true)
	b.preserveDummyLast = true
	b.isMathCharTrivial = true

	interpretAs := ""
	if containsTwoEndsOfLine(comment) {
		interpretAs = "\n\n"
	}
	b.addMarkupIA(comment, interpretAs)
}

func (b *builder) processWhitespace() {
	var whitespace string
	if b.curChar != '~' && b.curChar != '&' {
		whitespace = matchWhitespace(b.code, b.pos)
	} else {
		whitespace = b.curString
	}

	b.preserveDummyLast = true
	b.isMathCharTrivial = true

	if isTextMode(b.curMode) {
		switch {
		case b.lastSpace == "" && whitespace == " ":
			b.addText(" ")
		case b.lastSpace == "" && whitespace == "\n":
			// Upstream: addText(" ") (deviation 1).
			b.addTextAsMarkup("\n", " ")
		case b.lastSpace == "" && whitespace == "\n\n":
			b.addText("\n\n")
		case containsTwoEndsOfLine(whitespace):
			b.addMarkupIA(whitespace, "\n\n")
		case b.curChar == '~':
			interpretAs := ""
			if b.lastSpace == "" {
				interpretAs = " "
			}
			b.addMarkupIA(whitespace, interpretAs)
		default:
			interpretAs := ""
			if b.lastSpace == "" {
				interpretAs = " "
			}
			b.addMarkupIA(whitespace, interpretAs)
		}
	} else {
		b.addMarkup(whitespace)
	}

	if b.curChar == '~' || b.curChar == '&' {
		b.dummyLastSpace = " "
	}
}

func (b *builder) processQuotationMark() {
	if !isTextMode(b.curMode) {
		b.addMarkup(b.curString)
		return
	}

	quote := ""
	smartQuote := ""

	if b.pos+1 < len(b.code) {
		quote = b.code[b.pos : b.pos+2]
		switch quote {
		case "``", "\"'":
			smartQuote = "“"
		case "''":
			smartQuote = "”"
		case "\"`":
			smartQuote = "„"
		case "\"-", "\"\"", "\"|":
			smartQuote = ""
		case "\"=", "\"~":
			smartQuote = "-"
		default:
			quote = ""
		}
	}

	if quote == "" {
		b.addText(b.curString)
	} else if smartQuote == "" {
		b.addMarkup(quote)
	} else {
		b.addMarkupIA(quote, smartQuote)
	}
}

func (b *builder) processDefaultCharacter() {
	switch b.curChar {
	case '-':
		if isTextMode(b.curMode) {
			if strings.HasPrefix(b.code[b.pos:], "---") {
				b.addMarkupIA("---", "—")
				return
			}
			if strings.HasPrefix(b.code[b.pos:], "--") {
				b.addMarkupIA("--", "–")
				return
			}
		}
	case '[':
		if length := matchRegexp(lengthInBracketRegexp, b.code, b.pos); length != "" {
			b.isMathCharTrivial = true
			b.preserveDummyLast = true
			b.addMarkup(length)
			return
		}
	case '<':
		if b.codeLanguageID == "rsweave" {
			if rsweaveBegin := matchRegexp(rsweaveBeginRegexp, b.code, b.pos); rsweaveBegin != "" {
				b.pushMode(modeRsweave)
				b.addMarkup(rsweaveBegin)
				return
			}
		}
	}

	if isTextMode(b.curMode) {
		b.addText(b.curString)
		if isPunctuationChar(b.curChar) {
			b.lastPunctuation = b.curString
		}
	} else {
		b.addMarkup(b.curString)
		if isPunctuationChar(b.curChar) {
			b.dummyLastPunctuation = b.curString
		}

		if b.mathVowel == mathVowelUndecided {
			if isVowelChar(b.curChar) {
				b.mathVowel = mathStartsWithVowel
			} else {
				b.mathVowel = mathStartsWithConsonant
			}
		}
	}
}

func (b *builder) processEnvironmentArguments() {
	for b.pos < len(b.code) {
		if argument := matchArgumentFromPosition(b.code, b.pos, argBrace); argument != "" {
			b.addMarkup(argument)
			continue
		}
		if argument := matchArgumentFromPosition(b.code, b.pos, argBracket); argument != "" {
			b.addMarkup(argument)
			continue
		}
		if argument := matchArgumentFromPosition(b.code, b.pos, argParenthesis); argument != "" {
			b.addMarkup(argument)
			continue
		}
		break
	}
}

func convertAccentToUnicode(accentCommand, letter string) string {
	if accentCommand == "" || letter == "" {
		return ""
	}

	var base rune
	switch letter {
	case "\\i":
		base = 'ı'
	case "\\j":
		base = 'ȷ'
	default:
		base, _ = utf8.DecodeRuneInString(letter)
	}

	var mark rune
	switch accentCommand[1] {
	case '`':
		mark = '\u0300' // grave
	case '\'':
		mark = '\u0301' // acute
	case '^':
		mark = '\u0302' // circumflex
	case '~':
		mark = '\u0303' // tilde
	case '"':
		mark = '\u0308' // diaeresis/umlaut
	case '=':
		mark = '\u0304' // macron
	case '.':
		mark = '\u0307' // dot above
	case 'H':
		mark = '\u030b' // double acute
	case 'b':
		mark = '\u0331' // macron below
	case 'c':
		mark = '\u0327' // cedilla
	case 'd':
		mark = '\u0323' // dot below
	case 'k':
		mark = '\u0328' // ogonek
	case 'r':
		mark = '\u030a' // ring above
	case 'u':
		mark = '\u0306' // breve
	case 'v':
		mark = '\u030c' // caron
	default:
		return string(base)
	}

	if composed, ok := accentComposition[mark][base]; ok {
		return string(composed)
	}
	return string(base) + string(mark)
}
