.PHONY: build run clean

build:
	go build -o bwtui .

run: build
	./bwtui

clean:
	rm -f bwtui
