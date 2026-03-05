use crate::shapes::{CircleVertex, LineVertex};

const CIRCLE_SHADER: &str = r#"
struct Uniforms {
    projection: mat4x4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;
@group(1) @binding(0) var texture_atlas: texture_2d_array<f32>;
@group(1) @binding(1) var texture_sampler: sampler;

struct VertexInput {
    @location(0) position: vec2<f32>,
    @location(1) center: vec2<f32>,
    @location(2) radius: f32,
    @location(3) color: vec4<f32>,
    @location(4) texture_index: i32,
};

struct VertexOutput {
    @builtin(position) clip_position: vec4<f32>,
    @location(0) frag_pos: vec2<f32>,
    @location(1) center: vec2<f32>,
    @location(2) radius: f32,
    @location(3) color: vec4<f32>,
    @location(4) @interpolate(flat) texture_index: i32,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    out.clip_position = uniforms.projection * vec4<f32>(in.position, 0.0, 1.0);
    out.frag_pos = in.position;
    out.center = in.center;
    out.radius = in.radius;
    out.color = in.color;
    out.texture_index = in.texture_index;
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let dist = distance(in.frag_pos, in.center);
    let alpha = 1.0 - smoothstep(in.radius - 1.0, in.radius + 1.0, dist);
    if (alpha < 0.01) { discard; }

    var base_color = in.color;

    if (in.texture_index >= 0) {
        // Compute normalized disk coords in [-1, 1]
        let d = (in.frag_pos - in.center) / in.radius;
        let r2 = dot(d, d);
        if (r2 <= 1.0) {
            // Project to sphere surface: z = sqrt(1 - x^2 - y^2)
            let z = sqrt(1.0 - r2);
            // Spherical to equirectangular UV
            let u = 0.5 + atan2(d.x, z) / 6.283185;
            let v = 0.5 - asin(clamp(d.y, -1.0, 1.0)) / 3.141593;
            let tex_color = textureSample(texture_atlas, texture_sampler, vec2<f32>(u, v), in.texture_index);
            base_color = vec4<f32>(tex_color.rgb, in.color.a);
        }
    }

    return vec4<f32>(base_color.rgb, base_color.a * alpha);
}
"#;

const GLOW_SHADER: &str = r#"
struct Uniforms {
    projection: mat4x4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct VertexInput {
    @location(0) position: vec2<f32>,
    @location(1) center: vec2<f32>,
    @location(2) radius: f32,
    @location(3) color: vec4<f32>,
    @location(4) texture_index: i32,
};

struct VertexOutput {
    @builtin(position) clip_position: vec4<f32>,
    @location(0) frag_pos: vec2<f32>,
    @location(1) center: vec2<f32>,
    @location(2) radius: f32,
    @location(3) color: vec4<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    out.clip_position = uniforms.projection * vec4<f32>(in.position, 0.0, 1.0);
    out.frag_pos = in.position;
    out.center = in.center;
    out.radius = in.radius;
    out.color = in.color;
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    let dist = distance(in.frag_pos, in.center);
    let glow_radius = in.radius * 0.5;
    let intensity = exp(-(dist * dist) / (2.0 * glow_radius * glow_radius));
    if (intensity < 0.01) { discard; }
    return vec4<f32>(in.color.rgb, in.color.a * intensity * 0.4);
}
"#;

const LINE_SHADER: &str = r#"
struct Uniforms {
    projection: mat4x4<f32>,
};
@group(0) @binding(0) var<uniform> uniforms: Uniforms;

struct VertexInput {
    @location(0) position: vec2<f32>,
    @location(1) color: vec4<f32>,
};

struct VertexOutput {
    @builtin(position) clip_position: vec4<f32>,
    @location(0) color: vec4<f32>,
};

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out: VertexOutput;
    out.clip_position = uniforms.projection * vec4<f32>(in.position, 0.0, 1.0);
    out.color = in.color;
    return out;
}

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
    return in.color;
}
"#;

pub struct Pipelines {
    pub circle: wgpu::RenderPipeline,
    pub glow: wgpu::RenderPipeline,
    pub line: wgpu::RenderPipeline,
    pub projection_bind_group_layout: wgpu::BindGroupLayout,
    pub texture_bind_group_layout: wgpu::BindGroupLayout,
}

