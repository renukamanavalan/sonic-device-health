# Select bash for commands
.ONESHELL:
SHELL = /bin/bash
.SHELLFLAGS += -e

ENGINE_TARGET := lom/build/bin/LoMEngine
PLMGR_TARGET := lom/build/bin/pluginmgr
ENGINE_CONFIG := config/*.conf.json

MKDIR := mkdir
CP := cp
RM := rm

ifeq ($(DESTDIR),)
	override DESTDIR := /lom-root/debian/tmp
endif

%::
	@echo "+++ --- Making Go --- +++"
	cd lom && $(MAKE) -f Makefile $@

# e.g. make -j16 install DESTDIR=/sonic/src/sonic-eventd/debian/tmp AM_UPDATE_INFO_DIR=no "INSTALL=install --strip-program=true"
install:
	@echo 'install: Destdir:$(DESTDIR)'
	$(RM) -rf $(DESTDIR)
	$(MKDIR) -p $(DESTDIR)/usr/bin
	$(MKDIR) -p $(DESTDIR)/usr/share/lom
	$(CP) $(ENGINE_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(PLMGR_TARGET) $(DESTDIR)/usr/bin
	$(CP) $(ENGINE_CONFIG) $(DESTDIR)/usr/share/lom/

