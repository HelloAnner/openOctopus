PYTHON ?= python3

.PHONY: test e2e check

test:
	go test ./...

e2e:
	$(PYTHON) -m pytest e2e/config e2e/session e2e/eventbus e2e/orchestrator e2e/role-runtime e2e/artifact e2e/human-gate e2e/recovery e2e/cli -v

check: test
