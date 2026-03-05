/// GPU compute shader ray tracer for sphere primitives.
/// Provides shadows, ambient occlusion, and progressive accumulation.

const RT_SHADER: &str = r#"
struct Sphere {
    center: vec3<f32>,
    radius: f32,
    color: vec4<f32>,
    material: u32,
    _pad1: u32,
    _pad2: u32,
    _pad3: u32,
};

struct Camera {
    width: f32,
    height: f32,
    frame_count: u32,
    num_spheres: u32,
    sun_screen_x: f32,
    sun_screen_y: f32,
    samples_per_frame: u32,
    max_bounces: u32,
};

@group(0) @binding(0) var output: texture_storage_2d<rgba8unorm, write>;
@group(0) @binding(1) var<storage, read> spheres: array<Sphere>;
@group(0) @binding(2) var<uniform> cam: Camera;
@group(0) @binding(3) var<storage, read_write> accum: array<vec4<f32>>;

fn pcg_hash(input: u32) -> u32 {
    var state = input * 747796405u + 2891336453u;
    var word = ((state >> ((state >> 28u) + 4u)) ^ state) * 277803737u;
    return (word >> 22u) ^ word;
}

fn rand_float(seed: ptr<function, u32>) -> f32 {
    *seed = pcg_hash(*seed);
    return f32(*seed) / 4294967295.0;
}

fn intersect_sphere(ro: vec3<f32>, rd: vec3<f32>, center: vec3<f32>, radius: f32) -> f32 {
    let oc = ro - center;
    let b = dot(oc, rd);
    let c = dot(oc, oc) - radius * radius;
    let discriminant = b * b - c;
    if discriminant < 0.0 { return -1.0; }
    let t = -b - sqrt(discriminant);
    if t > 0.0 { return t; }
    return -1.0;
}

// Build an arbitrary tangent frame from a normal
fn build_tangent_frame(n: vec3<f32>, r1: f32, r2: f32) -> vec3<f32> {
    // Cosine-weighted hemisphere sampling
    let phi = 6.283185 * r1;
    let cos_theta = sqrt(r2);
    let sin_theta = sqrt(1.0 - r2);

    // Build orthonormal basis
    var t: vec3<f32>;
    if abs(n.x) > 0.9 {
        t = vec3<f32>(0.0, 1.0, 0.0);
    } else {
        t = vec3<f32>(1.0, 0.0, 0.0);
    }
    let b = normalize(cross(n, t));
    let tangent = cross(b, n);

    return normalize(tangent * cos(phi) * sin_theta + b * sin(phi) * sin_theta + n * cos_theta);
}

