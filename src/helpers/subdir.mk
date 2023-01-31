CC := g++

HOBJS += ./helpers/sonic_helper.o
HTEST_OBJS += ./helpers/sample_test.o

C_DEPS += ./helpers/helper.d ./helpers/sample_test.d

includes = $(wildcard ./common/*.h)
includes += $(wildcard ./helpers/*.h)

# Use Wildchar to go through all vendors subdir and make all .so
#
helpers/%.o: helpers/%.cpp ${includes}
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	$(CC) -D__FILENAME__="$(subst helpers/,,$<)" -I../common $(CFLAGS) -c -fmessage-length=0 -MMD -MP -MF"$(@:%.o=%.d)" -MT"$(@)" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '

