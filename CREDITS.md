# Credits & Acknowledgments

## Project Author

**Joshua Baney** — [GitHub](https://github.com/JoshBaneyCS/)

---

## Scientific Data & References

### NASA / JPL
- **Planetary ephemeris and orbital elements** — All planet orbital parameters (semi-major axis, eccentricity, inclination, longitude of ascending node, argument of perihelion) are derived from NASA/JPL published data.
- **Physical constants** — Gravitational constant (G), planetary masses, radii, and rotation periods sourced from NASA Planetary Fact Sheets.
- **General Relativity corrections** — 1PN post-Newtonian perihelion precession formulae validated against JPL's published Mercury precession rate of ~43 arcsec/century.
- Website: [https://ssd.jpl.nasa.gov/](https://ssd.jpl.nasa.gov/)

### Classical References
- Curtis, H.D. — *Orbital Mechanics for Engineering Students* (orbital element conversion, Hohmann transfers)
- Numerical Recipes — RK4 and Verlet integrator implementations

---

## Texture Assets

All planet textures sourced from **Solar System Scope** under **CC BY 4.0** license.

| Asset | Resolution | License |
|-------|-----------|---------|
| Mercury albedo | 8K | CC BY 4.0 — Solar System Scope |
| Venus atmosphere | 4K | CC BY 4.0 — Solar System Scope |
| Earth diffuse | 8K | Free License — TurboSquid |
| Mars albedo | 8K | CC BY 4.0 — Solar System Scope |
| Jupiter albedo | 8K | CC BY 4.0 — Solar System Scope |
| Saturn albedo | 8K | CC BY 4.0 — Solar System Scope |
| Saturn ring alpha | 8K | CC BY 4.0 — Solar System Scope |
| Uranus albedo | 2K | CC BY 4.0 — Solar System Scope |
| Neptune albedo | 2K | CC BY 4.0 — Solar System Scope |
| Sun albedo | 8K | CC BY 4.0 — Solar System Scope |
| Milky Way skybox | 8K | CC BY 4.0 — Solar System Scope |
| Moon albedo | 2K | CC BY 4.0 — Solar System Scope |
| Io, Europa, Ganymede, Callisto, Titan | 2K | CC BY 4.0 — Solar System Scope |
| Pluto, Ceres, Vesta | 2K | CC BY 4.0 — Solar System Scope |

Source: [https://www.solarsystemscope.com/textures/](https://www.solarsystemscope.com/textures/)

See [assets/CREDITS.md](assets/CREDITS.md) for detailed per-file attribution.

---

## Graphics APIs & GPU Technology

This project renders real-time 3D graphics through the **wgpu** abstraction layer, which targets the following platform-native GPU APIs:

### <img src="https://upload.wikimedia.org/wikipedia/commons/2/21/Nvidia_logo.svg" alt="Nvidia" height="16"/> Nvidia
- **CUDA** and GPU computing — Pioneered the programmable GPU revolution that makes real-time N-body simulation possible.
- Vulkan and OpenGL drivers on Linux and Windows.
- Website: [https://www.nvidia.com/](https://www.nvidia.com/)

### <img src="https://upload.wikimedia.org/wikipedia/commons/7/7c/AMD_Logo.svg" alt="AMD" height="16"/> AMD
- **Vulkan drivers** and open-source GPU stack (RADV, Mesa) on Linux.
- OpenGL and DirectX 12 support on Windows.
- Website: [https://www.amd.com/](https://www.amd.com/)

### <img src="https://upload.wikimedia.org/wikipedia/en/c/cb/Metal_%28API%29_logo.png" alt="Metal" height="16"/> Apple Metal
- **Metal API** — macOS and iOS native GPU framework providing low-overhead, high-performance graphics.
- Primary rendering backend on Apple Silicon and Intel Macs.
- Reference: [https://developer.apple.com/metal/](https://developer.apple.com/metal/)

### Khronos Group
- **Vulkan** — Cross-platform, low-overhead GPU API (primary backend on Linux and Windows).
- **OpenGL** — Fallback rendering API for older hardware.
- Website: [https://www.khronos.org/](https://www.khronos.org/)

---

## Software Dependencies

### Core Engine
| Library | License | Role |
|---------|---------|------|
| [Bevy Engine](https://bevyengine.org/) | MIT / Apache-2.0 | 3D game engine (ECS, rendering, asset management) |
| [wgpu](https://wgpu.rs/) | MIT / Apache-2.0 | Cross-platform GPU abstraction (Vulkan, Metal, DX12, OpenGL) |
| [egui](https://www.egui.rs/) | MIT / Apache-2.0 | Immediate-mode GUI framework |
| [bevy_egui](https://github.com/mvlabat/bevy_egui) | MIT / Apache-2.0 | Bevy integration for egui |
| [bevy_embedded_assets](https://github.com/vleue/bevy_embedded_assets) | MIT / Apache-2.0 | Compile-time asset embedding |

### Rust Ecosystem
| Tool | Role |
|------|------|
| [rustc](https://www.rust-lang.org/) | Compiler |
| [cargo](https://doc.rust-lang.org/cargo/) | Build system & package manager |
| [rayon](https://github.com/rayon-rs/rayon) | Parallel iteration for N-body force computation |
| [serde](https://serde.rs/) | Serialization framework |
| [rand](https://docs.rs/rand/) | Random number generation (asteroid belt) |

### Legacy (Go)
| Library | License | Role |
|---------|---------|------|
| [Go](https://go.dev/) | BSD-3-Clause | Legacy CLI and headless simulation |
| [Fyne](https://fyne.io/) | BSD-3-Clause | Legacy GUI framework |

---

## AI Assistance

Portions of this codebase were developed with assistance from [Claude](https://claude.ai/) by [Anthropic](https://www.anthropic.com/).

---

## License

This project is licensed under the **GNU Affero General Public License v3.0** — see [LICENSE](LICENSE) for details.
