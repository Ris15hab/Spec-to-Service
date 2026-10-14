.PHONY: build test run docker
build:
	go build -o artifact-registry-metadata .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./artifact-registry-metadata
docker:
	docker build -t artifact-registry-metadata:latest .
