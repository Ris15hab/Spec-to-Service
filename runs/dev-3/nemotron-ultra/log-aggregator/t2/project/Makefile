.PHONY: build test run docker
build:
	go build -o log-aggregator .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./log-aggregator
docker:
	docker build -t log-aggregator:latest .
