use bevy::prelude::*;

use crate::physics_plugin::{CelestialBody, SimulationConfig, Sun};

// ---------------------------------------------------------------------------
// Components
// ---------------------------------------------------------------------------

/// Ring buffer storing recent world positions for orbital trail rendering.
#[derive(Component)]
pub struct TrailBuffer {
    positions: Vec<Vec3>,
    head: usize,
    len: usize,
    capacity: usize,
}

impl TrailBuffer {
    pub fn new(capacity: usize) -> Self {
        Self {
            positions: vec![Vec3::ZERO; capacity],
            head: 0,
            len: 0,
            capacity,
        }
    }

    pub fn push(&mut self, pos: Vec3) {
        self.positions[self.head] = pos;
        self.head = (self.head + 1) % self.capacity;
        if self.len < self.capacity {
            self.len += 1;
        }
    }

    pub fn clear(&mut self) {
        self.head = 0;
        self.len = 0;
    }

    /// Iterate positions from oldest to newest.
    pub fn iter(&self) -> impl Iterator<Item = Vec3> + '_ {
        let start = if self.len < self.capacity {
            0
        } else {
            self.head
        };
        (0..self.len).map(move |i| {
            let idx = (start + i) % self.capacity;
            self.positions[idx]
        })
    }
}

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct TrailPlugin;

impl Plugin for TrailPlugin {
    fn build(&self, app: &mut App) {
        app.insert_resource(TrailSampleTimer(Timer::from_seconds(
            0.1,
            TimerMode::Repeating,
        )))
        .add_systems(PostUpdate, (sample_trail_positions, draw_trails).chain());
    }
}

#[derive(Resource)]
struct TrailSampleTimer(Timer);

// ---------------------------------------------------------------------------
// Systems
// ---------------------------------------------------------------------------

/// Sample body positions into trail buffers at a throttled rate.
fn sample_trail_positions(
    time: Res<Time>,
    mut timer: ResMut<TrailSampleTimer>,
    config: Res<SimulationConfig>,
    mut query: Query<(&Transform, &mut TrailBuffer), With<CelestialBody>>,
) {
    timer.0.tick(time.delta());
    if !timer.0.just_finished() || !config.is_playing {
        return;
    }
    for (transform, mut trail) in &mut query {
        trail.push(transform.translation);
    }
}

/// Planet trail colors keyed by name.
fn trail_color(name: &str) -> Color {
    match name {
        "Mercury" => Color::srgba(0.663, 0.663, 0.663, 0.5),
        "Venus" => Color::srgba(1.0, 0.776, 0.286, 0.5),
        "Earth" => Color::srgba(0.392, 0.584, 0.929, 0.5),
        "Mars" => Color::srgba(0.757, 0.267, 0.055, 0.5),
        "Jupiter" => Color::srgba(0.847, 0.792, 0.616, 0.5),
        "Saturn" => Color::srgba(0.980, 0.871, 0.643, 0.5),
        "Uranus" => Color::srgba(0.310, 0.816, 0.906, 0.5),
        "Neptune" => Color::srgba(0.247, 0.329, 0.729, 0.5),
        "Pluto" => Color::srgba(0.824, 0.745, 0.667, 0.5),
        _ => Color::srgba(0.5, 0.5, 0.5, 0.3),
    }
}

/// Draw trail gizmos for all bodies.
fn draw_trails(
    config: Res<SimulationConfig>,
    query: Query<(&CelestialBody, &TrailBuffer), Without<Sun>>,
    mut gizmos: Gizmos,
) {
    if !config.show_trails {
        return;
    }

    for (body, trail) in &query {
        if trail.len < 2 {
            continue;
        }

        let base_color = trail_color(&body.name);
        let points: Vec<Vec3> = trail.iter().collect();

        // Draw with alpha falloff (older = more transparent)
        let total = points.len();
        for i in 0..total - 1 {
            let alpha = (i as f32 / total as f32) * 0.6;
            let color = base_color.with_alpha(alpha);
            gizmos.line(points[i], points[i + 1], color);
        }
    }
}
