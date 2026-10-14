.PHONY: build test run docker
build:
	go build -o cluster-inventory-api .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./cluster-inventory-api
docker:
	docker build -t cluster-inventory-api:latest .
