# Select bash for commands
.ONESHELL:
SHELL = /bin/bash
.SHELLFLAGS += -e

GO_BUILD_DIR := lom/build
ENGINE_TARGET := $(GO_BUILD_DIR)/bin/LoMEngine
PLMGR_TARGET := $(GO_BUILD_DIR)/bin/LoMPluginMgr
CLI_TARGET := $(GO_BUILD_DIR)/bin/LoMCli
CMN_C_LIB := $(GO_BUILD_DIR)/lib/cmn_c_lib.so
ENGINE_CONFIG := config/*.conf.json
VERSION_FILE := LoM-Version

MKDIR := mkdir
CP := cp
RM := rm

all: go-all

go-all:
	@echo "+++ --- Making Go --- +++"
	cd lom && $(MAKE) -f Makefile all

# e.g. make -j16 install DESTDIR=/sonic/src/sonic-eventd/debian/tmp
#            AM_UPDATE_INFO_DIR=no "INSTALL=install --strip-program=true"
#
install:
	@echo 'install: Destdir:$(DESTDIR)'
	$(RM) -rf $(DESTDIR)
	$(MKDIR) -p $(DESTDIR)/usr/bin
	$(MKDIR) -p $(DESTDIR)/test-bin
	$(MKDIR) -p $(DESTDIR)/usr/lib
	$(MKDIR) -p $(DESTDIR)/usr/share/lom
	$(CP) $(ENGINE_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(PLMGR_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(CLI_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(CMN_C_LIB) $(DESTDIR)/usr/lib
	$(CP) $(ENGINE_CONFIG) $(DESTDIR)/usr/share/lom/
	$(CP) $(VERSION_FILE) $(DESTDIR)/usr/share/lom/


deinstall:
	$(RM) -rf $(DESTDIR)/usr
	$(RM) -rf $(DESTDIR)/share/lom

clean:
	cd lom && $(MAKE) -f Makefile $@
