.PHONY: build test run docker
build:
	go build -o feature-flag-service .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./feature-flag-service
docker:
	docker build -t feature-flag-service:latest .
