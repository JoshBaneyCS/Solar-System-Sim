/*
 * renderer.hip — HIP/ROCm GPU renderer host code (AMD GPUs).
 *
 * Implements the native_render.h C API using HIP runtime API.
 * Near-identical to the CUDA renderer with hip* API substitutions.
 */
#include <hip/hip_runtime.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

extern "C" {
#include "native_render.h"
#include "sphere.h"
#include "camera.h"
#include "texture_loader.h"
#include "rasterizer.h"
}

extern "C" void launch_raytrace_kernel(
    const RTSphere* d_spheres,
    RTCameraUniform cam,
    float4* d_accum,
    uint8_t* d_output,
    hipTextureObject_t atlas,
    uint32_t tex_width,
    uint32_t tex_height);

typedef struct {
    float screen_x, screen_y, radius;
    float color[4];
    int texture_index;
} BodyData;

typedef struct { float x1, y1, x2, y2; } DistLineData;
typedef struct { float x, y; float color[4]; } TrailVertex;

struct Renderer {
    int device_id;
    RTSphere* d_spheres;
    float4*   d_accum;
    uint8_t*  d_output;
    hipArray_t         tex_array;
    hipTextureObject_t tex_object;
    uint32_t tex_width, tex_height;
    uint32_t width, height;
    NativeCamera camera;
    BodyData bodies[MAX_RT_SPHERES];
    uint32_t num_bodies;
    BodyData sun;
    int has_sun;
    int rt_enabled;
    uint32_t rt_frame_count, rt_samples_per_frame, rt_max_bounces;
    TrailVertex* trail_vertices;
    uint32_t trail_vertex_count;
    int show_trails;
    DistLineData dist_line;
    int has_dist_line;
    int show_spacetime;
    uint8_t* pixels;
    size_t pixel_size;
};

static void allocate_gpu_buffers(struct Renderer* r) {
    size_t pixel_count = (size_t)r->width * r->height;
    size_t output_size = pixel_count * 4;
    size_t accum_size = pixel_count * sizeof(float4);
    if (r->d_output) hipFree(r->d_output);
    if (r->d_accum)  hipFree(r->d_accum);
    hipMalloc(&r->d_output, output_size);
    hipMalloc(&r->d_accum, accum_size);
    hipMemset(r->d_accum, 0, accum_size);
    free(r->pixels);
    r->pixel_size = output_size;
    r->pixels = (uint8_t*)malloc(output_size);
    if (r->pixels) memset(r->pixels, 0, output_size);
}

static hipTextureObject_t create_texture_atlas(TextureAtlasData* atlas, hipArray_t* out_array) {
    hipChannelFormatDesc desc = hipCreateChannelDesc(8, 8, 8, 8, hipChannelFormatKindUnsigned);
    hipExtent extent = make_hipExtent(atlas->width, atlas->height, atlas->layers);
    hipMalloc3DArray(out_array, &desc, extent, hipArrayLayered);

    for (uint32_t i = 0; i < atlas->layers; i++) {
        hipMemcpy3DParms params = {0};
        params.srcPtr = make_hipPitchedPtr(
            (void*)(atlas->data + i * atlas->width * atlas->height * 4),
            atlas->width * 4, atlas->width, atlas->height);
        params.dstArray = *out_array;
        params.dstPos = make_hipPos(0, 0, i);
        params.extent = make_hipExtent(atlas->width, atlas->height, 1);
        params.kind = hipMemcpyHostToDevice;
        hipMemcpy3D(&params);
    }

    hipResourceDesc resDesc;
    memset(&resDesc, 0, sizeof(resDesc));
    resDesc.resType = hipResourceTypeArray;
    resDesc.res.array.array = *out_array;

    hipTextureDesc texDesc;
    memset(&texDesc, 0, sizeof(texDesc));
    texDesc.addressMode[0] = hipAddressModeWrap;
    texDesc.addressMode[1] = hipAddressModeClamp;
    texDesc.filterMode = hipFilterModeLinear;
    texDesc.readMode = hipReadModeNormalizedFloat;
    texDesc.normalizedCoords = 1;

    hipTextureObject_t texObj = 0;
    hipCreateTextureObject(&texObj, &resDesc, &texDesc, NULL);
    return texObj;
}

extern "C" Renderer* render_create(uint32_t width, uint32_t height) {
    return render_create_with_textures(width, height, NULL);
}

extern "C" Renderer* render_create_with_textures(uint32_t width, uint32_t height, const char* asset_dir) {
    struct Renderer* r = (struct Renderer*)calloc(1, sizeof(struct Renderer));
    if (!r) return NULL;
    r->width = width; r->height = height;
    r->rt_samples_per_frame = 1; r->rt_max_bounces = 4;
    camera_init(&r->camera, width, height);

    int count = 0;
    if (hipGetDeviceCount(&count) != hipSuccess || count == 0) { free(r); return NULL; }
    r->device_id = 0;
    hipSetDevice(r->device_id);
    hipMalloc(&r->d_spheres, MAX_RT_SPHERES * sizeof(RTSphere));
    allocate_gpu_buffers(r);

    TextureAtlasData* atlas = NULL;
    if (asset_dir && strlen(asset_dir) > 0) atlas = texture_atlas_load(asset_dir);
    if (!atlas) atlas = texture_atlas_fallback();
    if (atlas) {
        r->tex_width = atlas->width; r->tex_height = atlas->height;
        r->tex_object = create_texture_atlas(atlas, &r->tex_array);
        texture_atlas_free(atlas);
    }
    return r;
}

