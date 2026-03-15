/*
 * raytracer.metal — Native Metal compute shader for sphere ray tracing.
 *
 * Direct port of the WGSL shader in crates/render_core/src/raytracer.rs.
 * Provides: sphere intersection, shadow rays, ambient occlusion (4 samples),
 * glossy reflections, texture sampling, progressive accumulation.
 *
 * Dispatched as 8x8 threadgroups, one thread per pixel.
 */
#include <metal_stdlib>
using namespace metal;

/* Must match RTSphere in sphere.h (48 bytes) */
struct Sphere {
    float3 center;
    float  radius;
    float4 color;
    uint   material;
    int    texture_index;
    uint   _pad2;
    uint   _pad3;
};

/* Must match RTCameraUniform in sphere.h (32 bytes) */
struct Camera {
    float  width;
    float  height;
    uint   frame_count;
    uint   num_spheres;
    float  sun_screen_x;
    float  sun_screen_y;
    uint   samples_per_frame;
    uint   max_bounces;
};

/* PCG hash for pseudo-random number generation */
static uint pcg_hash(uint input) {
    uint state = input * 747796405u + 2891336453u;
    uint word = ((state >> ((state >> 28u) + 4u)) ^ state) * 277803737u;
    return (word >> 22u) ^ word;
}

static float rand_float(thread uint& seed) {
    seed = pcg_hash(seed);
    return float(seed) / 4294967295.0f;
}

/* Ray-sphere intersection. Returns hit distance or -1. */
static float intersect_sphere(float3 ro, float3 rd, float3 center, float radius) {
    float3 oc = ro - center;
    float b = dot(oc, rd);
    float c = dot(oc, oc) - radius * radius;
    float discriminant = b * b - c;
    if (discriminant < 0.0f) return -1.0f;
    float t = -b - sqrt(discriminant);
    if (t > 0.0f) return t;
    return -1.0f;
}

/* Cosine-weighted hemisphere sampling from a normal */
static float3 build_tangent_frame(float3 n, float r1, float r2) {
    float phi = 6.283185f * r1;
    float cos_theta = sqrt(r2);
    float sin_theta = sqrt(1.0f - r2);

    float3 t;
    if (abs(n.x) > 0.9f) {
        t = float3(0.0f, 1.0f, 0.0f);
    } else {
        t = float3(1.0f, 0.0f, 0.0f);
    }
    float3 b = normalize(cross(n, t));
    float3 tangent = cross(b, n);

    return normalize(tangent * cos(phi) * sin_theta + b * sin(phi) * sin_theta + n * cos_theta);
}

/* Compute spherical UV from a surface normal for texture sampling */
static float2 sphere_uv(float3 normal) {
    float u = 0.5f + atan2(normal.x, -normal.z) / 6.283185f;
    float v = 0.5f - asin(clamp(normal.y, -1.0f, 1.0f)) / 3.141593f;
    return float2(u, v);
}

