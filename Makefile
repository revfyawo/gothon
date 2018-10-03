PACKAGE=github.com/revfyawo/gothon

generate:
	go generate $(PACKAGE)/tokenizer

build: generate
	go build

run: generate
	go run $(PACKAGE) test.py
