.PHONY: build test run docker
build:
	go build -o gpu-job-watcher .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./gpu-job-watcher
docker:
	docker build -t gpu-job-watcher:latest .
