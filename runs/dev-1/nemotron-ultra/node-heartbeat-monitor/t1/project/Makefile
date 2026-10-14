.PHONY: build test run docker
build:
	go build -o node-heartbeat-monitor .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./node-heartbeat-monitor
docker:
	docker build -t node-heartbeat-monitor:latest .
