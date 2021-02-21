.PHONY: test run build

test:
	go test -race ./...

run:
	go run -race cmd/labns/main.go

build:
	go build -o /bin/main cmd/labns/main.gn