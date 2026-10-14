.PHONY: build test run docker
build:
	go build -o config-diff .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./config-diff
docker:
	docker build -t config-diff:latest .
