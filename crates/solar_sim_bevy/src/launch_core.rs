//! Launch planning module — ported from internal/launch/*.go

use std::f64::consts::PI;

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

pub const MU_EARTH: f64 = 3.986004418e14;
pub const MU_SUN: f64 = 1.32712440018e20;
pub const MU_MOON: f64 = 4.9048695e12;
pub const R_EARTH: f64 = 6.371e6;
pub const R_MOON: f64 = 1.7371e6;
pub const G0: f64 = 9.80665;
pub const KSC_LATITUDE_RAD: f64 = 28.5724 * PI / 180.0;
pub const EARTH_ROTATIONAL_V: f64 = 407.0;
pub const GRAVITY_DRAG_LOSS: f64 = 1500.0;
pub const GEO_ALTITUDE: f64 = 35786e3;
pub const MOON_DISTANCE: f64 = 384400e3;
pub const MARS_ORBIT_SMA: f64 = 1.524 * 1.496e11;
pub const EARTH_ORBIT_SMA: f64 = 1.496e11;

// ---------------------------------------------------------------------------
// Orbital mechanics
// ---------------------------------------------------------------------------

pub fn circular_velocity(mu: f64, r: f64) -> f64 {
    (mu / r).sqrt()
}

pub fn escape_velocity(mu: f64, r: f64) -> f64 {
    (2.0 * mu / r).sqrt()
}

pub fn hohmann_delta_v(mu: f64, r1: f64, r2: f64) -> (f64, f64) {
    let a = (r1 + r2) / 2.0;
    let v1 = (mu / r1).sqrt();
    let vt1 = (mu * (2.0 / r1 - 1.0 / a)).sqrt();
    let dv1 = (vt1 - v1).abs();
    let v2 = (mu / r2).sqrt();
    let vt2 = (mu * (2.0 / r2 - 1.0 / a)).sqrt();
    let dv2 = (v2 - vt2).abs();
    (dv1, dv2)
}

pub fn hohmann_transfer_time(mu: f64, r1: f64, r2: f64) -> f64 {
    let a = (r1 + r2) / 2.0;
    PI * (a * a * a / mu).sqrt()
}

pub fn plane_change_dv(v: f64, delta_inc: f64) -> f64 {
    2.0 * v * (delta_inc / 2.0).sin()
}

pub fn hyperbolic_excess_dv(mu: f64, r: f64, v_inf: f64) -> f64 {
    let v_circ = circular_velocity(mu, r);
    let v_depart = (v_inf * v_inf + 2.0 * mu / r).sqrt();
    (v_depart - v_circ).abs()
}

// ---------------------------------------------------------------------------
// Rocket equation
// ---------------------------------------------------------------------------

#[derive(Clone)]
pub struct Stage {
    pub name: &'static str,
    pub isp: f64,
    pub thrust: f64,
    pub wet_mass: f64,
    pub dry_mass: f64,
}

#[derive(Clone)]
pub struct Vehicle {
    pub name: &'static str,
    pub stages: &'static [Stage],
}

pub fn total_vehicle_delta_v(v: &Vehicle) -> f64 {
    let mut total = 0.0;
    for i in 0..v.stages.len() {
        let mut payload = 0.0;
        for j in (i + 1)..v.stages.len() {
            payload += v.stages[j].wet_mass;
        }
        let m0 = v.stages[i].wet_mass + payload;
        let mf = v.stages[i].dry_mass + payload;
        if mf > 0.0 && m0 > mf {
            total += v.stages[i].isp * G0 * (m0 / mf).ln();
        }
    }
    total
}

// ---------------------------------------------------------------------------
// Vehicle catalog
// ---------------------------------------------------------------------------

pub static VEHICLES: &[Vehicle] = &[
    Vehicle {
        name: "Generic",
        stages: &[
            Stage {
                name: "Stage 1",
                isp: 290.0,
                thrust: 7e6,
                wet_mass: 400000.0,
                dry_mass: 30000.0,
            },
            Stage {
                name: "Stage 2",
                isp: 340.0,
                thrust: 1e6,
                wet_mass: 100000.0,
                dry_mass: 6000.0,
            },
        ],
    },
    Vehicle {
        name: "Falcon-like",
        stages: &[
            Stage {
                name: "Stage 1",
                isp: 282.0,
                thrust: 7.6e6,
                wet_mass: 433100.0,
                dry_mass: 25600.0,
            },
            Stage {
                name: "Stage 2",
                isp: 348.0,
                thrust: 934000.0,
                wet_mass: 111500.0,
                dry_mass: 4000.0,
            },
        ],
    },
    Vehicle {
        name: "Saturn V-like",
        stages: &[
            Stage {
                name: "S-IC",
                isp: 263.0,
                thrust: 35.1e6,
                wet_mass: 2290000.0,
                dry_mass: 131000.0,
            },
            Stage {
                name: "S-II",
                isp: 421.0,
                thrust: 5.141e6,
                wet_mass: 496200.0,
                dry_mass: 36200.0,
            },
            Stage {
                name: "S-IVB",
                isp: 421.0,
                thrust: 1.033e6,
                wet_mass: 123000.0,
                dry_mass: 13300.0,
            },
        ],
    },
];

// ---------------------------------------------------------------------------
// Destination catalog
// ---------------------------------------------------------------------------

#[derive(Clone, Copy, PartialEq, Eq)]
pub enum ReferenceFrame {
    EarthCentered,
    Heliocentric,
}

#[derive(Clone)]
pub struct Destination {
    pub name: &'static str,
    pub altitude: f64,
    pub apoapsis_alt: f64,
    pub semi_major_axis: f64,
    pub inclination: f64,
    pub frame: ReferenceFrame,
}

pub static DESTINATIONS: &[Destination] = &[
    Destination {
        name: "LEO (200 km)",
        altitude: 200e3,
        apoapsis_alt: 200e3,
        semi_major_axis: 0.0,
        inclination: KSC_LATITUDE_RAD,
        frame: ReferenceFrame::EarthCentered,
    },
    Destination {
        name: "ISS Orbit",
        altitude: 408e3,
        apoapsis_alt: 408e3,
        semi_major_axis: 0.0,
        inclination: 51.6 * PI / 180.0,
        frame: ReferenceFrame::EarthCentered,
    },
    Destination {
        name: "GTO",
        altitude: 200e3,
        apoapsis_alt: GEO_ALTITUDE,
        semi_major_axis: 0.0,
        inclination: KSC_LATITUDE_RAD,
        frame: ReferenceFrame::EarthCentered,
    },
    Destination {
        name: "Moon Transfer (TLI)",
        altitude: 200e3,
        apoapsis_alt: MOON_DISTANCE,
        semi_major_axis: 0.0,
        inclination: KSC_LATITUDE_RAD,
        frame: ReferenceFrame::EarthCentered,
    },
    Destination {
        name: "Mars Transfer (Hohmann)",
        altitude: 200e3,
        apoapsis_alt: 0.0,
        semi_major_axis: MARS_ORBIT_SMA,
        inclination: 1.85 * PI / 180.0,
        frame: ReferenceFrame::Heliocentric,
    },
];

// ---------------------------------------------------------------------------
// Launch plan
// ---------------------------------------------------------------------------

#[derive(Clone, Default)]
pub struct DeltaVBudget {
    pub ascent: f64,
    pub plane_change: f64,
    pub transfer: f64,
    pub arrival: f64,
    pub total: f64,
}

#[derive(Clone)]
pub struct LaunchPlan {
    pub vehicle_name: String,
    pub dest_name: String,
    pub budget: DeltaVBudget,
    pub transfer_time: f64,
    pub parking_orbit_v: f64,
    pub parking_altitude: f64,
    pub vehicle_delta_v: f64,
    pub feasible: bool,
}

pub fn plan(vehicle: &Vehicle, dest: &Destination) -> LaunchPlan {
    let parking_alt = if dest.altitude > 0.0 {
        dest.altitude
    } else {
        200e3
    };
    let r_park = R_EARTH + parking_alt;
    let park_v = circular_velocity(MU_EARTH, r_park);

    let ascent = park_v + GRAVITY_DRAG_LOSS - EARTH_ROTATIONAL_V;

    let mut plane_change = 0.0;
    if dest.frame == ReferenceFrame::EarthCentered {
        let inc_diff = (dest.inclination - KSC_LATITUDE_RAD).abs();
        if inc_diff > 0.001 {
            plane_change = plane_change_dv(park_v, inc_diff);
        }
    }

    let (transfer, arrival, transfer_time) = match dest.frame {
        ReferenceFrame::Heliocentric => {
            let (dv1h, dv2h) = hohmann_delta_v(MU_SUN, EARTH_ORBIT_SMA, dest.semi_major_axis);
            let t = hohmann_transfer_time(MU_SUN, EARTH_ORBIT_SMA, dest.semi_major_axis);
            let transfer_dv = hyperbolic_excess_dv(MU_EARTH, r_park, dv1h);
            let mu_mars = 4.283e13;
            let r_mars_orbit = 3.3895e6 + 300e3;
            let arrival_dv = hyperbolic_excess_dv(mu_mars, r_mars_orbit, dv2h);
            (transfer_dv, arrival_dv, t)
        }
        ReferenceFrame::EarthCentered if dest.apoapsis_alt > 100000e3 => {
            // Lunar
            let r_moon = R_EARTH + dest.apoapsis_alt;
            let (dv1, dv2) = hohmann_delta_v(MU_EARTH, r_park, r_moon);
            let r_lunar_orbit = R_MOON + 100e3;
            let arrival_dv = hyperbolic_excess_dv(MU_MOON, r_lunar_orbit, dv2);
            let t = hohmann_transfer_time(MU_EARTH, r_park, r_moon);
            (dv1, arrival_dv, t)
        }
        ReferenceFrame::EarthCentered if dest.apoapsis_alt > dest.altitude + 1e3 => {
            // Elliptical (GTO)
            let r_apo = R_EARTH + dest.apoapsis_alt;
            let (dv1, dv2) = hohmann_delta_v(MU_EARTH, r_park, r_apo);
            let t = hohmann_transfer_time(MU_EARTH, r_park, r_apo);
            (dv1, dv2, t)
        }
        _ => (0.0, 0.0, 0.0),
    };

    let total = ascent + plane_change + transfer + arrival;
    let vehicle_dv = total_vehicle_delta_v(vehicle);

    LaunchPlan {
        vehicle_name: vehicle.name.to_string(),
        dest_name: dest.name.to_string(),
        budget: DeltaVBudget {
            ascent,
            plane_change,
            transfer,
            arrival,
            total,
        },
        transfer_time,
        parking_orbit_v: park_v,
        parking_altitude: parking_alt,
        vehicle_delta_v: vehicle_dv,
        feasible: vehicle_dv >= total,
    }
}

// ---------------------------------------------------------------------------
// Trajectory propagation
// ---------------------------------------------------------------------------

#[derive(Clone)]
pub struct TrajectoryPoint {
    pub time: f64,
    pub position: [f64; 3],
    pub velocity: [f64; 3],
}

#[derive(Clone)]
pub struct Trajectory {
    pub points: Vec<TrajectoryPoint>,
    pub frame: ReferenceFrame,
}

pub fn propagate_trajectory(launch_plan: &LaunchPlan, dest: &Destination) -> Trajectory {
    let r_park = R_EARTH + launch_plan.parking_altitude;

    match dest.frame {
        ReferenceFrame::Heliocentric => {
            let pos = [EARTH_ORBIT_SMA, 0.0, 0.0];
            let v_circ = circular_velocity(MU_SUN, EARTH_ORBIT_SMA);
            let (dv1, _) = hohmann_delta_v(MU_SUN, EARTH_ORBIT_SMA, dest.semi_major_axis);
            let vel = [0.0, v_circ + dv1, 0.0];
            propagate_rk4(
                pos,
                vel,
                MU_SUN,
                3600.0,
                launch_plan.transfer_time,
                ReferenceFrame::Heliocentric,
            )
        }
        _ => {
            let pos = [r_park, 0.0, 0.0];
            let v_park = circular_velocity(MU_EARTH, r_park);
            let total_v = v_park + launch_plan.budget.transfer;
            let vel = [0.0, total_v, 0.0];
            let duration = if launch_plan.transfer_time > 0.0 {
                launch_plan.transfer_time
            } else {
                2.0 * PI * r_park / v_park
            };
            propagate_rk4(
                pos,
                vel,
                MU_EARTH,
                60.0,
                duration,
                ReferenceFrame::EarthCentered,
            )
        }
    }
}

fn propagate_rk4(
    mut pos: [f64; 3],
    mut vel: [f64; 3],
    mu: f64,
    dt: f64,
    duration: f64,
    frame: ReferenceFrame,
) -> Trajectory {
    let mut points = Vec::new();
    let total_steps = (duration / dt) as usize;
    let record_interval = if total_steps > 1000 {
        total_steps / 1000
    } else {
        1
    };

    let mut t = 0.0;
    let mut step = 0;

    while t < duration && step < 100000 {
        if step % record_interval == 0 {
            points.push(TrajectoryPoint {
                time: t,
                position: pos,
                velocity: vel,
            });
        }

        let (new_pos, new_vel) = rk4_step(pos, vel, dt, mu);
        pos = new_pos;
        vel = new_vel;
        t += dt;
        step += 1;
    }

    points.push(TrajectoryPoint {
        time: t,
        position: pos,
        velocity: vel,
    });
    Trajectory { points, frame }
}

