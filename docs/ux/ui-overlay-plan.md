# UI Overlay Plan

Mapping every Go/Fyne UI panel to a `bevy_egui` implementation.

---

## Panel Layout

```
+-------------------+-----------------------------------+-------------------+
|                   |                                   |                   |
|   Left Sidebar    |                                   |  Right Sidebar    |
|   (280px)         |         3D Viewport               |  (350px)          |
|                   |                                   |                   |
|  [Simulation]     |                                   |  Physics Panel    |
|  [Launch Plan]    |                                   |  (equations,      |
|  [Bodies]         |                                   |   Earth stats)    |
|                   |                                   |                   |
|                   |                                   |                   |
+-------------------+-----------------------------------+-------------------+
|                        Status Bar (28px)                                  |
|  FPS: 60  |  Time: 1.23 yr  |  Speed: 1.0x  |  Zoom: 1.00x  |  Info    |
+--------------------------------------------------------------------------+
```

**egui Panel Types:**

| Region | egui Container | Behavior |
|--------|---------------|----------|
| Left sidebar | `egui::SidePanel::left("left_panel")` | Fixed 280px width, scrollable content, tabbed |
| Right sidebar | `egui::SidePanel::right("right_panel")` | Fixed 350px width, scrollable |
| Bottom bar | `egui::TopBottomPanel::bottom("status_bar")` | Fixed 28px height |
| About dialog | `egui::Window::new("About")` | Modal, centered, closable |
| Settings dialog | `egui::Window::new("Settings")` | Modal, centered, closable |

---

## Per-Panel Mapping

### Master Mapping Table

| Go Panel | File | Key Widgets | egui Equivalent |
|----------|------|-------------|-----------------|
| Controls panel | `app.go:215-507` | Button, Slider, Check, Select | `SidePanel::left`, tab "Simulation" |
| Physics panel | `app.go:82-213` | Label (multi-line, live-updating) | `SidePanel::right` |
| Bodies panel | `bodies_panel.go` | VBox of cards, Label, Button, Check | `SidePanel::left`, tab "Bodies" |
| Launch panel | `launch_panel.go` | Select, Button, Label, Slider | `SidePanel::left`, tab "Launch Planner" |
| Mission playback | `mission_playback.go` | Button, Slider, Label | Embedded in Launch panel |
| Status bar | `statusbar.go` | Label (monospace) | `TopBottomPanel::bottom` |
| Diagnostics | `diagnostics.go` | Label (text) | Embedded in About dialog |
| About dialog | `about.go` | Label, Hyperlink, Image | `egui::Window` |
| Settings dialog | `settings.go` | Select, Check, Form | `egui::Window` |

---

## 1. Controls Panel (Left Sidebar -- "Simulation" Tab)

**Source:** `app.go:createControls()` (lines 215--507)

**egui system:** `ui_controls_panel`

```rust
fn ui_controls_panel(
    mut egui_ctx: ResMut<EguiContext>,
    mut config: ResMut<SimulationConfig>,
    mut camera: ResMut<OrbitCamera>,
    mut sim_commands: EventWriter<SimCommand>,
    mut auto_fit: EventWriter<AutoFitEvent>,
    mut panel_state: ResMut<UIPanelState>,
) {
    egui::SidePanel::left("left_panel")
        .default_width(280.0)
        .resizable(false)
        .show(egui_ctx.ctx_mut(), |ui| {
            // Tab bar
            ui.horizontal(|ui| {
                ui.selectable_value(&mut panel_state.active_left_tab, LeftTab::Simulation, "Simulation");
                ui.selectable_value(&mut panel_state.active_left_tab, LeftTab::LaunchPlanner, "Launch");
                ui.selectable_value(&mut panel_state.active_left_tab, LeftTab::Bodies, "Bodies");
            });
            ui.separator();

            egui::ScrollArea::vertical().show(ui, |ui| {
                match panel_state.active_left_tab {
                    LeftTab::Simulation => simulation_tab(ui, &mut config, &mut camera, &mut sim_commands, &mut auto_fit),
                    LeftTab::LaunchPlanner => launch_tab(ui, /* ... */),
                    LeftTab::Bodies => bodies_tab(ui, /* ... */),
                }
            });
        });
}
```

**Simulation tab widget breakdown:**

