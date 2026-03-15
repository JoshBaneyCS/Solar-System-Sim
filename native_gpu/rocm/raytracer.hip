/*
 * raytracer.hip — Native HIP compute kernel for sphere ray tracing (AMD ROCm).
 *
 * Port of the CUDA kernel in ../cuda/raytracer.cu using HIP API.
 * HIP is source-compatible with CUDA for compute kernels with minor API changes.
 */
#include <hip/hip_runtime.h>
#include <stdint.h>
#include <math.h>

extern "C" {
#include "sphere.h"
}

/* ---- Device helper functions ---- */

__device__ static uint32_t pcg_hash(uint32_t input) {
    uint32_t state = input * 747796405u + 2891336453u;
    uint32_t word = ((state >> ((state >> 28u) + 4u)) ^ state) * 277803737u;
    return (word >> 22u) ^ word;
}

__device__ static float rand_float(uint32_t& seed) {
    seed = pcg_hash(seed);
    return (float)seed / 4294967295.0f;
}

__device__ static float intersect_sphere(float3 ro, float3 rd, float3 center, float radius) {
    float3 oc = make_float3(ro.x - center.x, ro.y - center.y, ro.z - center.z);
    float b = oc.x * rd.x + oc.y * rd.y + oc.z * rd.z;
    float c = oc.x * oc.x + oc.y * oc.y + oc.z * oc.z - radius * radius;
    float discriminant = b * b - c;
    if (discriminant < 0.0f) return -1.0f;
    float t = -b - sqrtf(discriminant);
    if (t > 0.0f) return t;
    return -1.0f;
}

__device__ static float3 normalize3(float3 v) {
    float len = sqrtf(v.x * v.x + v.y * v.y + v.z * v.z);
    if (len < 1e-10f) return make_float3(0, 0, 0);
    float inv = 1.0f / len;
    return make_float3(v.x * inv, v.y * inv, v.z * inv);
}

__device__ static float dot3(float3 a, float3 b) {
    return a.x * b.x + a.y * b.y + a.z * b.z;
}

__device__ static float3 cross3(float3 a, float3 b) {
    return make_float3(
        a.y * b.z - a.z * b.y,
        a.z * b.x - a.x * b.z,
        a.x * b.y - a.y * b.x
    );
}

__device__ static float length2(float2 v) {
    return sqrtf(v.x * v.x + v.y * v.y);
}

__device__ static float3 build_tangent_frame(float3 n, float r1, float r2) {
    float phi = 6.283185f * r1;
    float cos_theta = sqrtf(r2);
    float sin_theta = sqrtf(1.0f - r2);

    float3 t;
    if (fabsf(n.x) > 0.9f) {
        t = make_float3(0.0f, 1.0f, 0.0f);
    } else {
        t = make_float3(1.0f, 0.0f, 0.0f);
    }
    float3 b = normalize3(cross3(n, t));
    float3 tangent = cross3(b, n);

    return normalize3(make_float3(
        tangent.x * cosf(phi) * sin_theta + b.x * sinf(phi) * sin_theta + n.x * cos_theta,
        tangent.y * cosf(phi) * sin_theta + b.y * sinf(phi) * sin_theta + n.y * cos_theta,
        tangent.z * cosf(phi) * sin_theta + b.z * sinf(phi) * sin_theta + n.z * cos_theta
    ));
}

__device__ static float2 sphere_uv(float3 normal) {
    float u = 0.5f + atan2f(normal.x, -normal.z) / 6.283185f;
    float v = 0.5f - asinf(fminf(fmaxf(normal.y, -1.0f), 1.0f)) / 3.141593f;
    return make_float2(u, v);
}

__device__ static float3 reflect3(float3 v, float3 n) {
    float d = 2.0f * dot3(v, n);
    return make_float3(v.x - d * n.x, v.y - d * n.y, v.z - d * n.z);
}

/* ---- Main ray tracing kernel ---- */

