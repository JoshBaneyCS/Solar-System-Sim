/*
 * texture_loader.c — Planet texture atlas loader using stb_image.
 */
#include "texture_loader.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define STB_IMAGE_IMPLEMENTATION
#include "stb_image.h"

#define STB_IMAGE_RESIZE2_IMPLEMENTATION
#include "stb_image_resize2.h"

static uint8_t* load_single_texture(const char* asset_dir, const char* body_name) {
    size_t layer_size = (size_t)TEXTURE_WIDTH * TEXTURE_HEIGHT * 4;
    uint8_t* output = (uint8_t*)malloc(layer_size);
    if (!output) return NULL;

    /* Try .jpg first, then .png */
    char path[1024];
    int w, h, channels;
    uint8_t* img = NULL;

    snprintf(path, sizeof(path), "%s/%s/albedo.jpg", asset_dir, body_name);
    img = stbi_load(path, &w, &h, &channels, 4);

    if (!img) {
        snprintf(path, sizeof(path), "%s/%s/albedo.png", asset_dir, body_name);
        img = stbi_load(path, &w, &h, &channels, 4);
    }

    if (!img) {
        /* Fallback: white texture */
        memset(output, 255, layer_size);
        return output;
    }

    /* Resize to target dimensions */
    if (w == TEXTURE_WIDTH && h == TEXTURE_HEIGHT) {
        memcpy(output, img, layer_size);
    } else {
        stbir_resize_uint8_linear(img, w, h, 0,
            output, TEXTURE_WIDTH, TEXTURE_HEIGHT, 0,
            STBIR_RGBA);
    }

    stbi_image_free(img);
    return output;
}

TextureAtlasData* texture_atlas_load(const char* asset_dir) {
    TextureAtlasData* atlas = (TextureAtlasData*)malloc(sizeof(TextureAtlasData));
    if (!atlas) return NULL;

    size_t layer_size = (size_t)TEXTURE_WIDTH * TEXTURE_HEIGHT * 4;
    size_t total_size = layer_size * NUM_BODY_TEXTURES;

    atlas->data = (uint8_t*)malloc(total_size);
    if (!atlas->data) {
        free(atlas);
        return NULL;
    }

    atlas->width = TEXTURE_WIDTH;
    atlas->height = TEXTURE_HEIGHT;
    atlas->layers = NUM_BODY_TEXTURES;

    for (uint32_t i = 0; i < NUM_BODY_TEXTURES; i++) {
        uint8_t* layer = load_single_texture(asset_dir, BODY_NAMES[i]);
        if (layer) {
            memcpy(atlas->data + i * layer_size, layer, layer_size);
            free(layer);
        } else {
            /* Allocation failed — fill with white */
            memset(atlas->data + i * layer_size, 255, layer_size);
        }
    }

    return atlas;
}

TextureAtlasData* texture_atlas_fallback(void) {
    TextureAtlasData* atlas = (TextureAtlasData*)malloc(sizeof(TextureAtlasData));
    if (!atlas) return NULL;

    /* 1x1 per layer, all white */
    atlas->width = 1;
    atlas->height = 1;
    atlas->layers = NUM_BODY_TEXTURES;
    atlas->data = (uint8_t*)malloc(4 * NUM_BODY_TEXTURES);
    if (!atlas->data) {
        free(atlas);
        return NULL;
    }

    memset(atlas->data, 255, 4 * NUM_BODY_TEXTURES);
    return atlas;
}

void texture_atlas_free(TextureAtlasData* atlas) {
    if (atlas) {
        free(atlas->data);
        free(atlas);
    }
}

const uint8_t* texture_atlas_layer(const TextureAtlasData* atlas, uint32_t layer) {
    if (!atlas || layer >= atlas->layers) return NULL;
    size_t layer_size = (size_t)atlas->width * atlas->height * 4;
    return atlas->data + layer * layer_size;
}
