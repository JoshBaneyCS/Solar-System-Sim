# Migration Validation Matrix

Cross-reference of every physics validation scenario between the Go baseline and the Rust migration target. All baseline values and tolerances are extracted directly from Go source code.

## Constants

| Constant | Value | Source |
|----------|-------|--------|
| `BaseTimeStep` | 7200.0 s (2 hours) | `pkg/constants/constants.go:9` |
| `G` | 6.674e-11 m^3 kg^-1 s^-2 | `pkg/constants/constants.go` |
| `C` | 2.998e8 m/s | `pkg/constants/constants.go` |
| `AU` | 1.496e11 m | `pkg/constants/constants.go` |

---

## Physics Validation Scenarios

| # | Scenario | Go Test File | Go Baseline Value | Tolerance | Rust Test | Phase Gate | Status |
|---|----------|-------------|------------------|-----------|-----------|------------|--------|
| 1 | Energy conservation (RK4, 1-year, Sun-only) | `internal/physics/integration_test.go:35-54` `TestEnergyConservation` | Relative energy drift ~0 over 1000 steps (Sun-only, no N-body) | `1e-6` relative drift | `crates/physics_core/src/sim.rs` `test_energy_conservation` (exists, 1000 steps, tolerance 1e-6) | Phase A | Rust test exists; uses single Earth body. Must extend to full 9-body system. |
| 2 | Energy conservation (RK4, N-body, 1000 steps) | `internal/physics/integration_test.go:138-157` `TestEnergyConservationWithNBody` | Relative drift over 1000 steps with planet-planet gravity | `1e-5` relative drift | Not yet in Rust | Phase A | **Missing in Rust** |
| 3 | Energy conservation (N-body, multi-year validation) | `internal/validation/energy.go:25-52` `ValidateEnergyConservation` | Relative drift < tolerance over N years. N-body enabled, GR disabled. | `1e-4 * years` relative drift (e.g., 1e-4 for 1 year, 1e-3 for 10 years) | Not yet in Rust | Phase A | **Missing in Rust** |
| 4 | Energy conservation (Verlet, 1000 steps) | `internal/physics/verlet_test.go:10-30` `TestVerletEnergyConservation` | Relative drift over 1000 steps. Sun-only, no N-body, Verlet integrator. | `1e-6` relative drift | Not yet in Rust (Verlet integrator missing) | Phase A | **Missing in Rust** -- Verlet not yet ported |
| 5 | Angular momentum conservation (RK4, 1000 steps) | `internal/physics/integration_test.go:56-90` `TestAngularMomentumConservation` | Per-component (Lx, Ly, Lz) relative drift over 1000 steps. Sun-only. | `1e-6` relative drift per component | Not yet in Rust | Phase A | **Missing in Rust** |
| 6 | Angular momentum conservation (Verlet, 1000 steps) | `internal/physics/verlet_test.go:32-52` `TestVerletAngularMomentumConservation` | Relative magnitude drift over 1000 steps. Sun-only, Verlet integrator. | `1e-6` relative drift | Not yet in Rust | Phase A | **Missing in Rust** |
| 7 | Angular momentum conservation (multi-year validation) | `internal/validation/angular_momentum.go:24-51` `ValidateAngularMomentumConservation` | Relative magnitude drift over N years. Sun-only (PlanetGravity=false). | `1e-6` relative drift | Not yet in Rust | Phase A | **Missing in Rust** |
| 8 | Earth orbital period (RK4) | `internal/physics/integration_test.go:92-136` `TestEarthOrbitalPeriod` | ~365.25 days. Measured by cumulative angle tracking via atan2. Sun-only. | `1%` relative error (0.01) | Not yet in Rust | Phase A | **Missing in Rust** |
| 9 | Earth orbital period (Verlet) | `internal/physics/verlet_test.go:54-99` `TestVerletEarthOrbitalPeriod` | ~365.25 days. Same angle-tracking method. | `1%` relative error (0.01) | Not yet in Rust | Phase A | **Missing in Rust** |
| 10 | Earth Kepler period (validation harness) | `internal/validation/kepler.go:29-109` `ValidateKeplerPeriod("Earth", 2.0)` | Expected: 365.256 days. Measured over 2 years of simulation. | `1%` relative error (0.01) | Not yet in Rust | Phase A | **Missing in Rust** |
| 11 | Mercury orbital period | `internal/validation/kepler.go:29-109` `ValidateKeplerPeriod("Mercury", 1.0)` | Expected: 87.969 days. Measured over 1 year of simulation. | `1%` relative error (0.01) | Not yet in Rust | Phase A | **Missing in Rust** |
| 12 | Mercury perihelion precession (GR) | `internal/validation/mercury_precession.go:18-46` `ValidateMercuryPrecession(10.0)` | Expected: 43.0 arcsec/century. GR-only component isolated by subtracting Newton-only rate from GR+Newton rate. Measured via LRL vector linear regression over 10 simulated years. | `70%` relative error (0.7), i.e., range ~13--73 arcsec/century | Not yet in Rust | Phase A | **Missing in Rust** -- requires body catalog, LRL vector, linear regression |
| 13 | Mercury precession without GR | `internal/validation/mercury_precession.go:23` (Newton-only rate subtracted) | ~0 arcsec/century (Newtonian precession rate is the baseline subtracted from GR rate) | Implicitly tested as part of scenario 12 | Not yet in Rust | Phase A | **Missing in Rust** |
| 14 | Golden test (100 steps, RK4) | `internal/physics/golden_test.go:69-71` `TestGoldenBaseline100` | Exact positions and velocities for all 9 planets after 100 RK4 steps. PlanetGravity=true, GR=true, dt=7200s. See `golden100` array. | Position: `1e-1` m absolute per component. Velocity: `1e-6` m/s absolute per component. | Not yet in Rust | Phase A | **Missing in Rust** -- requires identical initial conditions from body catalog |
| 15 | Golden test (1000 steps, RK4) | `internal/physics/golden_test.go:73-75` `TestGoldenBaseline1000` | Exact positions and velocities for all 9 planets after 1000 RK4 steps. Same config. See `golden1000` array. | Position: `1e-1` m absolute per component. Velocity: `1e-6` m/s absolute per component. | Not yet in Rust | Phase A | **Missing in Rust** |
| 16 | Orbital element roundtrip | `internal/physics/orbital_elements_test.go:10-31` `TestCircularOrbit` | Circular orbit (e=0, a=1 AU): position should be at (1 AU, 0, 0), velocity should be (0, v_circular, 0). | Position: `1e-10` relative. Velocity: `1e-10` relative. | Not yet in Rust | Phase A | **Missing in Rust** |
| 17 | Orbital radius from elements | `internal/physics/orbital_elements_test.go:33-50` `TestOrbitalRadius` | For each planet: computed |r| matches expected `a(1-e^2)/(1+e*cos(nu))`. | `1e-8` relative error | Not yet in Rust | Phase A | **Missing in Rust** |
| 18 | Velocity magnitude from elements | `internal/physics/orbital_elements_test.go:72-96` `TestVelocityFormula` | For each planet: computed |v| matches expected from mu/h perifocal velocity. | `1e-8` relative error | Not yet in Rust | Phase A | **Missing in Rust** |
| 19 | Rotation preserves magnitude | `internal/physics/orbital_elements_test.go:98-114` `TestRotationsPreserveMagnitude` | For each planet: |position| after 3D rotation matches expected orbital radius. | `1e-8` relative error | Not yet in Rust | Phase A | **Missing in Rust** |
| 20 | Zero inclination stays in ecliptic | `internal/physics/orbital_elements_test.go:116-130` `TestZeroInclinationInEcliptic` | Z position < 1e-3 m, Vz < 1e-6 m/s for i=0 orbit. | Absolute: Z < 1e-3, Vz < 1e-6 | Not yet in Rust | Phase A | **Missing in Rust** |
| 21 | Inclined orbit has non-zero Z | `internal/physics/orbital_elements_test.go:52-70` `TestInclinedOrbit` | |Z| > 1e6 m, |Vz| > 1e-3 m/s for 30-degree inclined orbit. | Lower bounds: Z > 1e6, Vz > 1e-3 | Not yet in Rust | Phase A | **Missing in Rust** |
| 22 | N-body stability (long run) | Not explicitly tested | No NaN/Infinity in positions after 100 simulated years | All position/velocity components finite | Not yet in Rust | Phase A | **Missing in both Go and Rust** -- recommended new test |
| 23 | Time reversal | Not explicitly tested | Forward 1000 steps then backward 1000 steps should return to approximately initial state | Positions within 1% of initial for RK4 (non-symplectic, expect some drift) | Not yet in Rust | Phase A | **Missing in both Go and Rust** -- recommended new test |

