use crate::camera::Camera;
use crate::pipeline::Pipelines;
use crate::shapes::{make_circle_vertices, CircleVertex, LineVertex};
use crate::spacetime::{generate_grid, SpacetimeBody};

/// Body data passed from Go via FFI.
pub struct BodyData {
    pub screen_x: f32,
    pub screen_y: f32,
    pub radius: f32,
    pub color: [f32; 4],
}

/// Trail segment data.
pub struct TrailData {
    pub vertices: Vec<LineVertex>,
}

/// Distance measurement line between two selected bodies.
pub struct DistanceLine {
    pub x1: f32,
    pub y1: f32,
    pub x2: f32,
    pub y2: f32,
}

pub struct Renderer {
    device: wgpu::Device,
    queue: wgpu::Queue,
    pipelines: Pipelines,
    texture: wgpu::Texture,
    texture_view: wgpu::TextureView,
    readback_buffer: wgpu::Buffer,
    uniform_buffer: wgpu::Buffer,
    bind_group: wgpu::BindGroup,
    pub width: u32,
    pub height: u32,
    format: wgpu::TextureFormat,
    pub camera: Camera,
    pixel_data: Vec<u8>,

    // Scene data set via FFI
    pub bodies: Vec<BodyData>,
    pub sun: Option<BodyData>,
    pub trails: Vec<TrailData>,
    pub show_trails: bool,
    pub show_spacetime: bool,
    pub spacetime_bodies: Vec<SpacetimeBody>,
    pub distance_line: Option<DistanceLine>,
}

impl Renderer {
    pub fn new(width: u32, height: u32) -> Option<Self> {
        let instance = wgpu::Instance::new(&wgpu::InstanceDescriptor {
            backends: wgpu::Backends::all(),
            ..Default::default()
        });

        let adapter = pollster::block_on(instance.request_adapter(&wgpu::RequestAdapterOptions {
            power_preference: wgpu::PowerPreference::HighPerformance,
            compatible_surface: None,
            force_fallback_adapter: false,
        }))?;

        let (device, queue) = pollster::block_on(adapter.request_device(
            &wgpu::DeviceDescriptor {
                label: Some("render_core_device"),
                required_features: wgpu::Features::empty(),
                required_limits: wgpu::Limits::default(),
                memory_hints: Default::default(),
            },
            None,
        ))
        .ok()?;

        let format = wgpu::TextureFormat::Rgba8UnormSrgb;
        let pipelines = Pipelines::new(&device, format);

        let (texture, texture_view) = Self::create_texture(&device, width, height, format);
        let readback_buffer = Self::create_readback_buffer(&device, width, height);

        let camera = Camera::new(width, height);
        let matrix = camera.ortho_matrix();
        let uniform_buffer = device.create_buffer(&wgpu::BufferDescriptor {
            label: Some("uniform_buffer"),
            size: 64,
            usage: wgpu::BufferUsages::UNIFORM | wgpu::BufferUsages::COPY_DST,
            mapped_at_creation: false,
        });
        queue.write_buffer(&uniform_buffer, 0, bytemuck::cast_slice(&matrix));

        let bind_group = device.create_bind_group(&wgpu::BindGroupDescriptor {
            label: Some("projection_bind_group"),
            layout: &pipelines.bind_group_layout,
            entries: &[wgpu::BindGroupEntry {
                binding: 0,
                resource: uniform_buffer.as_entire_binding(),
            }],
        });

        let buf_size = (width * height * 4) as usize;
        Some(Self {
            device,
            queue,
            pipelines,
            texture,
            texture_view,
            readback_buffer,
            uniform_buffer,
            bind_group,
            width,
            height,
            format,
            camera,
            pixel_data: vec![0u8; buf_size],
            bodies: Vec::new(),
            sun: None,
            trails: Vec::new(),
            show_trails: false,
            show_spacetime: false,
            spacetime_bodies: Vec::new(),
            distance_line: None,
        })
    }

    fn create_texture(
        device: &wgpu::Device,
        width: u32,
        height: u32,
        format: wgpu::TextureFormat,
    ) -> (wgpu::Texture, wgpu::TextureView) {
        let texture = device.create_texture(&wgpu::TextureDescriptor {
            label: Some("offscreen_texture"),
            size: wgpu::Extent3d { width, height, depth_or_array_layers: 1 },
            mip_level_count: 1,
            sample_count: 1,
            dimension: wgpu::TextureDimension::D2,
            format,
            usage: wgpu::TextureUsages::RENDER_ATTACHMENT | wgpu::TextureUsages::COPY_SRC,
            view_formats: &[],
        });
        let view = texture.create_view(&wgpu::TextureViewDescriptor::default());
        (texture, view)
    }

    fn create_readback_buffer(device: &wgpu::Device, width: u32, height: u32) -> wgpu::Buffer {
        let bytes_per_row = Self::padded_bytes_per_row(width);
        let buffer_size = (bytes_per_row * height) as u64;
        device.create_buffer(&wgpu::BufferDescriptor {
            label: Some("readback_buffer"),
            size: buffer_size,
            usage: wgpu::BufferUsages::COPY_DST | wgpu::BufferUsages::MAP_READ,
            mapped_at_creation: false,
        })
    }

