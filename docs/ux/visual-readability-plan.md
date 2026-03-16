# Visual Readability Plan

Bevy implementation approach for each visual feature in the Solar System Simulator.

---

## Orbital Trails

### Current Implementation
- `render/trail_buffer.go`: Ring buffer of up to 200 segments per body. Catmull-Rom interpolation with 4 sub-segments per trail point. Bresenham line drawing on a CPU raster. Per-pixel alpha blending (old = faint, new = opaque).
- Per-body `ShowTrail` toggle. Global `ShowTrails` toggle clears all trails.

### Bevy Implementation

**Recommended approach: `bevy_polyline` crate.**

```toml
[dependencies]
bevy_polyline = "0.10"
```

Each body with `Orbit { show_trail: true }` gets a child `Polyline` entity. The `TrailPlugin` updates vertices each frame.

**Trail data flow:**
1. `PhysicsPlugin::manage_trails` appends current position to `Orbit.trail_history` (a `VecDeque<DVec3>`, max 500 points).
2. `TrailPlugin::update_trail_polylines` reads `trail_history`, applies Catmull-Rom subdivision (4 sub-points per segment, yielding up to 2000 visual vertices), converts to render coordinates, and writes to the `Polyline` asset.

**Alpha gradient (trail fading):**

`bevy_polyline` supports per-vertex color. Assign alpha based on age:

```rust
fn trail_vertex_color(body_color: Color, index: usize, total: usize) -> Color {
    let t = index as f32 / total.max(1) as f32;  // 0.0 = oldest, 1.0 = newest
    let alpha = t * t;  // Quadratic fade -- newest is opaque, old fades fast
    body_color.with_alpha(alpha)
}
```

**Max trail length:** 500 physics points (configurable per body via `Orbit.max_trail_len`). At 60 Hz physics with 4x Catmull-Rom subdivision = 2000 visual vertices max. Well within GPU budget.

**Per-body toggle:** `Orbit.show_trail` controls per-body visibility. `SimulationConfig.show_trails` is the global master toggle. When global is off, set `Visibility::Hidden` on all trail polyline entities.

**Alternative considered: Bevy Gizmos API.** Gizmos are immediate-mode (re-drawn every frame). Simpler code but no alpha gradient and slightly higher CPU cost at 500+ segments. Suitable as a debug fallback but `bevy_polyline` is better for production.

**Alternative considered: Custom mesh.** A triangle-strip mesh with per-vertex color/alpha. More complex to build but allows trail width variation (thicker near body, thinner at tail). Consider this as a v2 enhancement.

---

## Asteroid Belt

### Current Implementation
- `render/belt.go`: 1500 particles with randomized Keplerian elements (2.1--3.3 AU, Kirkwood gaps at 2.5, 2.82, 2.95 AU excluded). Each frame, Kepler equation is solved per particle (5 Newton iterations) to compute position. Rendered as small circles on CPU raster.

### Bevy Implementation

**Recommended approach: Instanced mesh rendering.**

Spawn 1500 entities sharing a single `Mesh` handle (tiny `Sphere` with 8 segments) and a single gray `StandardMaterial`. Bevy automatically batches these into instanced draw calls.

```rust
fn spawn_belt_particles(
    mut commands: Commands,
    mut meshes: ResMut<Assets<Mesh>>,
    mut materials: ResMut<Assets<StandardMaterial>>,
) {
    let mesh = meshes.add(Sphere::new(0.02).mesh().uv(8, 4));
    let material = materials.add(StandardMaterial {
        base_color: Color::srgb(0.5, 0.5, 0.5),
        perceptual_roughness: 1.0,
        metallic: 0.0,
        unlit: true,  // No lighting on tiny particles
        ..default()
    });

    for _ in 0..1500 {
        let elements = random_belt_elements();
        commands.spawn((
            BeltParticle { /* elements */ },
            Mesh3d(mesh.clone()),
            MeshMaterial3d(material.clone()),
            Transform::default(),
            Visibility::default(),
        ));
    }
}
```

**Performance:** Bevy's automatic instancing renders all 1500 identical meshes in a single draw call. At 8-segment spheres (128 triangles each), this is 192K triangles total -- trivial for any GPU.

