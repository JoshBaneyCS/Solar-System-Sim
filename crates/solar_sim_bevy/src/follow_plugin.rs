use bevy::prelude::*;

use crate::camera_plugin::OrbitCamera;
use crate::physics_plugin::CelestialBody;
use crate::ui_bodies::FollowTarget;

// ---------------------------------------------------------------------------
// Plugin
// ---------------------------------------------------------------------------

pub struct FollowPlugin;

impl Plugin for FollowPlugin {
    fn build(&self, app: &mut App) {
        app.insert_resource(FollowTarget::default())
            .add_systems(Update, follow_body_system);
    }
}

// ---------------------------------------------------------------------------
// System
// ---------------------------------------------------------------------------

fn follow_body_system(
    follow: Res<FollowTarget>,
    mut orbit: ResMut<OrbitCamera>,
    query: Query<(&CelestialBody, &Transform)>,
) {
    if follow.entity.is_none() {
        return;
    };

    if let Some((_, transform)) = query.iter().find(|(b, _)| b.name == follow.name) {
        let target_pos = transform.translation;
        // Smooth interpolation (lerp towards target)
        let t = 0.1;
        orbit.focus = orbit.focus.lerp(target_pos, t);
    }
}

/// Compute a camera distance that fits all bodies on screen.
pub fn compute_auto_fit_distance(body_query: &Query<&Transform, With<CelestialBody>>) -> f32 {
    let mut max_dist: f32 = 1.0;
    for transform in body_query.iter() {
        let d = transform.translation.length();
        if d > max_dist {
            max_dist = d;
        }
    }
    // Add 20% padding and account for perspective
    max_dist * 1.5
}