| Go Widget | Go Code | egui Widget | Bevy Resource/Event |
|-----------|---------|-------------|---------------------|
| Play/Pause button | `widget.NewButton("Play", ...)` | `ui.button(if playing { "Pause" } else { "Play" })` | `SimulationConfig.is_playing` |
| Speed label | `widget.NewLabel("Speed: ...")` | `ui.label(format!(...))` | `SimulationConfig.time_speed` |
| Speed slider | `widget.NewSlider(-10, 10)` | `ui.add(egui::Slider::new(&mut speed_exp, -10.0..=10.0).text("Speed"))` | `time_speed = 2.0_f64.powf(speed_exp)` |
| Rewind button | `widget.NewButton("Rewind", ...)` | `ui.button("Rewind")` | Sets `time_speed` negative |
| Fast Forward button | `widget.NewButton("Fast Forward", ...)` | `ui.button("Fast Forward")` | Sets `time_speed` positive |
| Trails checkbox | `widget.NewCheck("Show Orbital Trails", ...)` | `ui.checkbox(&mut config.show_trails, "Show Orbital Trails")` | `SimulationConfig.show_trails` |
| Spacetime checkbox | `widget.NewCheck("Show Spacetime Fabric", ...)` | `ui.checkbox(&mut config.show_spacetime, "Show Spacetime Fabric")` | `SimulationConfig.show_spacetime` |
| Planet gravity checkbox | `widget.NewCheck("Planet-Planet Gravity", ...)` | `ui.checkbox(&mut config.planet_gravity, "Planet-Planet Gravity (N-Body)")` | `SimulationConfig.planet_gravity` |
| Relativity checkbox | `widget.NewCheck("General Relativity", ...)` | `ui.checkbox(&mut config.relativity, "General Relativity")` | `SimulationConfig.relativity` |
| Moons checkbox | `widget.NewCheck("Show Moons", ...)` | `ui.checkbox(&mut config.show_moons, "Show Moons")` | `SimCommand::SetShowMoons` |
| Comets checkbox | `widget.NewCheck("Show Comets", ...)` | `ui.checkbox(&mut config.show_comets, "Show Comets")` | `SimCommand::SetShowComets` |
| Asteroids checkbox | `widget.NewCheck("Show Asteroids", ...)` | `ui.checkbox(&mut config.show_asteroids, "Show Asteroids")` | `SimCommand::SetShowAsteroids` |
| Belt checkbox | `widget.NewCheck("Show Asteroid Belt", ...)` | `ui.checkbox(&mut config.show_belt, "Show Asteroid Belt")` | `BeltConfig.visible` |
| Integrator select | `widget.NewSelect(["Verlet", "RK4"], ...)` | `egui::ComboBox::from_label("Integrator").selected_text(...)` | `SimulationConfig.integrator` |
| Sun mass label | `widget.NewLabel("Sun Mass: ...")` | `ui.label(format!(...))` | `SimulationConfig.sun_mass_multiplier` |
| Sun mass slider | `widget.NewSlider(0.1, 5.0)` | `ui.add(egui::Slider::new(&mut config.sun_mass_multiplier, 0.1..=5.0).text("Sun Mass"))` | `SimCommand::SetSunMass` |
| Zoom label | `widget.NewLabel("Zoom: ...")` | `ui.label(format!(...))` | `OrbitCamera.zoom_level` |
| Zoom slider | `widget.NewSlider(-2, 23)` | `ui.add(egui::Slider::new(&mut zoom_exp, -2.0..=23.0).text("Zoom"))` | `OrbitCamera.target_distance` |
| Auto-Fit button | `widget.NewButton("Auto-Fit All Planets", ...)` | `ui.button("Auto-Fit All Planets")` | `EventWriter<AutoFitEvent>` |
| Follow select | `widget.NewSelect(followOptions, ...)` | `egui::ComboBox::from_label("Follow").selected_text(...)` | `OrbitCamera.follow_target` |
| Reset button | `widget.NewButton("Reset Simulation", ...)` | `ui.button("Reset Simulation")` | `SimCommand::Reset` |

**Note:** The current Go UI has 3D rotation sliders (Pitch/Yaw/Roll), Enable 3D checkbox, and Reset 3D View button. These are **removed** in the Bevy version because the viewport is always 3D and rotation is controlled via mouse drag. The GPU Rendering and Ray Tracing checkboxes are also removed since Bevy handles rendering natively.

**Simulation tab layout code:**

```rust
fn simulation_tab(
    ui: &mut egui::Ui,
    config: &mut SimulationConfig,
    camera: &mut OrbitCamera,
    sim_commands: &mut EventWriter<SimCommand>,
    auto_fit: &mut EventWriter<AutoFitEvent>,
) {
    // --- Time Control ---
    ui.heading("Time Control");
    if ui.button(if config.is_playing { "⏸ Pause" } else { "▶ Play" }).clicked() {
        config.is_playing = !config.is_playing;
        sim_commands.send(SimCommand::SetPlaying(config.is_playing));
    }

    let mut speed_exp = config.time_speed.abs().log2();
    let speed_negative = config.time_speed < 0.0;
    ui.label(format!("Speed: {:.1}x", config.time_speed));
    if ui.add(egui::Slider::new(&mut speed_exp, -10.0..=10.0).step_by(0.1)).changed() {
        let sign = if speed_negative { -1.0 } else { 1.0 };
        config.time_speed = sign * 2.0_f64.powf(speed_exp);
        sim_commands.send(SimCommand::SetTimeSpeed(config.time_speed));
    }

    ui.horizontal(|ui| {
        if ui.button("⏪ Rewind").clicked() {
            config.time_speed = -config.time_speed.abs();
            config.is_playing = true;
            sim_commands.send(SimCommand::SetTimeSpeed(config.time_speed));
            sim_commands.send(SimCommand::SetPlaying(true));
        }
        if ui.button("⏩ Fast Forward").clicked() {
            config.time_speed = config.time_speed.abs();
            config.is_playing = true;
            sim_commands.send(SimCommand::SetTimeSpeed(config.time_speed));
            sim_commands.send(SimCommand::SetPlaying(true));
        }
    });

    ui.separator();

    // --- Camera Controls ---
    ui.heading("Camera");
    let mut zoom_exp = camera.zoom_level.log2();
    ui.label(format!("Zoom: {:.2}x", camera.zoom_level));
    if ui.add(egui::Slider::new(&mut zoom_exp, -2.0..=23.0).step_by(0.1)).changed() {
        camera.target_distance = /* compute from zoom_exp */;
    }
    if ui.button("Auto-Fit All Planets").clicked() {
        auto_fit.send(AutoFitEvent);
    }

    // Follow body combo box (populated from query)
    // ...

    ui.separator();

    // --- Display Options ---
    ui.heading("Display");
    if ui.checkbox(&mut config.show_trails, "Show Orbital Trails").changed() {
        sim_commands.send(SimCommand::SetShowTrails(config.show_trails));
    }
    if ui.checkbox(&mut config.show_spacetime, "Show Spacetime Fabric").changed() {
        sim_commands.send(SimCommand::SetShowSpacetime(config.show_spacetime));
    }
    if ui.checkbox(&mut config.show_labels, "Show Labels").changed() {
        sim_commands.send(SimCommand::SetShowLabels(config.show_labels));
    }

    ui.separator();

    // --- Celestial Bodies ---
    ui.heading("Celestial Bodies");
    if ui.checkbox(&mut config.show_moons, "Show Moons").changed() {
        sim_commands.send(SimCommand::SetShowMoons(config.show_moons));
    }
    if ui.checkbox(&mut config.show_comets, "Show Comets").changed() {
        sim_commands.send(SimCommand::SetShowComets(config.show_comets));
    }
    if ui.checkbox(&mut config.show_asteroids, "Show Asteroids").changed() {
        sim_commands.send(SimCommand::SetShowAsteroids(config.show_asteroids));
    }
    if ui.checkbox(&mut config.show_belt, "Show Asteroid Belt").changed() {
        sim_commands.send(SimCommand::SetShowBelt(config.show_belt));
    }

    ui.separator();

    // --- Physics Options ---
    ui.heading("Physics");
    if ui.checkbox(&mut config.planet_gravity, "Planet-Planet Gravity (N-Body)").changed() {
        sim_commands.send(SimCommand::SetPlanetGravity(config.planet_gravity));
    }
    if ui.checkbox(&mut config.relativity, "General Relativity").changed() {
        sim_commands.send(SimCommand::SetRelativity(config.relativity));
    }

    egui::ComboBox::from_label("Integrator")
        .selected_text(match config.integrator {
            IntegratorType::Verlet => "Verlet (symplectic)",
            IntegratorType::RK4 => "RK4 (classic)",
        })
        .show_ui(ui, |ui| {
            ui.selectable_value(&mut config.integrator, IntegratorType::Verlet, "Verlet (symplectic)");
            ui.selectable_value(&mut config.integrator, IntegratorType::RK4, "RK4 (classic)");
        });

    ui.separator();

    // --- Sun Properties ---
    ui.heading("Sun Properties");
    ui.label(format!("Sun Mass: {:.2}x", config.sun_mass_multiplier));
    if ui.add(egui::Slider::new(&mut config.sun_mass_multiplier, 0.1..=5.0).step_by(0.1)).changed() {
        sim_commands.send(SimCommand::SetSunMass(config.sun_mass_multiplier));
    }

    ui.separator();

    if ui.button("Reset Simulation").clicked() {
        sim_commands.send(SimCommand::Reset);
    }
}
```

