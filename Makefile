build:
	go build

run: build
	./gothon test.py