**Alternative considered: Point sprite material.** A custom shader that renders each particle as a camera-facing textured quad. Fewer vertices (4 per particle vs 128) but requires a custom shader. Worth exploring if belt particle count increases to 10K+.

**Position update:** Each frame, `update_belt_positions` solves Kepler's equation for each `BeltParticle` and updates `Transform.translation`. This is pure math on 1500 entities -- well under 1ms on any modern CPU.

---

## Comet Tails

### Current Implementation
- `render/renderer.go:477-523`: 8-segment gradient line pointing away from the Sun. Color fades from body color to transparent. Length inversely proportional to Sun distance.

### Bevy Implementation

**Recommended approach: Billboard gradient quad.**

For each entity with `CometTail` marker, spawn a child entity with a thin elongated `Quad` mesh and a custom unlit material with alpha gradient. The quad is oriented to always face the camera (billboard) and points away from the Sun.

```rust
fn update_comet_tails(
    sun_query: Query<&Transform, (With<CelestialBody>, Without<CometTail>)>,
    mut comet_query: Query<(&Transform, &CometTail, &Children)>,
    mut tail_query: Query<&mut Transform, (With<CometTailMesh>, Without<CometTail>)>,
    camera_query: Query<&Transform, With<Camera3d>>,
) {
    let sun_pos = /* find Sun transform */;
    let cam_pos = camera_query.single().translation;

    for (comet_transform, _, children) in comet_query.iter() {
        // Tail direction: away from Sun
        let away = (comet_transform.translation - sun_pos).normalize();
        // Tail length: inversely proportional to Sun distance
        let sun_dist = comet_transform.translation.distance(sun_pos);
        let tail_length = (5.0 / sun_dist.max(0.1)).clamp(0.5, 20.0);

        for &child in children.iter() {
            if let Ok(mut tail_transform) = tail_query.get_mut(child) {
                tail_transform.translation = away * tail_length * 0.5;
                tail_transform.scale = Vec3::new(0.1, tail_length, 0.1);
                // Orient tail to face camera while pointing away from Sun
                *tail_transform = tail_transform.looking_at(
                    comet_transform.translation + away * tail_length,
                    (cam_pos - comet_transform.translation).normalize(),
                );
            }
        }
    }
}
```

**Alternative: Particle system.** Use `bevy_hanabi` or a custom GPU particle emitter spawning particles that drift away from the Sun with decreasing alpha. More visually impressive but heavier. Consider as a v2 enhancement.

**Material:** Use `StandardMaterial` with `alpha_mode: AlphaMode::Blend`, `unlit: true`, and a gradient texture (white center fading to transparent edges).

---

## Planet Labels

### Current Implementation
- `render/renderer.go:417-475`: Fyne `canvas.Text` objects positioned at each body's screen-projected location. Offset below the body sphere. White text, 11px font.
- GPU mode: Fyne text overlaid on top of the GPU raster image.

### Bevy Implementation

**Recommended approach: `bevy_egui` screen-space text overlays.**

The advantage over Bevy's `Text2d` is that egui text renders at exact pixel positions without depth-buffer artifacts, is never occluded by 3D objects, and respects the UI theme.

```rust
fn ui_body_labels(
    mut egui_ctx: ResMut<EguiContext>,
    camera_q: Query<(&Camera, &GlobalTransform)>,
    bodies: Query<(&CelestialBody, &GlobalTransform), Without<Camera3d>>,
    config: Res<SimulationConfig>,
) {
    if !config.show_labels { return; }

    let (camera, cam_transform) = camera_q.single();

    egui::Area::new(egui::Id::new("body_labels"))
        .fixed_pos(egui::pos2(0.0, 0.0))
        .show(egui_ctx.ctx_mut(), |ui| {
            for (body, transform) in bodies.iter() {
                // Project 3D position to screen coordinates
                if let Ok(screen_pos) = camera.world_to_viewport(cam_transform, transform.translation()) {
                    let label_pos = egui::pos2(
                        screen_pos.x - 20.0,
                        screen_pos.y + 12.0,  // Offset below body
                    );
                    ui.painter().text(
                        label_pos,
                        egui::Align2::CENTER_TOP,
                        &body.name,
                        egui::FontId::proportional(12.0),
                        egui::Color32::from_rgba_premultiplied(220, 220, 230, 200),
                    );
                }
            }
        });
}
```

