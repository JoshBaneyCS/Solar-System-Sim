/*
 * renderer.cu — CUDA GPU renderer host code.
 *
 * Implements the native_render.h C API using CUDA runtime API.
 * Manages device memory, texture objects, kernel dispatch, and pixel readback.
 */
#include <cuda_runtime.h>
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

/* Forward declaration of kernel launcher */
extern "C" void launch_raytrace_kernel(
    const RTSphere* d_spheres,
    RTCameraUniform cam,
    float4* d_accum,
    uint8_t* d_output,
    cudaTextureObject_t atlas,
    uint32_t tex_width,
    uint32_t tex_height);

/* ---- Internal renderer state ---- */

typedef struct {
    float screen_x, screen_y;
    float radius;
    float color[4];
    int texture_index;
} BodyData;

typedef struct {
    float x1, y1, x2, y2;
} DistLineData;

typedef struct {
    float x, y;
    float color[4];
} TrailVertex;

struct Renderer {
    /* CUDA device */
    int device_id;

    /* GPU buffers */
    RTSphere* d_spheres;
    float4*   d_accum;
    uint8_t*  d_output;

    /* Texture atlas */
    cudaArray_t         tex_array;
    cudaTextureObject_t tex_object;
    uint32_t            tex_width;
    uint32_t            tex_height;

    /* Dimensions */
    uint32_t width;
    uint32_t height;

    /* Camera */
    NativeCamera camera;

    /* Body data */
    BodyData bodies[MAX_RT_SPHERES];
    uint32_t num_bodies;
    BodyData sun;
    int has_sun;

    /* RT state */
    int      rt_enabled;
    uint32_t rt_frame_count;
    uint32_t rt_samples_per_frame;
    uint32_t rt_max_bounces;

    /* Trails */
    TrailVertex* trail_vertices;
    uint32_t     trail_vertex_count;
    int          show_trails;

    /* Distance line */
    DistLineData dist_line;
    int has_dist_line;

    /* Spacetime */
    int show_spacetime;

    /* Host readback buffer */
    uint8_t* pixels;
    size_t   pixel_size;
};

/* ---- Helper functions ---- */

static void allocate_gpu_buffers(struct Renderer* r) {
    size_t pixel_count = (size_t)r->width * r->height;
    size_t output_size = pixel_count * 4;
    size_t accum_size = pixel_count * sizeof(float4);

    if (r->d_output) cudaFree(r->d_output);
    if (r->d_accum)  cudaFree(r->d_accum);

    cudaMalloc(&r->d_output, output_size);
    cudaMalloc(&r->d_accum, accum_size);
    cudaMemset(r->d_accum, 0, accum_size);

    free(r->pixels);
    r->pixel_size = output_size;
    r->pixels = (uint8_t*)malloc(output_size);
    if (r->pixels) memset(r->pixels, 0, output_size);
}

static cudaTextureObject_t create_texture_atlas(TextureAtlasData* atlas,
    cudaArray_t* out_array)
{
    /* Create a layered CUDA array */
    cudaChannelFormatDesc desc = cudaCreateChannelDesc(8, 8, 8, 8, cudaChannelFormatKindUnsigned);

    cudaExtent extent;
    extent.width = atlas->width;
    extent.height = atlas->height;
    extent.depth = atlas->layers;

    cudaMalloc3DArray(out_array, &desc, extent, cudaArrayLayered);

    /* Copy each layer */
    for (uint32_t i = 0; i < atlas->layers; i++) {
        cudaMemcpy3DParms params = {0};
        params.srcPtr = make_cudaPitchedPtr(
            (void*)(atlas->data + i * atlas->width * atlas->height * 4),
            atlas->width * 4, atlas->width, atlas->height);
        params.dstArray = *out_array;
        params.dstPos = make_cudaPos(0, 0, i);
        params.extent = make_cudaExtent(atlas->width, atlas->height, 1);
        params.kind = cudaMemcpyHostToDevice;
        cudaMemcpy3D(&params);
    }

    /* Create texture object */
    cudaResourceDesc resDesc;
    memset(&resDesc, 0, sizeof(resDesc));
    resDesc.resType = cudaResourceTypeArray;
    resDesc.res.array.array = *out_array;

    cudaTextureDesc texDesc;
    memset(&texDesc, 0, sizeof(texDesc));
    texDesc.addressMode[0] = cudaAddressModeWrap;
    texDesc.addressMode[1] = cudaAddressModeClamp;
    texDesc.filterMode = cudaFilterModeLinear;
    texDesc.readMode = cudaReadModeNormalizedFloat;
    texDesc.normalizedCoords = 1;

    cudaTextureObject_t texObj = 0;
    cudaCreateTextureObject(&texObj, &resDesc, &texDesc, NULL);
    return texObj;
}