impl Pipelines {
    pub fn new(device: &wgpu::Device, format: wgpu::TextureFormat) -> Self {
        // Bind group layout 0: projection uniform
        let projection_bind_group_layout = device.create_bind_group_layout(&wgpu::BindGroupLayoutDescriptor {
            label: Some("projection_bind_group_layout"),
            entries: &[wgpu::BindGroupLayoutEntry {
                binding: 0,
                visibility: wgpu::ShaderStages::VERTEX,
                ty: wgpu::BindingType::Buffer {
                    ty: wgpu::BufferBindingType::Uniform,
                    has_dynamic_offset: false,
                    min_binding_size: None,
                },
                count: None,
            }],
        });

        // Bind group layout 1: texture array + sampler (for circle pipeline)
        let texture_bind_group_layout = device.create_bind_group_layout(&wgpu::BindGroupLayoutDescriptor {
            label: Some("texture_bind_group_layout"),
            entries: &[
                wgpu::BindGroupLayoutEntry {
                    binding: 0,
                    visibility: wgpu::ShaderStages::FRAGMENT,
                    ty: wgpu::BindingType::Texture {
                        sample_type: wgpu::TextureSampleType::Float { filterable: true },
                        view_dimension: wgpu::TextureViewDimension::D2Array,
                        multisampled: false,
                    },
                    count: None,
                },
                wgpu::BindGroupLayoutEntry {
                    binding: 1,
                    visibility: wgpu::ShaderStages::FRAGMENT,
                    ty: wgpu::BindingType::Sampler(wgpu::SamplerBindingType::Filtering),
                    count: None,
                },
            ],
        });

        // Circle pipeline layout: projection + textures
        let circle_pipeline_layout = device.create_pipeline_layout(&wgpu::PipelineLayoutDescriptor {
            label: Some("circle_pipeline_layout"),
            bind_group_layouts: &[&projection_bind_group_layout, &texture_bind_group_layout],
            push_constant_ranges: &[],
        });

        // Other pipeline layout: projection only
        let other_pipeline_layout = device.create_pipeline_layout(&wgpu::PipelineLayoutDescriptor {
            label: Some("other_pipeline_layout"),
            bind_group_layouts: &[&projection_bind_group_layout],
            push_constant_ranges: &[],
        });

        let alpha_blend = wgpu::BlendState {
            color: wgpu::BlendComponent {
                src_factor: wgpu::BlendFactor::SrcAlpha,
                dst_factor: wgpu::BlendFactor::OneMinusSrcAlpha,
                operation: wgpu::BlendOperation::Add,
            },
            alpha: wgpu::BlendComponent::OVER,
        };

        let additive_blend = wgpu::BlendState {
            color: wgpu::BlendComponent {
                src_factor: wgpu::BlendFactor::SrcAlpha,
                dst_factor: wgpu::BlendFactor::One,
                operation: wgpu::BlendOperation::Add,
            },
            alpha: wgpu::BlendComponent::OVER,
        };

        // Circle pipeline (with texture support)
        let circle_shader = device.create_shader_module(wgpu::ShaderModuleDescriptor {
            label: Some("circle_shader"),
            source: wgpu::ShaderSource::Wgsl(CIRCLE_SHADER.into()),
        });
        let circle = device.create_render_pipeline(&wgpu::RenderPipelineDescriptor {
            label: Some("circle_pipeline"),
            layout: Some(&circle_pipeline_layout),
            vertex: wgpu::VertexState {
                module: &circle_shader,
                entry_point: Some("vs_main"),
                buffers: &[CircleVertex::LAYOUT],
                compilation_options: Default::default(),
            },
            fragment: Some(wgpu::FragmentState {
                module: &circle_shader,
                entry_point: Some("fs_main"),
                targets: &[Some(wgpu::ColorTargetState {
                    format,
                    blend: Some(alpha_blend),
                    write_mask: wgpu::ColorWrites::ALL,
                })],
                compilation_options: Default::default(),
            }),
            primitive: wgpu::PrimitiveState {
                topology: wgpu::PrimitiveTopology::TriangleList,
                ..Default::default()
            },
            depth_stencil: None,
            multisample: wgpu::MultisampleState::default(),
            multiview: None,
            cache: None,
        });

        // Glow pipeline (additive blend, uses CircleVertex but ignores texture_index)
        let glow_shader = device.create_shader_module(wgpu::ShaderModuleDescriptor {
            label: Some("glow_shader"),
            source: wgpu::ShaderSource::Wgsl(GLOW_SHADER.into()),
        });
        let glow = device.create_render_pipeline(&wgpu::RenderPipelineDescriptor {
            label: Some("glow_pipeline"),
            layout: Some(&other_pipeline_layout),
            vertex: wgpu::VertexState {
                module: &glow_shader,
                entry_point: Some("vs_main"),
                buffers: &[CircleVertex::LAYOUT],
                compilation_options: Default::default(),
            },
            fragment: Some(wgpu::FragmentState {
                module: &glow_shader,
                entry_point: Some("fs_main"),
                targets: &[Some(wgpu::ColorTargetState {
                    format,
                    blend: Some(additive_blend),
                    write_mask: wgpu::ColorWrites::ALL,
                })],
                compilation_options: Default::default(),
            }),
            primitive: wgpu::PrimitiveState {
                topology: wgpu::PrimitiveTopology::TriangleList,
                ..Default::default()
            },
            depth_stencil: None,
            multisample: wgpu::MultisampleState::default(),
            multiview: None,
            cache: None,
        });

        // Line pipeline
        let line_shader = device.create_shader_module(wgpu::ShaderModuleDescriptor {
            label: Some("line_shader"),
            source: wgpu::ShaderSource::Wgsl(LINE_SHADER.into()),
        });
        let line = device.create_render_pipeline(&wgpu::RenderPipelineDescriptor {
            label: Some("line_pipeline"),
            layout: Some(&other_pipeline_layout),
            vertex: wgpu::VertexState {
                module: &line_shader,
                entry_point: Some("vs_main"),
                buffers: &[LineVertex::LAYOUT],
                compilation_options: Default::default(),
            },
            fragment: Some(wgpu::FragmentState {
                module: &line_shader,
                entry_point: Some("fs_main"),
                targets: &[Some(wgpu::ColorTargetState {
                    format,
                    blend: Some(alpha_blend),
                    write_mask: wgpu::ColorWrites::ALL,
                })],
                compilation_options: Default::default(),
            }),
            primitive: wgpu::PrimitiveState {
                topology: wgpu::PrimitiveTopology::LineList,
                ..Default::default()
            },
            depth_stencil: None,
            multisample: wgpu::MultisampleState::default(),
            multiview: None,
            cache: None,
        });

        Self { circle, glow, line, projection_bind_group_layout, texture_bind_group_layout }
    }
}
