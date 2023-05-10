GO := /usr/local/go/bin/go
INSTALL := /usr/bin/install

BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin
TEST_DIR := $(BUILD_DIR)/test

# Set to -v for verbose o/p. May set it in Env before calling Make
# export TEST_FLAG=-v in shell before calling make or
# TEST_FLAG=-v make all

export GO111MODULE=on

SRC_FILES=$(shell find . -name '*.go' | grep -v '_test.go' | grep -v '/tests/')
TEST_FILES=$(wildcard *_test.go)

# Binaries used in LoM system
ENGINE_BIN := $(BIN_DIR)/LoMEngine
PLUGIN_MANAGER_BIN := $(BIN_DIR)/pluginmgr

# Test binaries/flags
LIB_TEST = $(TEST_DIR)/LibTested
ENGINE_TEST = $(TEST_DIR)/EngineTested
PLUGIN_MANAGER_TEST = $(TEST_DIR)/PluginManagerTested
SONIC_DBCLIENT_TEST =$(TEST_DIR)/SonicDbClientTested

format_code:
	@if which gofmt > /dev/null; then \
		gofmt -l ./ | xargs -I {} gofmt -w {}; \
	else \
		echo "gofmt command not found, skipping code formatting"; \
	fi
	@if which sed > /dev/null; then \
		sed -i 's/\t/    /g' $$(find ./ -type f -name '*.go'); \
	else \
		echo "sed command not found, skipping code formatting"; \
	fi

.PHONY: pluginmgr lib all clean

all: | format_code $(ENGINE_BIN) $(PLUGIN_MANAGER_BIN) $(LIB_TEST) $(ENGINE_TEST) $(PLUGIN_MANAGER_TEST) $(SONIC_DBCLIENT_TEST)

pluginmgr: export TEST_FLAG=-v -p 1 -cover
pluginmgr:  | format_code clean $(PLUGIN_MANAGER_BIN) $(PLUGIN_MANAGER_TEST)

lib: export TEST_FLAG=-v -p 1 -cover
lib: clean $(SRC_FILES) $(LIB_TEST)

$(ENGINE_BIN): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(BIN_DIR)
	$(GO) build  -o $@ lom/src/engine
	$(info "Built lom-engine")

$(PLUGIN_MANAGER_BIN): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(BIN_DIR)
	$(GO) build -o $@ lom/src/pluginmgr
	$(info "Built plugin manager")


$(LIB_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/LibCoverprofile.out \
		-coverpkg lom/src/lib/lomipc,lom/src/lib/lomcommon \
		-covermode=atomic ./src/lib/lib_test || (echo "LibTest failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/LibCoverprofile.out -o $(TEST_DIR)/LibCoverprofile.html
	touch $@

$(ENGINE_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/EngineCoverprofile.out \
		-coverpkg lom/src/engine \
		-covermode=atomic lom/src/engine || (echo "EngineTest failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/EngineCoverprofile.out -o $(TEST_DIR)/EngineCoverprofile.html
	touch $@

$(PLUGIN_MANAGER_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/PluginManagerCoverprofile.out \
		-coverpkg lom/src/pluginmgr_common \
		-coverpkg lom/src/plugins/plugins_common \
		-covermode=atomic ./src/pluginmgr/pluginmgr_test  || (echo "Plugin manager test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/PluginManagerCoverprofile.out -o $(TEST_DIR)/PluginManagerCoverprofile.html
	touch $@

$(SONIC_DBCLIENT_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/SonicDbClientCoverprofile.out \
		-coverpkg lom/src/plugins/sonic/client/dbclient \
		-covermode=atomic lom/src/plugins/sonic/client/dbclient || (echo "SonicDbClient Test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/SonicDbClientCoverprofile.out -o $(TEST_DIR)/SonicDbClientCoverprofile.html
	touch $@

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(PLUGIN_MANAGER_BIN)