---

## 2. Physics Panel (Right Sidebar)

**Source:** `app.go:createPhysicsPanel()` (lines 82--213)

**egui system:** `ui_physics_panel`

The current Go panel displays a large multi-line label with physics equations and live Earth orbital parameters, updated at 10 Hz.

```rust
fn ui_physics_panel(
    mut egui_ctx: ResMut<EguiContext>,
    config: Res<SimulationConfig>,
    clock: Res<SimulationClock>,
    bodies: Query<(&CelestialBody, &Transform, &Velocity)>,
    camera: Res<OrbitCamera>,
) {
    egui::SidePanel::right("right_panel")
        .default_width(350.0)
        .resizable(true)
        .show(egui_ctx.ctx_mut(), |ui| {
            ui.heading("Physics Equations");
            egui::ScrollArea::vertical().show(ui, |ui| {
                ui.style_mut().override_font_id = Some(egui::FontId::monospace(11.0));

                // Newton's Law of Universal Gravitation
                ui.label("Newton's Law of Universal Gravitation (3D):");
                ui.label("  F = -GMm/r^2 * r_hat");
                ui.label(format!("  G = {:.3e} m^3 kg^-1 s^-2", 6.674e-11));
                ui.label(format!("  M = {:.3e} kg (Sun)", config.sun_mass_multiplier * 1.989e30));
                ui.separator();

                // N-Body
                ui.label("N-Body Problem:");
                ui.label("  F_i = sum_j(!=i) (-G*M_j*m_i / r_ij^2) * r_hat_ij");
                ui.label(format!("  Planet-Planet: {}",
                    if config.planet_gravity { "ENABLED" } else { "DISABLED" }
                ));
                ui.separator();

                // GR
                ui.label("General Relativity Correction:");
                ui.label("  a_GR = (GM/(c^2*r^3))[(4GM/r-v^2)r + 4(r.v)v]");
                ui.label(format!("  GR: {}",
                    if config.relativity { "ENABLED (~43\"/century)" } else { "DISABLED" }
                ));
                ui.separator();

                // Live Earth values (find Earth entity)
                if let Some((_, transform, velocity)) = bodies.iter()
                    .find(|(b, _, _)| b.name == "Earth")
                {
                    let pos = transform.translation;
                    let r = (pos.x * pos.x + pos.y * pos.y + pos.z * pos.z).sqrt();
                    let v = velocity.0.length();
                    let r_meters = r as f64 / RENDER_SCALE * AU;
                    let v_ms = v;

                    ui.heading("Current Earth Values");
                    ui.label(format!("  Distance: {:.3e} m ({:.3f} AU)", r_meters, r_meters / AU));
                    ui.label(format!("  Velocity: {:.3e} m/s ({:.1} km/s)", v_ms, v_ms / 1000.0));
                    ui.label(format!("  Period: {:.1} days",
                        2.0 * std::f64::consts::PI * r_meters / v_ms / 86400.0));
                }

                ui.separator();
                ui.heading("Simulation");
                ui.label(format!("  Time: {:.1} days ({:.2} years)",
                    clock.elapsed_days, clock.elapsed_years));
                ui.label(format!("  Speed: {:.1}x", config.time_speed));
                ui.label(format!("  Zoom: {:.2}x", camera.zoom_level));
            });
        });
}
```

---

## 3. Bodies Panel (Left Sidebar -- "Bodies" Tab)

