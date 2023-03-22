build:
	go build -o bin/go-cache main.go

run: build
	./bin/go-cache

test:
	go test -v -cover ./.../ --race
