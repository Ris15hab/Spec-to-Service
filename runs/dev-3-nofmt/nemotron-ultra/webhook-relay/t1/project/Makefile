.PHONY: build test run docker
build:
	go build -o webhook-relay .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./webhook-relay
docker:
	docker build -t webhook-relay:latest .