@compute @workgroup_size(8, 8)
fn main(@builtin(global_invocation_id) id: vec3<u32>) {
    let px = id.x;
    let py = id.y;
    if px >= u32(cam.width) || py >= u32(cam.height) { return; }

    var seed = px + py * u32(cam.width) + cam.frame_count * 196613u;

    // Orthographic ray: cast in +z direction from above the screen plane
    let ray_origin = vec3<f32>(f32(px) + 0.5, f32(py) + 0.5, -1000.0);
    let ray_dir = vec3<f32>(0.0, 0.0, 1.0);

    // Find nearest sphere intersection
    var min_t: f32 = 1e20;
    var hit_idx: i32 = -1;

    for (var i = 0u; i < cam.num_spheres; i++) {
        let s = spheres[i];
        let t = intersect_sphere(ray_origin, ray_dir, s.center, s.radius);
        if t > 0.0 && t < min_t {
            min_t = t;
            hit_idx = i32(i);
        }
    }

    var pixel_color: vec3<f32>;
    let bg = vec3<f32>(5.0 / 255.0, 5.0 / 255.0, 15.0 / 255.0);

    if hit_idx < 0 {
        pixel_color = bg;
    } else {
        let sphere = spheres[u32(hit_idx)];
        let hit_point = ray_origin + ray_dir * min_t;
        let hit_normal = normalize(hit_point - sphere.center);

        if sphere.material == 1u {
            // Emissive (sun) — self-lit with glow
            let glow = 1.0 + 0.3 * max(0.0, 1.0 - length(hit_point.xy - sphere.center.xy) / sphere.radius);
            pixel_color = sphere.color.rgb * glow;
        } else {
            // Light direction toward sun
            let sun_pos = vec3<f32>(cam.sun_screen_x, cam.sun_screen_y, 0.0);
            let to_light = normalize(sun_pos - hit_point);
            let ndotl = max(dot(hit_normal, to_light), 0.0);

            // Shadow test
            var in_shadow = false;
            let shadow_origin = hit_point + hit_normal * 0.1;
            for (var i = 0u; i < cam.num_spheres; i++) {
                if i == u32(hit_idx) { continue; }
                let t = intersect_sphere(shadow_origin, to_light, spheres[i].center, spheres[i].radius);
                if t > 0.0 {
                    in_shadow = true;
                    break;
                }
            }

            // Ambient occlusion (4 hemisphere samples)
            var ao_factor: f32 = 0.0;
            let ao_samples = 4u;
            let ao_origin = hit_point + hit_normal * 0.1;
            let ao_range = sphere.radius * 5.0;

            for (var s = 0u; s < ao_samples; s++) {
                let r1 = rand_float(&seed);
                let r2 = rand_float(&seed);
                let ao_dir = build_tangent_frame(hit_normal, r1, r2);

                var occluded = false;
                for (var i = 0u; i < cam.num_spheres; i++) {
                    if i == u32(hit_idx) { continue; }
                    let t = intersect_sphere(ao_origin, ao_dir, spheres[i].center, spheres[i].radius);
                    if t > 0.0 && t < ao_range {
                        occluded = true;
                        break;
                    }
                }
                if !occluded { ao_factor += 1.0; }
            }
            ao_factor = ao_factor / f32(ao_samples);

            let ambient = 0.08;
            var diffuse: f32 = 0.0;
            if !in_shadow { diffuse = ndotl; }

            pixel_color = sphere.color.rgb * (ambient + diffuse * 0.9) * ao_factor;

            // Glossy reflection for gas giants (material == 2)
            if sphere.material == 2u {
                let reflect_dir = reflect(ray_dir, hit_normal);
                // Perturb for glossy
                let pr1 = rand_float(&seed);
                let pr2 = rand_float(&seed);
                let glossy_dir = normalize(reflect_dir + build_tangent_frame(reflect_dir, pr1, pr2) * 0.3);

                var ref_color = bg;
                var ref_min_t: f32 = 1e20;
                for (var i = 0u; i < cam.num_spheres; i++) {
                    if i == u32(hit_idx) { continue; }
                    let t = intersect_sphere(shadow_origin, glossy_dir, spheres[i].center, spheres[i].radius);
                    if t > 0.0 && t < ref_min_t {
                        ref_min_t = t;
                        ref_color = spheres[i].color.rgb;
                    }
                }
                pixel_color = mix(pixel_color, ref_color, 0.15);
            }
        }
    }

    // sRGB gamma correction
    pixel_color = pow(clamp(pixel_color, vec3<f32>(0.0), vec3<f32>(1.0)), vec3<f32>(1.0 / 2.2));

    // Progressive accumulation
    let idx = py * u32(cam.width) + px;
    let prev = accum[idx];
    let weight = 1.0 / f32(cam.frame_count + 1u);
    let accumulated = mix(prev.rgb, pixel_color, weight);
    accum[idx] = vec4<f32>(accumulated, 1.0);

    textureStore(output, vec2<i32>(id.xy), vec4<f32>(accumulated, 1.0));
}
"#;

/// Sphere data sent to the GPU compute shader.
#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable)]
pub struct RTSphere {
    pub center: [f32; 3],
    pub radius: f32,
    pub color: [f32; 4],
    pub material: u32,
    pub _pad: [u32; 3],
}

/// Camera uniform for the RT compute shader.
#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable)]
pub struct RTCameraUniform {
    pub width: f32,
    pub height: f32,
    pub frame_count: u32,
    pub num_spheres: u32,
    pub sun_screen_x: f32,
    pub sun_screen_y: f32,
    pub samples_per_frame: u32,
    pub max_bounces: u32,
}

