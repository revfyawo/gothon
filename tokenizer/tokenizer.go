package tokenizer

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type TokenValue int
type TokenLit struct {
	Token   TokenValue
	Literal string
}

var keywords = []string{
	"False", "await", "else", "import", "pass",
	"None", "break", "except", "in", "raise",
	"True", "class", "finally", "is", "return",
	"and", "continue", "for", "lambda", "try",
	"as", "def", "from", "nonlocal", "while",
	"assert", "del", "global", "not", "with",
	"async", "elif", "if", "or", "yield",
}

var regexBlankLine = regexp.MustCompile(`^\s*$`)
var regexCommentLine = regexp.MustCompile(`^\s*#.*$`)
var regexJoinLine = regexp.MustCompile(`^.*\\$`)
var regexWhitespace = regexp.MustCompile(`\s`)
var regexStartIdent = regexp.MustCompile(`[a-zA-Z_]`)
var regexContIdent = regexp.MustCompile(`[a-zA-Z0-9_]`)
var regexDigit = regexp.MustCompile(`\d`)

type Tokenizer struct {
	scanner     *bufio.Scanner
	line        *strings.Reader
	lineText    string
	indentStack []int
	joinLines   bool
	embedLvl    int
	Tokens      chan TokenLit
}

func New(f *os.File) (t *Tokenizer) {
	return &Tokenizer{
		scanner:     bufio.NewScanner(f),
		line:        strings.NewReader(""),
		indentStack: []int{0},
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
	return
}

func (t *Tokenizer) tokenIdentifier() {
	var char rune
	var err error
	var buf strings.Builder

	// Add first rune to buf if it starts an identifier
	// otherwise unread & return
	char, err = t.read()
	if err != nil || !regexStartIdent.MatchString(string(char)) {
		fmt.Printf("err, match: %v, %v", err, !regexStartIdent.MatchString(string(char)))
		t.unread()
		return
	}
	buf.WriteRune(char)

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
	t.Tokens <- TokenLit{IDENTIFIER, buf.String()}
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

func (t *Tokenizer) tokenOther() {
	char, err := t.read()
	if err != nil {
		t.unread()
		return
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
	default:
		t.Tokens <- TokenLit{ILLEGAL, string(char)}
	}
}

func (t *Tokenizer) tokenizeLine() {
	if regexBlankLine.MatchString(t.lineText) || regexCommentLine.MatchString(t.lineText) {
		// Ignore blank lines and comment lines
		return
	}

	var char, err = t.readUnread()
	var charString = string(char)
	for err == nil {
		char, err = t.read()
		charString = string(char)
		t.unread()
		if regexWhitespace.MatchString(charString) {
			t.tokenWhitespace()
		} else if regexStartIdent.MatchString(charString) {
			t.tokenIdentifier()
		} else if regexDigit.MatchString(charString) {
			t.tokenNumber()
		} else {
			t.tokenOther()
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

	if !t.joinLines {
		t.Tokens <- TokenLit{Token: NEWLINE}
	}
}

func (t *Tokenizer) Tokenize() {
	for t.scanner.Scan() {
		t.lineText = t.scanner.Text()
		t.line.Reset(t.lineText)
		t.tokenizeLine()
	}
	t.Tokens <- TokenLit{Token: EOF}
}
