package tokenizer

func checkKeyword(literal string) bool {
	for _, kw := range keywords {
		if literal == kw {
			return true
		}
	}
	return false
}