**Source:** `bodies_panel.go` (lines 16--135)

**egui system:** `ui_bodies_panel` (called within the left sidebar when `LeftTab::Bodies` is active)

```rust
fn bodies_tab(
    ui: &mut egui::Ui,
    bodies: &Query<(Entity, &CelestialBody, &Transform, &Velocity, &mut Orbit)>,
    sun_q: &Query<&Transform, /* sun filter */>,
    camera: &mut OrbitCamera,
) {
    let sun_pos = sun_q.single().translation;

    // Group by BodyType
    let groups = [
        ("Star", BodyType::Star),
        ("Planets", BodyType::Planet),
        ("Dwarf Planets", BodyType::DwarfPlanet),
        ("Moons", BodyType::Moon),
        ("Comets", BodyType::Comet),
        ("Asteroids", BodyType::Asteroid),
    ];

    for (group_name, body_type) in &groups {
        let group_bodies: Vec<_> = bodies.iter()
            .filter(|(_, b, _, _, _)| b.body_type == *body_type)
            .collect();

        if group_bodies.is_empty() { continue; }

        ui.collapsing(*group_name, |ui| {
            for (entity, body, transform, velocity, orbit) in &group_bodies {
                ui.group(|ui| {
                    ui.strong(&body.name);

                    let dist = transform.translation.distance(sun_pos);
                    let dist_au = dist as f64 / RENDER_SCALE * AU / AU;
                    let vel = velocity.0.length();

                    ui.label(format!("Mass: {:.3e} kg", body.mass));
                    ui.label(format!("Distance: {:.4} AU", dist_au));
                    ui.label(format!("Velocity: {:.2} km/s", vel / 1000.0));

                    let period = 2.0 * std::f64::consts::PI * (dist as f64) / vel / 86400.0;
                    ui.label(format!("Period: {:.1} days", period));

                    ui.horizontal(|ui| {
                        if ui.button("Follow").clicked() {
                            camera.follow_target = Some(*entity);
                        }
                        let mut show_trail = orbit.show_trail;
                        if ui.checkbox(&mut show_trail, "Trail").changed() {
                            // orbit.show_trail = show_trail; (need mutable access)
                        }
                    });
                });
                ui.add_space(4.0);
            }
        });
    }
}
```

**Current Go behavior:** Live-updates at 500ms intervals via a goroutine. In Bevy, the system runs every frame (egui is immediate-mode), so values are always up-to-date. No goroutine needed.

---

## 4. Launch Panel (Left Sidebar -- "Launch Planner" Tab)

**Source:** `launch_panel.go` (lines 58--235)

**egui system:** `ui_launch_panel` (called within the left sidebar when `LeftTab::LaunchPlanner` is active)

```rust
fn launch_tab(
    ui: &mut egui::Ui,
    launch_state: &mut LaunchState,
    launch_commands: &mut EventWriter<LaunchCommand>,
    launch_results: &Query<&LaunchComputed>,
) {
    ui.heading("Launch Planner");
    ui.separator();
    ui.label("Launch Site: Kennedy Space Center");
    ui.separator();

    // Destination selector
    let destinations = ["LEO (400 km)", "ISS (408 km)", "GTO", "Moon", "Mars"];
    ui.label("Destination:");
    egui::ComboBox::from_id_salt("dest_select")
        .selected_text(destinations[launch_state.destination_index])
        .show_ui(ui, |ui| {
            for (i, name) in destinations.iter().enumerate() {
                ui.selectable_value(&mut launch_state.destination_index, i, *name);
            }
        });

    // Vehicle selector
    let vehicles = ["Generic Rocket", "Falcon-class (9.4 km/s)", "Saturn V-class (13.1 km/s)"];
    ui.label("Vehicle:");
    egui::ComboBox::from_id_salt("vehicle_select")
        .selected_text(vehicles[launch_state.vehicle_index])
        .show_ui(ui, |ui| {
            for (i, name) in vehicles.iter().enumerate() {
                ui.selectable_value(&mut launch_state.vehicle_index, i, *name);
            }
        });

    ui.separator();

    ui.horizontal(|ui| {
        if ui.button("Simulate Launch").clicked() {
            launch_commands.send(LaunchCommand::Simulate);
        }
        if ui.button("Clear").clicked() {
            launch_commands.send(LaunchCommand::Clear);
        }
    });

    ui.separator();

    // Results display
    if let Some(plan) = &launch_state.plan {
        egui::ScrollArea::vertical().max_height(200.0).show(ui, |ui| {
            ui.label(&plan.summary);  // Pre-formatted summary text
        });
    } else {
        ui.label("Select a destination and vehicle, then click Simulate.");
    }

    // Mission playback controls (shown when trajectory exists)
    if launch_state.trajectory.is_some() {
        ui.separator();
        ui.heading("Mission Playback");

        ui.horizontal(|ui| {
            let label = if launch_state.playback_paused { "Play" } else { "Pause" };
            if ui.button(label).clicked() {
                launch_commands.send(LaunchCommand::PlaybackToggle);
            }
        });

        // Speed slider (1x to 64x)
        let mut speed_exp = launch_state.playback_speed.log2() as f32;
        ui.label(format!("Playback: {:.1}x", launch_state.playback_speed));
        if ui.add(egui::Slider::new(&mut speed_exp, 0.0..=6.0).step_by(0.5)).changed() {
            let speed = 2.0_f64.powf(speed_exp as f64);
            launch_commands.send(LaunchCommand::PlaybackSetSpeed(speed));
        }

        // Timeline scrub slider
        ui.label("Timeline:");
        let mut pct = if launch_state.playback_time > 0.0 {
            (launch_state.playback_time / launch_state.total_time() * 100.0) as f32
        } else { 0.0 };
        if ui.add(egui::Slider::new(&mut pct, 0.0..=100.0).step_by(0.5).show_value(false)).changed() {
            let t = pct as f64 / 100.0 * launch_state.total_time();
            launch_commands.send(LaunchCommand::PlaybackSeek(t));
        }

        // Telemetry readout
        if let Some(pos) = &launch_state.vehicle_world_pos {
            let speed_km = /* interpolated velocity */ 0.0;
            let dist_au = pos.length() / AU;
            let elapsed_days = launch_state.playback_time / 86400.0;
            let progress = launch_state.playback_time / launch_state.total_time() * 100.0;
            ui.label(format!("Elapsed: {:.1} days", elapsed_days));
            ui.label(format!("Distance: {:.4} AU", dist_au));
            ui.label(format!("Progress: {:.1}%", progress));
        }
    }
}
```

