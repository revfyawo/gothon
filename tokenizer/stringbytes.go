package tokenizer

import (
	"errors"
)

func isQuote(r rune) bool {
	return r == '\'' || r == '"'
}

func (t *Tokenizer) eofLongLiteral(err error, delim rune, isString, isBytes, isRaw, isFormat bool, literal string) (bool, string, error) {
	// EOF while parsing long string/bytes
	// Value spans on multiple lines
	literal += "\n"
	t.longDelim = delim
	if isString {
		if isRaw && isFormat {
			t.longBuild = &TokenLit{LONGRAWFSTRING, literal}
		} else if isRaw {
			t.longBuild = &TokenLit{LONGRAWSTRING, literal}
		} else if isFormat {
			t.longBuild = &TokenLit{LONGFSTRING, literal}
		} else {
			t.longBuild = &TokenLit{LONGSTRING, literal}
		}
	} else if isBytes {
		if isRaw {
			t.longBuild = &TokenLit{LONGRAWBYTES, literal}
		} else {
			t.longBuild = &TokenLit{LONGBYTES, literal}
		}
	} else {
		return false, "", errors.New("token is neither a string nor a bytes")
	}
	return true, "", err
}

func (t *Tokenizer) endOfLiteral(isLong, isString, isBytes, isRaw, isFormat bool, literal string) (bool, string, error) {
	if !isLong {
		if isString {
			if isRaw && isFormat {
				t.Tokens <- TokenLit{RAWFSTRING, literal}
			} else if isRaw {
				t.Tokens <- TokenLit{RAWSTRING, literal}
			} else if isFormat {
				t.Tokens <- TokenLit{FSTRING, literal}
			} else {
				t.Tokens <- TokenLit{STRING, literal}
			}
		} else if isBytes {
			if isRaw {
				t.Tokens <- TokenLit{RAWBYTES, literal}
			} else {
				t.Tokens <- TokenLit{BYTES, literal}
			}
		} else {
			return false, "", errors.New("token is neither a string nor a bytes")
		}
		return true, "", nil
	} else {
		t.longBuild = nil
		t.longDelim = 0
		if isString {
			if isRaw && isFormat {
				t.Tokens <- TokenLit{LONGRAWFSTRING, literal}
			} else if isRaw {
				t.Tokens <- TokenLit{LONGRAWSTRING, literal}
			} else if isFormat {
				t.Tokens <- TokenLit{LONGFSTRING, literal}
			} else {
				t.Tokens <- TokenLit{LONGSTRING, literal}
			}
		} else if isBytes {
			if isRaw {
				t.Tokens <- TokenLit{LONGRAWBYTES, literal}
			} else {
				t.Tokens <- TokenLit{LONGBYTES, literal}
			}
		} else {
			return false, "", errors.New("token is neither a string nor a bytes")
		}
		return true, "", nil
	}
}
