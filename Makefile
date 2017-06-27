SHELL := /bin/bash

SOURCES = $(filter-out %_test.go test_utils.go, $(wildcard *.go))
TEST_SOURCES=$(SOURCES) test_utils.go

VERSION = $(shell git rev-parse HEAD)
LDFLAGS = -ldflags "-X main.VERSION=$(VERSION)"

.PHONY: run
run:
	go run $(LDFLAGS) $(SOURCES)

.PHONY: test
test:
	go test -v -cover

# Runs a specific test suite
# supports a regex as argument, as long as it only matches one suite
.PHONY: test_%
test_%:
	@ SUITE=$$(if [ -f "$*_test.go" ]; then \
		echo "$*_test.go"; \
	else \
		FIND_RESULT=$$(find . -name "*$**_test\.go"); \
		[ -z "$$FIND_RESULT" ] && echo "No suite found with input '$*'" 1>&2 && exit 1; \
		NB_MACTHES=$$(echo "$$FIND_RESULT" | wc -l) && [[ $$NB_MACTHES != 1 ]] && echo -e "Found $$NB_MACTHES suites matching input:\n$$FIND_RESULT" 1>&2 && exit 1; \
		echo "$$FIND_RESULT"; \
	fi) && COMMAND="go test -v $$SUITE $(TEST_SOURCES)" && echo $$COMMAND && eval $$COMMAND;

.PHONY: build
build: get
	go build $(LDFLAGS) -o k9 $(SOURCES)

.PHONY: get
get:
	go get -t -d -v ./...