---

## GR Correction Validation

| # | Scenario | Go Test File | Baseline / Acceptance Criteria | Tolerance | Rust Test | Status |
|---|----------|-------------|-------------------------------|-----------|-----------|--------|
| G1 | GR correction non-zero for Mercury | `internal/physics/gr_test.go:10-42` | GR correction magnitude > 1e-20 m/s^2. GR/Newtonian ratio in [1e-10, 1e-5]. | Ratio range: 1e-10 to 1e-5 | `crates/physics_core/src/gr.rs` `test_gr_correction_nonzero_for_mercury` (exists) | Rust test exists |
| G2 | GR correction non-zero for Venus, Earth, Mars | `internal/physics/gr_test.go:44-76` | diff magnitude > 1e-20. GR/Newtonian ratio in [1e-15, 1e-5]. | Ratio range: 1e-15 to 1e-5 | Not yet in Rust (Rust tests only cover Mercury) | **Missing in Rust** |
| G3 | GR formula matches manual 1PN | `internal/physics/gr_test.go:78-117` | Code GR component matches hand-computed `(GM/(c^2 r^3))[(4GM/r - v^2)r + 4(r.v)v]` per component. | `1e-8` relative error per XYZ component | `crates/physics_core/src/gr.rs` `test_gr_correction_formula_1pn` (exists) | Rust test exists |
| G4 | GR correction perpendicular to L | `internal/physics/gr_test.go:119-147` | dot(GR_correction, L) < |GR|*|L|*1e-8 | 1e-8 scaled tolerance | Not yet in Rust | **Missing in Rust** |
| G5 | GR correction at zero velocity | Not in Go | Direction matches position vector. y,z < 1e-30 | Absolute < 1e-30 | `crates/physics_core/src/gr.rs` `test_gr_correction_zero_velocity` (exists) | Rust-only test (good) |

