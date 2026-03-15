/*
 * camera.c — Camera world-to-screen projection.
 * Port of crates/render_core/src/camera.rs.
 */
#include "camera.h"
#include <math.h>

#define AU 1.496e11
#define DEFAULT_DISPLAY_SCALE 100.0

void camera_init(NativeCamera* cam, uint32_t width, uint32_t height) {
    cam->zoom = 1.0;
    cam->pan_x = 0.0;
    cam->pan_y = 0.0;
    cam->rotation_x = 0.0;
    cam->rotation_y = 0.0;
    cam->rotation_z = 0.0;
    cam->use_3d = 0;
    cam->follow_x = 0.0;
    cam->follow_y = 0.0;
    cam->follow_z = 0.0;
    cam->width = width;
    cam->height = height;
}

void camera_world_to_screen(const NativeCamera* cam,
    double wx, double wy, double wz,
    float* out_sx, float* out_sy)
{
    double x = wx;
    double y = wy;
    double z = wz;

    if (cam->use_3d) {
        if (cam->rotation_x != 0.0) {
            double cos_x = cos(cam->rotation_x);
            double sin_x = sin(cam->rotation_x);
            double new_y = y * cos_x - z * sin_x;
            double new_z = y * sin_x + z * cos_x;
            y = new_y;
            z = new_z;
        }
        if (cam->rotation_y != 0.0) {
            double cos_y = cos(cam->rotation_y);
            double sin_y = sin(cam->rotation_y);
            double new_x = x * cos_y + z * sin_y;
            double new_z = -x * sin_y + z * cos_y;
            x = new_x;
            z = new_z;
        }
        if (cam->rotation_z != 0.0) {
            double cos_z = cos(cam->rotation_z);
            double sin_z = sin(cam->rotation_z);
            double new_x = x * cos_z - y * sin_z;
            double new_y = x * sin_z + y * cos_z;
            x = new_x;
            y = new_y;
        }
    }

    double ds = DEFAULT_DISPLAY_SCALE * cam->zoom;
    double cw = (double)cam->width;
    double ch = (double)cam->height;

    double sx = (x - cam->follow_x) / AU * ds - cam->pan_x * ds + cw / 2.0;
    double sy = (y - cam->follow_y) / AU * ds - cam->pan_y * ds + ch / 2.0;

    if (cam->use_3d) {
        sx -= z / AU * ds * 0.5;
        sy -= z / AU * ds * 0.8;
    }

    *out_sx = (float)sx;
    *out_sy = (float)sy;
}
