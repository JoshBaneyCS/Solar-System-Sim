/*
 * hardware.hip — ROCm/HIP GPU hardware detection for AMD GPUs.
 */
#include <hip/hip_runtime.h>
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
    int count = 0;
    if (hipGetDeviceCount(&count) != hipSuccess || count == 0) return NULL;

    hipDeviceProp_t prop;
    if (hipGetDeviceProperties(&prop, 0) != hipSuccess) return NULL;

    HardwareInfo* info = (HardwareInfo*)calloc(1, sizeof(HardwareInfo));
    if (!info) return NULL;

    info->vendor = strdup_safe("AMD");
    info->device_name = strdup_safe(prop.name);
    info->backend = strdup_safe("ROCm/HIP");
    info->device_type = prop.integrated ? strdup_safe("Integrated (APU)") : strdup_safe("Discrete");
    info->max_texture_size = (uint32_t)prop.maxTexture2DLayered[0];

    /* Tier based on GCN architecture version */
    int gcn_arch = prop.gcnArch;
    if (gcn_arch >= 1030) {
        info->tier = 2; /* High — RDNA 2+ (RX 6000, 7000 series) */
    } else if (gcn_arch >= 1010) {
        info->tier = 1; /* Medium — RDNA 1 (RX 5000 series) */
    } else {
        info->tier = 0; /* Low — GCN / Vega */
    }

    return info;
}

extern "C" void render_free_hardware_info(HardwareInfo* info) {
    if (!info) return;
    free(info->vendor); free(info->device_name);
    free(info->backend); free(info->device_type);
    free(info);
}
