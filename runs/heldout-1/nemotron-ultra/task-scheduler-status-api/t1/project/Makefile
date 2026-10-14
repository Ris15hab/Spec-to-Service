.PHONY: build test run docker
build:
	go build -o task-scheduler-status-api .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./task-scheduler-status-api
docker:
	docker build -t task-scheduler-status-api:latest .
