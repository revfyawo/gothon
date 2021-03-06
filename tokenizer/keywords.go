package tokenizer

var keywords = map[string]TokenValue{
	"False":    FALSE,
	"None":     NONE,
	"True":     TRUE,
	"and":      AND,
	"as":       AS,
	"assert":   ASSERT,
	"async":    ASYNC,
	"await":    AWAIT,
	"break":    BREAK,
	"class":    CLASS,
	"continue": CONTINUE,
	"def":      DEF,
	"del":      DEL,
	"elif":     ELIF,
	"else":     ELSE,
	"except":   EXCEPT,
	"finally":  FINALLY,
	"for":      FOR,
	"from":     FROM,
	"global":   GLOBAL,
	"if":       IF,
	"import":   IMPORT,
	"in":       IN,
	"is":       IS,
	"lambda":   LAMBDA,
	"nonlocal": NONLOCAL,
	"not":      NOT,
	"or":       OR,
	"pass":     PASS,
	"raise":    RAISE,
	"return":   RETURN,
	"try":      TRY,
	"while":    WHILE,
	"with":     WITH,
	"yield":    YIELD,
}