/* ---- Public API ---- */

extern "C" Renderer* render_create(uint32_t width, uint32_t height) {
    return render_create_with_textures(width, height, NULL);
}

extern "C" Renderer* render_create_with_textures(uint32_t width, uint32_t height,
    const char* asset_dir)
{
    struct Renderer* r = (struct Renderer*)calloc(1, sizeof(struct Renderer));
    if (!r) return NULL;

    r->width = width;
    r->height = height;
    r->rt_samples_per_frame = 1;
    r->rt_max_bounces = 4;
    camera_init(&r->camera, width, height);

    /* Initialize CUDA */
    int device_count = 0;
    if (cudaGetDeviceCount(&device_count) != cudaSuccess || device_count == 0) {
        free(r);
        return NULL;
    }
    r->device_id = 0;
    cudaSetDevice(r->device_id);

    /* Allocate sphere buffer on device */
    cudaMalloc(&r->d_spheres, MAX_RT_SPHERES * sizeof(RTSphere));

    /* Allocate output and accumulation buffers */
    allocate_gpu_buffers(r);

    /* Load texture atlas */
    TextureAtlasData* atlas = NULL;
    if (asset_dir && strlen(asset_dir) > 0) {
        atlas = texture_atlas_load(asset_dir);
    }
    if (!atlas) {
        atlas = texture_atlas_fallback();
    }
    if (atlas) {
        r->tex_width = atlas->width;
        r->tex_height = atlas->height;
        r->tex_object = create_texture_atlas(atlas, &r->tex_array);
        texture_atlas_free(atlas);
    }

    return r;
}

extern "C" void render_set_camera(Renderer* h,
    double zoom, double pan_x, double pan_y,
    double rot_x, double rot_y, double rot_z,
    uint8_t use_3d,
    double follow_x, double follow_y, double follow_z)
{
    struct Renderer* r = (struct Renderer*)h;
    r->camera.zoom = zoom;
    r->camera.pan_x = pan_x;
    r->camera.pan_y = pan_y;
    r->camera.rotation_x = rot_x;
    r->camera.rotation_y = rot_y;
    r->camera.rotation_z = rot_z;
    r->camera.use_3d = (use_3d != 0);
    r->camera.follow_x = follow_x;
    r->camera.follow_y = follow_y;
    r->camera.follow_z = follow_z;
}

extern "C" void render_set_bodies(Renderer* h,
    uint32_t n_bodies,
    const double* positions,
    const double* colors,
    const double* radii,
    const double* sun_pos,
    const double* sun_color,
    double sun_radius)
{
    struct Renderer* r = (struct Renderer*)h;
    uint32_t n = n_bodies < MAX_RT_SPHERES - 1 ? n_bodies : MAX_RT_SPHERES - 1;
    r->num_bodies = n;

    for (uint32_t i = 0; i < n; i++) {
        float sx, sy;
        camera_world_to_screen(&r->camera,
            positions[i*3], positions[i*3+1], positions[i*3+2], &sx, &sy);
        r->bodies[i].screen_x = sx;
        r->bodies[i].screen_y = sy;
        r->bodies[i].radius = (float)radii[i];
        r->bodies[i].color[0] = (float)colors[i*4];
        r->bodies[i].color[1] = (float)colors[i*4+1];
        r->bodies[i].color[2] = (float)colors[i*4+2];
        r->bodies[i].color[3] = (float)colors[i*4+3];
        r->bodies[i].texture_index = (int)i;
    }

    float sx, sy;
    camera_world_to_screen(&r->camera, sun_pos[0], sun_pos[1], sun_pos[2], &sx, &sy);
    r->sun.screen_x = sx;
    r->sun.screen_y = sy;
    r->sun.radius = (float)sun_radius;
    r->sun.color[0] = (float)sun_color[0];
    r->sun.color[1] = (float)sun_color[1];
    r->sun.color[2] = (float)sun_color[2];
    r->sun.color[3] = (float)sun_color[3];
    r->sun.texture_index = NUM_BODY_TEXTURES - 1;
    r->has_sun = 1;
}

