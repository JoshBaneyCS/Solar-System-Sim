/*
 * renderer.m — Metal GPU renderer host code (Objective-C).
 *
 * Implements the native_render.h C API using Apple Metal compute pipeline.
 * Manages device, buffers, texture atlas, compute pipeline, and pixel readback.
 */
#import <Metal/Metal.h>
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>

#include "native_render.h"
#include "sphere.h"
#include "camera.h"
#include "texture_loader.h"
#include "rasterizer.h"

/* ---- Internal renderer state ---- */

typedef struct {
    float screen_x;
    float screen_y;
    float radius;
    float color[4];
    int   texture_index;
} BodyData;

typedef struct {
    float x1, y1, x2, y2;
} DistLineData;

/* Trail vertex (screen-space line segment endpoint) */
typedef struct {
    float x, y;
    float color[4]; /* rgba */
} TrailVertex;

struct Renderer {
    /* Metal objects */
    id<MTLDevice>              device;
    id<MTLCommandQueue>        command_queue;
    id<MTLComputePipelineState> pipeline;
    id<MTLLibrary>             library;

    /* GPU buffers */
    id<MTLBuffer> sphere_buffer;
    id<MTLBuffer> camera_buffer;
    id<MTLBuffer> accum_buffer;
    id<MTLBuffer> output_buffer;

    /* Texture atlas */
    id<MTLTexture> texture_atlas;
    id<MTLSamplerState> sampler;

    /* Dimensions */
    uint32_t width;
    uint32_t height;

    /* Camera */
    NativeCamera camera;

    /* Body data */
    BodyData bodies[MAX_RT_SPHERES];
    uint32_t num_bodies;
    BodyData sun;
    int      has_sun;

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
    int          has_dist_line;

    /* Spacetime grid (stored for CPU overlay) */
    int show_spacetime;

    /* Host readback buffer */
    uint8_t* pixels;
    size_t   pixel_size;
};

/* ---- Helper: find metallib or metal source path ---- */

static NSString* find_metallib_path(void) {
    NSBundle* bundle = [NSBundle mainBundle];
    NSString* path = [bundle pathForResource:@"raytracer" ofType:@"metallib"];
    if (path) return path;

    NSString* exe = [[NSProcessInfo processInfo] arguments][0];
    NSString* dir = [exe stringByDeletingLastPathComponent];

    path = [dir stringByAppendingPathComponent:@"raytracer.metallib"];
    if ([[NSFileManager defaultManager] fileExistsAtPath:path]) return path;

    path = @"native_gpu/metal/raytracer.metallib";
    if ([[NSFileManager defaultManager] fileExistsAtPath:path]) return path;

    return nil;
}

static NSString* find_metal_source_path(void) {
    NSString* exe = [[NSProcessInfo processInfo] arguments][0];
    NSString* dir = [exe stringByDeletingLastPathComponent];

    NSString* path = [dir stringByAppendingPathComponent:@"raytracer.metal"];
    if ([[NSFileManager defaultManager] fileExistsAtPath:path]) return path;

    path = @"native_gpu/metal/raytracer.metal";
    if ([[NSFileManager defaultManager] fileExistsAtPath:path]) return path;

    return nil;
}

