PYTHON ?= python3

.PHONY: test e2e check

test:
	go test ./...

e2e:
	$(PYTHON) -m pytest e2e/config -v

check: test e2e