pub struct RayTracer {
    compute_pipeline: wgpu::ComputePipeline,
    bind_group_layout: wgpu::BindGroupLayout,
    sphere_buffer: wgpu::Buffer,
    camera_buffer: wgpu::Buffer,
    accumulation_buffer: wgpu::Buffer,
    pub output_texture: wgpu::Texture,
    output_view: wgpu::TextureView,
    width: u32,
    height: u32,
}

impl RayTracer {
    pub fn new(device: &wgpu::Device, width: u32, height: u32) -> Option<Self> {
        let shader = device.create_shader_module(wgpu::ShaderModuleDescriptor {
            label: Some("rt_compute_shader"),
            source: wgpu::ShaderSource::Wgsl(RT_SHADER.into()),
        });

        let bind_group_layout = device.create_bind_group_layout(&wgpu::BindGroupLayoutDescriptor {
            label: Some("rt_bind_group_layout"),
            entries: &[
                // output texture
                wgpu::BindGroupLayoutEntry {
                    binding: 0,
                    visibility: wgpu::ShaderStages::COMPUTE,
                    ty: wgpu::BindingType::StorageTexture {
                        access: wgpu::StorageTextureAccess::WriteOnly,
                        format: wgpu::TextureFormat::Rgba8Unorm,
                        view_dimension: wgpu::TextureViewDimension::D2,
                    },
                    count: None,
                },
                // spheres storage buffer
                wgpu::BindGroupLayoutEntry {
                    binding: 1,
                    visibility: wgpu::ShaderStages::COMPUTE,
                    ty: wgpu::BindingType::Buffer {
                        ty: wgpu::BufferBindingType::Storage { read_only: true },
                        has_dynamic_offset: false,
                        min_binding_size: None,
                    },
                    count: None,
                },
                // camera uniform
                wgpu::BindGroupLayoutEntry {
                    binding: 2,
                    visibility: wgpu::ShaderStages::COMPUTE,
                    ty: wgpu::BindingType::Buffer {
                        ty: wgpu::BufferBindingType::Uniform,
                        has_dynamic_offset: false,
                        min_binding_size: None,
                    },
                    count: None,
                },
                // accumulation buffer
                wgpu::BindGroupLayoutEntry {
                    binding: 3,
                    visibility: wgpu::ShaderStages::COMPUTE,
                    ty: wgpu::BindingType::Buffer {
                        ty: wgpu::BufferBindingType::Storage { read_only: false },
                        has_dynamic_offset: false,
                        min_binding_size: None,
                    },
                    count: None,
                },
            ],
        });

        let pipeline_layout = device.create_pipeline_layout(&wgpu::PipelineLayoutDescriptor {
            label: Some("rt_pipeline_layout"),
            bind_group_layouts: &[&bind_group_layout],
            push_constant_ranges: &[],
        });

        let compute_pipeline = device.create_compute_pipeline(&wgpu::ComputePipelineDescriptor {
            label: Some("rt_compute_pipeline"),
            layout: Some(&pipeline_layout),
            module: &shader,
            entry_point: Some("main"),
            compilation_options: Default::default(),
            cache: None,
        });

        let max_spheres = 16u64;
        let sphere_buffer = device.create_buffer(&wgpu::BufferDescriptor {
            label: Some("rt_sphere_buffer"),
            size: max_spheres * std::mem::size_of::<RTSphere>() as u64,
            usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_DST,
            mapped_at_creation: false,
        });

        let camera_buffer = device.create_buffer(&wgpu::BufferDescriptor {
            label: Some("rt_camera_buffer"),
            size: std::mem::size_of::<RTCameraUniform>() as u64,
            usage: wgpu::BufferUsages::UNIFORM | wgpu::BufferUsages::COPY_DST,
            mapped_at_creation: false,
        });

        let accum_size = (width as u64) * (height as u64) * 16; // vec4<f32> per pixel
        let accumulation_buffer = device.create_buffer(&wgpu::BufferDescriptor {
            label: Some("rt_accumulation_buffer"),
            size: accum_size,
            usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_DST,
            mapped_at_creation: false,
        });

        let (output_texture, output_view) = Self::create_output_texture(device, width, height);

        Some(Self {
            compute_pipeline,
            bind_group_layout,
            sphere_buffer,
            camera_buffer,
            accumulation_buffer,
            output_texture,
            output_view,
            width,
            height,
        })
    }

