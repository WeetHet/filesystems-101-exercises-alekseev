.PHONY: build
build:
	go build -o ./bin/tests-runner -v -trimpath -tags 'postgres,nogrpctracing' ./cmd/tests-runner
