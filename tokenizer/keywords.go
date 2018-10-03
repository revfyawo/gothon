package tokenizer

var keywords = []string{
	"False", "await", "else", "import", "pass",
	"None", "break", "except", "in", "raise",
	"True", "class", "finally", "is", "return",
	"and", "continue", "for", "lambda", "try",
	"as", "def", "from", "nonlocal", "while",
	"assert", "del", "global", "not", "with",
	"async", "elif", "if", "or", "yield",
}

func checkKeyword(literal string) bool {
	for _, kw := range keywords {
		if literal == kw {
			return true
		}
	}
	return false
}
