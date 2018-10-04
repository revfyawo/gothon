package tokenizer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type TokenValue int
type TokenLit struct {
	Token   TokenValue
	Literal string
}

var regexBlankLine = regexp.MustCompile(`^\s*$`)
var regexCommentLine = regexp.MustCompile(`^\s*#.*$`)
var regexJoinLine = regexp.MustCompile(`^.*\\$`)

var regexStringLit = regexp.MustCompile(`[uU]|([rR][fF]?)|([fF][rR]?)`)
var regexBytesLit = regexp.MustCompile(`([bB][rR]?)|([rR][bB])`)

var regexWhitespace = regexp.MustCompile(`\s`)
var regexStartIdent = regexp.MustCompile(`[a-zA-Z_]`)
var regexContIdent = regexp.MustCompile(`[a-zA-Z0-9_]`)
var regexDigit = regexp.MustCompile(`\d`)

type Tokenizer struct {
	scanner  *bufio.Scanner
	line     *strings.Reader
	lineText string

	indentStack []int
	joinLines   bool
	embedLvl    int

	longBuild *TokenLit
	longDelim rune

	Tokens chan TokenLit
}

func New(f *os.File) (t *Tokenizer) {
	return &Tokenizer{
		scanner:     bufio.NewScanner(f),
		line:        strings.NewReader(""),
		indentStack: []int{0},
		longBuild:   nil,
		Tokens:      make(chan TokenLit),
	}
}

func (t *Tokenizer) read() (rune, error) {
	ch, _, err := t.line.ReadRune()
	return ch, err
}

func (t *Tokenizer) unread() {
	_ = t.line.UnreadRune()
}

func (t *Tokenizer) peek(n int) (string, error) {
	if n < 0 {
		return "", errors.New("can't peek negative rune number")
	}

	var b strings.Builder
	var length = n
	for length > 0 {
		ch, err := t.read()
		if err != nil {
			t.line.Seek(-int64(b.Len()), io.SeekCurrent)
			return b.String(), errors.New("EOF reached")
		}
		b.WriteRune(ch)
		length--
	}
	t.line.Seek(-int64(n), io.SeekCurrent)
	return b.String(), nil
}

func (t *Tokenizer) readUnread() (rune, error) {
	ch, err := t.read()
	t.unread()
	return ch, err
}

func (t *Tokenizer) tokenWhitespace() {
	var char rune
	var err error
	var buf = new(bytes.Buffer)

	// Fill buffer with whitespace
	for {
		char, err = t.read()
		if err != nil {
			break
		} else if !regexWhitespace.MatchString(string(char)) {
			t.unread()
			break
		} else {
			buf.WriteRune(char)
		}
	}
}

func (t *Tokenizer) tokenIndentation() {
	var char rune
	var err error
	var indent int

	for {
		char, err = t.read()
		if err != nil {
			break
		} else if !regexWhitespace.MatchString(string(char)) {
			t.unread()
			break
		} else if char == ' ' {
			indent++
		} else if char == '\t' {
			indent += 8 - indent%8
		}
	}

	var topStack = t.indentStack[len(t.indentStack)-1]
	if indent > topStack {
		t.indentStack = append(t.indentStack, indent)
		t.Tokens <- TokenLit{Token: INDENT}
	} else if indent < topStack {
		for indent < topStack {
			t.indentStack = t.indentStack[:len(t.indentStack)-1]
			topStack = t.indentStack[len(t.indentStack)-1]
			t.Tokens <- TokenLit{Token: DEINDENT}
		}
	}
}

