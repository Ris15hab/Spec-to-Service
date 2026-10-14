.PHONY: build test run docker
build:
	go build -o job-queue-api .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./job-queue-api
docker:
	docker build -t job-queue-api:latest .