extern "C" void render_set_trails(Renderer* h,
    uint32_t n_bodies,
    const uint32_t* trail_lengths,
    const double* trail_positions,
    const double* trail_colors,
    uint8_t show_trails)
{
    struct Renderer* r = (struct Renderer*)h;
    r->show_trails = (show_trails != 0);
    free(r->trail_vertices);
    r->trail_vertices = NULL;
    r->trail_vertex_count = 0;

    if (!r->show_trails || !trail_lengths || !trail_positions) return;

    uint32_t total_points = 0;
    for (uint32_t i = 0; i < n_bodies; i++) total_points += trail_lengths[i];
    if (total_points < 2) return;

    r->trail_vertices = (TrailVertex*)malloc(total_points * 2 * sizeof(TrailVertex));
    if (!r->trail_vertices) return;

    uint32_t offset = 0, vcount = 0;
    for (uint32_t i = 0; i < n_bodies; i++) {
        uint32_t len = trail_lengths[i];
        if (len < 2) { offset += len; continue; }
        float base_r = (float)trail_colors[i*4];
        float base_g = (float)trail_colors[i*4+1];
        float base_b = (float)trail_colors[i*4+2];
        uint32_t step = len > 200 ? len / 200 : 1;
        uint32_t j = 0;
        while (j + step < len) {
            uint32_t idx1 = offset + j, idx2 = offset + j + step;
            float x1, y1, x2, y2;
            camera_world_to_screen(&r->camera,
                trail_positions[idx1*3], trail_positions[idx1*3+1], trail_positions[idx1*3+2], &x1, &y1);
            camera_world_to_screen(&r->camera,
                trail_positions[idx2*3], trail_positions[idx2*3+1], trail_positions[idx2*3+2], &x2, &y2);
            float alpha = (float)j / (float)len;
            r->trail_vertices[vcount++] = {x1, y1, {base_r, base_g, base_b, alpha}};
            r->trail_vertices[vcount++] = {x2, y2, {base_r, base_g, base_b, alpha}};
            j += step;
        }
        offset += len;
    }
    r->trail_vertex_count = vcount;
}

extern "C" void render_set_spacetime(Renderer* h,
    uint8_t show_spacetime,
    const double* masses,
    const double* positions,
    uint32_t n_bodies)
{
    struct Renderer* r = (struct Renderer*)h;
    r->show_spacetime = (show_spacetime != 0);
    (void)masses; (void)positions; (void)n_bodies;
}

extern "C" void render_set_distance_line(Renderer* h,
    uint8_t has_line,
    double x1, double y1, double z1,
    double x2, double y2, double z2)
{
    struct Renderer* r = (struct Renderer*)h;
    r->has_dist_line = (has_line != 0);
    if (r->has_dist_line) {
        camera_world_to_screen(&r->camera, x1, y1, z1, &r->dist_line.x1, &r->dist_line.y1);
        camera_world_to_screen(&r->camera, x2, y2, z2, &r->dist_line.x2, &r->dist_line.y2);
    }
}

extern "C" void render_set_rt_mode(Renderer* h, uint8_t enabled) {
    struct Renderer* r = (struct Renderer*)h;
    r->rt_enabled = (enabled != 0);
    r->rt_frame_count = 0;
}

extern "C" void render_set_rt_quality(Renderer* h,
    uint32_t samples_per_frame, uint32_t max_bounces)
{
    struct Renderer* r = (struct Renderer*)h;
    r->rt_samples_per_frame = samples_per_frame;
    r->rt_max_bounces = max_bounces;
    r->rt_frame_count = 0;
}

