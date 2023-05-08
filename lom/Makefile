# Select bash for commands
.ONESHELL:
SHELL = /bin/bash
.SHELLFLAGS += -e
USER = $(shell id -un)
UID = $(shell id -u)
GUID = $(shell id -g)


GO := /usr/local/go/bin/go
INSTALL := /usr/bin/install

BUILD_DIR := build
LIB_DIR := $(BUILD_DIR)/lib
BIN_DIR := $(BUILD_DIR)/bin
TEST_DIR := $(BUILD_DIR)/test

# Set to -v for verbose o/p. May set it in Env before calling Make
# export TEST_FLAG=-v in shell before calling make or
# TEST_FLAG=-v make all

ifeq "${TEST_FLAG}" "-v"
PYTEST_VERBOSE= --addopts "-s"
endif

export GO111MODULE=on
export LOM_LIB_PATH=/lom-root/lom/build/lib

SRC_FILES=$(shell find . -name '*.go' | grep -v '_test.go' | grep -v '/tests/')
PY_FILES=$(shell find . -name '*.py' | grep -v '_test.py' | grep -v '/tests/')
TEST_FILES=$(wildcard *_test.go)

CMN_C_LIB_FILES=src/lib/clib/lom_clib.go
CMN_C_LIB=$(BUILD_DIR)/lib/cmn_c_lib.so

# Binaries used in LoM system
ENGINE_BIN := $(BIN_DIR)/LoMEngine


# Test binaries/flags
LIB_TEST = $(TEST_DIR)/LibTested
ENGINE_TEST = $(TEST_DIR)/EngineTested
SONIC_DBCLIENT_TEST =$(TEST_DIR)/SonicDbClientTested
PY_TEST = $(TEST_DIR)/pytested

all: $(CMN_C_LIB) $(ENGINE_BIN) $(LIB_TEST) $(ENGINE_TEST) $(SONIC_DBCLIENT_TEST) $(PY_TEST)

$(CMN_C_LIB): $(SRC_FILES)
	mkdir -p $(LIB_DIR)
	$(GO) build -buildmode=c-shared -o $@ $(CMN_C_LIB_FILES)
	$(info "Built $(CMN_C_LIB)")

$(ENGINE_BIN): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(BIN_DIR)
	$(GO) build  -o $@ lom/src/engine
	$(info "Built lom-engine")


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

$(SONIC_DBCLIENT_TEST): $(SRC_FILES) $(TEST_FILES)
	mkdir -p $(TEST_DIR)
	${GO} test $(TEST_FLAG) -coverprofile=$(TEST_DIR)/SonicDbClientCoverprofile.out \
		-coverpkg lom/src/plugins/sonic/client/dbclient \
		-covermode=atomic lom/src/plugins/sonic/client/dbclient || (echo "SonicDbClient Test failed $$?"; exit 1)
	${GO} tool cover -html=$(TEST_DIR)/SonicDbClientCoverprofile.out -o $(TEST_DIR)/SonicDbClientCoverprofile.html
	touch $@

$(PY_TEST): $(SRC_FILES) $(PY_FILES)
	mkdir -p $(TEST_DIR)
	pushd /lom-root/lom/python; \
	python3 setup.py test $(PYTEST_VERBOSE); \
    popd

clean:
	rm -rf $(BUILD_DIR)