kernel void raytrace(
    device const Sphere*    spheres       [[buffer(0)]],
    constant Camera&        cam           [[buffer(1)]],
    device float4*          accum         [[buffer(2)]],
    device uchar4*          output        [[buffer(3)]],
    texture2d_array<float>  texture_atlas [[texture(0)]],
    sampler                 tex_sampler   [[sampler(0)]],
    uint2                   gid           [[thread_position_in_grid]])
{
    uint px = gid.x;
    uint py = gid.y;
    if (px >= uint(cam.width) || py >= uint(cam.height)) return;

    /* Load spheres into threadgroup memory for faster access */
    threadgroup Sphere s_spheres[16];
    uint tid = gid.x % 8 + (gid.y % 8) * 8; /* local thread index in 8x8 group */
    if (tid < cam.num_spheres && tid < 16) {
        s_spheres[tid] = spheres[tid];
    }
    threadgroup_barrier(mem_flags::mem_threadgroup);

    uint seed = px + py * uint(cam.width) + cam.frame_count * 196613u;

    /* Orthographic ray: cast in +z direction */
    float3 ray_origin = float3(float(px) + 0.5f, float(py) + 0.5f, -1000.0f);
    float3 ray_dir = float3(0.0f, 0.0f, 1.0f);

    /* Find nearest sphere intersection */
    float min_t = 1e20f;
    int hit_idx = -1;

    for (uint i = 0; i < cam.num_spheres && i < 16; i++) {
        float t = intersect_sphere(ray_origin, ray_dir, s_spheres[i].center, s_spheres[i].radius);
        if (t > 0.0f && t < min_t) {
            min_t = t;
            hit_idx = int(i);
        }
    }

    float3 pixel_color;
    float3 bg = float3(5.0f / 255.0f, 5.0f / 255.0f, 15.0f / 255.0f);

    if (hit_idx < 0) {
        pixel_color = bg;
    } else {
        Sphere sphere = s_spheres[uint(hit_idx)];
        float3 hit_point = ray_origin + ray_dir * min_t;
        float3 hit_normal = normalize(hit_point - sphere.center);

        /* Get base color (from texture if available, otherwise solid) */
        float3 base_color = sphere.color.rgb;
        if (sphere.texture_index >= 0) {
            float2 uv = sphere_uv(hit_normal);
            float4 tex_color = texture_atlas.sample(tex_sampler, uv, uint(sphere.texture_index), level(0.0f));
            base_color = tex_color.rgb;
        }

        if (sphere.material == 1u) {
            /* Emissive (sun) — self-lit with glow */
            float glow = 1.0f + 0.3f * max(0.0f, 1.0f - length(hit_point.xy - sphere.center.xy) / sphere.radius);
            pixel_color = base_color * glow;
        } else {
            /* Light direction toward sun */
            float3 sun_pos = float3(cam.sun_screen_x, cam.sun_screen_y, 0.0f);
            float3 to_light = normalize(sun_pos - hit_point);
            float ndotl = max(dot(hit_normal, to_light), 0.0f);

            /* Shadow test */
            bool in_shadow = false;
            float3 shadow_origin = hit_point + hit_normal * 0.1f;
            for (uint i = 0; i < cam.num_spheres && i < 16; i++) {
                if (i == uint(hit_idx)) continue;
                float t = intersect_sphere(shadow_origin, to_light, s_spheres[i].center, s_spheres[i].radius);
                if (t > 0.0f) {
                    in_shadow = true;
                    break;
                }
            }

            /* Ambient occlusion (4 hemisphere samples) */
            float ao_factor = 0.0f;
            uint ao_samples = 4u;
            float3 ao_origin = hit_point + hit_normal * 0.1f;
            float ao_range = sphere.radius * 5.0f;

            for (uint s = 0; s < ao_samples; s++) {
                float r1 = rand_float(seed);
                float r2 = rand_float(seed);
                float3 ao_dir = build_tangent_frame(hit_normal, r1, r2);

                bool occluded = false;
                for (uint i = 0; i < cam.num_spheres && i < 16; i++) {
                    if (i == uint(hit_idx)) continue;
                    float t = intersect_sphere(ao_origin, ao_dir, s_spheres[i].center, s_spheres[i].radius);
                    if (t > 0.0f && t < ao_range) {
                        occluded = true;
                        break;
                    }
                }
                if (!occluded) ao_factor += 1.0f;
            }
            ao_factor = ao_factor / float(ao_samples);

            float ambient = 0.08f;
            float diffuse = 0.0f;
            if (!in_shadow) diffuse = ndotl;

            pixel_color = base_color * (ambient + diffuse * 0.9f) * ao_factor;

            /* Glossy reflection for gas giants (material == 2) */
            if (sphere.material == 2u) {
                float3 reflect_dir = reflect(ray_dir, hit_normal);
                float pr1 = rand_float(seed);
                float pr2 = rand_float(seed);
                float3 glossy_dir = normalize(reflect_dir + build_tangent_frame(reflect_dir, pr1, pr2) * 0.3f);

                float3 ref_color = bg;
                float ref_min_t = 1e20f;
                for (uint i = 0; i < cam.num_spheres && i < 16; i++) {
                    if (i == uint(hit_idx)) continue;
                    float t = intersect_sphere(shadow_origin, glossy_dir, s_spheres[i].center, s_spheres[i].radius);
                    if (t > 0.0f && t < ref_min_t) {
                        ref_min_t = t;
                        ref_color = s_spheres[i].color.rgb;
                    }
                }
                pixel_color = mix(pixel_color, ref_color, 0.15f);
            }
        }
    }

    /* sRGB gamma correction */
    pixel_color = pow(clamp(pixel_color, float3(0.0f), float3(1.0f)), float3(1.0f / 2.2f));

    /* Progressive accumulation */
    uint idx = py * uint(cam.width) + px;
    float4 prev = accum[idx];
    float weight = 1.0f / float(cam.frame_count + 1u);
    float3 accumulated = mix(prev.rgb, pixel_color, weight);
    accum[idx] = float4(accumulated, 1.0f);

    /* Write output as RGBA8 */
    output[idx] = uchar4(
        uchar(accumulated.r * 255.0f),
        uchar(accumulated.g * 255.0f),
        uchar(accumulated.b * 255.0f),
        255
    );
}
