
#ifdef __cplusplus
extern "C" {
#endif

/*
 * Init publisher at the start.
 * As init is async and can take time, it could make publish
 * take longer if it is done right before publish call.
 */
int lom_init_publish(void);

/*
 * Publish the JSON serialized object
 *
 * Expected format 
 * '{ "<publish tag>": { <key>: <val that can be serialized to string>, ... }}'
 *
 * e.g. "{ \"TEST_INFO\": { \"Foo\": \"Bar\", \"Hello\" : \"World\"} }"
 */
int lom_do_publish(const char *sdata);

/* close the handle. Gracious close goes long way. So please close at the end. */
void lom_deinit_publish(void);


/* Sample code for test purpose only */
int sample_test(const char *src, const char *sdata);

#ifdef __cplusplus
}
#endif

