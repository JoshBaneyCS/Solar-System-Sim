#ifndef RENDER_CORE_H
#define RENDER_CORE_H

#include <stdint.h>

typedef struct Renderer Renderer;

Renderer* render_create(uint32_t width, uint32_t height);

void render_set_camera(Renderer* h,
    double zoom, double pan_x, double pan_y,
    double rot_x, double rot_y, double rot_z,
    uint8_t use_3d,
    double follow_x, double follow_y, double follow_z);

void render_set_bodies(Renderer* h,
    uint32_t n_bodies,
    const double* positions,
    const double* colors,
    const double* radii,
    const double* sun_pos,
    const double* sun_color,
    double sun_radius);

void render_set_trails(Renderer* h,
    uint32_t n_bodies,
    const uint32_t* trail_lengths,
    const double* trail_positions,
    const double* trail_colors,
    uint8_t show_trails);

void render_set_spacetime(Renderer* h,
    uint8_t show_spacetime,
    const double* masses,
    const double* positions,
    uint32_t n_bodies);

void render_set_distance_line(Renderer* h,
    uint8_t has_line,
    double x1, double y1, double z1,
    double x2, double y2, double z2);

const uint8_t* render_frame(Renderer* h);

void render_resize(Renderer* h, uint32_t width, uint32_t height);

void render_free(Renderer* h);

#endif
