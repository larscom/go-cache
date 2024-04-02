build:
	go mod download
	go build -v ./...

test:
	go test -timeout 5s -v ./.../ --race

coverage:
	go test -v -coverprofile=cover.out -covermode=atomic ./.../
	go tool cover -html=cover.out -o cover.html