/* Load Metal library: try pre-compiled metallib first, fall back to source compilation */
static id<MTLLibrary> load_metal_library(id<MTLDevice> device) {
    NSError* error = nil;

    /* 1. Try pre-compiled metallib */
    NSString* libPath = find_metallib_path();
    if (libPath) {
        NSURL* url = [NSURL fileURLWithPath:libPath];
        id<MTLLibrary> lib = [device newLibraryWithURL:url error:&error];
        if (lib) return lib;
        NSLog(@"Metal: Failed to load metallib %@: %@", libPath, error);
    }

    /* 2. Try app bundle default library */
    id<MTLLibrary> lib = [device newDefaultLibrary];
    if (lib) return lib;

    /* 3. Compile from .metal source at runtime */
    NSString* srcPath = find_metal_source_path();
    if (srcPath) {
        NSString* source = [NSString stringWithContentsOfFile:srcPath
                                                    encoding:NSUTF8StringEncoding
                                                       error:&error];
        if (source) {
            MTLCompileOptions* opts = [[MTLCompileOptions alloc] init];
            if (@available(macOS 15.0, *)) {
                opts.mathMode = MTLMathModeFast;
            } else {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
                opts.fastMathEnabled = YES;
#pragma clang diagnostic pop
            }
            lib = [device newLibraryWithSource:source options:opts error:&error];
            if (lib) {
                NSLog(@"Metal: Compiled shader from source: %@", srcPath);
                return lib;
            }
            NSLog(@"Metal: Failed to compile shader source: %@", error);
        } else {
            NSLog(@"Metal: Failed to read shader source %@: %@", srcPath, error);
        }
    }

    return nil;
}

/* ---- Helper: create texture atlas on GPU ---- */

static id<MTLTexture> create_metal_texture_atlas(id<MTLDevice> device,
    id<MTLCommandQueue> queue, TextureAtlasData* atlas)
{
    MTLTextureDescriptor* desc = [[MTLTextureDescriptor alloc] init];
    desc.textureType = MTLTextureType2DArray;
    desc.pixelFormat = MTLPixelFormatRGBA8Unorm_sRGB;
    desc.width = atlas->width;
    desc.height = atlas->height;
    desc.arrayLength = atlas->layers;
    desc.mipmapLevelCount = 1;
    desc.usage = MTLTextureUsageShaderRead;

    id<MTLTexture> texture = [device newTextureWithDescriptor:desc];
    if (!texture) return nil;

    size_t bytes_per_row = atlas->width * 4;
    size_t bytes_per_layer = bytes_per_row * atlas->height;

    for (uint32_t i = 0; i < atlas->layers; i++) {
        MTLRegion region = MTLRegionMake2D(0, 0, atlas->width, atlas->height);
        [texture replaceRegion:region
                   mipmapLevel:0
                         slice:i
                     withBytes:atlas->data + i * bytes_per_layer
                   bytesPerRow:bytes_per_row
                 bytesPerImage:bytes_per_layer];
    }

    return texture;
}

/* ---- Helper: allocate/reallocate output and accum buffers ---- */

static void allocate_render_buffers(struct Renderer* r) {
    size_t pixel_count = (size_t)r->width * r->height;
    size_t output_size = pixel_count * 4; /* RGBA8 */
    size_t accum_size = pixel_count * sizeof(float) * 4; /* float4 per pixel */

    r->output_buffer = [r->device newBufferWithLength:output_size
                                              options:MTLResourceStorageModeShared];
    r->accum_buffer = [r->device newBufferWithLength:accum_size
                                             options:MTLResourceStorageModeShared];

    /* Clear accumulation buffer */
    memset([r->accum_buffer contents], 0, accum_size);

    /* Host readback buffer */
    free(r->pixels);
    r->pixel_size = output_size;
    r->pixels = (uint8_t*)malloc(output_size);
    if (r->pixels) memset(r->pixels, 0, output_size);
}

/* ---- Public API implementation ---- */

Renderer* render_create(uint32_t width, uint32_t height) {
    return render_create_with_textures(width, height, NULL);
}

