RM := rm -rf

CP := cp
MKDIR := mkdir
CC := g++
LIBS := -lpthread -luuid
EVENTS_LIBS := -levent -lhiredis -lswsscommon -lpthread -lboost_thread -lboost_system -lzmq -lboost_serialization -luuid -llua5.1
LOM_LIB := liblom_lib.so
LOM_TEST := tests/tests
LOM_FILE := ./tests/test_data_ut.json

HELPER_LIB := libhelpers.so
HELPER_TEST := helpers/tests

PWD := $(shell pwd)
CFLAGS += -fPIC -Wall -std=c++17 -I$(PWD)/common

ifneq ($(MAKECMDGOALS),clean)
ifneq ($(strip $(C_DEPS)),)
-include $(C_DEPS) $(OBJS)
endif
endif

-include lib/subdir.mk
-include tests/subdir.mk
-include helpers/subdir.mk

all: $(LOM_TEST) $(HELPER_LIB) $(HELPER_TEST)

$(LOM_TEST): $(TEST_OBJS) $(LOM_LIB)
	@echo 'Building target: $@'
	@echo 'Invoking: G++ Linker'
	$(CC) $(LDFLAGS) -Wl,-R -Wl,./ -o $(LOM_TEST) $(TEST_OBJS) $(LOM_LIB) $(LIBS)
	@echo 'Finished building target: $@'
	$(LOM_TEST) $(LOM_FILE)
	@echo 'Finished running tests'
	@echo ' '

$(LOM_LIB): $(OBJS)
	@echo 'Building target: $@'
	@echo 'Invoking: G++ Linker'
	$(CC) -fPIC $(LDFLAGS) -shared -o $(LOM_LIB) $(OBJS) $(LIBS)
	@echo 'Finished building target: $@'
	@echo ' '

$(HELPER_LIB): $(HOBJS)
	@echo 'Building target: $@'
	@echo 'Invoking: G++ Linker'
	$(CC) -fPIC $(LDFLAGS) -shared -o $@ $(HOBJS) $(LIBS) $(EVENTS_LIBS)
	@echo 'Finished building target: $@'
	@echo ' '

$(HELPER_TEST): $(HTEST_OBJS)
	@echo 'Building target: $@'
	@echo 'Invoking: G++ Linker'
	$(CC) $(LDFLAGS) -Wl,-R -Wl,./ -o $(HELPER_TEST) $(HTEST_OBJS) $(LIBS) -ldl
	@echo 'Finished building target: $@'
	LD_LIBRARY_PATH=. $(HELPER_TEST) $(LOM_FILE)
	@echo 'Finished running tests'
	@echo ' '

install:
	$(MKDIR) -p $(DESTDIR)/usr/lib
	$(CP) $(LOM_LIB) $(DESTDIR)/usr/lib

deinstall:
	$(RM) -f $(DESTDIR)/usr/lib/$(LOM_LIB)

clean:
	-$(RM) $(LOM_LIB) $(OBJS) $(TEST_OBJS) $(C_DEPS) $(LOM_TEST) $(HOBJS) $(HTEST_OBJS)
	-$(RM) $(HELPER_TEST)
	-@echo ' '

.PHONY: all clean dependents
