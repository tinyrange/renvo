#include <layer.h>
#include <layer.h>

#define FEATURE_HEADER "feature.h"
#include FEATURE_HEADER

#define CAT_INNER(a, b) a ## b
#define CAT(a, b) CAT_INNER(a, b)
#define STRINGIZE(x) #x
#define APPLY(name, ...) CAT(api_, name)(__VA_ARGS__)
#define SELF SELF

#if defined(__x86_64__) && __STDC_VERSION__ >= 201112L && \
    (FORCED_VALUE * LAYER_VALUE + FEATURE_VALUE == 17)
int CAT(selected_, value) = FORCED_VALUE + LAYER_VALUE + FEATURE_VALUE;
#else
int wrong_branch;
#endif

const char *text = STRINGIZE(token + LAYER_VALUE);
APPLY(call, selected_value, __LINE__);

#line 200 "virtual-kernel.h"
int source_line = __LINE__;
const char *source_file = __FILE__;
SELF
