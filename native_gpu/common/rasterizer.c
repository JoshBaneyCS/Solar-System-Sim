/*
 * rasterizer.c — CPU line/circle rasterization for overlay compositing.
 */
#include "rasterizer.h"
#include <stdlib.h>

void raster_blend_pixel(uint8_t* pixels, uint32_t width, uint32_t height,
    int x, int y,
    uint8_t r, uint8_t g, uint8_t b, uint8_t a)
{
    if (x < 0 || x >= (int)width || y < 0 || y >= (int)height) return;
    if (a == 0) return;

    uint8_t* p = pixels + ((uint32_t)y * width + (uint32_t)x) * 4;

    if (a == 255) {
        p[0] = r; p[1] = g; p[2] = b; p[3] = 255;
        return;
    }

    /* Alpha blend: out = src * alpha + dst * (1 - alpha) */
    uint32_t sa = a;
    uint32_t da = 255 - sa;
    p[0] = (uint8_t)((r * sa + p[0] * da) / 255);
    p[1] = (uint8_t)((g * sa + p[1] * da) / 255);
    p[2] = (uint8_t)((b * sa + p[2] * da) / 255);
    p[3] = (uint8_t)(sa + (p[3] * da) / 255);
}

void raster_draw_line(uint8_t* pixels, uint32_t width, uint32_t height,
    int x0, int y0, int x1, int y1,
    uint8_t r, uint8_t g, uint8_t b, uint8_t a)
{
    int dx = abs(x1 - x0);
    int dy = -abs(y1 - y0);
    int sx = x0 < x1 ? 1 : -1;
    int sy = y0 < y1 ? 1 : -1;
    int err = dx + dy;

    for (;;) {
        raster_blend_pixel(pixels, width, height, x0, y0, r, g, b, a);
        if (x0 == x1 && y0 == y1) break;
        int e2 = 2 * err;
        if (e2 >= dy) { err += dy; x0 += sx; }
        if (e2 <= dx) { err += dx; y0 += sy; }
    }
}

void raster_draw_circle(uint8_t* pixels, uint32_t width, uint32_t height,
    int cx, int cy, int radius,
    uint8_t r, uint8_t g, uint8_t b, uint8_t a)
{
    int r2 = radius * radius;
    for (int dy = -radius; dy <= radius; dy++) {
        for (int dx = -radius; dx <= radius; dx++) {
            if (dx * dx + dy * dy <= r2) {
                raster_blend_pixel(pixels, width, height,
                    cx + dx, cy + dy, r, g, b, a);
            }
        }
    }
}
