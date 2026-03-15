/// Vertex for circle rendering (SDF approach via quad).
#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable)]
pub struct CircleVertex {
    pub position: [f32; 2],
    pub center: [f32; 2],
    pub radius: f32,
    pub color: [f32; 4],
    pub texture_index: i32,
}

impl CircleVertex {
    pub const LAYOUT: wgpu::VertexBufferLayout<'static> = wgpu::VertexBufferLayout {
        array_stride: std::mem::size_of::<CircleVertex>() as wgpu::BufferAddress,
        step_mode: wgpu::VertexStepMode::Vertex,
        attributes: &[
            wgpu::VertexAttribute {
                offset: 0,
                shader_location: 0,
                format: wgpu::VertexFormat::Float32x2,
            },
            wgpu::VertexAttribute {
                offset: 8,
                shader_location: 1,
                format: wgpu::VertexFormat::Float32x2,
            },
            wgpu::VertexAttribute {
                offset: 16,
                shader_location: 2,
                format: wgpu::VertexFormat::Float32,
            },
            wgpu::VertexAttribute {
                offset: 20,
                shader_location: 3,
                format: wgpu::VertexFormat::Float32x4,
            },
            wgpu::VertexAttribute {
                offset: 36,
                shader_location: 4,
                format: wgpu::VertexFormat::Sint32,
            },
        ],
    };
}

/// Generate 6 vertices (2 triangles) for a circle quad.
#[inline]
pub fn make_circle_vertices(
    cx: f32,
    cy: f32,
    radius: f32,
    color: [f32; 4],
    texture_index: i32,
) -> [CircleVertex; 6] {
    let r = radius + 2.0; // extra margin for anti-aliasing
    let tl = CircleVertex {
        position: [cx - r, cy - r],
        center: [cx, cy],
        radius,
        color,
        texture_index,
    };
    let tr = CircleVertex {
        position: [cx + r, cy - r],
        center: [cx, cy],
        radius,
        color,
        texture_index,
    };
    let bl = CircleVertex {
        position: [cx - r, cy + r],
        center: [cx, cy],
        radius,
        color,
        texture_index,
    };
    let br = CircleVertex {
        position: [cx + r, cy + r],
        center: [cx, cy],
        radius,
        color,
        texture_index,
    };
    [tl, tr, bl, tr, br, bl]
}

/// Vertex for line rendering.
#[repr(C)]
#[derive(Copy, Clone, Debug, bytemuck::Pod, bytemuck::Zeroable)]
pub struct LineVertex {
    pub position: [f32; 2],
    pub color: [f32; 4],
}

impl LineVertex {
    pub const LAYOUT: wgpu::VertexBufferLayout<'static> = wgpu::VertexBufferLayout {
        array_stride: std::mem::size_of::<LineVertex>() as wgpu::BufferAddress,
        step_mode: wgpu::VertexStepMode::Vertex,
        attributes: &[
            wgpu::VertexAttribute {
                offset: 0,
                shader_location: 0,
                format: wgpu::VertexFormat::Float32x2,
            },
            wgpu::VertexAttribute {
                offset: 8,
                shader_location: 1,
                format: wgpu::VertexFormat::Float32x4,
            },
        ],
    };
}
