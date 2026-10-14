.PHONY: build test run docker
build:
	go build -o ttl-cache-service .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./ttl-cache-service
docker:
	docker build -t ttl-cache-service:latest .
