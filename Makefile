# Select bash for commands
.ONESHELL:
SHELL = /bin/bash
.SHELLFLAGS += -e

GO_BUILD_DIR := lom/build
ENGINE_TARGET := $(GO_BUILD_DIR)/bin/LoMEngine
PLMGR_TARGET := $(GO_BUILD_DIR)/bin/LoMPluginMgr
CLI_TARGET := $(GO_BUILD_DIR)/bin/LoMCli
CMN_C_LIB := $(GO_BUILD_DIR)/lib/cmn_c_lib.so
CONFIG_DIR := config
LOM_CONFIG := $(CONFIG_DIR)/*.json
VERSION_CONFIG := $(CONFIG_DIR)/LoM-Version.json
gNMI_SERVER_TARGET := $(GO_BUILD_DIR)/bin/LoMgNMIServer
gNMI_CLI_TARGET := $(GO_BUILD_DIR)/bin/gnmi_cli
LOM_PLUGIN_SCRIPTS := lom/src/plugins/plugins_files/ScriptBasedPlugin_scripts/*

MKDIR := mkdir
CP := cp
RM := rm

ifdef SONIC_OS_VERSION
VENDOR = sonic
endif

ifeq ($(VENDOR),)
$(error Specify vendor as sonic/arista/cisco!)
endif

VERSION_SRC_FILE = $(CONFIG_DIR)/$(VENDOR)/VersionSrc
VERSION_TEMPLATE_FILE = $(CONFIG_DIR)/LoM-Version.json.j2


ifeq ("$(wildcard $(VERSION_SRC_FILE))","")
$(error Expect $(VERSION_SRC_FILE) to exist. Refer VersionSrc.sample for details!)
endif

all: go-all $(VERSION_CONFIG)

go-all:
	@echo "+++ --- Making Go --- +++"
	pushd lom
	$(MAKE) -f Makefile all
	popd
	@echo "+++ --- Making Go DONE --- +++"

# Generate conf files
$(VERSION_CONFIG): $(CONFIG_DIR)/LoM-Version.json.j2 
	@echo "+++ --- Creating Version JSON $(HOST_OS_VERSION)--- +++"
	$(VERSION_SRC_FILE) $(VERSION_TEMPLATE_FILE) $(VERSION_CONFIG)
	@echo "+++ --- Creating Version JSON DONE --- +++"


# e.g. make -j16 install DESTDIR=/sonic/src/sonic-eventd/debian/tmp
#            AM_UPDATE_INFO_DIR=no "INSTALL=install --strip-program=true"
#
install:
	@echo 'install: Destdir:$(DESTDIR)'
	$(RM) -rf $(DESTDIR)
	$(MKDIR) -p $(DESTDIR)/usr/bin
	$(MKDIR) -p $(DESTDIR)/test-bin
	$(MKDIR) -p $(DESTDIR)/usr/lib
	$(MKDIR) -p $(DESTDIR)/usr/share/lom/scripts
	$(CP) $(ENGINE_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(PLMGR_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(CLI_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(CMN_C_LIB) $(DESTDIR)/usr/lib
	$(CP) $(LOM_CONFIG) $(DESTDIR)/usr/share/lom/
	$(CP) $(LOM_PLUGIN_SCRIPTS) $(DESTDIR)/usr/share/lom/scripts
	$(CP) $(VERSION_CONFIG) $(DESTDIR)/usr/share/lom/
	$(CP) $(gNMI_SERVER_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(gNMI_CLI_TARGET) $(DESTDIR)/usr/bin
	$(CP) LoM-Install/${VENDOR}/LoM-install.sh $(DESTDIR)/usr/share/lom
	$(CP) LoM-Install/${VENDOR}/common.sh $(DESTDIR)/usr/share/lom


deinstall:
	$(RM) -rf $(DESTDIR)/usr
	$(RM) -rf $(DESTDIR)/share/lom

clean:
	pushd lom
	$(MAKE) -f Makefile $@
	popd