**Alternative: Bevy `Text2d`.** Spawn `Text2d` as a child of each body entity. Simpler setup but text scales with zoom (becomes huge/tiny), requires billboard orientation, and can be occluded by other 3D objects. Not recommended for labels.

**Visibility culling:** Skip labels for bodies behind the camera (check `world_to_viewport` returns `Ok`). Skip labels for very distant bodies (screen distance < 2px from another label) to avoid clutter.

---

## Distance Measurement

### Current Implementation
- `render/renderer.go:292-319`: Click two bodies to draw a line between them. Line displays distance in AU, km, and light-minutes. White dashed line with text at midpoint.

### Bevy Implementation

**Body picking via ray cast:**

When a user clicks on the viewport (not on egui panels), cast a ray from the camera through the cursor position and test against all body bounding spheres. Use `Camera::viewport_to_world()` to get the ray.

```rust
fn body_picking(
    mouse: Res<ButtonInput<MouseButton>>,
    windows: Query<&Window>,
    camera_q: Query<(&Camera, &GlobalTransform)>,
    bodies: Query<(Entity, &GlobalTransform, &DisplayRadius, &CelestialBody)>,
    mut selected: Query<Entity, With<Selected>>,
    mut commands: Commands,
) {
    if !mouse.just_pressed(MouseButton::Left) { return; }

    let window = windows.single();
    let Some(cursor) = window.cursor_position() else { return };
    let (camera, cam_transform) = camera_q.single();
    let Ok(ray) = camera.viewport_to_world(cam_transform, cursor) else { return };

    // Find nearest body intersecting the ray
    let mut best: Option<(Entity, f32)> = None;
    for (entity, transform, radius, _) in bodies.iter() {
        let center = transform.translation();
        let to_center = center - ray.origin;
        let t = to_center.dot(*ray.direction);
        if t < 0.0 { continue; }  // Behind camera
        let closest_point = ray.origin + *ray.direction * t;
        let dist = closest_point.distance(center);
        let pick_radius = radius.0.max(5.0);  // Minimum 5 render-unit pick radius
        if dist < pick_radius {
            if best.map_or(true, |(_, best_t)| t < best_t) {
                best = Some((entity, t));
            }
        }
    }

    if let Some((entity, _)) = best {
        // Toggle Selected marker. Keep at most 2 selected.
        let selected_count = selected.iter().count();
        if selected_count >= 2 {
            // Clear all selections
            for e in selected.iter() {
                commands.entity(e).remove::<Selected>();
            }
        }
        commands.entity(entity).insert(Selected);
    }
}
```

**Distance line rendering:**

When exactly 2 entities have the `Selected` component, draw a line between them with distance text.

```rust
fn render_distance_line(
    selected: Query<(&GlobalTransform, &CelestialBody), With<Selected>>,
    mut gizmos: Gizmos,
    mut egui_ctx: ResMut<EguiContext>,
    camera_q: Query<(&Camera, &GlobalTransform), Without<Selected>>,
) {
    let items: Vec<_> = selected.iter().collect();
    if items.len() != 2 { return; }

    let (t1, b1) = items[0];
    let (t2, b2) = items[1];
    let p1 = t1.translation();
    let p2 = t2.translation();

    // Draw dashed line using Gizmos
    gizmos.line(p1, p2, Color::WHITE);

    // Compute real distance (convert render units back to meters)
    let render_dist = p1.distance(p2) as f64;
    let meters = render_dist / RENDER_SCALE * AU;
    let au = meters / AU;
    let km = meters / 1000.0;
    let light_min = meters / (299_792_458.0 * 60.0);

    // Project midpoint to screen for text overlay
    let midpoint = (p1 + p2) * 0.5;
    let (camera, cam_t) = camera_q.single();
    if let Ok(screen) = camera.world_to_viewport(cam_t, midpoint) {
        // Draw via egui
        egui::Area::new(egui::Id::new("distance_label"))
            .fixed_pos(egui::pos2(screen.x, screen.y - 20.0))
            .show(egui_ctx.ctx_mut(), |ui| {
                ui.label(format!(
                    "{} -- {}: {:.4} AU | {:.0} km | {:.2} light-min",
                    b1.name, b2.name, au, km, light_min
                ));
            });
    }
}
```

