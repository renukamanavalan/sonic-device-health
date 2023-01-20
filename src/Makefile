RM := rm -rf
LOM_TEST := tests/tests

CP := cp
MKDIR := mkdir
CC := g++
LIBS := -lpthread -luuid
LOM_LIB := lom_lib.so
LOM_TEST := tests/tests
LOM_FILE := ./tests/test_data_ut.json

PWD := $(shell pwd)
CFLAGS += -fPIC -Wall -std=c++17 -I$(PWD)/common

ifneq ($(MAKECMDGOALS),clean)
ifneq ($(strip $(C_DEPS)),)
-include $(C_DEPS) $(OBJS)
endif
endif

-include lib/subdir.mk
-include tests/subdir.mk

all: lom-tests

lom-tests: $(TEST_OBJS) $(LOM_LIB)
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

install:
	$(MKDIR) -p $(DESTDIR)/usr/bin
	$(CP) $(LOM_TEST) $(DESTDIR)/usr/bin

deinstall:
	$(RM) -rf $(DESTDIR)/usr

clean:
	-@echo ' '

.PHONY: all clean dependents