    /// wgpu requires rows to be aligned to 256 bytes.
    fn padded_bytes_per_row(width: u32) -> u32 {
        let unpadded = width * 4;
        let align = wgpu::COPY_BYTES_PER_ROW_ALIGNMENT;
        (unpadded + align - 1) / align * align
    }

    pub fn resize(&mut self, width: u32, height: u32) {
        if width == 0 || height == 0 || (width == self.width && height == self.height) {
            return;
        }
        self.width = width;
        self.height = height;
        self.camera.width = width;
        self.camera.height = height;

        let (texture, view) = Self::create_texture(&self.device, width, height, self.format);
        self.texture = texture;
        self.texture_view = view;
        self.readback_buffer = Self::create_readback_buffer(&self.device, width, height);
        self.pixel_data.resize((width * height * 4) as usize, 0);

        // Update projection
        let matrix = self.camera.ortho_matrix();
        self.queue.write_buffer(&self.uniform_buffer, 0, bytemuck::cast_slice(&matrix));
    }

    /// Render a frame and return a pointer to RGBA pixel data.
    pub fn render_frame(&mut self) -> &[u8] {
        // Update projection uniform
        let matrix = self.camera.ortho_matrix();
        self.queue.write_buffer(&self.uniform_buffer, 0, bytemuck::cast_slice(&matrix));

        let mut encoder = self.device.create_command_encoder(&wgpu::CommandEncoderDescriptor {
            label: Some("render_encoder"),
        });

        // Collect all vertex data before the render pass
        let spacetime_verts = if self.show_spacetime && !self.spacetime_bodies.is_empty() {
            generate_grid(
                self.width as f64,
                self.height as f64,
                self.camera.zoom,
                self.camera.pan_x,
                self.camera.pan_y,
                &self.spacetime_bodies,
            )
        } else {
            Vec::new()
        };

        let trail_verts: Vec<LineVertex> = if self.show_trails {
            self.trails.iter().flat_map(|t| t.vertices.iter().copied()).collect()
        } else {
            Vec::new()
        };

        // Build circle vertices for all bodies
        let mut circle_verts: Vec<CircleVertex> = Vec::new();
        let mut glow_verts: Vec<CircleVertex> = Vec::new();

        // Sun (rendered first as glow, then solid)
        if let Some(ref sun) = self.sun {
            // Glow quad (larger radius for glow effect)
            let glow_radius = sun.radius * 3.0;
            let glow_cvs = make_circle_vertices(sun.screen_x, sun.screen_y, glow_radius, sun.color);
            glow_verts.extend_from_slice(&glow_cvs);

            // Solid sun
            let sun_cvs = make_circle_vertices(sun.screen_x, sun.screen_y, sun.radius, sun.color);
            circle_verts.extend_from_slice(&sun_cvs);
        }

        // Planets
        for body in &self.bodies {
            let cvs = make_circle_vertices(body.screen_x, body.screen_y, body.radius, body.color);
            circle_verts.extend_from_slice(&cvs);
        }

        // Distance line
        let mut dist_line_verts: Vec<LineVertex> = Vec::new();
        if let Some(ref dl) = self.distance_line {
            let color = [1.0, 1.0, 0.0, 0.706]; // yellow, alpha ~180/255
            dist_line_verts.push(LineVertex { position: [dl.x1, dl.y1], color });
            dist_line_verts.push(LineVertex { position: [dl.x2, dl.y2], color });
        }

        // Create GPU buffers
        let spacetime_buf = if !spacetime_verts.is_empty() {
            Some(self.device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("spacetime_vb"),
                size: (spacetime_verts.len() * std::mem::size_of::<LineVertex>()) as u64,
                usage: wgpu::BufferUsages::VERTEX | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            }))
        } else {
            None
        };

        let trail_buf = if !trail_verts.is_empty() {
            Some(self.device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("trail_vb"),
                size: (trail_verts.len() * std::mem::size_of::<LineVertex>()) as u64,
                usage: wgpu::BufferUsages::VERTEX | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            }))
        } else {
            None
        };

        let glow_buf = if !glow_verts.is_empty() {
            Some(self.device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("glow_vb"),
                size: (glow_verts.len() * std::mem::size_of::<CircleVertex>()) as u64,
                usage: wgpu::BufferUsages::VERTEX | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            }))
        } else {
            None
        };

        let circle_buf = if !circle_verts.is_empty() {
            Some(self.device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("circle_vb"),
                size: (circle_verts.len() * std::mem::size_of::<CircleVertex>()) as u64,
                usage: wgpu::BufferUsages::VERTEX | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            }))
        } else {
            None
        };

        let dist_buf = if !dist_line_verts.is_empty() {
            Some(self.device.create_buffer(&wgpu::BufferDescriptor {
                label: Some("dist_line_vb"),
                size: (dist_line_verts.len() * std::mem::size_of::<LineVertex>()) as u64,
                usage: wgpu::BufferUsages::VERTEX | wgpu::BufferUsages::COPY_DST,
                mapped_at_creation: false,
            }))
        } else {
            None
        };

        // Write data to buffers
        if let Some(ref buf) = spacetime_buf {
            self.queue.write_buffer(buf, 0, bytemuck::cast_slice(&spacetime_verts));
        }
        if let Some(ref buf) = trail_buf {
            self.queue.write_buffer(buf, 0, bytemuck::cast_slice(&trail_verts));
        }
        if let Some(ref buf) = glow_buf {
            self.queue.write_buffer(buf, 0, bytemuck::cast_slice(&glow_verts));
        }
        if let Some(ref buf) = circle_buf {
            self.queue.write_buffer(buf, 0, bytemuck::cast_slice(&circle_verts));
        }
        if let Some(ref buf) = dist_buf {
            self.queue.write_buffer(buf, 0, bytemuck::cast_slice(&dist_line_verts));
        }

        // Render pass
        {
            let mut pass = encoder.begin_render_pass(&wgpu::RenderPassDescriptor {
                label: Some("main_pass"),
                color_attachments: &[Some(wgpu::RenderPassColorAttachment {
                    view: &self.texture_view,
                    resolve_target: None,
                    ops: wgpu::Operations {
                        load: wgpu::LoadOp::Clear(wgpu::Color {
                            r: 5.0 / 255.0,
                            g: 5.0 / 255.0,
                            b: 15.0 / 255.0,
                            a: 1.0,
                        }),
                        store: wgpu::StoreOp::Store,
                    },
                })],
                depth_stencil_attachment: None,
                timestamp_writes: None,
                occlusion_query_set: None,
            });

            pass.set_bind_group(0, Some(&self.bind_group), &[]);

            // 1) Spacetime grid
            if let Some(ref buf) = spacetime_buf {
                pass.set_pipeline(&self.pipelines.line);
                pass.set_vertex_buffer(0, buf.slice(..));
                pass.draw(0..spacetime_verts.len() as u32, 0..1);
            }

            // 2) Trails
            if let Some(ref buf) = trail_buf {
                pass.set_pipeline(&self.pipelines.line);
                pass.set_vertex_buffer(0, buf.slice(..));
                pass.draw(0..trail_verts.len() as u32, 0..1);
            }

            // 3) Sun glow (additive blend)
            if let Some(ref buf) = glow_buf {
                pass.set_pipeline(&self.pipelines.glow);
                pass.set_vertex_buffer(0, buf.slice(..));
                pass.draw(0..glow_verts.len() as u32, 0..1);
            }

            // 4) Circles (sun solid + planets)
            if let Some(ref buf) = circle_buf {
                pass.set_pipeline(&self.pipelines.circle);
                pass.set_vertex_buffer(0, buf.slice(..));
                pass.draw(0..circle_verts.len() as u32, 0..1);
            }

            // 5) Distance measurement line
            if let Some(ref buf) = dist_buf {
                pass.set_pipeline(&self.pipelines.line);
                pass.set_vertex_buffer(0, buf.slice(..));
                pass.draw(0..dist_line_verts.len() as u32, 0..1);
            }
        }

        // Copy texture to readback buffer
        let padded_bpr = Self::padded_bytes_per_row(self.width);
        encoder.copy_texture_to_buffer(
            wgpu::TexelCopyTextureInfo {
                texture: &self.texture,
                mip_level: 0,
                origin: wgpu::Origin3d::ZERO,
                aspect: wgpu::TextureAspect::All,
            },
            wgpu::TexelCopyBufferInfo {
                buffer: &self.readback_buffer,
                layout: wgpu::TexelCopyBufferLayout {
                    offset: 0,
                    bytes_per_row: Some(padded_bpr),
                    rows_per_image: Some(self.height),
                },
            },
            wgpu::Extent3d {
                width: self.width,
                height: self.height,
                depth_or_array_layers: 1,
            },
        );

        self.queue.submit(std::iter::once(encoder.finish()));

        // Map and read back
        let buffer_slice = self.readback_buffer.slice(..);
        let (sender, receiver) = std::sync::mpsc::channel();
        buffer_slice.map_async(wgpu::MapMode::Read, move |result| {
            let _ = sender.send(result);
        });
        self.device.poll(wgpu::Maintain::Wait);
        let _ = receiver.recv();

        {
            let data = buffer_slice.get_mapped_range();
            let unpadded_bpr = (self.width * 4) as usize;
            let padded = padded_bpr as usize;

            for row in 0..self.height as usize {
                let src_start = row * padded;
                let dst_start = row * unpadded_bpr;
                self.pixel_data[dst_start..dst_start + unpadded_bpr]
                    .copy_from_slice(&data[src_start..src_start + unpadded_bpr]);
            }
        }
        self.readback_buffer.unmap();

        &self.pixel_data
    }
}