extern "C" void render_set_camera(Renderer* h,
    double zoom, double pan_x, double pan_y,
    double rot_x, double rot_y, double rot_z, uint8_t use_3d,
    double follow_x, double follow_y, double follow_z) {
    struct Renderer* r = (struct Renderer*)h;
    r->camera.zoom = zoom; r->camera.pan_x = pan_x; r->camera.pan_y = pan_y;
    r->camera.rotation_x = rot_x; r->camera.rotation_y = rot_y; r->camera.rotation_z = rot_z;
    r->camera.use_3d = (use_3d != 0);
    r->camera.follow_x = follow_x; r->camera.follow_y = follow_y; r->camera.follow_z = follow_z;
}

extern "C" void render_set_bodies(Renderer* h,
    uint32_t n_bodies, const double* positions, const double* colors,
    const double* radii, const double* sun_pos, const double* sun_color, double sun_radius) {
    struct Renderer* r = (struct Renderer*)h;
    uint32_t n = n_bodies < MAX_RT_SPHERES - 1 ? n_bodies : MAX_RT_SPHERES - 1;
    r->num_bodies = n;
    for (uint32_t i = 0; i < n; i++) {
        float sx, sy;
        camera_world_to_screen(&r->camera, positions[i*3], positions[i*3+1], positions[i*3+2], &sx, &sy);
        r->bodies[i] = {sx, sy, (float)radii[i],
            {(float)colors[i*4], (float)colors[i*4+1], (float)colors[i*4+2], (float)colors[i*4+3]},
            (int)i};
    }
    float sx, sy;
    camera_world_to_screen(&r->camera, sun_pos[0], sun_pos[1], sun_pos[2], &sx, &sy);
    r->sun = {sx, sy, (float)sun_radius,
        {(float)sun_color[0], (float)sun_color[1], (float)sun_color[2], (float)sun_color[3]},
        NUM_BODY_TEXTURES - 1};
    r->has_sun = 1;
}

extern "C" void render_set_trails(Renderer* h,
    uint32_t n_bodies, const uint32_t* trail_lengths, const double* trail_positions,
    const double* trail_colors, uint8_t show_trails) {
    struct Renderer* r = (struct Renderer*)h;
    r->show_trails = (show_trails != 0);
    free(r->trail_vertices); r->trail_vertices = NULL; r->trail_vertex_count = 0;
    if (!r->show_trails || !trail_lengths || !trail_positions) return;
    uint32_t total = 0;
    for (uint32_t i = 0; i < n_bodies; i++) total += trail_lengths[i];
    if (total < 2) return;
    r->trail_vertices = (TrailVertex*)malloc(total * 2 * sizeof(TrailVertex));
    if (!r->trail_vertices) return;
    uint32_t offset = 0, vcount = 0;
    for (uint32_t i = 0; i < n_bodies; i++) {
        uint32_t len = trail_lengths[i];
        if (len < 2) { offset += len; continue; }
        float br = (float)trail_colors[i*4], bg = (float)trail_colors[i*4+1], bb = (float)trail_colors[i*4+2];
        uint32_t step = len > 200 ? len / 200 : 1;
        for (uint32_t j = 0; j + step < len; j += step) {
            uint32_t idx1 = offset + j, idx2 = offset + j + step;
            float x1, y1, x2, y2;
            camera_world_to_screen(&r->camera, trail_positions[idx1*3], trail_positions[idx1*3+1], trail_positions[idx1*3+2], &x1, &y1);
            camera_world_to_screen(&r->camera, trail_positions[idx2*3], trail_positions[idx2*3+1], trail_positions[idx2*3+2], &x2, &y2);
            float alpha = (float)j / (float)len;
            r->trail_vertices[vcount++] = {x1, y1, {br, bg, bb, alpha}};
            r->trail_vertices[vcount++] = {x2, y2, {br, bg, bb, alpha}};
        }
        offset += len;
    }
    r->trail_vertex_count = vcount;
}

extern "C" void render_set_spacetime(Renderer* h, uint8_t show, const double* m, const double* p, uint32_t n) {
    ((struct Renderer*)h)->show_spacetime = (show != 0);
    (void)m; (void)p; (void)n;
}

extern "C" void render_set_distance_line(Renderer* h, uint8_t has,
    double x1, double y1, double z1, double x2, double y2, double z2) {
    struct Renderer* r = (struct Renderer*)h;
    r->has_dist_line = (has != 0);
    if (r->has_dist_line) {
        camera_world_to_screen(&r->camera, x1, y1, z1, &r->dist_line.x1, &r->dist_line.y1);
        camera_world_to_screen(&r->camera, x2, y2, z2, &r->dist_line.x2, &r->dist_line.y2);
    }
}

