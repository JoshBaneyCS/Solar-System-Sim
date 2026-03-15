/*
 * texture_loader.h — Planet texture atlas loader using stb_image.
 *
 * Loads planet albedo textures from disk and produces a flat RGBA atlas
 * suitable for upload to any GPU texture API (Metal, CUDA, HIP).
 */
#ifndef TEXTURE_LOADER_H
#define TEXTURE_LOADER_H

#include <stdint.h>
#include "sphere.h"

/*
 * TextureAtlasData — CPU-side texture atlas ready for GPU upload.
 * data: flat RGBA8 pixel data, NUM_BODY_TEXTURES layers concatenated.
 *       Total size: TEXTURE_WIDTH * TEXTURE_HEIGHT * 4 * NUM_BODY_TEXTURES bytes.
 *       Layer order matches BODY_NAMES[]: mercury, venus, ..., neptune, sun.
 * Each layer is TEXTURE_WIDTH * TEXTURE_HEIGHT * 4 bytes (RGBA8).
 */
typedef struct {
    uint8_t* data;
    uint32_t width;
    uint32_t height;
    uint32_t layers;
} TextureAtlasData;

/*
 * Load texture atlas from an asset directory.
 * Expects {asset_dir}/{body_name}/albedo.jpg (or .png) for each body.
 * Missing textures get a white fallback layer.
 * Returns atlas data; caller must free with texture_atlas_free().
 * Returns NULL on allocation failure.
 */
TextureAtlasData* texture_atlas_load(const char* asset_dir);

/*
 * Create a minimal fallback atlas (all white, 1x1 per layer).
 * Used when no asset directory is provided.
 */
TextureAtlasData* texture_atlas_fallback(void);

/*
 * Free a texture atlas.
 */
void texture_atlas_free(TextureAtlasData* atlas);

/*
 * Get pointer to a specific layer's pixel data within the atlas.
 */
const uint8_t* texture_atlas_layer(const TextureAtlasData* atlas, uint32_t layer);

#endif