---

## Gravity Law Validation

| # | Scenario | Go Test File | Baseline / Acceptance Criteria | Tolerance | Rust Test | Status |
|---|----------|-------------|-------------------------------|-----------|-----------|--------|
| V1 | Sun-only gravity magnitude | `internal/physics/gravity_test.go:10-36` | |a| = GM_sun / r^2. Direction toward Sun (-X). Y,Z ~ 0. | `1e-8` relative error | `crates/physics_core/src/sim.rs` `test_sun_only_gravity` (exists) | Both exist |
| V2 | Inverse square law (1 AU vs 2 AU) | `internal/physics/gravity_test.go:38-56` | Ratio of |a(1AU)| / |a(2AU)| = 4.0 | `1e-8` relative error (Go), `1e-10` (Rust) | `crates/physics_core/src/sim.rs` `test_inverse_square_law` (exists) | Both exist |
| V3 | N-body perturbation is small but non-zero | `internal/physics/gravity_test.go:58-84` | diff magnitude > 1e-20. ratio < 0.01 of Sun-only. | Upper bound: 1% | Not yet in Rust | **Missing in Rust** |
| V4 | Gravity symmetry (+X vs -X) | `internal/physics/gravity_test.go:86-100` | |a(+x)| = |a(-x)| | `1e-10` relative error | Not yet in Rust | **Missing in Rust** |

---

## UI State Validation (Functional Regression)

| # | Scenario | Go Test File | Acceptance Criteria | Rust Test | Phase Gate | Status |
|---|----------|-------------|--------------------|-----------|-----------| --------|
| U1 | SetShowTrails propagates to simulator | `internal/ui/state_test.go:11-29` | state.ShowTrails() and sim.ShowTrails match after set | N/A | Phase C | **Bevy resource equivalent needed** |
| U2 | Listener fires on state change | `internal/ui/state_test.go:31-46` | Callback invoked within 100ms | N/A | Phase C | **Bevy event system equivalent needed** |
| U3 | SetIntegrator propagates to simulator | `internal/ui/state_test.go:48-62` | state.Integrator() and sim.Integrator match | N/A | Phase C | **Bevy resource equivalent needed** |
| U4 | ToSettings serialization | `internal/ui/state_test.go:64-82` | Settings struct reflects current state | N/A | Phase D | **RON serialization equivalent needed** |
| U5 | ApplyFromSettings | `internal/ui/state_test.go:84-111` | State + simulator updated from settings | N/A | Phase D | **RON deserialization equivalent needed** |
| U6 | RebindSimulator | `internal/ui/state_test.go:113-126` | Commands sent to new simulator | N/A | Phase B | **Bevy resource swap equivalent needed** |
| U7 | Concurrent access safety | `internal/ui/state_test.go:128-147` | No panics under concurrent read/write (100 goroutines) | N/A | Phase B | **Bevy's ECS provides this natively** |
| U8 | ResetToDefaults | `internal/ui/state_test.go:149-168` | All state values return to defaults | N/A | Phase C | **Bevy equivalent needed** |

