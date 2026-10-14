.PHONY: build test run docker
build:
	go build -o alert-rule-evaluator .
test:
	go test ./... -count=1
run: build
	PORT=8080 ./alert-rule-evaluator
docker:
	docker build -t alert-rule-evaluator:latest .
