// clear;g++ -rdynamic -o t sample_test.cc -ldl
// Run as "LD_LIBRARY_PATH=.. ./t"
//
#include <stdlib.h>
#include <stdio.h>
#include <string>
#include <dlfcn.h>

using namespace std;

#define DLL_FILE "libhelpers.so"


int main()
{
    int (*sample_test_fn)(const char *src, const char *sdata);
    const char *sample_data = "{ \"TEST_INFO\": { \"Foo\": \"Bar\", \"Hello\" : \"World\"} }";
    char *error;
    
    void *helpers_so = dlopen(DLL_FILE, RTLD_NOW);
    if (!helpers_so) {
        fprintf(stderr, "%s\n", dlerror());
        exit(EXIT_FAILURE);
    }
    dlerror();    /* Clear any existing error */

    /*
     * Writing: cosine = (double (*)(double)) dlsym(handle, "cos");
     * would seem more natural, but the C99 standard leaves
     * casting from "void *" to a function pointer undefined.
     * The assignment used below is the POSIX.1-2003 (Technical
     * Corrigendum 1) workaround; see the Rationale for the
     * POSIX specification of dlsym(). 
     *    *(void **) (&cosine) = dlsym(handle, "cos");
     */

    *(void **)(&sample_test_fn) = dlsym(helpers_so, "sample_test");
    if ((error = dlerror()) != NULL)  {
        fprintf(stderr, "%s\n", error);
        exit(EXIT_FAILURE);
    }
 
    (*sample_test_fn)("Hello", sample_data);
    dlclose(helpers_so);
}


#if 0

ref: https://linux.die.net/man/3/dlopen

#include <stdio.h>
#include <stdlib.h>
#include <dlfcn.h>

int
main(int argc, char **argv)
{
    void *handle;
    double (*cosine)(double);
    char *error;

   handle = dlopen("libm.so", RTLD_LAZY);
    if (!handle) {
        fprintf(stderr, "%s\n", dlerror());
        exit(EXIT_FAILURE);
    }

   dlerror();    /* Clear any existing error */

   /* Writing: cosine = (double (*)(double)) dlsym(handle, "cos");
       would seem more natural, but the C99 standard leaves
       casting from "void *" to a function pointer undefined.
       The assignment used below is the POSIX.1-2003 (Technical
       Corrigendum 1) workaround; see the Rationale for the
       POSIX specification of dlsym(). */

   *(void **) (&cosine) = dlsym(handle, "cos");

   if ((error = dlerror()) != NULL)  {
        fprintf(stderr, "%s\n", error);
        exit(EXIT_FAILURE);
    }

   printf("%f\n", (*cosine)(2.0));
    dlclose(handle);
    exit(EXIT_SUCCESS);
}
If this program were in a file named "foo.c", you wou

If this program were in a file named "foo.c", you would build the program with the following command:
gcc -rdynamic -o foo foo.c -ldl
Libraries exporting _init() and _fini() will want to be compiled as follows, using bar.c as the example name:
gcc -shared -nostartfiles -o bar bar.c

typedef int (*some_func)(char *param);

void *myso = dlopen("/path/to/my.so", RTLD_NOW);
some_func *func = dlsym(myso, "function_name_to_fetch");
func("foo");
dlclose(myso);

#endif