Renderer* render_create_with_textures(uint32_t width, uint32_t height,
    const char* asset_dir)
{
    @autoreleasepool {
        struct Renderer* r = (struct Renderer*)calloc(1, sizeof(struct Renderer));
        if (!r) return NULL;

        r->width = width;
        r->height = height;
        r->rt_samples_per_frame = 1;
        r->rt_max_bounces = 4;
        camera_init(&r->camera, width, height);

        /* Create Metal device */
        r->device = MTLCreateSystemDefaultDevice();
        if (!r->device) {
            free(r);
            return NULL;
        }

        r->command_queue = [r->device newCommandQueue];
        if (!r->command_queue) {
            free(r);
            return NULL;
        }

        /* Load Metal shader library (pre-compiled or source compilation) */
        NSError* error = nil;
        r->library = load_metal_library(r->device);
        if (!r->library) {
            NSLog(@"Metal: Failed to load shader library");
            free(r);
            return NULL;
        }

        /* Create compute pipeline */
        id<MTLFunction> kernel = [r->library newFunctionWithName:@"raytrace"];
        if (!kernel) {
            NSLog(@"Metal: Failed to find 'raytrace' kernel function");
            free(r);
            return NULL;
        }

        r->pipeline = [r->device newComputePipelineStateWithFunction:kernel error:&error];
        if (!r->pipeline) {
            NSLog(@"Metal: Failed to create compute pipeline: %@", error);
            free(r);
            return NULL;
        }

        /* Create sphere and camera buffers */
        r->sphere_buffer = [r->device newBufferWithLength:MAX_RT_SPHERES * sizeof(RTSphere)
                                                  options:MTLResourceStorageModeShared];
        r->camera_buffer = [r->device newBufferWithLength:sizeof(RTCameraUniform)
                                                  options:MTLResourceStorageModeShared];

        /* Allocate output and accumulation buffers */
        allocate_render_buffers(r);

        /* Load texture atlas */
        TextureAtlasData* atlas = NULL;
        if (asset_dir && strlen(asset_dir) > 0) {
            atlas = texture_atlas_load(asset_dir);
        }
        if (!atlas) {
            atlas = texture_atlas_fallback();
        }

        if (atlas) {
            r->texture_atlas = create_metal_texture_atlas(r->device, r->command_queue, atlas);
            texture_atlas_free(atlas);
        }

        /* Create sampler */
        MTLSamplerDescriptor* samplerDesc = [[MTLSamplerDescriptor alloc] init];
        samplerDesc.sAddressMode = MTLSamplerAddressModeRepeat;
        samplerDesc.tAddressMode = MTLSamplerAddressModeClampToEdge;
        samplerDesc.magFilter = MTLSamplerMinMagFilterLinear;
        samplerDesc.minFilter = MTLSamplerMinMagFilterLinear;
        samplerDesc.mipFilter = MTLSamplerMipFilterLinear;
        r->sampler = [r->device newSamplerStateWithDescriptor:samplerDesc];

        return r;
    }
}