---

## Default State Values (for Reset verification)

Extracted from `internal/ui/state.go:41-56` `NewAppState()`:

| Property | Default Value |
|----------|--------------|
| showTrails | true |
| showSpacetime | false |
| showLabels | true |
| planetGravity | true |
| relativity | true |
| integrator | Verlet |
| timeSpeed | 1.0 |
| isPlaying | true |
| showMoons | true |
| showComets | false |
| showAsteroids | false |
| showBelt | true |

---

## Golden Test Reference Data

### After 100 RK4 Steps (dt=7200s, PlanetGravity=true, GR=true)

Source: `internal/physics/golden_test.go:17-27`

| Body | X (m) | Y (m) | Z (m) |
|------|-------|-------|-------|
| Mercury | -2.998050661944995e+10 | 3.837189492086417e+10 | 5.886361551274197e+09 |
| Venus | -1.059535261813235e+11 | -1.882757140603617e+10 | 5.859097985072027e+09 |
| Earth | -1.398307919832044e+11 | -5.405512215904287e+10 | 5.391508244695966e+03 |
| Mars | -1.022683467478876e+11 | 2.204221664159921e+11 | 7.132305515695683e+09 |
| Jupiter | -7.885520918321320e+11 | -2.107096405710771e+11 | 1.850821009815355e+10 |
| Saturn | 1.105566423231063e+12 | -9.851994769593331e+11 | -2.678345310936499e+10 |
| Uranus | 4.431381727358097e+11 | 2.830268371248371e+12 | 4.774331040625253e+09 |
| Neptune | 4.454140655478871e+12 | 2.456947873653347e+11 | -1.076939082717095e+11 |
| Pluto | -3.012084005221812e+12 | -3.030084869871788e+12 | 1.196923700919022e+12 |

### After 1000 RK4 Steps

Source: `internal/physics/golden_test.go:29-39`

| Body | X (m) | Y (m) | Z (m) |
|------|-------|-------|-------|
| Mercury | 3.100889058745589e+10 | 3.529016438760774e+10 | 3.668515073653610e+07 |
| Venus | 6.934564877366739e+10 | -8.380826706131313e+10 | -5.148989308761284e+09 |
| Earth | 9.242571074567766e+09 | -1.517819751426132e+11 | 3.759308580520449e+05 |
| Mars | -2.118384253708217e+11 | 1.307228764499832e+11 | 7.946415175455531e+09 |
| Jupiter | -7.637076718557493e+11 | -2.873401037117965e+11 | 1.826906735946800e+10 |
| Saturn | 1.142661557154642e+12 | -9.378194900617136e+11 | -2.908330542597362e+10 |
| Uranus | 3.991969419049934e+11 | 2.834762662807740e+12 | 5.360954637255160e+09 |
| Neptune | 4.451749653816566e+12 | 2.810952549436245e+11 | -1.083677399303338e+11 |
| Pluto | -2.985053803975452e+12 | -3.058642083850327e+12 | 1.192154648470679e+12 |

**Tolerances:** Position components within 0.1 m. Velocity components within 1e-6 m/s.

---

## Expected Kepler Periods

Source: `internal/validation/kepler.go:12-25`

| Planet | Expected Period (days) | Tolerance |
|--------|----------------------|-----------|
| Mercury | 87.969 | 1% relative |
| Venus | 224.701 | 1% relative |
| Earth | 365.256 | 1% relative |
| Mars | 686.980 | 1% relative |

---

## Summary: Migration Readiness

| Category | Total Scenarios | Exist in Rust | Missing in Rust | Missing in Both |
|----------|----------------|---------------|-----------------|-----------------|
| Energy conservation | 4 | 1 (partial) | 3 | 0 |
| Angular momentum | 3 | 0 | 3 | 0 |
| Orbital periods | 4 | 0 | 4 | 0 |
| Mercury precession | 2 | 0 | 2 | 0 |
| Golden tests | 2 | 0 | 2 | 0 |
| Orbital elements | 6 | 0 | 6 | 0 |
| N-body stability | 1 | 0 | 0 | 1 |
| Time reversal | 1 | 0 | 0 | 1 |
| GR correction | 5 | 3 | 2 | 0 |
| Gravity law | 4 | 2 | 2 | 0 |
| UI state | 8 | 0 | 8 | 0 |
| **Total** | **40** | **6** | **32** | **2** |

**Phase A gate requires:** Scenarios 1-21, G1-G5, V1-V4 (31 scenarios).
**Phase D gate requires:** All 40 scenarios plus the 2 new recommended tests.
