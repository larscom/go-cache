build:
	go mod download
	go build -v ./...

test:
	go test -timeout 5s -v ./.../ --race