__global__ void raytrace_kernel(
    const RTSphere* __restrict__ spheres,
    RTCameraUniform cam,
    float4* __restrict__ accum,
    uint8_t* __restrict__ output,
    hipTextureObject_t atlas,
    uint32_t tex_width,
    uint32_t tex_height)
{
    uint32_t px = blockIdx.x * blockDim.x + threadIdx.x;
    uint32_t py = blockIdx.y * blockDim.y + threadIdx.y;
    if (px >= (uint32_t)cam.width || py >= (uint32_t)cam.height) return;

    __shared__ RTSphere s_spheres[MAX_RT_SPHERES];
    uint32_t tid = threadIdx.x + threadIdx.y * blockDim.x;
    if (tid < cam.num_spheres && tid < MAX_RT_SPHERES) {
        s_spheres[tid] = spheres[tid];
    }
    __syncthreads();

    uint32_t seed = px + py * (uint32_t)cam.width + cam.frame_count * 196613u;

    float3 ray_origin = make_float3((float)px + 0.5f, (float)py + 0.5f, -1000.0f);
    float3 ray_dir = make_float3(0.0f, 0.0f, 1.0f);

    float min_t = 1e20f;
    int hit_idx = -1;

    for (uint32_t i = 0; i < cam.num_spheres && i < MAX_RT_SPHERES; i++) {
        float3 center = make_float3(s_spheres[i].center[0], s_spheres[i].center[1], s_spheres[i].center[2]);
        float t = intersect_sphere(ray_origin, ray_dir, center, s_spheres[i].radius);
        if (t > 0.0f && t < min_t) { min_t = t; hit_idx = (int)i; }
    }

    float3 pixel_color;
    float3 bg = make_float3(5.0f/255.0f, 5.0f/255.0f, 15.0f/255.0f);

    if (hit_idx < 0) {
        pixel_color = bg;
    } else {
        RTSphere sphere = s_spheres[(uint32_t)hit_idx];
        float3 sphere_center = make_float3(sphere.center[0], sphere.center[1], sphere.center[2]);
        float3 hit_point = make_float3(
            ray_origin.x + ray_dir.x * min_t,
            ray_origin.y + ray_dir.y * min_t,
            ray_origin.z + ray_dir.z * min_t);
        float3 hit_normal = normalize3(make_float3(
            hit_point.x - sphere_center.x,
            hit_point.y - sphere_center.y,
            hit_point.z - sphere_center.z));

        float3 base_color = make_float3(sphere.color[0], sphere.color[1], sphere.color[2]);

        if (sphere.texture_index >= 0 && atlas != 0) {
            float2 uv = sphere_uv(hit_normal);
            float4 tex_color = tex2DLayered<float4>(atlas, uv.x, uv.y, sphere.texture_index);
            base_color = make_float3(tex_color.x, tex_color.y, tex_color.z);
        }

        if (sphere.material == 1u) {
            float dist = length2(make_float2(hit_point.x - sphere_center.x, hit_point.y - sphere_center.y));
            float glow = 1.0f + 0.3f * fmaxf(0.0f, 1.0f - dist / sphere.radius);
            pixel_color = make_float3(base_color.x * glow, base_color.y * glow, base_color.z * glow);
        } else {
            float3 sun_pos = make_float3(cam.sun_screen_x, cam.sun_screen_y, 0.0f);
            float3 to_light = normalize3(make_float3(
                sun_pos.x - hit_point.x, sun_pos.y - hit_point.y, sun_pos.z - hit_point.z));
            float ndotl = fmaxf(dot3(hit_normal, to_light), 0.0f);

            bool in_shadow = false;
            float3 shadow_origin = make_float3(
                hit_point.x + hit_normal.x * 0.1f,
                hit_point.y + hit_normal.y * 0.1f,
                hit_point.z + hit_normal.z * 0.1f);
            for (uint32_t i = 0; i < cam.num_spheres && i < MAX_RT_SPHERES; i++) {
                if (i == (uint32_t)hit_idx) continue;
                float3 c = make_float3(s_spheres[i].center[0], s_spheres[i].center[1], s_spheres[i].center[2]);
                float t = intersect_sphere(shadow_origin, to_light, c, s_spheres[i].radius);
                if (t > 0.0f) { in_shadow = true; break; }
            }

            float ao_factor = 0.0f;
            float ao_range = sphere.radius * 5.0f;
            for (uint32_t s = 0; s < 4u; s++) {
                float r1 = rand_float(seed);
                float r2 = rand_float(seed);
                float3 ao_dir = build_tangent_frame(hit_normal, r1, r2);
                bool occluded = false;
                for (uint32_t i = 0; i < cam.num_spheres && i < MAX_RT_SPHERES; i++) {
                    if (i == (uint32_t)hit_idx) continue;
                    float3 c = make_float3(s_spheres[i].center[0], s_spheres[i].center[1], s_spheres[i].center[2]);
                    float t = intersect_sphere(shadow_origin, ao_dir, c, s_spheres[i].radius);
                    if (t > 0.0f && t < ao_range) { occluded = true; break; }
                }
                if (!occluded) ao_factor += 1.0f;
            }
            ao_factor /= 4.0f;

            float ambient = 0.08f;
            float diffuse = in_shadow ? 0.0f : ndotl;
            float lighting = (ambient + diffuse * 0.9f) * ao_factor;
            pixel_color = make_float3(base_color.x * lighting, base_color.y * lighting, base_color.z * lighting);

            if (sphere.material == 2u) {
                float3 reflect_dir = reflect3(ray_dir, hit_normal);
                float pr1 = rand_float(seed);
                float pr2 = rand_float(seed);
                float3 perturb = build_tangent_frame(reflect_dir, pr1, pr2);
                float3 glossy_dir = normalize3(make_float3(
                    reflect_dir.x + perturb.x * 0.3f,
                    reflect_dir.y + perturb.y * 0.3f,
                    reflect_dir.z + perturb.z * 0.3f));

                float3 ref_color = bg;
                float ref_min_t = 1e20f;
                for (uint32_t i = 0; i < cam.num_spheres && i < MAX_RT_SPHERES; i++) {
                    if (i == (uint32_t)hit_idx) continue;
                    float3 c = make_float3(s_spheres[i].center[0], s_spheres[i].center[1], s_spheres[i].center[2]);
                    float t = intersect_sphere(shadow_origin, glossy_dir, c, s_spheres[i].radius);
                    if (t > 0.0f && t < ref_min_t) {
                        ref_min_t = t;
                        ref_color = make_float3(s_spheres[i].color[0], s_spheres[i].color[1], s_spheres[i].color[2]);
                    }
                }
                pixel_color = make_float3(
                    pixel_color.x * 0.85f + ref_color.x * 0.15f,
                    pixel_color.y * 0.85f + ref_color.y * 0.15f,
                    pixel_color.z * 0.85f + ref_color.z * 0.15f);
            }
        }
    }

    pixel_color.x = powf(fminf(fmaxf(pixel_color.x, 0.0f), 1.0f), 1.0f / 2.2f);
    pixel_color.y = powf(fminf(fmaxf(pixel_color.y, 0.0f), 1.0f), 1.0f / 2.2f);
    pixel_color.z = powf(fminf(fmaxf(pixel_color.z, 0.0f), 1.0f), 1.0f / 2.2f);

    uint32_t idx = py * (uint32_t)cam.width + px;
    float4 prev = accum[idx];
    float weight = 1.0f / (float)(cam.frame_count + 1u);
    float3 accumulated = make_float3(
        prev.x * (1.0f - weight) + pixel_color.x * weight,
        prev.y * (1.0f - weight) + pixel_color.y * weight,
        prev.z * (1.0f - weight) + pixel_color.z * weight);
    accum[idx] = make_float4(accumulated.x, accumulated.y, accumulated.z, 1.0f);

    uint32_t out_idx = idx * 4;
    output[out_idx]     = (uint8_t)(accumulated.x * 255.0f);
    output[out_idx + 1] = (uint8_t)(accumulated.y * 255.0f);
    output[out_idx + 2] = (uint8_t)(accumulated.z * 255.0f);
    output[out_idx + 3] = 255;
}

extern "C" void launch_raytrace_kernel(
    const RTSphere* d_spheres,
    RTCameraUniform cam,
    float4* d_accum,
    uint8_t* d_output,
    hipTextureObject_t atlas,
    uint32_t tex_width,
    uint32_t tex_height)
{
    dim3 block(8, 8);
    dim3 grid(((uint32_t)cam.width + 7) / 8, ((uint32_t)cam.height + 7) / 8);
    hipLaunchKernelGGL(raytrace_kernel, grid, block, 0, 0,
        d_spheres, cam, d_accum, d_output, atlas, tex_width, tex_height);
}