extern "C" const uint8_t* render_frame(Renderer* h) {
    struct Renderer* r = (struct Renderer*)h;
    if (!r->pixels) return NULL;

    if (r->rt_enabled) {
        /* Build RT sphere array */
        uint32_t total_spheres = 0;
        RTSphere rt_spheres[MAX_RT_SPHERES];
        memset(rt_spheres, 0, sizeof(rt_spheres));

        for (uint32_t i = 0; i < r->num_bodies && total_spheres < MAX_RT_SPHERES; i++) {
            BodyData* b = &r->bodies[i];
            RTSphere* s = &rt_spheres[total_spheres];
            s->center[0] = b->screen_x; s->center[1] = b->screen_y; s->center[2] = 0.0f;
            s->radius = b->radius;
            s->color[0] = b->color[0]; s->color[1] = b->color[1];
            s->color[2] = b->color[2]; s->color[3] = b->color[3];
            s->material = (i >= 4 && i <= 7) ? 2 : 0;
            s->texture_index = b->texture_index;
            total_spheres++;
        }

        float sun_sx = 0, sun_sy = 0;
        if (r->has_sun && total_spheres < MAX_RT_SPHERES) {
            RTSphere* s = &rt_spheres[total_spheres];
            s->center[0] = r->sun.screen_x; s->center[1] = r->sun.screen_y; s->center[2] = 0.0f;
            s->radius = r->sun.radius;
            s->color[0] = r->sun.color[0]; s->color[1] = r->sun.color[1];
            s->color[2] = r->sun.color[2]; s->color[3] = r->sun.color[3];
            s->material = 1;
            s->texture_index = r->sun.texture_index;
            sun_sx = r->sun.screen_x;
            sun_sy = r->sun.screen_y;
            total_spheres++;
        }

        /* Upload sphere data */
        cudaMemcpy(r->d_spheres, rt_spheres, total_spheres * sizeof(RTSphere),
                   cudaMemcpyHostToDevice);

        /* Build camera uniform */
        RTCameraUniform cam;
        cam.width = (float)r->width;
        cam.height = (float)r->height;
        cam.frame_count = r->rt_frame_count;
        cam.num_spheres = total_spheres;
        cam.sun_screen_x = sun_sx;
        cam.sun_screen_y = sun_sy;
        cam.samples_per_frame = r->rt_samples_per_frame;
        cam.max_bounces = r->rt_max_bounces;

        /* Dispatch kernel */
        launch_raytrace_kernel(r->d_spheres, cam, r->d_accum, r->d_output,
                              r->tex_object, r->tex_width, r->tex_height);
        cudaDeviceSynchronize();

        r->rt_frame_count++;

        /* Readback */
        cudaMemcpy(r->pixels, r->d_output, r->pixel_size, cudaMemcpyDeviceToHost);
    } else {
        /* CPU fallback rasterization */
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

    /* CPU overlay compositing */
    if (r->show_trails && r->trail_vertices && r->trail_vertex_count >= 2) {
        for (uint32_t i = 0; i + 1 < r->trail_vertex_count; i += 2) {
            TrailVertex* v0 = &r->trail_vertices[i];
            TrailVertex* v1 = &r->trail_vertices[i+1];
            raster_draw_line(r->pixels, r->width, r->height,
                (int)v0->x, (int)v0->y, (int)v1->x, (int)v1->y,
                (uint8_t)(v0->color[0]*255), (uint8_t)(v0->color[1]*255),
                (uint8_t)(v0->color[2]*255), (uint8_t)(v0->color[3]*255));
        }
    }
    if (r->has_dist_line) {
        raster_draw_line(r->pixels, r->width, r->height,
            (int)r->dist_line.x1, (int)r->dist_line.y1,
            (int)r->dist_line.x2, (int)r->dist_line.y2,
            255, 255, 0, 200);
    }

    return r->pixels;
}

extern "C" void render_resize(Renderer* h, uint32_t width, uint32_t height) {
    struct Renderer* r = (struct Renderer*)h;
    if (width == 0 || height == 0) return;
    r->width = width;
    r->height = height;
    r->camera.width = width;
    r->camera.height = height;
    allocate_gpu_buffers(r);
    r->rt_frame_count = 0;
}

extern "C" void render_free(Renderer* h) {
    if (!h) return;
    struct Renderer* r = (struct Renderer*)h;
    cudaFree(r->d_spheres);
    cudaFree(r->d_accum);
    cudaFree(r->d_output);
    if (r->tex_object) cudaDestroyTextureObject(r->tex_object);
    if (r->tex_array)  cudaFreeArray(r->tex_array);
    free(r->pixels);
    free(r->trail_vertices);
    free(r);
}