---

## Skybox

### Current Implementation
- `render/textures.go:81-95`: Loads a milky way JPEG texture and draws it as the canvas background, stretched to fill.

### Bevy Implementation

Bevy has built-in skybox support via the `Skybox` component on the camera entity. Load a cubemap or equirectangular HDR image.

```rust
fn setup_skybox(
    mut commands: Commands,
    asset_server: Res<AssetServer>,
    camera_entity: Query<Entity, With<Camera3d>>,
) {
    let skybox_handle: Handle<Image> = asset_server.load("textures/skybox/milky_way.hdr");

    let entity = camera_entity.single();
    commands.entity(entity).insert(Skybox {
        image: skybox_handle,
        brightness: 200.0,
        rotation: Quat::IDENTITY,
    });
}
```

**Texture format:** Convert the existing milky way JPEG to an equirectangular HDR or KTX2 cubemap. Bevy's asset pipeline can load equirectangular images and convert to cubemap at runtime. For best performance, pre-convert to KTX2 with `ktx2enc`.

**Asset path:** `assets/textures/skybox/milky_way.ktx2` (or `.hdr`).

---

## Sun Glow

### Current Implementation
- `render/lighting.go:147-181`: Radial gradient circle drawn in CPU raster. Cached by diameter. Yellow-to-transparent gradient.

### Bevy Implementation

**Bevy's built-in bloom post-processing.**

The Sun entity already has an emissive material (defined in `SunBundle` in ecs-data-model.md). Enable `Bloom` on the camera:

```rust
fn spawn_camera(mut commands: Commands) {
    commands.spawn((
        Camera3d::default(),
        Camera {
            hdr: true,  // Required for bloom
            ..default()
        },
        Bloom {
            intensity: 0.3,
            low_frequency_boost: 0.5,
            composite_mode: BloomCompositeMode::Additive,
            ..default()
        },
        Tonemapping::TonyMcMapface,
        // ... skybox, transform, etc.
    ));
}
```

The Sun's emissive material (`emissive: LinearRgba::new(10.0, 8.0, 2.0, 1.0)`) will naturally produce a bloom glow. No custom shader needed. Bloom intensity can be exposed as a UI slider.

---

## Spacetime Grid

### Current Implementation
- `spacetime/spacetime.go`: Computes gravitational potential field on a 2D grid (40--150 lines based on zoom). Grid vertices are displaced by potential. Colors follow a purple-red-orange gradient. Cached and invalidated when camera moves > 5%.

### Bevy Implementation

**Dynamic mesh with vertex displacement and vertex colors.**

