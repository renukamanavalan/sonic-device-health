# Select bash for commands
.ONESHELL:
SHELL = /bin/bash
.SHELLFLAGS += -e
USER = $(shell id -un)
UID = $(shell id -u)
GUID = $(shell id -g)


GO := /usr/local/go1.20.3/go/bin/go
GOFMT := $(GO)fmt
INSTALL := /usr/bin/install

BUILD_DIR := build
LIB_DIR := $(BUILD_DIR)/lib
BIN_DIR := $(BUILD_DIR)/bin
TEST_DIR := $(BUILD_DIR)/test

# Set to -v for verbose o/p. May set it in Env before calling Make
# export TEST_FLAG=-v in shell before calling make or
# TEST_FLAG=-v make all

# TEST_FLAG := "-v" Set it for verbose test o/p

ifeq "${TEST_FLAG}" "-v"
PYTEST_VERBOSE= --addopts "-s"
endif

export GO111MODULE=on
export LOM_LIB_PATH=$(PWD)/build/lib

SRC_FILES=$(shell find . -name '*.go' | grep -v '_test.go' | grep -v '/tests/')
PY_FILES=$(shell find . -name '*.py' | grep -v '_test.py' | grep -v '/tests/')
TEST_FILES=$(shell find . -name '*test.go')

CMN_C_LIB_FILES=src/lib/clib/lom_clib.go
CMN_C_LIB=$(BUILD_DIR)/lib/cmn_c_lib.so

# Binaries used in LoM system
ENGINE_BIN := $(BIN_DIR)/LoMEngine
PLUGIN_MANAGER_BIN := $(BIN_DIR)/LoMPluginMgr

# Test binaries/flags
LIB_TEST = $(TEST_DIR)/LibTested
ENGINE_TEST = $(TEST_DIR)/EngineTested
PLUGIN_MANAGER_TEST = $(TEST_DIR)/PluginManagerTested
SONIC_DBCLIENT_TEST =$(TEST_DIR)/SonicDbClientTested
PY_TEST = $(TEST_DIR)/pytested
SONIC_LINK_CRC_TEST = $(TEST_DIR)/SonicLinkCrcTested
PLUGINS_COMMON_TEST = $(TEST_DIR)/PluginsCommonTested

.NOTPARALLEL:

format_code:
	@if ! which sed > /dev/null; then \
		echo "Installing sed..."; \
		apt-get update && apt-get install -y sed; \
	fi
	@$(GOFMT) -l ./ | xargs -I {} $(GOFMT) -w {};
	@sed -i 's/\t/    /g' $$(find ./ -type f -name '*.go');

# Define the file coverage extraction function
define extract_file_coverage
	@echo "*******************************************************"
	@echo "Extracting file coverages:"
	@file_coverage="$$(sed -n 's/.*<option value="[^"]\+">\([^<]\+\).* (\([^)]\+\%.*\)<\/option>.*/\1 (\2)/p' $(1))"; 
	echo "$${file_coverage}"
	@echo "*******************************************************"
	@echo ""
	@echo ""
endef

.PHONY: all clean format_code

all: format_code $(CMN_C_LIB) $(ENGINE_BIN) $(LIB_TEST) $(ENGINE_TEST) $(PLUGIN_MANAGER_TEST) \
	$(SONIC_DBCLIENT_TEST) $(SONIC_LINK_CRC_TEST) $(PLUGINS_COMMON_TEST) $(PY_TEST) \
	$(PLUGIN_MANAGER_BIN)


$(CMN_C_LIB): $(SRC_FILES)
	mkdir -p $(LIB_DIR)
	$(GO) build -buildmode=c-shared -o $@ $(CMN_C_LIB_FILES)
	$(info "Built $(CMN_C_LIB)")

$(ENGINE_BIN): $(SRC_FILES)
	mkdir -p $(BIN_DIR)
	$(GO) build  -o $@ lom/src/engine
	$(info "Built lom-engine")

$(PLUGIN_MANAGER_BIN): $(SRC_FILES)
	mkdir -p $(BIN_DIR)
	$(GO) build -o $@ lom/src/pluginmgr
	$(info "Built plugin manager")


$(LIB_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test -p 1 $(TEST_FLAG) -coverprofile=$(TEST_DIR)/LibCoverprofile.out \
		-coverpkg lom/src/lib/lomipc,lom/src/lib/lomcommon \
		-covermode=atomic -v ./src/lib/lib_test ./src/lib/lib_test/config || (echo "LibTest failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/LibCoverprofile.out -o $(TEST_DIR)/LibCoverprofile.html
	$(call extract_file_coverage,$(TEST_DIR)/LibCoverprofile.html)
	touch $@

$(ENGINE_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/EngineCoverprofile.out \
		-coverpkg lom/src/engine \
		-covermode=atomic -v lom/src/engine || (echo "EngineTest failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/EngineCoverprofile.out -o $(TEST_DIR)/EngineCoverprofile.html
	$(call extract_file_coverage,$(TEST_DIR)/EngineCoverprofile.html)
	touch $@

$(PLUGIN_MANAGER_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test -p 1 $(TEST_FLAG) -coverprofile=$(TEST_DIR)/PluginManagerCoverprofile.out \
		-coverpkg lom/src/pluginmgr/pluginmgr_common \
		-covermode=atomic -v ./src/pluginmgr/pluginmgr_common  || (echo "Plugin manager test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/PluginManagerCoverprofile.out -o $(TEST_DIR)/PluginManagerCoverprofile.html
	$(call extract_file_coverage,$(TEST_DIR)/PluginManagerCoverprofile.html)
	touch $@

$(SONIC_DBCLIENT_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/SonicDbClientCoverprofile.out \
		-coverpkg lom/src/plugins/sonic/client/dbclient \
		-covermode=atomic -v lom/src/plugins/sonic/client/dbclient || (echo "SonicDbClient Test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/SonicDbClientCoverprofile.out -o $(TEST_DIR)/SonicDbClientCoverprofile.html
	$(call extract_file_coverage,$(TEST_DIR)/SonicDbClientCoverprofile.html)
	touch $@

$(PY_TEST): $(SRC_FILES) $(PY_FILES)
	mkdir -p $(TEST_DIR)
	pushd python; \
	python3 setup.py test $(PYTEST_VERBOSE); \
	popd

$(SONIC_LINK_CRC_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/SonicLinkCrcCoverprofile.out \
				-coverpkg lom/src/plugins/sonic/plugin/linkcrc \
				-covermode=atomic -v lom/src/plugins/sonic/plugin/linkcrc || (echo "SonicLinkCrc plugin Test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/SonicLinkCrcCoverprofile.out -o $(TEST_DIR)/SonicLinkCrcCoverprofile.html
	$(call extract_file_coverage,$(TEST_DIR)/SonicLinkCrcCoverprofile.html)
	touch $@
	
$(PLUGINS_COMMON_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/PluginsCommonCoverprofile.out \
		-coverpkg lom/src/plugins/plugins_common \
				-covermode=atomic -v lom/src/plugins/plugins_common || (echo "PluginsCommon Tests failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/PluginsCommonCoverprofile.out -o $(TEST_DIR)/PluginsCommonCoverprofile.html
	$(call extract_file_coverage,$(TEST_DIR)/PluginsCommonCoverprofile.html)
	touch $@

clean:
	rm -rf $(BUILD_DIR)
