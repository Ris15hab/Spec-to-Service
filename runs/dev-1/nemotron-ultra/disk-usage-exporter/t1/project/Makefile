.PHONY: build test run docker
build:
	go build -o disk-usage-exporter .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./disk-usage-exporter
docker:
	docker build -t disk-usage-exporter:latest .
