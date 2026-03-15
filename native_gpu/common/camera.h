/*
 * camera.h — Camera world-to-screen projection.
 * Port of crates/render_core/src/camera.rs.
 */
#ifndef CAMERA_H
#define CAMERA_H

#include <stdint.h>

typedef struct {
    double zoom;
    double pan_x;
    double pan_y;
    double rotation_x;
    double rotation_y;
    double rotation_z;
    int    use_3d;
    double follow_x;
    double follow_y;
    double follow_z;
    uint32_t width;
    uint32_t height;
} NativeCamera;

/* Initialize camera with default values. */
void camera_init(NativeCamera* cam, uint32_t width, uint32_t height);

/* Convert world position (meters) to screen pixel coordinates. */
void camera_world_to_screen(const NativeCamera* cam,
    double wx, double wy, double wz,
    float* out_sx, float* out_sy);

#endif