func (t *Tokenizer) tokenStringBytesLiteral(inLong bool) (bool, string, error) {
	ch, err := t.read()
	if err != nil {
		return false, "", err
	}

	var build strings.Builder
	var delim rune
	var empty bool
	var isLong bool

	var isString bool
	var isBytes bool
	var isRaw bool
	var isFormat bool
	if !inLong {
		if isQuote(ch) {
			delim = ch
			isString = true
		} else if regexBytesLit.MatchString(string(ch)) {
			isBytes = true
			build.WriteRune(ch)
			ch, err = t.read()
			if err != nil {
				// b identifier then EOF
				return false, build.String(), err
			} else if isQuote(ch) {
				// Just b options
				delim = ch
				build.Reset()
			} else {
				build.WriteRune(ch)
				if regexBytesLit.MatchString(build.String()) {
					ch, err := t.read()
					if err != nil {
						// br identifier then EOF
						return false, build.String(), err
					} else if isQuote(ch) {
						// br raw bytes
						isRaw = true
						delim = ch
						build.Reset()
					} else if regexContIdent.MatchString(build.String() + string(ch)) {
						// identifier starting with br
						build.WriteRune(ch)
						return false, build.String(), nil
					} else {
						// identifier br then something else
						t.unread()
						return false, build.String(), nil
					}
				} else {
					// identifier starting with b
					return false, build.String(), nil
				}
			}
		} else if regexStringLit.MatchString(string(ch)) {
			build.WriteRune(ch)
			ch, err = t.read()
			if err != nil {
				// u r or f identifier then EOF
				return false, build.String(), err
			} else if isQuote(ch) {
				// Just u r of f options
				isString = true
				delim = ch
				options := build.String()
				if options == "r" {
					isRaw = true
				} else if options == "f" {
					isFormat = true
				}
				build.Reset()
			} else {
				build.WriteRune(ch)
				findResult := regexStringLit.FindStringIndex(build.String())
				if findResult[0] == 0 && findResult[1] == 2 {
					ch, err = t.read()
					if err != nil {
						// rf or fr identifier then EOF
						return false, build.String(), err
					} else if isQuote(ch) {
						// rf or fr string
						isString = true
						isRaw = true
						isFormat = true
						delim = ch
						build.Reset()
					} else if regexContIdent.MatchString(build.String() + string(ch)) {
						// identifier starting with rf or fr
						build.WriteRune(ch)
						return false, build.String(), nil
					} else {
						// rf or fr identifier then something else
						t.unread()
						return false, build.String(), nil
					}
				} else if regexBytesLit.MatchString(build.String()) {
					ch, err = t.read()
					if err != nil {
						// rb identifier then EOF
						return false, build.String(), err
					} else if isQuote(ch) {
						// rb raw bytes
						isBytes = true
						isRaw = true
						delim = ch
						build.Reset()
					} else if regexContIdent.MatchString(build.String() + string(ch)) {
						// identifier starting with rb
						build.WriteRune(ch)
						return false, build.String(), nil
					} else {
						// rb identifier then something else
						t.unread()
						return false, build.String(), nil
					}
				} else {
					// identifier starting with r f or u
					return false, build.String(), nil
				}
			}
		} else {
			// Not a quote, b, r, f or u
			build.WriteRune(ch)
			return false, build.String(), nil
		}

		// string/bytes options & delimiter set
		// reader after first delimiter
		ch, err = t.read()
		if err != nil {
			// Weird case: end of line after first quote
			return false, "", errors.New("EOL while parsing string")
		} else if ch == delim {
			// Two delimiters
			ch, err = t.read()
			if err != nil {
				// Empty string/bytes
				empty = true
			} else if ch == delim {
				// Long string/bytes
				isLong = true
			} else {
				// Weird case: two delimiters & something else
				// To be handled by parser
				return t.endOfLiteral(isLong, isString, isBytes, isRaw, isFormat, "")
			}
		} else {
			// Start of string: handled below
			t.unread()
		}
	} else {
		t.unread()
		isLong = true
		delim = t.longDelim
		build.WriteString(t.longBuild.Literal)
		switch t.longBuild.Token {
		case LONGBYTES:
			isBytes = true
		case LONGRAWBYTES:
			isBytes = true
			isRaw = true
		case LONGSTRING:
			isString = true
		case LONGFSTRING:
			isString = true
			isFormat = true
		case LONGRAWSTRING:
			isString = true
			isRaw = true
		case LONGRAWFSTRING:
			isString = true
			isFormat = true
			isRaw = true
		}
	}

	// We know here if string/bytes is empty/long
	// reader is after
	if !empty {
		for {
			ch, err = t.read()
			if err != nil && isLong {
				return t.eofLongLiteral(err, delim, isString, isBytes, isRaw, isFormat, build.String())
			} else if err != nil {
				// EOF while parsing string
				return false, "", errors.New("EOL while parsing string")
			}

			if !isLong && ch == delim {
				cpt := 0
				for build.String()[build.Len()-cpt-1] == '\\' {
					cpt++
				}
				if cpt%2 == 0 {
					// end string/bytes delimiter
					return t.endOfLiteral(isLong, isString, isBytes, isRaw, isFormat, build.String())
				} else {
					build.WriteRune(ch)
				}
			} else if isLong && ch == delim {
				// long string/bytes delimiter
				ch, err = t.read()
				if err != nil {
					// EOF while parsing long string/bytes
					build.WriteRune(delim)
					return t.eofLongLiteral(err, delim, isString, isBytes, isRaw, isFormat, build.String())
				} else if ch == delim {
					// Two long string/bytes delimiter in a row
					ch, err = t.read()
					if err != nil {
						// EOF while parsing long string/bytes
						build.WriteRune(delim)
						build.WriteRune(delim)
						return t.eofLongLiteral(err, delim, isString, isBytes, isRaw, isFormat, build.String())
					} else if ch == delim {
						// Three long string/bytes delimiters in a row
						cpt := 0
						for build.String()[build.Len()-cpt-1] == '\\' {
							cpt++
						}
						if cpt%2 == 0 {
							// End of long string/bytes literal
							return t.endOfLiteral(isLong, isString, isBytes, isRaw, isFormat, build.String())
						} else {
							build.WriteRune(delim)
							build.WriteRune(delim)
							build.WriteRune(delim)
						}
					} else {
						// Two long string/bytes delimiter and something else
						build.WriteRune(delim)
						build.WriteRune(delim)
						build.WriteRune(ch)
					}
				} else {
					// One long string/bytes delimiter and something else
					build.WriteRune(delim)
					build.WriteRune(ch)
				}
			} else {
				// Not a delimiter, or escaped delimiter
				build.WriteRune(ch)
			}
		}
	} else {
		return t.endOfLiteral(isLong, isString, isBytes, isRaw, isFormat, build.String())
	}
}