void render_set_camera(Renderer* h,
    double zoom, double pan_x, double pan_y,
    double rot_x, double rot_y, double rot_z,
    uint8_t use_3d,
    double follow_x, double follow_y, double follow_z)
{
    struct Renderer* r = h;
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

void render_set_bodies(Renderer* h,
    uint32_t n_bodies,
    const double* positions,
    const double* colors,
    const double* radii,
    const double* sun_pos,
    const double* sun_color,
    double sun_radius)
{
    struct Renderer* r = h;
    uint32_t n = n_bodies < MAX_RT_SPHERES - 1 ? n_bodies : MAX_RT_SPHERES - 1;
    r->num_bodies = n;

    for (uint32_t i = 0; i < n; i++) {
        float sx, sy;
        camera_world_to_screen(&r->camera,
            positions[i*3], positions[i*3+1], positions[i*3+2],
            &sx, &sy);
        r->bodies[i].screen_x = sx;
        r->bodies[i].screen_y = sy;
        r->bodies[i].radius = (float)radii[i];
        r->bodies[i].color[0] = (float)colors[i*4];
        r->bodies[i].color[1] = (float)colors[i*4+1];
        r->bodies[i].color[2] = (float)colors[i*4+2];
        r->bodies[i].color[3] = (float)colors[i*4+3];
        r->bodies[i].texture_index = (int)i;
    }

    /* Sun */
    float sx, sy;
    camera_world_to_screen(&r->camera, sun_pos[0], sun_pos[1], sun_pos[2], &sx, &sy);
    r->sun.screen_x = sx;
    r->sun.screen_y = sy;
    r->sun.radius = (float)sun_radius;
    r->sun.color[0] = (float)sun_color[0];
    r->sun.color[1] = (float)sun_color[1];
    r->sun.color[2] = (float)sun_color[2];
    r->sun.color[3] = (float)sun_color[3];
    r->sun.texture_index = NUM_BODY_TEXTURES - 1; /* sun is last layer */
    r->has_sun = 1;
}

void render_set_trails(Renderer* h,
    uint32_t n_bodies,
    const uint32_t* trail_lengths,
    const double* trail_positions,
    const double* trail_colors,
    uint8_t show_trails)
{
    struct Renderer* r = h;
    r->show_trails = (show_trails != 0);

    free(r->trail_vertices);
    r->trail_vertices = NULL;
    r->trail_vertex_count = 0;

    if (!r->show_trails || !trail_lengths || !trail_positions) return;

    /* Count total trail points to allocate vertices */
    uint32_t total_points = 0;
    for (uint32_t i = 0; i < n_bodies; i++) {
        total_points += trail_lengths[i];
    }
    if (total_points < 2) return;

    /* Allocate worst-case vertex buffer */
    r->trail_vertices = (TrailVertex*)malloc(total_points * 2 * sizeof(TrailVertex));
    if (!r->trail_vertices) return;

    uint32_t offset = 0;
    uint32_t vcount = 0;

    for (uint32_t i = 0; i < n_bodies; i++) {
        uint32_t len = trail_lengths[i];
        if (len < 2) { offset += len; continue; }

        float base_r = (float)trail_colors[i*4];
        float base_g = (float)trail_colors[i*4+1];
        float base_b = (float)trail_colors[i*4+2];

        uint32_t step = len > 200 ? len / 200 : 1;
        uint32_t j = 0;
        while (j + step < len) {
            uint32_t idx1 = offset + j;
            uint32_t idx2 = offset + j + step;
            float x1, y1, x2, y2;
            camera_world_to_screen(&r->camera,
                trail_positions[idx1*3], trail_positions[idx1*3+1], trail_positions[idx1*3+2],
                &x1, &y1);
            camera_world_to_screen(&r->camera,
                trail_positions[idx2*3], trail_positions[idx2*3+1], trail_positions[idx2*3+2],
                &x2, &y2);
            float alpha = (float)j / (float)len;

            r->trail_vertices[vcount].x = x1;
            r->trail_vertices[vcount].y = y1;
            r->trail_vertices[vcount].color[0] = base_r;
            r->trail_vertices[vcount].color[1] = base_g;
            r->trail_vertices[vcount].color[2] = base_b;
            r->trail_vertices[vcount].color[3] = alpha;
            vcount++;

            r->trail_vertices[vcount].x = x2;
            r->trail_vertices[vcount].y = y2;
            r->trail_vertices[vcount].color[0] = base_r;
            r->trail_vertices[vcount].color[1] = base_g;
            r->trail_vertices[vcount].color[2] = base_b;
            r->trail_vertices[vcount].color[3] = alpha;
            vcount++;

            j += step;
        }
        offset += len;
    }
    r->trail_vertex_count = vcount;
}

void render_set_spacetime(Renderer* h,
    uint8_t show_spacetime,
    const double* masses,
    const double* positions,
    uint32_t n_bodies)
{
    struct Renderer* r = h;
    r->show_spacetime = (show_spacetime != 0);
    /* Spacetime grid is deferred — overlays are CPU-composited in render_frame */
    (void)masses;
    (void)positions;
    (void)n_bodies;
}

void render_set_distance_line(Renderer* h,
    uint8_t has_line,
    double x1, double y1, double z1,
    double x2, double y2, double z2)
{
    struct Renderer* r = h;
    r->has_dist_line = (has_line != 0);
    if (r->has_dist_line) {
        camera_world_to_screen(&r->camera, x1, y1, z1, &r->dist_line.x1, &r->dist_line.y1);
        camera_world_to_screen(&r->camera, x2, y2, z2, &r->dist_line.x2, &r->dist_line.y2);
    }
}

void render_set_rt_mode(Renderer* h, uint8_t enabled) {
    struct Renderer* r = h;
    r->rt_enabled = (enabled != 0);
    r->rt_frame_count = 0;
}

void render_set_rt_quality(Renderer* h,
    uint32_t samples_per_frame, uint32_t max_bounces)
{
    struct Renderer* r = h;
    r->rt_samples_per_frame = samples_per_frame;
    r->rt_max_bounces = max_bounces;
    r->rt_frame_count = 0;
}

const uint8_t* render_frame(Renderer* h) {
    @autoreleasepool {
        struct Renderer* r = h;
        if (!r->pixels || !r->pipeline) return NULL;

        if (r->rt_enabled) {
            /* Build RTSphere array */
            uint32_t total_spheres = 0;
            RTSphere rt_spheres[MAX_RT_SPHERES];
            memset(rt_spheres, 0, sizeof(rt_spheres));

            /* Add planet bodies */
            for (uint32_t i = 0; i < r->num_bodies && total_spheres < MAX_RT_SPHERES; i++) {
                BodyData* b = &r->bodies[i];
                RTSphere* s = &rt_spheres[total_spheres];
                s->center[0] = b->screen_x;
                s->center[1] = b->screen_y;
                s->center[2] = 0.0f;
                s->radius = b->radius;
                s->color[0] = b->color[0];
                s->color[1] = b->color[1];
                s->color[2] = b->color[2];
                s->color[3] = b->color[3];
                /* Determine material: gas giants (Jupiter=4, Saturn=5, Uranus=6, Neptune=7) are glossy */
                if (i >= 4 && i <= 7) {
                    s->material = 2; /* glossy */
                } else {
                    s->material = 0; /* diffuse */
                }
                s->texture_index = b->texture_index;
                total_spheres++;
            }

            /* Add sun */
            float sun_sx = 0, sun_sy = 0;
            if (r->has_sun) {
                RTSphere* s = &rt_spheres[total_spheres];
                s->center[0] = r->sun.screen_x;
                s->center[1] = r->sun.screen_y;
                s->center[2] = 0.0f;
                s->radius = r->sun.radius;
                s->color[0] = r->sun.color[0];
                s->color[1] = r->sun.color[1];
                s->color[2] = r->sun.color[2];
                s->color[3] = r->sun.color[3];
                s->material = 1; /* emissive */
                s->texture_index = r->sun.texture_index;
                sun_sx = r->sun.screen_x;
                sun_sy = r->sun.screen_y;
                total_spheres++;
            }

            /* Upload sphere data */
            memcpy([r->sphere_buffer contents], rt_spheres,
                   total_spheres * sizeof(RTSphere));

            /* Build camera uniform */
            RTCameraUniform cam_uniform;
            cam_uniform.width = (float)r->width;
            cam_uniform.height = (float)r->height;
            cam_uniform.frame_count = r->rt_frame_count;
            cam_uniform.num_spheres = total_spheres;
            cam_uniform.sun_screen_x = sun_sx;
            cam_uniform.sun_screen_y = sun_sy;
            cam_uniform.samples_per_frame = r->rt_samples_per_frame;
            cam_uniform.max_bounces = r->rt_max_bounces;
            memcpy([r->camera_buffer contents], &cam_uniform, sizeof(cam_uniform));

            /* Dispatch compute kernel */
            id<MTLCommandBuffer> cmd = [r->command_queue commandBuffer];
            id<MTLComputeCommandEncoder> encoder = [cmd computeCommandEncoder];

            [encoder setComputePipelineState:r->pipeline];
            [encoder setBuffer:r->sphere_buffer offset:0 atIndex:0];
            [encoder setBuffer:r->camera_buffer offset:0 atIndex:1];
            [encoder setBuffer:r->accum_buffer  offset:0 atIndex:2];
            [encoder setBuffer:r->output_buffer offset:0 atIndex:3];

            if (r->texture_atlas) {
                [encoder setTexture:r->texture_atlas atIndex:0];
            }
            if (r->sampler) {
                [encoder setSamplerState:r->sampler atIndex:0];
            }

            MTLSize threadgroups = MTLSizeMake((r->width + 7) / 8, (r->height + 7) / 8, 1);
            MTLSize threads_per_group = MTLSizeMake(8, 8, 1);
            [encoder dispatchThreadgroups:threadgroups threadsPerThreadgroup:threads_per_group];
            [encoder endEncoding];

            [cmd commit];
            [cmd waitUntilCompleted];

            r->rt_frame_count++;

            /* Copy GPU output to host buffer */
            memcpy(r->pixels, [r->output_buffer contents], r->pixel_size);
        } else {
            /* Non-RT mode: render simple circle rasterization on CPU */
            memset(r->pixels, 13, r->pixel_size); /* dark background */
            /* Set alpha channel */
            for (size_t i = 3; i < r->pixel_size; i += 4) {
                r->pixels[i] = 255;
            }
            /* Background: rgb(5, 5, 15) */
            for (size_t i = 0; i < r->pixel_size; i += 4) {
                r->pixels[i]   = 5;
                r->pixels[i+1] = 5;
                r->pixels[i+2] = 15;
                r->pixels[i+3] = 255;
            }

            /* Draw sun */
            if (r->has_sun) {
                raster_draw_circle(r->pixels, r->width, r->height,
                    (int)r->sun.screen_x, (int)r->sun.screen_y,
                    (int)r->sun.radius,
                    (uint8_t)(r->sun.color[0] * 255),
                    (uint8_t)(r->sun.color[1] * 255),
                    (uint8_t)(r->sun.color[2] * 255), 255);
            }

            /* Draw bodies */
            for (uint32_t i = 0; i < r->num_bodies; i++) {
                BodyData* b = &r->bodies[i];
                raster_draw_circle(r->pixels, r->width, r->height,
                    (int)b->screen_x, (int)b->screen_y,
                    (int)b->radius,
                    (uint8_t)(b->color[0] * 255),
                    (uint8_t)(b->color[1] * 255),
                    (uint8_t)(b->color[2] * 255), 255);
            }
        }

        /* Composite CPU overlays */

        /* Trails */
        if (r->show_trails && r->trail_vertices && r->trail_vertex_count >= 2) {
            for (uint32_t i = 0; i + 1 < r->trail_vertex_count; i += 2) {
                TrailVertex* v0 = &r->trail_vertices[i];
                TrailVertex* v1 = &r->trail_vertices[i+1];
                raster_draw_line(r->pixels, r->width, r->height,
                    (int)v0->x, (int)v0->y, (int)v1->x, (int)v1->y,
                    (uint8_t)(v0->color[0] * 255),
                    (uint8_t)(v0->color[1] * 255),
                    (uint8_t)(v0->color[2] * 255),
                    (uint8_t)(v0->color[3] * 255));
            }
        }

        /* Distance line */
        if (r->has_dist_line) {
            raster_draw_line(r->pixels, r->width, r->height,
                (int)r->dist_line.x1, (int)r->dist_line.y1,
                (int)r->dist_line.x2, (int)r->dist_line.y2,
                255, 255, 0, 200); /* yellow */
        }

        return r->pixels;
    }
}

void render_resize(Renderer* h, uint32_t width, uint32_t height) {
    struct Renderer* r = h;
    if (width == 0 || height == 0) return;
    r->width = width;
    r->height = height;
    r->camera.width = width;
    r->camera.height = height;
    allocate_render_buffers(r);
    r->rt_frame_count = 0;
}

void render_free(Renderer* h) {
    if (!h) return;
    struct Renderer* r = h;
    free(r->pixels);
    free(r->trail_vertices);
    /* ARC handles Metal object release */
    free(r);
}