---

## 5. Status Bar (Bottom Panel)

**Source:** `statusbar.go` (lines 16--117)

**egui system:** `ui_status_bar`

```rust
fn ui_status_bar(
    mut egui_ctx: ResMut<EguiContext>,
    clock: Res<SimulationClock>,
    config: Res<SimulationConfig>,
    camera: Res<OrbitCamera>,
    diagnostics: Res<DiagnosticsStore>,
    runtime_info: Res<RuntimeInfoResource>,
) {
    egui::TopBottomPanel::bottom("status_bar")
        .exact_height(28.0)
        .show(egui_ctx.ctx_mut(), |ui| {
            ui.horizontal_centered(|ui| {
                ui.style_mut().override_font_id = Some(egui::FontId::monospace(12.0));

                // FPS (from Bevy's built-in diagnostics)
                if let Some(fps) = diagnostics.get(&FrameTimeDiagnosticsPlugin::FPS) {
                    if let Some(value) = fps.smoothed() {
                        ui.label(format!("FPS: {:.0}", value));
                    }
                }
                ui.separator();

                // Simulation time
                if clock.elapsed_years >= 1.0 {
                    ui.label(format!("Time: {:.2} yr", clock.elapsed_years));
                } else {
                    ui.label(format!("Time: {:.1} d", clock.elapsed_days));
                }
                ui.separator();

                // Speed
                ui.label(format!("Speed: {:.1}x", config.time_speed));
                ui.separator();

                // Zoom
                ui.label(format!("Zoom: {:.2}x", camera.zoom_level));
                ui.separator();

                // Runtime info
                ui.label(&runtime_info.summary);
            });
        });
}
```

**Note:** The current Go implementation throttles updates to every 4th frame and skips when values are unchanged. In Bevy with egui, the status bar is redrawn every frame (egui is immediate-mode and very fast). The FPS counter uses Bevy's built-in `FrameTimeDiagnosticsPlugin` instead of a manual counter.

**Bevy diagnostics setup:**

```rust
// In main.rs, add the diagnostics plugin:
app.add_plugins(bevy::diagnostic::FrameTimeDiagnosticsPlugin);
```

---

## 6. Diagnostics Panel

**Source:** `diagnostics.go` (lines 8--66)

The current Go implementation detects OS, architecture, CPU count, Go version, and GPU info at startup.

In Bevy, this becomes a `RuntimeInfoResource` populated at startup:

```rust
#[derive(Resource)]
pub struct RuntimeInfoResource {
    pub os: String,
    pub arch: String,
    pub cpu_count: usize,
    pub gpu_info: String,
    pub bevy_version: String,
    pub summary: String,  // One-line summary for status bar
}

fn detect_runtime(mut commands: Commands) {
    let info = RuntimeInfoResource {
        os: std::env::consts::OS.to_string(),
        arch: std::env::consts::ARCH.to_string(),
        cpu_count: num_cpus::get(),
        gpu_info: String::new(),  // Populated after renderer init via RenderAdapterInfo
        bevy_version: env!("CARGO_PKG_VERSION").to_string(),
        summary: format!("{}/{} | CPUs: {}", std::env::consts::OS, std::env::consts::ARCH, num_cpus::get()),
    };
    commands.insert_resource(info);
}
```

GPU info is available via Bevy's `RenderAdapterInfo` resource after the renderer initializes. Update the resource in a startup system that runs after `RenderPlugin`.

Diagnostics data is displayed inside the About dialog rather than as a separate panel.

---

## 7. Mission Playback

**Source:** `mission_playback.go` (lines 10--167)

Mission playback is embedded in the Launch Panel (see section 4 above). The `LaunchState` resource holds all playback state. The `update_mission_playback` system in `LaunchPlugin` handles time advancement, interpolation, and telemetry computation.

No separate egui system is needed. The launch tab function reads `LaunchState` fields directly.

---

## 8. About Dialog

**Source:** `about.go` (lines 14--108)

**egui system:** `ui_about_dialog`

