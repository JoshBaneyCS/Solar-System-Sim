# Camera Controls Plan

## Current Camera System Analysis

### Input Handlers (Go/Fyne)

All input handling lives in `internal/ui/input_handler.go` via the `InteractiveCanvas` widget, which implements Fyne's `Scrollable`, `Draggable`, `Focusable`, `MouseUp/MouseDown`, and `Tapped` interfaces.

| Input | Handler | Action | File:Line |
|-------|---------|--------|-----------|
| Scroll wheel | `Scrolled()` | Zoom in/out. `factor = 1.15^(DY/10)`, calls `viewport.AdjustZoom(factor)`. Logarithmic feel. | `input_handler.go:56-60` |
| Drag (no modifier) | `Dragged()` | 3D orbit rotation. `RotationY += dx * 0.01`, `RotationX += dy * 0.01`. Auto-enables `Use3D = true`. Pitch clamped to +/-90 degrees. | `input_handler.go:64-89` |
| Shift+Drag | `Dragged()` | Pan. `panScale = 0.002 / zoom`. Moves `PanX`/`PanY` in AU-space. | `input_handler.go:68-74` |
| Click (any) | `MouseDown()` | Requests keyboard focus. Records Shift modifier state for drag mode detection. | `input_handler.go:143-149` |
| W/A/S/D keys | `TypedKey()` | Pan camera. Step = `0.1 / zoom` AU. | `input_handler.go:118-125` |
| R key | `TypedKey()` | Zoom in by `1.15x` factor. | `input_handler.go:126-127` |
| F key | `TypedKey()` | Zoom out by `1/1.15x` factor. | `input_handler.go:128-129` |
| Q key | `TypedKey()` | Roll camera left (RotationZ -= 0.05). | `input_handler.go:130-133` |
| E key | `TypedKey()` | Roll camera right (RotationZ += 0.05). | `input_handler.go:134-137` |
| Zoom slider | `createControls()` | Logarithmic slider (-2 to 23), zoom = `2^value`. | `app.go:303-311` |
| Auto-Fit button | `createControls()` | Calls `viewport.AutoFit()` -- computes bounding box of all planets with 10% margin, sets zoom and resets pan. | `app.go:313-319` |
| Follow dropdown | `createControls()` | Select from "None (Free Camera)", "Sun", or any planet name. Sets `viewport.FollowBody` pointer. | `app.go:321-349` |
| Enable 3D checkbox | `createControls()` | Toggles `viewport.Use3D`. | `app.go:351-355` |
| Pitch/Yaw/Roll sliders | `createControls()` | Three sliders from -pi to +pi, step 0.1. Directly set `RotationX/Y/Z`. | `app.go:357-388` |
| Reset 3D View button | `createControls()` | Sets all rotations to 0. | `app.go:390-399` |

### Viewport State (`internal/viewport/viewport.go`)

| Field | Type | Range | Description |
|-------|------|-------|-------------|
| `Zoom` | `float64` | 0.01 -- 10,000,000 | Multiplier on `DefaultDisplayScale` (100 px/AU) |
| `PanX`, `PanY` | `float64` | unbounded | Pan offset in AU |
| `RotationX` | `float64` | -pi to +pi | Pitch (clamped to +/-90 degrees in drag handler) |
| `RotationY` | `float64` | -pi to +pi | Yaw |
| `RotationZ` | `float64` | -pi to +pi | Roll |
| `Use3D` | `bool` | -- | When false, skips rotation math in projection |
| `FollowBody` | `*physics.Body` | nil or pointer | Camera centers on this body's position |

### Projection Model

The current system is **not** a true 3D camera. It applies Euler rotations to world positions, then projects via an orthographic-like formula with a pseudo-perspective Z offset:

```
x_screen = (worldX - centerX) / AU * displayScale - panX * displayScale + canvasWidth / 2
y_screen = (worldY - centerY) / AU * displayScale - panY * displayScale + canvasHeight / 2
// Pseudo-3D offset:
x_screen -= worldZ / AU * displayScale * 0.5
y_screen -= worldZ / AU * displayScale * 0.8
```

This is a fixed oblique projection, not a perspective camera. There is no near/far plane, no field-of-view, and no depth buffer.

### Pain Points

1. **No true 3D camera.** The "3D view" is a rotated orthographic projection with hard-coded oblique Z offsets. No perspective foreshortening.
2. **Rotation via sliders is awkward.** The sliders provide 0.1-radian steps, making smooth orbiting difficult. Mouse drag orbit works but feels imprecise due to low sensitivity (0.01 rad/px).
3. **No smooth transitions.** Changing follow target, auto-fit, or zoom slider all snap instantly -- no interpolation or easing.
4. **Pan is in AU-space.** Pan sensitivity is zoom-dependent but the mapping is non-intuitive at extreme zoom levels.
5. **No body picking.** No way to click on a body to select/follow it. Must use the dropdown.
6. **Shift detection is fragile.** Shift state is only captured on MouseDown; if the user presses Shift during a drag, the mode does not change.
7. **Keyboard zoom (R/F) conflicts.** R is zoom-in in current Go code but conventionally R means "reset" in 3D viewers.
8. **No right-click context menu or secondary camera mode.**
9. **Thread safety is manual.** Every viewport access requires explicit Lock/RLock pairs.
10. **Auto-fit only considers X/Y.** Bodies with high Z (inclined orbits) can be off-screen after auto-fit.

---

## Target Bevy Camera Spec

### Approach: `bevy_panorbit_camera` as Starting Point

Use the [`bevy_panorbit_camera`](https://github.com/Plonq/bevy_panorbit_camera) crate as the foundation. It provides orbit, pan, and zoom out of the box with smooth interpolation. We extend it with follow-body, auto-fit, and keyboard shortcut systems.

```toml
# Cargo.toml
[dependencies]
bevy_panorbit_camera = "0.21"
```

If `bevy_panorbit_camera` proves too opinionated, implement a custom `OrbitCamera` resource (already defined in `ecs-data-model.md`) with equivalent logic.

### Camera Modes

| Mode | Activation | Behavior |
|------|------------|----------|
| **Orbit** (default) | Always active | Left-click drag rotates around focus point. Scroll wheel zooms. Middle-click drag pans. |
| **Follow** | Double-click a body, or select from UI | Camera smoothly transitions focus point to the followed body. Orbit controls still work around the body. |
| **Free-fly** (stretch goal) | Right-click drag | First-person camera: right-drag rotates view direction, WASD moves camera position. For close inspection of body surfaces. |

### Input Mapping

#### Mouse Controls

| Input | Bevy API | Action |
|-------|----------|--------|
| Left-click drag | `EventReader<MouseMotion>` + `Res<ButtonInput<MouseButton>>` (Left) | Orbit: yaw += dx * sensitivity, pitch += dy * sensitivity |
| Scroll wheel | `EventReader<MouseWheel>` | Zoom: `distance *= 1.0 - scroll_y * 0.1`. Logarithmic. |
| Middle-click drag | `EventReader<MouseMotion>` + `Res<ButtonInput<MouseButton>>` (Middle) | Pan: offset camera focus point in screen-space plane |
| Double-click (left) | Custom double-click detection (time between two `ButtonInput` press events < 300ms) | Pick body under cursor (ray cast), follow it |
| Right-click drag | `EventReader<MouseMotion>` + `Res<ButtonInput<MouseButton>>` (Right) | Free-fly rotation (stretch goal) |

#### Keyboard Controls

| Key | Bevy API | Action |
|-----|----------|--------|
| Space | `Res<ButtonInput<KeyCode>>` (Space, just_pressed) | Toggle play/pause. Sends `SimCommand::SetPlaying`. |
| `+` / `=` | `Res<ButtonInput<KeyCode>>` (Equal/NumpadAdd, pressed) | Increase time speed by 2x. Sends `SimCommand::SetTimeSpeed`. |
| `-` | `Res<ButtonInput<KeyCode>>` (Minus/NumpadSubtract, pressed) | Decrease time speed by 0.5x. |
| F | `Res<ButtonInput<KeyCode>>` (KeyF, just_pressed) | Auto-fit all visible bodies. Sends `AutoFitEvent`. |
| R | `Res<ButtonInput<KeyCode>>` (KeyR, just_pressed) | Reset camera to default position (top-down, zoom 1x, focus origin). |
| 1-9 | `Res<ButtonInput<KeyCode>>` (Digit1..Digit9, just_pressed) | Jump to planet (1=Mercury, 2=Venus, ... 8=Neptune, 9=Pluto). Sets follow target. |
| 0 | `Res<ButtonInput<KeyCode>>` (Digit0, just_pressed) | Follow Sun. |
| Escape | `Res<ButtonInput<KeyCode>>` (Escape, just_pressed) | Unfollow / deselect. Clears `follow_target` and `Selected` markers. |
| W/A/S/D | `Res<ButtonInput<KeyCode>>` (pressed, per-frame) | Pan camera (moves focus point in camera-local XZ plane). |
| Q/E | `Res<ButtonInput<KeyCode>>` (pressed, per-frame) | Roll camera. |
| F11 | `Res<ButtonInput<KeyCode>>` (F11, just_pressed) | Toggle fullscreen. |
| T | `Res<ButtonInput<KeyCode>>` (KeyT, just_pressed) | Toggle trails. |
| L | `Res<ButtonInput<KeyCode>>` (KeyL, just_pressed) | Toggle labels. |
| G | `Res<ButtonInput<KeyCode>>` (KeyG, just_pressed) | Toggle spacetime grid. |

### OrbitCamera Resource (from ECS Data Model)

```rust
#[derive(Resource)]
pub struct OrbitCamera {
    // Current state
    pub distance: f64,          // Distance from focus (render units)
    pub yaw: f64,               // Horizontal angle (radians)
    pub pitch: f64,             // Vertical angle (radians), clamped [-89deg, +89deg]
    pub roll: f64,              // Roll angle (radians)
    pub pan_offset: DVec3,      // Camera-local pan offset
    pub focus_point: DVec3,     // World-space focus target

    // Follow
    pub follow_target: Option<Entity>,

    // Animation targets (for smooth transitions)
    pub target_distance: f64,
    pub target_yaw: f64,
    pub target_pitch: f64,
    pub target_focus: DVec3,

    // UI readback
    pub zoom_level: f64,        // Logarithmic zoom for display

    // Configuration
    pub orbit_sensitivity: f32, // Default: 0.003
    pub pan_sensitivity: f32,   // Default: 0.001
    pub zoom_sensitivity: f32,  // Default: 0.1
    pub smoothing: f32,         // Lerp factor per frame. 0.0 = instant, 0.9 = very smooth. Default: 0.1
}
```

### System Design

```rust
pub struct CameraPlugin;

impl Plugin for CameraPlugin {
    fn build(&self, app: &mut App) {
        app
            .insert_resource(OrbitCamera::default())
            .add_event::<AutoFitEvent>()
            .add_event::<BodySelected>()
            .add_systems(Startup, spawn_camera)
            .add_systems(Update, (
                camera_orbit_input,
                camera_zoom_input,
                camera_pan_input,
                camera_keyboard_shortcuts,
                camera_body_picking,
                camera_follow_body,
                camera_auto_fit,
                camera_smooth_interpolation,
                camera_apply_transform,
            ).chain());
    }
}
```

#### System Details

**`camera_orbit_input`**
```rust
fn camera_orbit_input(
    mut camera: ResMut<OrbitCamera>,
    mouse_button: Res<ButtonInput<MouseButton>>,
    mut mouse_motion: EventReader<MouseMotion>,
) {
    if mouse_button.pressed(MouseButton::Left) {
        for ev in mouse_motion.read() {
            camera.target_yaw -= ev.delta.x as f64 * camera.orbit_sensitivity as f64;
            camera.target_pitch -= ev.delta.y as f64 * camera.orbit_sensitivity as f64;
            camera.target_pitch = camera.target_pitch
                .clamp(-89.0_f64.to_radians(), 89.0_f64.to_radians());
        }
    }
}
```

**`camera_zoom_input`**
```rust
fn camera_zoom_input(
    mut camera: ResMut<OrbitCamera>,
    mut scroll: EventReader<MouseWheel>,
    keyboard: Res<ButtonInput<KeyCode>>,
) {
    for ev in scroll.read() {
        let zoom_delta = ev.y as f64 * camera.zoom_sensitivity as f64;
        camera.target_distance *= (1.0 - zoom_delta).max(0.01);
        camera.target_distance = camera.target_distance.clamp(0.001, 1e8);
    }
}
```

**`camera_pan_input`**
```rust
fn camera_pan_input(
    mut camera: ResMut<OrbitCamera>,
    mouse_button: Res<ButtonInput<MouseButton>>,
    mut mouse_motion: EventReader<MouseMotion>,
) {
    if mouse_button.pressed(MouseButton::Middle) {
        for ev in mouse_motion.read() {
            // Pan in camera-local right/up plane
            let right = /* compute from yaw/pitch */;
            let up = /* compute from yaw/pitch */;
            camera.target_focus += right * (-ev.delta.x as f64) * camera.pan_sensitivity as f64
                                 + up * (ev.delta.y as f64) * camera.pan_sensitivity as f64;
        }
    }
}
```

**`camera_body_picking`**

Uses Bevy ray casting to detect which body the user clicked/double-clicked.

```rust
fn camera_body_picking(
    mut camera: ResMut<OrbitCamera>,
    mouse_button: Res<ButtonInput<MouseButton>>,
    windows: Query<&Window>,
    camera_q: Query<(&Camera, &GlobalTransform)>,
    bodies: Query<(Entity, &Transform, &CelestialBody, &DisplayRadius)>,
    mut double_click_timer: Local<Option<f64>>,
    time: Res<Time>,
) {
    if mouse_button.just_pressed(MouseButton::Left) {
        let now = time.elapsed_secs_f64();
        if let Some(last) = *double_click_timer {
            if now - last < 0.3 {
                // Double-click detected -- pick nearest body under cursor
                // Use camera.viewport_to_world() ray, test against body bounding spheres
                // If hit, set camera.follow_target = Some(entity)
                *double_click_timer = None;
                return;
            }
        }
        *double_click_timer = Some(now);
    }
}
```

**`camera_smooth_interpolation`**

The key system for smooth camera feel. Every frame, lerp/slerp the actual camera state toward the target state.

```rust
fn camera_smooth_interpolation(
    mut camera: ResMut<OrbitCamera>,
    time: Res<Time>,
) {
    let dt = time.delta_secs_f64();
    let t = 1.0 - (1.0 - camera.smoothing as f64).powf(dt * 60.0);  // frame-rate independent

    camera.distance = lerp(camera.distance, camera.target_distance, t);
    camera.yaw = lerp_angle(camera.yaw, camera.target_yaw, t);
    camera.pitch = lerp(camera.pitch, camera.target_pitch, t);
    camera.focus_point = camera.focus_point.lerp(camera.target_focus, t);
}

fn lerp(a: f64, b: f64, t: f64) -> f64 {
    a + (b - a) * t
}

fn lerp_angle(a: f64, b: f64, t: f64) -> f64 {
    let diff = ((b - a) + std::f64::consts::PI) % std::f64::consts::TAU - std::f64::consts::PI;
    a + diff * t
}
```

**`camera_apply_transform`**

Computes the final `Transform` from the orbit camera state:

```rust
fn camera_apply_transform(
    camera: Res<OrbitCamera>,
    mut query: Query<&mut Transform, With<Camera3d>>,
) {
    let Ok(mut transform) = query.get_single_mut() else { return };

    let focus = Vec3::new(
        camera.focus_point.x as f32,
        camera.focus_point.y as f32,
        camera.focus_point.z as f32,
    );

    // Spherical coordinates -> Cartesian offset
    let cos_pitch = camera.pitch.cos() as f32;
    let offset = Vec3::new(
        cos_pitch * camera.yaw.sin() as f32,
        camera.pitch.sin() as f32,
        cos_pitch * camera.yaw.cos() as f32,
    ) * camera.distance as f32;

    let eye = focus + offset + camera.pan_offset.as_vec3();

    *transform = Transform::from_translation(eye).looking_at(focus, Vec3::Y);

    // Apply roll
    if camera.roll.abs() > 1e-6 {
        transform.rotate_local_z(camera.roll as f32);
    }
}
```

**`camera_auto_fit`**

```rust
fn camera_auto_fit(
    mut camera: ResMut<OrbitCamera>,
    mut events: EventReader<AutoFitEvent>,
    bodies: Query<&Transform, With<CelestialBody>>,
) {
    for _ in events.read() {
        // Compute bounding sphere of all body positions
        let mut max_dist: f32 = 0.0;
        let mut center = Vec3::ZERO;
        let count = bodies.iter().count() as f32;

        for t in bodies.iter() {
            center += t.translation;
        }
        center /= count.max(1.0);

        for t in bodies.iter() {
            let d = t.translation.distance(center);
            if d > max_dist { max_dist = d; }
        }

        // Set camera to see entire bounding sphere with 10% margin
        camera.target_focus = DVec3::new(center.x as f64, center.y as f64, center.z as f64);
        camera.target_distance = (max_dist * 1.1 / (std::f32::consts::FRAC_PI_4).tan()) as f64;
        camera.target_pitch = 45.0_f64.to_radians();
        camera.target_yaw = 0.0;
        camera.follow_target = None;
        camera.pan_offset = DVec3::ZERO;
    }
}
```

### Migration Mapping Summary

| Go Input | Go Mechanism | Bevy System | Bevy API |
|----------|-------------|-------------|----------|
| Scroll wheel zoom | `InteractiveCanvas.Scrolled()` | `camera_zoom_input` | `EventReader<MouseWheel>` |
| Drag orbit | `InteractiveCanvas.Dragged()` (no Shift) | `camera_orbit_input` | `EventReader<MouseMotion>` + `Res<ButtonInput<MouseButton>>` |
| Shift+drag pan | `InteractiveCanvas.Dragged()` (Shift) | `camera_pan_input` | `EventReader<MouseMotion>` + `Res<ButtonInput<MouseButton>>` (Middle) |
| W/A/S/D pan | `InteractiveCanvas.TypedKey()` | `camera_keyboard_shortcuts` | `Res<ButtonInput<KeyCode>>` |
| R/F zoom keys | `InteractiveCanvas.TypedKey()` | Remapped: R=reset, F=auto-fit | `Res<ButtonInput<KeyCode>>` |
| Q/E roll | `InteractiveCanvas.TypedKey()` | `camera_keyboard_shortcuts` | `Res<ButtonInput<KeyCode>>` |
| Zoom slider | Fyne widget | `ui_controls_panel` egui slider | Writes `OrbitCamera.target_distance` |
| Pitch/Yaw/Roll sliders | Fyne widgets | Removed (mouse drag is primary) | -- |
| Follow dropdown | Fyne `Select` widget | `ui_controls_panel` egui combo + double-click picking | Writes `OrbitCamera.follow_target` |
| Auto-Fit button | Fyne `Button` | `ui_controls_panel` egui button + F key | Sends `AutoFitEvent` |
| Enable 3D checkbox | Fyne `Check` | Removed (always 3D in Bevy) | -- |
| Reset 3D View button | Fyne `Button` | R key + egui button | Resets `OrbitCamera` to defaults |

### Key Improvements Over Current System

1. **True perspective 3D camera** with depth buffer, field-of-view, and proper frustum culling.
2. **Smooth interpolation on all transitions** -- follow target changes, auto-fit, keyboard zoom all animate smoothly via per-frame lerp.
3. **Mouse-driven orbit** as primary rotation (no sliders needed). More intuitive and responsive.
4. **Middle-click pan** replaces Shift+drag -- standard 3D viewport convention.
5. **Double-click to follow** -- direct interaction rather than searching through a dropdown.
6. **Body picking via ray cast** -- click on a body to select it for info display or distance measurement.
7. **No manual thread safety** -- Bevy's ECS handles all resource access automatically.
8. **Frame-rate-independent smoothing** -- consistent feel at any framerate.
9. **3D auto-fit** considers all three axes, not just X/Y.
10. **Number keys for quick planet access** -- press 3 to jump to Earth, 4 to jump to Mars.
