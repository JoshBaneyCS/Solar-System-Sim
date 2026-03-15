/*
 * sphere.h — Shared GPU data structures for native ray tracing kernels.
 *
 * These structs match the WGSL shader layout in crates/render_core/src/raytracer.rs
 * and the Rust RTSphere/RTCameraUniform types (48 bytes / 32 bytes respectively).
 */
#ifndef SPHERE_H
#define SPHERE_H

#include <stdint.h>

/* Maximum number of spheres (planets + sun) supported by the ray tracer. */
#define MAX_RT_SPHERES 16

/* Number of planet texture layers (8 planets + 1 sun). */
#define NUM_BODY_TEXTURES 9

/* Target texture atlas resolution per layer. */
#define TEXTURE_WIDTH  2048
#define TEXTURE_HEIGHT 1024

/*
 * RTSphere — Per-sphere data sent to the GPU compute kernel.
 * center: screen-space position [x, y, z] (z used for depth sorting)
 * radius: display radius in pixels
 * color:  [r, g, b, a] in 0-1 range
 * material: 0=diffuse (rocky planet), 1=emissive (sun), 2=glossy (gas giant)
 * texture_index: index into texture atlas layers (-1 = no texture)
 */
typedef struct {
    float center[3];
    float radius;
    float color[4];
    uint32_t material;
    int32_t texture_index;
    uint32_t _pad[2];
} RTSphere; /* 48 bytes */

/*
 * RTCameraUniform — Camera parameters for the compute kernel.
 * width/height: render target dimensions in pixels
 * frame_count: progressive accumulation frame index (0 = first frame)
 * num_spheres: number of valid entries in the sphere buffer
 * sun_screen_x/y: screen-space position of the sun (light source)
 * samples_per_frame: RT quality parameter
 * max_bounces: RT quality parameter
 */
typedef struct {
    float width;
    float height;
    uint32_t frame_count;
    uint32_t num_spheres;
    float sun_screen_x;
    float sun_screen_y;
    uint32_t samples_per_frame;
    uint32_t max_bounces;
} RTCameraUniform; /* 32 bytes */

/* Body names matching texture atlas layer order. */
static const char* BODY_NAMES[NUM_BODY_TEXTURES] = {
    "mercury", "venus", "earth", "mars",
    "jupiter", "saturn", "uranus", "neptune",
    "sun"
};

#endif
