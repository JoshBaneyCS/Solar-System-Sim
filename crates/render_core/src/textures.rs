use std::path::Path;

/// Body names in the order they map to texture array layers.
/// Indices 0-7 = planets, index 8 = sun.
pub const BODY_NAMES: &[&str] = &[
    "mercury", "venus", "earth", "mars", "jupiter", "saturn", "uranus", "neptune", "sun",
];

/// Target resolution for all texture array layers.
const TARGET_WIDTH: u32 = 2048;
const TARGET_HEIGHT: u32 = 1024;

pub struct TextureAtlas {
    pub texture: wgpu::Texture,
    pub view: wgpu::TextureView,
    pub sampler: wgpu::Sampler,
    pub layer_count: u32,
}

impl TextureAtlas {
    /// Load textures from an asset directory.
    /// Expects `{asset_dir}/{body_name}/albedo.jpg` (or .png) for each body.
    /// Missing textures get a 1x1 white fallback layer.
    pub fn from_directory(
        device: &wgpu::Device,
        queue: &wgpu::Queue,
        asset_dir: &str,
    ) -> Option<Self> {
        let layer_count = BODY_NAMES.len() as u32;

        let texture = device.create_texture(&wgpu::TextureDescriptor {
            label: Some("planet_texture_atlas"),
            size: wgpu::Extent3d {
                width: TARGET_WIDTH,
                height: TARGET_HEIGHT,
                depth_or_array_layers: layer_count,
            },
            mip_level_count: 1,
            sample_count: 1,
            dimension: wgpu::TextureDimension::D2,
            format: wgpu::TextureFormat::Rgba8UnormSrgb,
            usage: wgpu::TextureUsages::TEXTURE_BINDING | wgpu::TextureUsages::COPY_DST,
            view_formats: &[],
        });

        for (i, name) in BODY_NAMES.iter().enumerate() {
            let rgba_data = load_body_texture(asset_dir, name);
            queue.write_texture(
                wgpu::TexelCopyTextureInfo {
                    texture: &texture,
                    mip_level: 0,
                    origin: wgpu::Origin3d {
                        x: 0,
                        y: 0,
                        z: i as u32,
                    },
                    aspect: wgpu::TextureAspect::All,
                },
                &rgba_data,
                wgpu::TexelCopyBufferLayout {
                    offset: 0,
                    bytes_per_row: Some(TARGET_WIDTH * 4),
                    rows_per_image: Some(TARGET_HEIGHT),
                },
                wgpu::Extent3d {
                    width: TARGET_WIDTH,
                    height: TARGET_HEIGHT,
                    depth_or_array_layers: 1,
                },
            );
        }

        let view = texture.create_view(&wgpu::TextureViewDescriptor {
            dimension: Some(wgpu::TextureViewDimension::D2Array),
            array_layer_count: Some(layer_count),
            ..Default::default()
        });

        let sampler = device.create_sampler(&wgpu::SamplerDescriptor {
            label: Some("planet_sampler"),
            address_mode_u: wgpu::AddressMode::Repeat,
            address_mode_v: wgpu::AddressMode::ClampToEdge,
            mag_filter: wgpu::FilterMode::Linear,
            min_filter: wgpu::FilterMode::Linear,
            mipmap_filter: wgpu::FilterMode::Linear,
            ..Default::default()
        });

        Some(Self {
            texture,
            view,
            sampler,
            layer_count,
        })
    }

    /// Create a minimal 1x1 white fallback atlas (used when no asset directory is provided).
    pub fn fallback(device: &wgpu::Device, queue: &wgpu::Queue) -> Self {
        let layer_count = BODY_NAMES.len() as u32;
        let texture = device.create_texture(&wgpu::TextureDescriptor {
            label: Some("fallback_texture_atlas"),
            size: wgpu::Extent3d {
                width: 1,
                height: 1,
                depth_or_array_layers: layer_count,
            },
            mip_level_count: 1,
            sample_count: 1,
            dimension: wgpu::TextureDimension::D2,
            format: wgpu::TextureFormat::Rgba8UnormSrgb,
            usage: wgpu::TextureUsages::TEXTURE_BINDING | wgpu::TextureUsages::COPY_DST,
            view_formats: &[],
        });

        // Write white pixels for all layers
        let white = [255u8, 255, 255, 255];
        for i in 0..layer_count {
            queue.write_texture(
                wgpu::TexelCopyTextureInfo {
                    texture: &texture,
                    mip_level: 0,
                    origin: wgpu::Origin3d { x: 0, y: 0, z: i },
                    aspect: wgpu::TextureAspect::All,
                },
                &white,
                wgpu::TexelCopyBufferLayout {
                    offset: 0,
                    bytes_per_row: Some(4),
                    rows_per_image: Some(1),
                },
                wgpu::Extent3d {
                    width: 1,
                    height: 1,
                    depth_or_array_layers: 1,
                },
            );
        }

        let view = texture.create_view(&wgpu::TextureViewDescriptor {
            dimension: Some(wgpu::TextureViewDimension::D2Array),
            array_layer_count: Some(layer_count),
            ..Default::default()
        });

        let sampler = device.create_sampler(&wgpu::SamplerDescriptor {
            label: Some("fallback_sampler"),
            address_mode_u: wgpu::AddressMode::Repeat,
            address_mode_v: wgpu::AddressMode::ClampToEdge,
            mag_filter: wgpu::FilterMode::Nearest,
            min_filter: wgpu::FilterMode::Nearest,
            ..Default::default()
        });

        Self {
            texture,
            view,
            sampler,
            layer_count,
        }
    }
}

/// Load a body's albedo texture, resize to TARGET_WIDTH x TARGET_HEIGHT, return RGBA bytes.
/// Falls back to a white pixel pattern if the file is missing or unreadable.
fn load_body_texture(asset_dir: &str, body_name: &str) -> Vec<u8> {
    let fallback = || -> Vec<u8> { vec![255u8; (TARGET_WIDTH * TARGET_HEIGHT * 4) as usize] };

    // Try .jpg first, then .png
    let base = Path::new(asset_dir).join(body_name);
    let path = if base.join("albedo.jpg").exists() {
        base.join("albedo.jpg")
    } else if base.join("albedo.png").exists() {
        base.join("albedo.png")
    } else {
        log::warn!("No texture found for {}, using fallback", body_name);
        return fallback();
    };

    match image::open(&path) {
        Ok(img) => {
            let resized = img.resize_exact(
                TARGET_WIDTH,
                TARGET_HEIGHT,
                image::imageops::FilterType::Lanczos3,
            );
            resized.to_rgba8().into_raw()
        }
        Err(e) => {
            log::warn!("Failed to load texture {:?}: {}, using fallback", path, e);
            fallback()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_body_names_count() {
        assert_eq!(BODY_NAMES.len(), 9); // 8 planets + sun
    }

    #[test]
    fn test_body_names_order() {
        assert_eq!(BODY_NAMES[0], "mercury");
        assert_eq!(BODY_NAMES[7], "neptune");
        assert_eq!(BODY_NAMES[8], "sun");
    }
}