func (t *Tokenizer) tokenIdentifier() error {
	var char rune
	var err error
	var buf strings.Builder

	// if EOL reached or char does not start an identifier: unread & return
	char, err = t.readUnread()
	if err != nil || !regexStartIdent.MatchString(string(char)) {
		return err
	}

	isString, startIdent, err := t.tokenStringBytesLiteral(false)
	if !isString && err != nil && err.Error() != "EOF" {
		return err
	} else if !isString {
		buf.WriteString(startIdent)
	} else if isString {
		return nil
	}

	// Fill buf with the next identifier characters
	for {
		char, err = t.read()
		if err != nil {
			break
		} else if !regexContIdent.MatchString(string(char)) {
			t.unread()
			break
		} else {
			buf.WriteRune(char)
		}
	}

	// Send TokenLit to channel
	var literal = buf.String()
	var token, ok = keywords[literal]
	if ok {
		t.Tokens <- TokenLit{Token: token}
	} else {
		t.Tokens <- TokenLit{IDENTIFIER, literal}
	}
	return nil
}

func (t *Tokenizer) tokenNumber() {
	var char rune
	var err error
	var buf = new(bytes.Buffer)

	// Fill buf with digits
	for {
		char, err = t.read()
		if err != nil {
			break
		} else if !regexDigit.MatchString(string(char)) {
			t.unread()
			break
		} else {
			buf.WriteRune(char)
		}
	}

	// Send TokenLit to channel
	t.Tokens <- TokenLit{INTEGER, buf.String()}
}

func (t *Tokenizer) tokenOther() error {
	char, err := t.read()
	if err != nil {
		t.unread()
		return nil
	}

	switch char {
	case '(':
		t.Tokens <- TokenLit{Token: PARENLEFT}
		t.embedLvl++
	case ')':
		t.Tokens <- TokenLit{Token: PARENRIGHT}
		t.embedLvl--
	case '[':
		t.Tokens <- TokenLit{Token: BRACKETLEFT}
		t.embedLvl++
	case ']':
		t.Tokens <- TokenLit{Token: BRACKETRIGHT}
		t.embedLvl--
	case '{':
		t.Tokens <- TokenLit{Token: BRACELEFT}
		t.embedLvl++
	case '}':
		t.Tokens <- TokenLit{Token: BRACERIGHT}
		t.embedLvl--
	case '\\':
		if !regexJoinLine.MatchString(t.lineText) {
			t.Tokens <- TokenLit{ILLEGAL, string(char)}
		}
	case '\'', '"':
		t.unread()
		isString, _, err := t.tokenStringBytesLiteral(false)
		if !isString && err != nil && err.Error() != "EOF" {
			return err
		} else if !isString {
			t.Tokens <- TokenLit{ILLEGAL, string(char)}
		}
	default:
		t.Tokens <- TokenLit{ILLEGAL, string(char)}
	}
	return nil
}

func (t *Tokenizer) tokenizeLine() error {
	if regexBlankLine.MatchString(t.lineText) || regexCommentLine.MatchString(t.lineText) {
		// Ignore blank lines and comment lines
		return nil
	}

	if t.longBuild != nil {
		t.tokenStringBytesLiteral(true)
	} else if !t.joinLines {
		t.tokenIndentation()
	}

	var char, err = t.readUnread()
	var charString string
	for err == nil {
		char, err = t.read()
		charString = string(char)
		t.unread()
		if regexWhitespace.MatchString(charString) {
			t.tokenWhitespace()
		} else if regexStartIdent.MatchString(charString) {
			panicError := t.tokenIdentifier()
			if panicError != nil {
				return panicError
			}
		} else if regexDigit.MatchString(charString) {
			t.tokenNumber()
		} else {
			panicError := t.tokenOther()
			if panicError != nil {
				return panicError
			}
		}
	}

	// Check if need to join lines
	if regexJoinLine.MatchString(t.lineText) || t.embedLvl > 0 {
		t.joinLines = true
	} else if t.embedLvl < 0 {
		t.embedLvl = 0
		t.joinLines = false
	} else if t.joinLines {
		t.joinLines = false
	}

	if !t.joinLines && t.longBuild == nil {
		t.Tokens <- TokenLit{Token: NEWLINE}
	}
	return nil
}

func (t *Tokenizer) Tokenize() {
	cpt := 0
	for t.scanner.Scan() {
		cpt++
		t.lineText = t.scanner.Text()
		t.line.Reset(t.lineText)
		err := t.tokenizeLine()
		if err != nil {
			panic(fmt.Sprintf("error parsing line %v: %v", cpt, err))
		}
	}
	t.Tokens <- TokenLit{Token: EOF}
}