fn rk4_step(pos: [f64; 3], vel: [f64; 3], dt: f64, mu: f64) -> ([f64; 3], [f64; 3]) {
    let accel = |p: [f64; 3]| -> [f64; 3] {
        let r = (p[0] * p[0] + p[1] * p[1] + p[2] * p[2]).sqrt();
        if r < 1e3 {
            return [0.0; 3];
        }
        let f = -mu / (r * r * r);
        [p[0] * f, p[1] * f, p[2] * f]
    };

    let add = |a: [f64; 3], b: [f64; 3]| -> [f64; 3] { [a[0] + b[0], a[1] + b[1], a[2] + b[2]] };
    let scale = |a: [f64; 3], s: f64| -> [f64; 3] { [a[0] * s, a[1] * s, a[2] * s] };

    let a1 = accel(pos);
    let k1v = vel;
    let k1a = a1;

    let p2 = add(pos, scale(k1v, dt / 2.0));
    let v2 = add(vel, scale(k1a, dt / 2.0));
    let a2 = accel(p2);

    let p3 = add(pos, scale(v2, dt / 2.0));
    let v3 = add(vel, scale(a2, dt / 2.0));
    let a3 = accel(p3);

    let p4 = add(pos, scale(v3, dt));
    let v4 = add(vel, scale(a3, dt));
    let a4 = accel(p4);

    let new_pos = add(
        pos,
        scale(
            add(add(k1v, scale(v2, 2.0)), add(scale(v3, 2.0), v4)),
            dt / 6.0,
        ),
    );
    let new_vel = add(
        vel,
        scale(
            add(add(k1a, scale(a2, 2.0)), add(scale(a3, 2.0), a4)),
            dt / 6.0,
        ),
    );

    (new_pos, new_vel)
}

// ---------------------------------------------------------------------------
// Summary
// ---------------------------------------------------------------------------

pub fn summary(plan: &LaunchPlan) -> String {
    let status = if plan.feasible {
        "FEASIBLE"
    } else {
        "NOT FEASIBLE (insufficient dv)"
    };
    let transfer_days = plan.transfer_time / 86400.0;

    format!(
        "Kennedy Space Center Launch Plan\n\
         ================================\n\
         Vehicle: {} (total dv: {:.1} m/s = {:.2} km/s)\n\
         Destination: {}\n\
         Status: {}\n\
         \n\
         Delta-V Budget:\n\
         \x20 Ascent to parking orbit:  {:8.1} m/s  ({:.2} km/s)\n\
         \x20 Plane change:             {:8.1} m/s  ({:.2} km/s)\n\
         \x20 Transfer burn:            {:8.1} m/s  ({:.2} km/s)\n\
         \x20 Arrival burn:             {:8.1} m/s  ({:.2} km/s)\n\
         \x20 ─────────────────────────────────\n\
         \x20 Total required:           {:8.1} m/s  ({:.2} km/s)\n\
         \n\
         Parking Orbit:\n\
         \x20 Altitude: {:.0} km\n\
         \x20 Velocity: {:.1} m/s ({:.2} km/s)\n\
         \n\
         Transfer Time: {:.1} days ({:.2} hours)\n\
         \n\
         Formulas Used:\n\
         \x20 Tsiolkovsky: dv = Isp * g0 * ln(m0/mf)\n\
         \x20 Orbital velocity: v = sqrt(mu/r)\n\
         \x20 Hohmann transfer: dv1 = sqrt(mu/r1) * (sqrt(2*r2/(r1+r2)) - 1)\n\
         \x20 Transfer time: t = pi * sqrt(a^3/mu)",
        plan.vehicle_name,
        plan.vehicle_delta_v,
        plan.vehicle_delta_v / 1000.0,
        plan.dest_name,
        status,
        plan.budget.ascent,
        plan.budget.ascent / 1000.0,
        plan.budget.plane_change,
        plan.budget.plane_change / 1000.0,
        plan.budget.transfer,
        plan.budget.transfer / 1000.0,
        plan.budget.arrival,
        plan.budget.arrival / 1000.0,
        plan.budget.total,
        plan.budget.total / 1000.0,
        plan.parking_altitude / 1000.0,
        plan.parking_orbit_v,
        plan.parking_orbit_v / 1000.0,
        transfer_days,
        plan.transfer_time / 3600.0,
    )
}