```rust
fn ui_about_dialog(
    mut egui_ctx: ResMut<EguiContext>,
    mut panel_state: ResMut<UIPanelState>,
    runtime_info: Res<RuntimeInfoResource>,
) {
    if !panel_state.about_open { return; }

    egui::Window::new("About Solar System Simulator")
        .collapsible(false)
        .resizable(false)
        .default_width(400.0)
        .anchor(egui::Align2::CENTER_CENTER, [0.0, 0.0])
        .show(egui_ctx.ctx_mut(), |ui| {
            ui.vertical_centered(|ui| {
                // Logo (if loaded as egui texture)
                // ui.image(logo_texture, [128.0, 128.0]);

                ui.heading("Solar System Simulator");
                ui.label("Version 0.1.2");
                ui.separator();
                ui.label("Author: Joshua Baney");
                ui.hyperlink_to("GitHub Repository", "https://github.com/JoshBaneyCS/Solar-System-Sim");
                ui.hyperlink_to("Sponsor / Donate", "https://www.paypal.com/donate/?business=HWM2DENMWG4K2");
                ui.separator();

                // System info
                ui.heading("System Information");
                ui.label(format!("Platform: {}/{}", runtime_info.os, runtime_info.arch));
                ui.label(format!("CPU Cores: {}", runtime_info.cpu_count));
                ui.label(format!("Bevy Version: {}", runtime_info.bevy_version));
                if !runtime_info.gpu_info.is_empty() {
                    ui.label(format!("GPU: {}", runtime_info.gpu_info));
                }
                ui.separator();

                // Credits
                ui.heading("Credits");
                ui.label("Bevy Engine - Rust game engine");
                ui.label("egui - Immediate-mode GUI");
                ui.label("NASA/JPL - Planetary ephemeris data");
                ui.label("Rust - Programming language");
                ui.label("");
                ui.label("License: MIT");
                ui.separator();

                if ui.button("Close").clicked() {
                    panel_state.about_open = false;
                }
            });
        });
}
```

---

## 9. Settings Dialog

**Source:** `settings.go` (lines 82--153)

**egui system:** `ui_settings_dialog`

```rust
fn ui_settings_dialog(
    mut egui_ctx: ResMut<EguiContext>,
    mut panel_state: ResMut<UIPanelState>,
    mut config: ResMut<SimulationConfig>,
    mut sim_commands: EventWriter<SimCommand>,
    mut app_settings: ResMut<AppSettings>,
) {
    if !panel_state.settings_open { return; }

    egui::Window::new("Settings")
        .collapsible(false)
        .resizable(false)
        .default_width(400.0)
        .anchor(egui::Align2::CENTER_CENTER, [0.0, 0.0])
        .show(egui_ctx.ctx_mut(), |ui| {
            egui::Grid::new("settings_grid")
                .num_columns(2)
                .spacing([20.0, 8.0])
                .show(ui, |ui| {
                    // Integrator
                    ui.label("Integrator:");
                    egui::ComboBox::from_id_salt("settings_integrator")
                        .selected_text(match config.integrator {
                            IntegratorType::Verlet => "Verlet (symplectic)",
                            IntegratorType::RK4 => "RK4 (classic)",
                        })
                        .show_ui(ui, |ui| {
                            ui.selectable_value(&mut config.integrator, IntegratorType::Verlet, "Verlet");
                            ui.selectable_value(&mut config.integrator, IntegratorType::RK4, "RK4");
                        });
                    ui.end_row();

                    // Display toggles
                    ui.label("");
                    ui.checkbox(&mut config.show_trails, "Show Trails");
                    ui.end_row();

                    ui.label("");
                    ui.checkbox(&mut config.show_spacetime, "Show Spacetime Fabric");
                    ui.end_row();

                    ui.label("");
                    ui.checkbox(&mut config.show_labels, "Show Labels");
                    ui.end_row();

                    // Physics toggles
                    ui.label("");
                    ui.checkbox(&mut config.planet_gravity, "Planet-Planet Gravity");
                    ui.end_row();

                    ui.label("");
                    ui.checkbox(&mut config.relativity, "General Relativity");
                    ui.end_row();
                });

            ui.separator();
            ui.horizontal(|ui| {
                if ui.button("Apply & Save").clicked() {
                    // Persist settings to disk
                    app_settings.save_from_config(&config);
                    panel_state.settings_open = false;
                }
                if ui.button("Cancel").clicked() {
                    panel_state.settings_open = false;
                }
            });
        });
}
```

**Note:** The Go settings dialog includes GPU Mode, Ray Tracing, and Quality Preset selectors. These are removed in Bevy since rendering is handled by Bevy's built-in PBR pipeline. If quality presets are needed later (MSAA level, bloom intensity, shadow quality), they can be added as Bevy render settings.

---

## Theme

### Current SpaceTheme Colors (`theme.go`)

| Color Name | RGBA | Hex | Usage |
|------------|------|-----|-------|
| Background | `(10, 10, 26, 255)` | `#0A0A1A` | Main window background |
| Button | `(26, 26, 46, 255)` | `#1A1A2E` | Button fill |
| Disabled Button | `(20, 20, 35, 255)` | `#141423` | Disabled button |
| Primary (accent) | `(0, 180, 216, 255)` | `#00B4D8` | Cyan accent, selection highlight |
| Focus | `(0, 180, 216, 128)` | `#00B4D880` | Focus ring |
| Hover | `(30, 30, 55, 255)` | `#1E1E37` | Hover state |
| Input Background | `(20, 20, 38, 255)` | `#141426` | Text input, slider background |
| Placeholder | `(120, 120, 140, 255)` | `#78788C` | Placeholder text |
| Scrollbar | `(60, 60, 80, 255)` | `#3C3C50` | Scrollbar thumb |
| Shadow | `(0, 0, 0, 100)` | `#00000064` | Drop shadows |
| Foreground (text) | `(220, 220, 230, 255)` | `#DCDCE6` | Primary text |
| Separator | `(40, 40, 60, 255)` | `#28283C` | Divider lines |
| Status bar bg | `(15, 15, 30, 255)` | `#0F0F1E` | Bottom status bar |

### Sizing

| Property | Value |
|----------|-------|
| Padding | 6px |
| Inner Padding | 4px |
| Text Size | 13px |
| Separator Thickness | 1px |

### egui Theme Application

Apply at `EguiPlugin` initialization in the UIPlugin's startup system:

