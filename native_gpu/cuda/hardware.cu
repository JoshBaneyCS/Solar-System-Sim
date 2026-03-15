/*
 * hardware.cu — CUDA GPU hardware detection.
 *
 * Implements render_get_hardware_info() and render_free_hardware_info()
 * from the native_render.h API using CUDA runtime API.
 */
#include <cuda_runtime.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

extern "C" {
#include "native_render.h"
}

static char* strdup_safe(const char* s) {
    if (!s) return NULL;
    size_t len = strlen(s) + 1;
    char* copy = (char*)malloc(len);
    if (copy) memcpy(copy, s, len);
    return copy;
}

extern "C" HardwareInfo* render_get_hardware_info(void) {
    int device_count = 0;
    if (cudaGetDeviceCount(&device_count) != cudaSuccess || device_count == 0) {
        return NULL;
    }

    cudaDeviceProp prop;
    if (cudaGetDeviceProperties(&prop, 0) != cudaSuccess) {
        return NULL;
    }

    HardwareInfo* info = (HardwareInfo*)calloc(1, sizeof(HardwareInfo));
    if (!info) return NULL;

    info->vendor = strdup_safe("NVIDIA");
    info->device_name = strdup_safe(prop.name);
    info->backend = strdup_safe("CUDA");

    /* Device type based on integrated flag */
    if (prop.integrated) {
        info->device_type = strdup_safe("Integrated");
    } else {
        info->device_type = strdup_safe("Discrete");
    }

    info->max_texture_size = (uint32_t)prop.maxTexture2DLayered[0];

    /* Tier based on compute capability */
    int cc = prop.major * 10 + prop.minor;
    if (cc >= 86) {
        info->tier = 2; /* High — Ampere+ (RTX 30xx, 40xx) */
    } else if (cc >= 70) {
        info->tier = 1; /* Medium — Volta/Turing (RTX 20xx, GTX 16xx) */
    } else {
        info->tier = 0; /* Low — Pascal and older */
    }

    return info;
}

extern "C" void render_free_hardware_info(HardwareInfo* info) {
    if (!info) return;
    free(info->vendor);
    free(info->device_name);
    free(info->backend);
    free(info->device_type);
    free(info);
}