    fn create_output_texture(
        device: &wgpu::Device,
        width: u32,
        height: u32,
    ) -> (wgpu::Texture, wgpu::TextureView) {
        let texture = device.create_texture(&wgpu::TextureDescriptor {
            label: Some("rt_output_texture"),
            size: wgpu::Extent3d { width, height, depth_or_array_layers: 1 },
            mip_level_count: 1,
            sample_count: 1,
            dimension: wgpu::TextureDimension::D2,
            format: wgpu::TextureFormat::Rgba8Unorm,
            usage: wgpu::TextureUsages::STORAGE_BINDING | wgpu::TextureUsages::COPY_SRC,
            view_formats: &[],
        });
        let view = texture.create_view(&wgpu::TextureViewDescriptor::default());
        (texture, view)
    }

    pub fn resize(&mut self, device: &wgpu::Device, width: u32, height: u32) {
        self.width = width;
        self.height = height;
        let (texture, view) = Self::create_output_texture(device, width, height);
        self.output_texture = texture;
        self.output_view = view;

        let accum_size = (width as u64) * (height as u64) * 16;
        self.accumulation_buffer = device.create_buffer(&wgpu::BufferDescriptor {
            label: Some("rt_accumulation_buffer"),
            size: accum_size,
            usage: wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_DST,
            mapped_at_creation: false,
        });
    }

    pub fn reset_accumulation(&self, queue: &wgpu::Queue) {
        let zeros = vec![0u8; (self.width as usize) * (self.height as usize) * 16];
        queue.write_buffer(&self.accumulation_buffer, 0, &zeros);
    }

    pub fn dispatch(
        &self,
        device: &wgpu::Device,
        queue: &wgpu::Queue,
        encoder: &mut wgpu::CommandEncoder,
        spheres: &[RTSphere],
        camera: &RTCameraUniform,
    ) {
        // Upload sphere data
        queue.write_buffer(&self.sphere_buffer, 0, bytemuck::cast_slice(spheres));
        // Upload camera
        queue.write_buffer(&self.camera_buffer, 0, bytemuck::bytes_of(camera));

        let bind_group = device.create_bind_group(&wgpu::BindGroupDescriptor {
            label: Some("rt_bind_group"),
            layout: &self.bind_group_layout,
            entries: &[
                wgpu::BindGroupEntry {
                    binding: 0,
                    resource: wgpu::BindingResource::TextureView(&self.output_view),
                },
                wgpu::BindGroupEntry {
                    binding: 1,
                    resource: self.sphere_buffer.as_entire_binding(),
                },
                wgpu::BindGroupEntry {
                    binding: 2,
                    resource: self.camera_buffer.as_entire_binding(),
                },
                wgpu::BindGroupEntry {
                    binding: 3,
                    resource: self.accumulation_buffer.as_entire_binding(),
                },
            ],
        });

        {
            let mut cpass = encoder.begin_compute_pass(&wgpu::ComputePassDescriptor {
                label: Some("rt_compute_pass"),
                timestamp_writes: None,
            });
            cpass.set_pipeline(&self.compute_pipeline);
            cpass.set_bind_group(0, Some(&bind_group), &[]);
            let wg_x = (self.width + 7) / 8;
            let wg_y = (self.height + 7) / 8;
            cpass.dispatch_workgroups(wg_x, wg_y, 1);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_rt_sphere_size() {
        assert_eq!(std::mem::size_of::<RTSphere>(), 48);
    }

    #[test]
    fn test_rt_camera_uniform_size() {
        assert_eq!(std::mem::size_of::<RTCameraUniform>(), 32);
    }
}