```rust
fn compute_spacetime_grid(
    config: Res<SpacetimeConfig>,
    bodies: Query<(&Transform, &CelestialBody)>,
    camera: Res<OrbitCamera>,
    mut meshes: ResMut<Assets<Mesh>>,
    grid_entity: Query<&Mesh3d, With<SpacetimeGrid>>,
) {
    if !config.enabled || config.cache_valid { return; }

    let resolution = config.grid_resolution;  // 40-150 based on zoom

    // Build grid positions in the ecliptic plane (Y=0)
    let mut positions = Vec::new();
    let mut colors = Vec::new();
    let mut indices = Vec::new();

    let bounds = camera_view_bounds(&camera);  // XZ extent visible to camera

    for i in 0..=resolution {
        for j in 0..=resolution {
            let x = bounds.min.x + (bounds.max.x - bounds.min.x) * i as f32 / resolution as f32;
            let z = bounds.min.z + (bounds.max.z - bounds.min.z) * j as f32 / resolution as f32;

            // Compute gravitational potential at this point
            let mut potential: f64 = 0.0;
            for (transform, body) in bodies.iter() {
                let dx = x as f64 - transform.translation().x as f64;
                let dz = z as f64 - transform.translation().z as f64;
                let r = (dx * dx + dz * dz).sqrt().max(0.1);
                potential += body.mass / r;
            }

            // Displace Y by potential (clamped)
            let displacement = (potential * 1e-20).clamp(-5.0, 0.0) as f32;
            positions.push([x, displacement, z]);

            // Color by potential magnitude
            let t = (-displacement / 5.0).clamp(0.0, 1.0);
            let color = potential_color(t);  // purple -> red -> orange
            colors.push(color);
        }
    }

    // Generate line indices (wireframe grid)
    for i in 0..=resolution {
        for j in 0..resolution {
            indices.push(i * (resolution + 1) + j);
            indices.push(i * (resolution + 1) + j + 1);
        }
    }
    for j in 0..=resolution {
        for i in 0..resolution {
            indices.push(i * (resolution + 1) + j);
            indices.push((i + 1) * (resolution + 1) + j);
        }
    }

    // Update mesh asset
    // Use Mesh with PrimitiveTopology::LineList and vertex colors
}

fn potential_color(t: f32) -> [f32; 4] {
    // t: 0.0 = low potential (flat), 1.0 = high potential (deep well)
    if t < 0.5 {
        let s = t * 2.0;
        [0.5 * s + 0.3 * (1.0 - s), 0.0, 0.8 * (1.0 - s), 0.6]  // purple -> red
    } else {
        let s = (t - 0.5) * 2.0;
        [1.0, 0.5 * s, 0.0, 0.6]  // red -> orange
    }
}
```

**Material:** Use a wireframe-capable material. Bevy's `StandardMaterial` with `wireframe: true` (requires `WireframePlugin`) or a custom line-rendering shader. The vertex colors provide the heat-map coloring.

**Cache invalidation:** Store the last camera bounds in `SpacetimeConfig`. Recompute only when camera moves > 5% of view extent or a body moves significantly.

---

## Trajectory Overlay

### Current Implementation
- `render/renderer.go:365-414`: Line strip from launch planner trajectory points. Color gradient from green (departure) to red (arrival). For Earth-centered trajectories, positions offset by Earth's current position.

### Bevy Implementation

Use `bevy_polyline` (same crate as orbital trails). Single `Polyline` entity with `TrajectoryOverlay` marker.

```rust
fn render_trajectory_overlay(
    launch_state: Res<LaunchState>,
    mut polylines: ResMut<Assets<Polyline>>,
    overlay_q: Query<&PolylineHandle, With<TrajectoryOverlay>>,
    earth_q: Query<&Transform, (With<CelestialBody>, /* earth filter */)>,
) {
    let Some(trajectory) = &launch_state.trajectory else { return };

    let earth_offset = if trajectory.frame == Frame::EarthCentered {
        earth_q.single().translation
    } else {
        Vec3::ZERO
    };

    let total = trajectory.points.len();
    let vertices: Vec<PolylineVertex> = trajectory.points.iter().enumerate().map(|(i, pt)| {
        let t = i as f32 / total.max(1) as f32;
        let pos = physics_to_render(pt.position) + earth_offset;
        let color = Color::srgb(t, 1.0 - t, 0.0);  // green -> red gradient
        PolylineVertex { position: pos, color }
    }).collect();

    // Update polyline asset
    if let Ok(handle) = overlay_q.get_single() {
        if let Some(polyline) = polylines.get_mut(&handle.0) {
            polyline.vertices = vertices;
        }
    }
}
```

**Vehicle marker:** During mission playback, a small green sphere (`VehicleMarker` entity) is positioned at the interpolated trajectory point. Transform updated each frame by `update_mission_playback` system.

---

## Sense of Scale

### Size Exaggeration Strategy

The real solar system is impossible to visualize at true scale -- planets are invisible dots separated by vast distances. The current Go app uses `PhysicalRadius` for zoom-dependent scaling with a minimum pixel size.

**Bevy approach: Three-tier display radius.**

