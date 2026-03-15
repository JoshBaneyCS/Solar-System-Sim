use std::ffi::CString;
use std::os::raw::c_char;
use std::ptr;

/// Hardware performance tier based on GPU capabilities.
#[repr(u8)]
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum HardwareTier {
    /// Discrete GPU or Apple Silicon — full quality, RT enabled
    High = 2,
    /// Integrated GPU — reduced RT samples
    Medium = 1,
    /// CPU/software fallback — RT disabled
    Low = 0,
}

/// GPU hardware information detected at renderer creation time.
#[repr(C)]
pub struct HardwareInfo {
    /// GPU vendor name (e.g. "NVIDIA", "AMD", "Intel", "Apple")
    pub vendor: *mut c_char,
    /// GPU device name (e.g. "Apple M1 Pro", "NVIDIA GeForce RTX 4090")
    pub device_name: *mut c_char,
    /// Graphics backend in use (e.g. "Metal", "Vulkan", "Dx12")
    pub backend: *mut c_char,
    /// Device type: "DiscreteGpu", "IntegratedGpu", "Cpu", "Other"
    pub device_type: *mut c_char,
    /// Maximum supported 2D texture dimension
    pub max_texture_size: u32,
    /// Hardware tier (0=Low, 1=Medium, 2=High)
    pub tier: u8,
}

/// Detect GPU hardware by probing the wgpu adapter.
pub fn detect_hardware() -> Option<HardwareInfo> {
    let instance = wgpu::Instance::new(&wgpu::InstanceDescriptor {
        backends: wgpu::Backends::all(),
        ..Default::default()
    });

    let adapter = pollster::block_on(instance.request_adapter(&wgpu::RequestAdapterOptions {
        power_preference: wgpu::PowerPreference::HighPerformance,
        compatible_surface: None,
        force_fallback_adapter: false,
    }))?;

    let info = adapter.get_info();
    let limits = adapter.limits();

    let vendor_str = vendor_name(info.vendor);
    let device_str = info.name.clone();
    let backend_str = format!("{:?}", info.backend);
    let device_type_str = format!("{:?}", info.device_type);
    let max_tex = limits.max_texture_dimension_2d;

    let tier = classify_tier(&info);

    Some(HardwareInfo {
        vendor: to_c_string(&vendor_str),
        device_name: to_c_string(&device_str),
        backend: to_c_string(&backend_str),
        device_type: to_c_string(&device_type_str),
        max_texture_size: max_tex,
        tier: tier as u8,
    })
}

/// Detect hardware from an existing adapter (avoids double-probing when renderer is created).
pub fn detect_hardware_from_adapter(adapter: &wgpu::Adapter) -> HardwareInfo {
    let info = adapter.get_info();
    let limits = adapter.limits();

    let vendor_str = vendor_name(info.vendor);
    let device_str = info.name.clone();
    let backend_str = format!("{:?}", info.backend);
    let device_type_str = format!("{:?}", info.device_type);
    let max_tex = limits.max_texture_dimension_2d;

    let tier = classify_tier(&info);

    HardwareInfo {
        vendor: to_c_string(&vendor_str),
        device_name: to_c_string(&device_str),
        backend: to_c_string(&backend_str),
        device_type: to_c_string(&device_type_str),
        max_texture_size: max_tex,
        tier: tier as u8,
    }
}

/// Map wgpu vendor ID to a human-readable name.
fn vendor_name(vendor_id: u32) -> String {
    match vendor_id {
        0x10DE => "NVIDIA".to_string(),
        0x1002 => "AMD".to_string(),
        0x8086 => "Intel".to_string(),
        0x106B => "Apple".to_string(),
        // wgpu on Metal often reports 0 for Apple GPUs
        0 => "Apple".to_string(),
        other => format!("Unknown(0x{:04X})", other),
    }
}

/// Classify hardware into a performance tier.
fn classify_tier(info: &wgpu::AdapterInfo) -> HardwareTier {
    match info.device_type {
        wgpu::DeviceType::DiscreteGpu => HardwareTier::High,
        wgpu::DeviceType::IntegratedGpu => {
            // Apple Silicon integrated GPUs are high-tier
            if info.vendor == 0x106B || info.vendor == 0 {
                if info.backend == wgpu::Backend::Metal {
                    return HardwareTier::High;
                }
            }
            HardwareTier::Medium
        }
        wgpu::DeviceType::Cpu => HardwareTier::Low,
        _ => HardwareTier::Medium,
    }
}

fn to_c_string(s: &str) -> *mut c_char {
    CString::new(s)
        .map(|c| c.into_raw())
        .unwrap_or(ptr::null_mut())
}

// --- FFI exports ---

/// Detect GPU hardware info. Returns null on failure.
/// Caller must free the returned pointer with `render_free_hardware_info`.
#[unsafe(no_mangle)]
pub extern "C" fn render_get_hardware_info() -> *mut HardwareInfo {
    match detect_hardware() {
        Some(info) => Box::into_raw(Box::new(info)),
        None => ptr::null_mut(),
    }
}

/// Free a HardwareInfo struct returned by `render_get_hardware_info`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn render_free_hardware_info(info: *mut HardwareInfo) {
    if info.is_null() {
        return;
    }
    let info = unsafe { Box::from_raw(info) };
    // Free the CStrings
    if !info.vendor.is_null() {
        let _ = unsafe { CString::from_raw(info.vendor) };
    }
    if !info.device_name.is_null() {
        let _ = unsafe { CString::from_raw(info.device_name) };
    }
    if !info.backend.is_null() {
        let _ = unsafe { CString::from_raw(info.backend) };
    }
    if !info.device_type.is_null() {
        let _ = unsafe { CString::from_raw(info.device_type) };
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_vendor_name() {
        assert_eq!(vendor_name(0x10DE), "NVIDIA");
        assert_eq!(vendor_name(0x1002), "AMD");
        assert_eq!(vendor_name(0x8086), "Intel");
        assert_eq!(vendor_name(0x106B), "Apple");
        assert_eq!(vendor_name(0), "Apple");
    }
}
