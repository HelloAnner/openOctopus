PYTHON ?= python3
BIN_NAME ?= openoctopus
INSTALL_PATH ?= $(HOME)/.local/bin

.PHONY: test e2e check install build

test:
	go test ./...

e2e:
	$(PYTHON) -m pytest e2e/config e2e/session e2e/eventbus e2e/orchestrator e2e/role-runtime e2e/artifact e2e/human-gate e2e/recovery e2e/cli e2e/tmux -v

build:
	go build -o $(BIN_NAME) ./cmd/openoctopus

install: build
	mkdir -p $(INSTALL_PATH)
	rm -f $(INSTALL_PATH)/.$(BIN_NAME).tmp
	cp $(BIN_NAME) $(INSTALL_PATH)/.$(BIN_NAME).tmp
	mv -f $(INSTALL_PATH)/.$(BIN_NAME).tmp $(INSTALL_PATH)/$(BIN_NAME)
	rm $(BIN_NAME)

check: test