| Zoom Level | Strategy | Description |
|------------|----------|-------------|
| Zoomed out (solar system view) | `DisplayRadius` minimum | All planets are at least 4 render-units across. Sizes are relative (Jupiter > Earth > Mars) but exaggerated ~1000x. |
| Mid zoom (planetary neighborhood) | Interpolated | Blend between display radius and physical radius based on camera distance. |
| Zoomed in (single body) | `PhysicalRadius` | True physical radius in render units. At extreme zoom, body fills the viewport. Cap at 5000 render-units to avoid precision issues. |

```rust
fn update_display_radius(
    camera: Res<OrbitCamera>,
    mut bodies: Query<(&PhysicalRadius, &DisplayRadius, &mut Transform, &GlobalTransform)>,
) {
    for (phys_r, disp_r, mut transform, global) in bodies.iter_mut() {
        let cam_dist = camera.focus_point.distance(
            DVec3::new(
                global.translation().x as f64,
                global.translation().y as f64,
                global.translation().z as f64,
            )
        );

        // Physical radius in render units
        let phys_render = (phys_r.0 / AU * RENDER_SCALE) as f32;
        // Minimum visible radius (display radius)
        let min_render = disp_r.0;

        // Screen-space angular size
        let angular = phys_render / cam_dist.max(0.001) as f32;

        // Use physical radius when it would be > minimum pixel size, otherwise use display radius
        let radius = if angular > 0.005 {
            phys_render.min(5000.0)
        } else {
            min_render
        };

        transform.scale = Vec3::splat(radius);
    }
}
```

**Distance markers:** At certain zoom levels, draw concentric circles at 1 AU intervals using Bevy `Gizmos`:

```rust
fn draw_distance_markers(
    mut gizmos: Gizmos,
    camera: Res<OrbitCamera>,
) {
    let max_au = (camera.distance / RENDER_SCALE as f64) as i32 + 5;
    for au in 1..=max_au.min(50) {
        let radius = au as f32 * RENDER_SCALE as f32;
        gizmos.circle(
            Isometry3d::from_rotation(Quat::from_rotation_x(std::f32::consts::FRAC_PI_2)),
            radius,
            Color::srgba(0.3, 0.3, 0.5, 0.15),
        ).resolution(128);
    }
}
```

**Zoom-dependent detail levels (LOD):** Reduce mesh complexity for distant bodies.

| Distance from Camera | Sphere Segments | Triangle Count |
|---------------------|----------------|----------------|
| < 10 render-units | 64x32 | ~4096 |
| 10--100 | 32x18 | ~1152 |
| > 100 | 16x8 | ~256 |

Bevy does not have built-in LOD, but we can swap `Mesh3d` handles based on camera distance. Three pre-generated sphere meshes at startup.

---

## Summary Table

| Feature | Approach | Crate/API | Performance Notes |
|---------|----------|-----------|-------------------|
| Orbital trails | `bevy_polyline` with per-vertex alpha | `bevy_polyline` | 28 polylines, 2000 verts each max. Negligible GPU cost. |
| Asteroid belt | Instanced mesh (8-seg sphere) | Bevy built-in instancing | 1 draw call for 1500 particles. ~192K tris. |
| Comet tails | Billboard gradient quad | `StandardMaterial` + alpha blend | 4 quads. Trivial. |
| Planet labels | Screen-space egui text | `bevy_egui` | ~28 text draws. Negligible. |
| Distance line | Gizmos line + egui text | Bevy `Gizmos` + `bevy_egui` | 1 line + 1 text. Trivial. |
| Skybox | Bevy `Skybox` component | Bevy built-in | 1 cubemap sample per pixel. Standard. |
| Sun glow | Bloom post-processing | Bevy `Bloom` | Full-screen post-process. Built-in and optimized. |
| Spacetime grid | Dynamic mesh with vertex displacement | Custom mesh + `WireframePlugin` | 150x150 grid max = 45K verts. Cached. |
| Trajectory overlay | `bevy_polyline` with color gradient | `bevy_polyline` | 1 polyline, ~1000 verts. Trivial. |
| Scale handling | Three-tier display radius + distance markers | Custom system + `Gizmos` | Per-body scale update each frame. |