extern "C" void render_set_rt_mode(Renderer* h, uint8_t enabled) {
    struct Renderer* r = (struct Renderer*)h;
    r->rt_enabled = (enabled != 0); r->rt_frame_count = 0;
}

extern "C" void render_set_rt_quality(Renderer* h, uint32_t spf, uint32_t mb) {
    struct Renderer* r = (struct Renderer*)h;
    r->rt_samples_per_frame = spf; r->rt_max_bounces = mb; r->rt_frame_count = 0;
}

extern "C" const uint8_t* render_frame(Renderer* h) {
    struct Renderer* r = (struct Renderer*)h;
    if (!r->pixels) return NULL;

    if (r->rt_enabled) {
        uint32_t total = 0;
        RTSphere rt[MAX_RT_SPHERES]; memset(rt, 0, sizeof(rt));
        for (uint32_t i = 0; i < r->num_bodies && total < MAX_RT_SPHERES; i++) {
            BodyData* b = &r->bodies[i];
            rt[total] = {{b->screen_x, b->screen_y, 0}, b->radius,
                {b->color[0], b->color[1], b->color[2], b->color[3]},
                (i >= 4 && i <= 7) ? 2u : 0u, b->texture_index, {0, 0}};
            total++;
        }
        float ssx = 0, ssy = 0;
        if (r->has_sun && total < MAX_RT_SPHERES) {
            rt[total] = {{r->sun.screen_x, r->sun.screen_y, 0}, r->sun.radius,
                {r->sun.color[0], r->sun.color[1], r->sun.color[2], r->sun.color[3]},
                1u, r->sun.texture_index, {0, 0}};
            ssx = r->sun.screen_x; ssy = r->sun.screen_y;
            total++;
        }
        hipMemcpy(r->d_spheres, rt, total * sizeof(RTSphere), hipMemcpyHostToDevice);
        RTCameraUniform cam = {(float)r->width, (float)r->height, r->rt_frame_count, total,
            ssx, ssy, r->rt_samples_per_frame, r->rt_max_bounces};
        launch_raytrace_kernel(r->d_spheres, cam, r->d_accum, r->d_output,
            r->tex_object, r->tex_width, r->tex_height);
        hipDeviceSynchronize();
        r->rt_frame_count++;
        hipMemcpy(r->pixels, r->d_output, r->pixel_size, hipMemcpyDeviceToHost);
    } else {
        for (size_t i = 0; i < r->pixel_size; i += 4) {
            r->pixels[i] = 5; r->pixels[i+1] = 5; r->pixels[i+2] = 15; r->pixels[i+3] = 255;
        }
        if (r->has_sun)
            raster_draw_circle(r->pixels, r->width, r->height,
                (int)r->sun.screen_x, (int)r->sun.screen_y, (int)r->sun.radius,
                (uint8_t)(r->sun.color[0]*255), (uint8_t)(r->sun.color[1]*255),
                (uint8_t)(r->sun.color[2]*255), 255);
        for (uint32_t i = 0; i < r->num_bodies; i++) {
            BodyData* b = &r->bodies[i];
            raster_draw_circle(r->pixels, r->width, r->height,
                (int)b->screen_x, (int)b->screen_y, (int)b->radius,
                (uint8_t)(b->color[0]*255), (uint8_t)(b->color[1]*255),
                (uint8_t)(b->color[2]*255), 255);
        }
    }

    if (r->show_trails && r->trail_vertices && r->trail_vertex_count >= 2)
        for (uint32_t i = 0; i + 1 < r->trail_vertex_count; i += 2) {
            TrailVertex* v0 = &r->trail_vertices[i]; TrailVertex* v1 = &r->trail_vertices[i+1];
            raster_draw_line(r->pixels, r->width, r->height,
                (int)v0->x, (int)v0->y, (int)v1->x, (int)v1->y,
                (uint8_t)(v0->color[0]*255), (uint8_t)(v0->color[1]*255),
                (uint8_t)(v0->color[2]*255), (uint8_t)(v0->color[3]*255));
        }
    if (r->has_dist_line)
        raster_draw_line(r->pixels, r->width, r->height,
            (int)r->dist_line.x1, (int)r->dist_line.y1,
            (int)r->dist_line.x2, (int)r->dist_line.y2, 255, 255, 0, 200);
    return r->pixels;
}

extern "C" void render_resize(Renderer* h, uint32_t w, uint32_t he) {
    struct Renderer* r = (struct Renderer*)h;
    if (w == 0 || he == 0) return;
    r->width = w; r->height = he;
    r->camera.width = w; r->camera.height = he;
    allocate_gpu_buffers(r);
    r->rt_frame_count = 0;
}

extern "C" void render_free(Renderer* h) {
    if (!h) return;
    struct Renderer* r = (struct Renderer*)h;
    hipFree(r->d_spheres); hipFree(r->d_accum); hipFree(r->d_output);
    if (r->tex_object) hipDestroyTextureObject(r->tex_object);
    if (r->tex_array) hipFreeArray(r->tex_array);
    free(r->pixels); free(r->trail_vertices); free(r);
}
