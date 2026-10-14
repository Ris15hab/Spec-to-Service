.PHONY: build test run docker
build:
	go build -o config-validator .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./config-validator
docker:
	docker build -t config-validator:latest .
