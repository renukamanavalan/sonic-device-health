common.cpp  lom_lib.cpp  subdir.mk  transport.cpp  transport.h
CC := g++

TEST_OBJS += ./lib/common.o ./lib/lom_lib.o ./lib/transport.o
OBJS += ./lib/common.o ./lib/lom_lib.o ./lib/transport.o

C_DEPS += ./lib/common.d ./lib/lom_lib.d ./lib/transport.d

src/%.o: src/%.cpp
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	$(CC) -D__FILENAME__="$(subst src/,,$<)" $(CFLAGS) -c -fmessage-length=0 -MMD -MP -MF"$(@:%.o=%.d)" -MT"$(@)" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '
