use bevy::input::mouse::{MouseMotion, MouseWheel};
use bevy::prelude::*;
use bevy_egui::EguiContexts;

// ---------------------------------------------------------------------------
// Resources
// ---------------------------------------------------------------------------

/// Orbit camera state: the camera orbits around a focus point.
#[derive(Resource)]
pub struct OrbitCamera {
    /// Distance from the focus point (Bevy world units).
    pub distance: f32,
    /// Horizontal angle in radians.
    pub yaw: f32,
    /// Vertical angle in radians (clamped to avoid gimbal lock).
    pub pitch: f32,
    /// The point the camera orbits around (world coordinates).
    pub focus: Vec3,
    /// Sensitivity multipliers.
    pub rotate_sensitivity: f32,
    pub zoom_sensitivity: f32,
    pub pan_sensitivity: f32,
}

impl Default for OrbitCamera {
    fn default() -> Self {
        Self {
            distance: 50.0,
            yaw: 0.0,
            // Start looking down at 45 degrees
            pitch: std::f32::consts::FRAC_PI_4,
            focus: Vec3::ZERO,
            rotate_sensitivity: 0.005,
            zoom_sensitivity: 0.15,
            pan_sensitivity: 0.02,
        }
    }
}

/// Marker component for the orbit camera entity.
#[derive(Component)]
pub struct OrbitCameraMarker;

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct CameraPlugin;

impl Plugin for CameraPlugin {
    fn build(&self, app: &mut App) {
        app.insert_resource(OrbitCamera::default())
            .insert_resource(EguiWantsInput::default())
            .add_systems(Startup, spawn_camera)
            .add_systems(
                Update,
                (
                    update_egui_wants_input,
                    camera_zoom,
                    camera_rotate,
                    camera_pan,
                    camera_keyboard,
                    camera_apply_transform,
                )
                    .chain(),
            );
    }
}

// ---------------------------------------------------------------------------
// Startup
// ---------------------------------------------------------------------------

fn spawn_camera(mut commands: Commands) {
    commands.spawn((
        Camera3d::default(),
        Projection::Perspective(PerspectiveProjection {
            far: 5000.0,
            ..default()
        }),
        Transform::from_xyz(0.0, 50.0, 50.0).looking_at(Vec3::ZERO, Vec3::Y),
        OrbitCameraMarker,
    ));
}

// ---------------------------------------------------------------------------
// Input systems
// ---------------------------------------------------------------------------

/// Returns true if egui wants the input this frame.
#[derive(Resource, Default)]
pub struct EguiWantsInput {
    pub pointer: bool,
    pub keyboard: bool,
}

fn update_egui_wants_input(
    mut contexts: EguiContexts,
    mut wants: ResMut<EguiWantsInput>,
) {
    let ctx = contexts.ctx_mut();
    wants.pointer = ctx.wants_pointer_input();
    wants.keyboard = ctx.wants_keyboard_input();
}

/// Mouse scroll to zoom in/out.
fn camera_zoom(
    mut orbit: ResMut<OrbitCamera>,
    mut scroll_events: EventReader<MouseWheel>,
    egui_wants: Res<EguiWantsInput>,
) {
    if egui_wants.pointer {
        scroll_events.clear();
        return;
    }
    for ev in scroll_events.read() {
        let delta = ev.y;
        // Logarithmic zoom: multiply distance by a factor
        let factor = 1.0 - delta * orbit.zoom_sensitivity;
        orbit.distance *= factor;
        orbit.distance = orbit.distance.clamp(0.5, 5000.0);
    }
}

/// Right-mouse drag to rotate the orbit camera.
fn camera_rotate(
    mut orbit: ResMut<OrbitCamera>,
    mouse_button: Res<ButtonInput<MouseButton>>,
    mut motion_events: EventReader<MouseMotion>,
    egui_wants: Res<EguiWantsInput>,
) {
    if egui_wants.pointer {
        motion_events.clear();
        return;
    }
    if !mouse_button.pressed(MouseButton::Right) && !mouse_button.pressed(MouseButton::Left) {
        // Drain events to avoid stale deltas
        motion_events.clear();
        return;
    }

    for ev in motion_events.read() {
        orbit.yaw -= ev.delta.x * orbit.rotate_sensitivity;
        orbit.pitch -= ev.delta.y * orbit.rotate_sensitivity;
        // Clamp pitch to avoid flipping
        orbit.pitch = orbit
            .pitch
            .clamp(-std::f32::consts::FRAC_PI_2 + 0.01, std::f32::consts::FRAC_PI_2 - 0.01);
    }
}

/// Middle-mouse drag to pan the focus point.
fn camera_pan(
    mut orbit: ResMut<OrbitCamera>,
    mouse_button: Res<ButtonInput<MouseButton>>,
    mut motion_events: EventReader<MouseMotion>,
    egui_wants: Res<EguiWantsInput>,
) {
    if egui_wants.pointer {
        motion_events.clear();
        return;
    }
    if !mouse_button.pressed(MouseButton::Middle) {
        motion_events.clear();
        return;
    }

    for ev in motion_events.read() {
        // Pan in the camera's local XY plane, scaled by distance
        let pan_speed = orbit.pan_sensitivity * orbit.distance * 0.01;
        let right = Vec3::new(orbit.yaw.cos(), 0.0, -orbit.yaw.sin());
        let up = Vec3::Y;
        orbit.focus -= right * ev.delta.x * pan_speed;
        orbit.focus += up * ev.delta.y * pan_speed;
    }
}

/// Keyboard controls: WASD for pan, QE for rotation, RF or +/- for zoom.
fn camera_keyboard(
    mut orbit: ResMut<OrbitCamera>,
    keys: Res<ButtonInput<KeyCode>>,
    time: Res<Time>,
    egui_wants: Res<EguiWantsInput>,
) {
    if egui_wants.keyboard {
        return;
    }
    let dt = time.delta_secs();
    let move_speed = orbit.distance * 0.5 * dt;
    let rot_speed = 1.0 * dt;

    // Compute camera-relative forward and right vectors on the XZ plane
    let forward = Vec3::new(-orbit.yaw.sin(), 0.0, -orbit.yaw.cos());
    let right = Vec3::new(orbit.yaw.cos(), 0.0, -orbit.yaw.sin());

    if keys.pressed(KeyCode::KeyW) || keys.pressed(KeyCode::ArrowUp) {
        orbit.focus += forward * move_speed;
    }
    if keys.pressed(KeyCode::KeyS) || keys.pressed(KeyCode::ArrowDown) {
        orbit.focus -= forward * move_speed;
    }
    if keys.pressed(KeyCode::KeyA) || keys.pressed(KeyCode::ArrowLeft) {
        orbit.focus -= right * move_speed;
    }
    if keys.pressed(KeyCode::KeyD) || keys.pressed(KeyCode::ArrowRight) {
        orbit.focus += right * move_speed;
    }

    // Q/E to rotate
    if keys.pressed(KeyCode::KeyQ) {
        orbit.yaw += rot_speed;
    }
    if keys.pressed(KeyCode::KeyE) {
        orbit.yaw -= rot_speed;
    }

    // R/F to zoom
    if keys.pressed(KeyCode::KeyR) {
        orbit.distance *= 1.0 - 0.5 * dt;
        orbit.distance = orbit.distance.max(0.5);
    }
    if keys.pressed(KeyCode::KeyF) {
        orbit.distance *= 1.0 + 0.5 * dt;
        orbit.distance = orbit.distance.min(5000.0);
    }
}

// ---------------------------------------------------------------------------
// Apply orbit camera state to the actual camera Transform
// ---------------------------------------------------------------------------

fn camera_apply_transform(
    orbit: Res<OrbitCamera>,
    mut query: Query<&mut Transform, With<OrbitCameraMarker>>,
) {
    for mut transform in &mut query {
        // Spherical coordinates -> Cartesian offset
        let x = orbit.distance * orbit.pitch.cos() * orbit.yaw.sin();
        let y = orbit.distance * orbit.pitch.sin();
        let z = orbit.distance * orbit.pitch.cos() * orbit.yaw.cos();

        let eye = orbit.focus + Vec3::new(x, y, z);
        *transform = Transform::from_translation(eye).looking_at(orbit.focus, Vec3::Y);
    }
}