```rust
fn configure_egui_theme(mut egui_ctx: ResMut<EguiContext>) {
    let mut visuals = egui::Visuals::dark();

    // Window/panel backgrounds
    visuals.window_fill = egui::Color32::from_rgb(10, 10, 26);
    visuals.panel_fill = egui::Color32::from_rgb(10, 10, 26);
    visuals.extreme_bg_color = egui::Color32::from_rgb(20, 20, 38);  // Input fields
    visuals.faint_bg_color = egui::Color32::from_rgb(26, 26, 46);     // Subtle backgrounds

    // Widget visuals
    visuals.widgets.noninteractive.bg_fill = egui::Color32::from_rgb(20, 20, 35);
    visuals.widgets.noninteractive.fg_stroke = egui::Stroke::new(1.0, egui::Color32::from_rgb(220, 220, 230));

    visuals.widgets.inactive.bg_fill = egui::Color32::from_rgb(26, 26, 46);
    visuals.widgets.inactive.fg_stroke = egui::Stroke::new(1.0, egui::Color32::from_rgb(220, 220, 230));

    visuals.widgets.hovered.bg_fill = egui::Color32::from_rgb(30, 30, 55);
    visuals.widgets.hovered.fg_stroke = egui::Stroke::new(1.0, egui::Color32::from_rgb(0, 180, 216));

    visuals.widgets.active.bg_fill = egui::Color32::from_rgb(0, 180, 216);
    visuals.widgets.active.fg_stroke = egui::Stroke::new(1.0, egui::Color32::from_rgb(10, 10, 26));

    // Selection
    visuals.selection.bg_fill = egui::Color32::from_rgba_premultiplied(0, 180, 216, 128);
    visuals.selection.stroke = egui::Stroke::new(1.0, egui::Color32::from_rgb(0, 180, 216));

    // Separators
    visuals.widgets.noninteractive.bg_stroke = egui::Stroke::new(1.0, egui::Color32::from_rgb(40, 40, 60));

    // Scrollbar
    visuals.widgets.inactive.bg_fill = egui::Color32::from_rgb(60, 60, 80);

    // Window shadow
    visuals.window_shadow = egui::Shadow {
        offset: egui::vec2(0.0, 2.0),
        blur: 8.0,
        spread: 0.0,
        color: egui::Color32::from_rgba_premultiplied(0, 0, 0, 100),
    };

    // Window rounding
    visuals.window_rounding = egui::Rounding::same(6.0);

    let ctx = egui_ctx.ctx_mut();
    ctx.set_visuals(visuals);

    // Font sizing
    let mut style = (*ctx.style()).clone();
    style.spacing.item_spacing = egui::vec2(6.0, 4.0);
    style.spacing.button_padding = egui::vec2(6.0, 4.0);
    style.text_styles.insert(
        egui::TextStyle::Body,
        egui::FontId::new(13.0, egui::FontFamily::Proportional),
    );
    style.text_styles.insert(
        egui::TextStyle::Button,
        egui::FontId::new(13.0, egui::FontFamily::Proportional),
    );
    style.text_styles.insert(
        egui::TextStyle::Monospace,
        egui::FontId::new(12.0, egui::FontFamily::Monospace),
    );
    ctx.set_style(style);
}
```

---

## Settings Persistence

### Current (Go/Fyne)

- Uses Fyne's `Preferences` API (key-value store backed by JSON).
- `LoadSettings()` reads with `prefs.StringWithFallback()` / `prefs.BoolWithFallback()`.
- `Save()` writes via `prefs.SetString()` / `prefs.SetBool()`.
- File location: OS-specific Fyne preferences directory.

### Target (Rust/Bevy)

Use `serde` with RON (Rusty Object Notation) file format. RON is human-readable, Rust-native, and commonly used in the Bevy ecosystem.

```toml
[dependencies]
serde = { version = "1", features = ["derive"] }
ron = "0.8"
directories = "5"  # For OS-specific config paths
```

```rust
use serde::{Deserialize, Serialize};

#[derive(Resource, Serialize, Deserialize, Clone)]
pub struct AppSettings {
    // Physics
    pub integrator: String,       // "verlet" or "rk4"
    pub planet_gravity: bool,
    pub relativity: bool,
    pub sun_mass_multiplier: f64,

    // Display
    pub show_trails: bool,
    pub show_spacetime: bool,
    pub show_labels: bool,
    pub show_belt: bool,

    // Bodies
    pub show_moons: bool,
    pub show_comets: bool,
    pub show_asteroids: bool,

    // Window
    pub window_width: f32,
    pub window_height: f32,
    pub fullscreen: bool,
}

impl Default for AppSettings {
    fn default() -> Self {
        Self {
            integrator: "verlet".into(),
            planet_gravity: true,
            relativity: true,
            sun_mass_multiplier: 1.0,
            show_trails: true,
            show_spacetime: false,
            show_labels: true,
            show_belt: true,
            show_moons: true,
            show_comets: false,
            show_asteroids: false,
            window_width: 1600.0,
            window_height: 900.0,
            fullscreen: false,
        }
    }
}

impl AppSettings {
    pub fn config_path() -> std::path::PathBuf {
        let dirs = directories::ProjectDirs::from("com", "joshbaney", "solar-sim")
            .expect("Could not determine config directory");
        dirs.config_dir().join("settings.ron")
    }

    pub fn load() -> Self {
        let path = Self::config_path();
        if path.exists() {
            let contents = std::fs::read_to_string(&path).unwrap_or_default();
            ron::from_str(&contents).unwrap_or_default()
        } else {
            Self::default()
        }
    }

    pub fn save(&self) {
        let path = Self::config_path();
        if let Some(parent) = path.parent() {
            let _ = std::fs::create_dir_all(parent);
        }
        let pretty = ron::ser::PrettyConfig::default();
        if let Ok(contents) = ron::ser::to_string_pretty(self, pretty) {
            let _ = std::fs::write(&path, contents);
        }
    }

    pub fn save_from_config(&mut self, config: &SimulationConfig) {
        self.show_trails = config.show_trails;
        self.show_spacetime = config.show_spacetime;
        self.show_labels = config.show_labels;
        self.show_belt = config.show_belt;
        self.planet_gravity = config.planet_gravity;
        self.relativity = config.relativity;
        self.integrator = match config.integrator {
            IntegratorType::Verlet => "verlet".into(),
            IntegratorType::RK4 => "rk4".into(),
        };
        self.sun_mass_multiplier = config.sun_mass_multiplier;
        self.show_moons = config.show_moons;
        self.show_comets = config.show_comets;
        self.show_asteroids = config.show_asteroids;
        self.save();
    }
}
```

