/*
 * hardware.m — Metal GPU hardware detection.
 *
 * Implements render_get_hardware_info() and render_free_hardware_info()
 * from the native_render.h API using Apple Metal framework.
 */
#import <Metal/Metal.h>
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

#include "native_render.h"

static char* strdup_safe(const char* s) {
    if (!s) return NULL;
    size_t len = strlen(s) + 1;
    char* copy = (char*)malloc(len);
    if (copy) memcpy(copy, s, len);
    return copy;
}

HardwareInfo* render_get_hardware_info(void) {
    @autoreleasepool {
        id<MTLDevice> device = MTLCreateSystemDefaultDevice();
        if (!device) return NULL;

        HardwareInfo* info = (HardwareInfo*)calloc(1, sizeof(HardwareInfo));
        if (!info) return NULL;

        /* Device name */
        NSString* name = [device name];
        info->device_name = strdup_safe([name UTF8String]);

        /* Vendor — Apple Silicon vs Intel */
        info->vendor = strdup_safe("Apple");

        /* Backend */
        info->backend = strdup_safe("Metal");

        /* Device type */
        if ([device hasUnifiedMemory]) {
            info->device_type = strdup_safe("Integrated (Apple Silicon)");
        } else {
            info->device_type = strdup_safe("Discrete");
        }

        /* Max texture size */
        /* Metal guarantees at least 8192 on iOS, 16384 on macOS */
        info->max_texture_size = 16384;

        /* Hardware tier */
        /* Apple Silicon with unified memory = High, Intel integrated = Medium */
        if ([device hasUnifiedMemory]) {
            info->tier = 2; /* High — Apple Silicon */
        } else {
            info->tier = 1; /* Medium — discrete or Intel integrated */
        }

        return info;
    }
}

void render_free_hardware_info(HardwareInfo* info) {
    if (!info) return;
    free(info->vendor);
    free(info->device_name);
    free(info->backend);
    free(info->device_type);
    free(info);
}
