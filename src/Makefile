RM := rm -rf
LOM_TEST := tests/tests

CP := cp
MKDIR := mkdir
CC := g++
LIBS := -lpthread -lzmq -luuid

PWD := $(shell pwd)
CFLAGS += -Wall -std=c++17 -fPIE -I$(PWD)/common

ifneq ($(MAKECMDGOALS),clean)
ifneq ($(strip $(C_DEPS)),)
-include $(C_DEPS) $(OBJS)
endif
endif

-include lib/subdir.mk
-include tests/subdir.mk

all: lom-tests

lom-tests: $(TEST_OBJS)
	@echo 'Building target: $@'
	@echo 'Invoking: G++ Linker'
	$(CC) $(LDFLAGS) -o $(LOM_TEST) $(TEST_OBJS) $(LIBS)
	@echo 'Finished building target: $@'
	$(EVENTD_TEST)
	@echo 'Finished running tests'
	@echo ' '

.PHONY: all clean dependents