**File location:** `~/.config/solar-sim/settings.ron` (Linux), `~/Library/Application Support/com.joshbaney.solar-sim/settings.ron` (macOS), `%APPDATA%/joshbaney/solar-sim/config/settings.ron` (Windows). Handled by the `directories` crate.

**Load at startup:**
```rust
fn load_settings(mut commands: Commands) {
    let settings = AppSettings::load();
    // Apply to SimulationConfig
    // ...
    commands.insert_resource(settings);
}
```

**Save on change:** Save whenever the user clicks "Apply & Save" in the Settings dialog, or on app exit via a Bevy `AppExit` observer:

```rust
fn save_on_exit(
    mut exit_events: EventReader<AppExit>,
    settings: Res<AppSettings>,
) {
    for _ in exit_events.read() {
        settings.save();
    }
}
```

---

## Main Menu

### Current (Go/Fyne)

The Go app has a native OS menu bar with File, View, Simulation, Settings, and About menus (`menu.go`).

### Bevy Approach

egui does not provide a native OS menu bar. Instead, use an egui `TopBottomPanel::top` with a horizontal menu bar:

```rust
fn ui_menu_bar(
    mut egui_ctx: ResMut<EguiContext>,
    mut panel_state: ResMut<UIPanelState>,
    mut config: ResMut<SimulationConfig>,
    mut sim_commands: EventWriter<SimCommand>,
    mut exit: EventWriter<AppExit>,
) {
    egui::TopBottomPanel::top("menu_bar").show(egui_ctx.ctx_mut(), |ui| {
        egui::menu::bar(ui, |ui| {
            ui.menu_button("File", |ui| {
                if ui.button("Export Screenshot...").clicked() {
                    // Trigger screenshot (Bevy screenshot API)
                    ui.close_menu();
                }
                ui.separator();
                if ui.button("Quit").clicked() {
                    exit.send(AppExit::Success);
                }
            });

            ui.menu_button("View", |ui| {
                if ui.checkbox(&mut config.show_trails, "Trails").changed() {
                    sim_commands.send(SimCommand::SetShowTrails(config.show_trails));
                }
                if ui.checkbox(&mut config.show_spacetime, "Spacetime Fabric").changed() {
                    sim_commands.send(SimCommand::SetShowSpacetime(config.show_spacetime));
                }
                if ui.checkbox(&mut config.show_labels, "Labels").changed() {
                    sim_commands.send(SimCommand::SetShowLabels(config.show_labels));
                }
                ui.separator();
                if ui.checkbox(&mut config.show_moons, "Moons").changed() {
                    sim_commands.send(SimCommand::SetShowMoons(config.show_moons));
                }
                if ui.checkbox(&mut config.show_comets, "Comets").changed() {
                    sim_commands.send(SimCommand::SetShowComets(config.show_comets));
                }
                if ui.checkbox(&mut config.show_asteroids, "Asteroids").changed() {
                    sim_commands.send(SimCommand::SetShowAsteroids(config.show_asteroids));
                }
                if ui.checkbox(&mut config.show_belt, "Asteroid Belt").changed() {
                    sim_commands.send(SimCommand::SetShowBelt(config.show_belt));
                }
            });

            ui.menu_button("Simulation", |ui| {
                if ui.button("Play/Pause").clicked() {
                    config.is_playing = !config.is_playing;
                    sim_commands.send(SimCommand::SetPlaying(config.is_playing));
                    ui.close_menu();
                }
                if ui.button("Reset").clicked() {
                    sim_commands.send(SimCommand::Reset);
                    ui.close_menu();
                }
            });

            ui.menu_button("Settings", |ui| {
                if ui.button("Settings...").clicked() {
                    panel_state.settings_open = true;
                    ui.close_menu();
                }
            });

            ui.menu_button("Help", |ui| {
                if ui.button("About...").clicked() {
                    panel_state.about_open = true;
                    ui.close_menu();
                }
            });
        });
    });
}
```

---

## System Registration Summary

```rust
pub struct UIPlugin;

impl Plugin for UIPlugin {
    fn build(&self, app: &mut App) {
        app
            .add_plugins(bevy_egui::EguiPlugin)
            .add_plugins(bevy::diagnostic::FrameTimeDiagnosticsPlugin)
            .insert_resource(UIPanelState::default())
            .insert_resource(AppSettings::load())
            .insert_resource(RuntimeInfoResource::detect())
            .add_systems(Startup, configure_egui_theme)
            .add_systems(Update, (
                ui_menu_bar,
                ui_controls_panel,     // Left sidebar (contains all 3 tabs)
                ui_physics_panel,      // Right sidebar
                ui_status_bar,         // Bottom panel
                ui_settings_dialog,    // Modal window
                ui_about_dialog,       // Modal window
                handle_keyboard_shortcuts,
            ));
    }
}
```

Note: `ui_controls_panel` internally dispatches to `simulation_tab`, `launch_tab`, or `bodies_tab` based on `UIPanelState.active_left_tab`. This keeps the system list clean while supporting the tabbed interface.
