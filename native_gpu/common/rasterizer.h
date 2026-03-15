/*
 * rasterizer.h — CPU line/circle rasterization for overlay compositing.
 *
 * Used by native GPU backends to draw trails, spacetime grid, and distance
 * lines onto the ray-traced pixel buffer without needing a GPU raster pipeline.
 */
#ifndef RASTERIZER_H
#define RASTERIZER_H

#include <stdint.h>

/*
 * Draw a line from (x0,y0) to (x1,y1) using Bresenham's algorithm.
 * Color is [r,g,b,a] in 0-255 range. Alpha-blends onto the pixel buffer.
 * pixels: RGBA8 buffer, stride = width * 4.
 */
void raster_draw_line(uint8_t* pixels, uint32_t width, uint32_t height,
    int x0, int y0, int x1, int y1,
    uint8_t r, uint8_t g, uint8_t b, uint8_t a);

/*
 * Draw a filled circle at (cx, cy) with given radius.
 * Color is [r,g,b,a] in 0-255. Alpha-blends onto the pixel buffer.
 */
void raster_draw_circle(uint8_t* pixels, uint32_t width, uint32_t height,
    int cx, int cy, int radius,
    uint8_t r, uint8_t g, uint8_t b, uint8_t a);

/*
 * Alpha-blend a single pixel at (x, y) onto the buffer.
 */
void raster_blend_pixel(uint8_t* pixels, uint32_t width, uint32_t height,
    int x, int y,
    uint8_t r, uint8_t g, uint8_t b, uint8_t a);

#endif
