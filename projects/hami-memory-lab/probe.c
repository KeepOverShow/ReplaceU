/* CUDA Driver API probe. Run only on a dedicated NVIDIA test environment. */
#include <cuda.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <limits.h>
#include <stdint.h>

static void report(const char *stage, CUresult result) {
    const char *name = "UNKNOWN";
    cuGetErrorName(result, &name);
    printf("{\"stage\":\"%s\",\"code\":%d,\"name\":\"%s\"}\n", stage,
           (int)result, name ? name : "UNKNOWN");
    fflush(stdout);
}
static size_t mib(const char *text) {
    char *end; errno = 0;
    unsigned long value = strtoul(text, &end, 10);
    if (errno || *end || !value || value > 8192 || value > SIZE_MAX / 1048576) {
        fprintf(stderr, "MiB argument must be 1..8192\n"); exit(2);
    }
    return (size_t)value * 1048576;
}
static CUresult allocate(const char *stage, CUdeviceptr *ptr, size_t bytes) {
    CUresult result = cuMemAlloc(ptr, bytes); report(stage, result);
    if (result == CUDA_SUCCESS) {
        report("touch", cuMemsetD8(*ptr, 1, bytes));
        report("synchronize", cuCtxSynchronize());
    }
    return result;
}
int main(int argc, char **argv) {
    if (argc != 3) { fprintf(stderr, "usage: probe FIRST_MIB SECOND_MIB\n"); return 2; }
    size_t first = mib(argv[1]), second = mib(argv[2]);
    CUdevice dev; CUcontext ctx; CUdeviceptr a = 0, b = 0, c = 0;
    CUresult result = cuInit(0); report("init", result); if (result) return 2;
    result = cuDeviceGet(&dev, 0); report("device", result); if (result) return 2;
    result = cuCtxCreate(&ctx, 0, dev); report("context", result); if (result) return 2;
    printf("{\"first_mib\":%zu,\"second_mib\":%zu}\n", first/1048576, second/1048576);
    CUresult ra = allocate("first", &a, first);
    CUresult rb = allocate("second_while_first_held", &b, second);
    if (rb == CUDA_SUCCESS) report("free_second", cuMemFree(b));
    if (ra == CUDA_SUCCESS) report("free_first", cuMemFree(a));
    CUresult rc = allocate("second_after_release", &c, second);
    if (rc == CUDA_SUCCESS) report("free_retry", cuMemFree(c));
    report("destroy_context", cuCtxDestroy(ctx));
    /* Outcome is data, not a passed isolation assertion. */
    return 0;
}
