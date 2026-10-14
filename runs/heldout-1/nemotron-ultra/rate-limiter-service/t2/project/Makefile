.PHONY: build test run docker
build:
	go build -o rate-limiter-service .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./rate-limiter-service
docker:
	docker build -t rate-limiter-service:latest .
