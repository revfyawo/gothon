package main

import (
	"fmt"
	"github.com/revfyawo/gothon/tokenizer"
	"os"
)

func main() {
	file, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	t := tokenizer.New(file)
	go t.Tokenize()
	for toklit := range t.Tokens {
		fmt.Printf("Received %v\n", toklit)
		if toklit.Token == tokenizer.EOF {
			break
		}
	}
}
