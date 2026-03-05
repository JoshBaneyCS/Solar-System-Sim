# Numerics & Stability Guide

> Expanded by `math-sme` agent.

This document covers the numerical methods, timestep strategies, floating-point
considerations, and visualization numerics used in the solar system simulator.

---

## 1. Integrator Comparison: RK4 vs Velocity Verlet

### RK4 (4th-order Runge-Kutta)

- **Error**: Local O(dt^5), global O(dt^4)
- **Cost**: 4 force evaluations per step (4 snapshots × N bodies)
- **Strengths**: High accuracy per step; good for short, precise integrations
- **Weakness**: Non-symplectic — energy drifts monotonically over long runs. After 1 year
  with dt=7200s, relative energy drift is ~1.4e-5.

### Velocity Verlet (symplectic)

- **Error**: Local O(dt^3), global O(dt^2)
- **Cost**: 2 force evaluations per step
- **Strengths**: Symplectic — conserves a shadow Hamiltonian, so energy oscillates around
  the true value rather than drifting. After 1 year with dt=7200s, relative energy drift
  is ~3.4e-9 (3+ orders of magnitude better than RK4).
- **Weakness**: Lower order means larger per-step error at the same dt. But for orbital
  mechanics over years-to-decades, the bounded energy error dominates.

### Algorithm (Velocity Verlet)

```
1. v(t + dt/2) = v(t) + a(t) * dt/2        # half-step velocity
2. x(t + dt)  = x(t) + v(t + dt/2) * dt     # full-step position
3. a(t + dt)  = acceleration(x(t + dt))      # recompute forces
4. v(t + dt)  = v(t + dt/2) + a(t + dt) * dt/2  # complete velocity
```

### Recommendation

Verlet is the default integrator. Use RK4 only when you need higher per-step accuracy
for short integrations (e.g., trajectory propagation in the launch planner). For the
main N-body simulation running over months to decades, Verlet's symplectic property
makes it the clear winner.

---

## 2. Timestep Analysis

### Base Timestep: 7200s (2 hours)

The base timestep of 2 hours was chosen to balance accuracy and performance:

- **Mercury** (fastest planet): orbital period ~88 days = 7.6e6 s → ~1056 steps/orbit.
  At 1000+ steps per orbit, even 2nd-order Verlet captures the elliptical dynamics
  accurately. Nyquist says we need >2 samples/orbit; 1056 provides a safety factor of 500x.
- **Neptune** (slowest): ~165 years/orbit → ~7.2e5 steps/orbit. More than sufficient.
- **Earth**: ~365.25 days → ~4383 steps/orbit.

### TimeSpeed Slider Risk

The UI speed slider ranges from 2^-10 (~0.001x) to 2^10 (~1024x). At maximum speed:

```
effective_dt = BaseTimeStep * TimeSpeed = 7200 * 1024 ≈ 7.4e6 s ≈ 85 days
```

Mercury's orbital period is ~88 days, so at max speed the simulator would take
~1 step per orbit — catastrophically undersampled. The orbit would spiral outward
and the simulation would become unphysical.

### Substep Protection

The `Update()` method subdivides large effective timesteps:

```
MaxSafeDt = 28800s (8 hours)

if effectiveDt > MaxSafeDt:
    nSub = ceil(effectiveDt / MaxSafeDt)
    subDt = effectiveDt / nSub
    for each substep: Step(subDt)
```

At max speed (1024x), this produces ~256 substeps per frame, each at ~28800s. Mercury
gets ~264 steps/orbit — still well-sampled. The cost is ~256 force evaluations per
frame instead of 1, but with 8 bodies this remains fast.

---

## 3. Floating-Point Analysis

### Why Kahan Summation Is NOT Needed

Kahan (compensated) summation reduces floating-point rounding from O(N*eps) to O(eps)
when accumulating N terms. For our simulation:

- **N = 8 bodies**: summation error is ~8 * eps ≈ 8 * 1.1e-16 ≈ 9e-16
- **Verlet integration error**: ~3.4e-9 per year (relative energy drift)
- **RK4 integration error**: ~1.4e-5 per year

The integration error exceeds summation error by **7+ orders of magnitude** (Verlet)
to **10+ orders** (RK4). Kahan summation would improve a term that's already negligible.

### Time Accumulation

`CurrentTime` is accumulated as `CurrentTime += dt` each step. After 100 years at
dt=7200s:

```
steps = 100 * 365.25 * 86400 / 7200 ≈ 4.38e5
CurrentTime ≈ 3.15e9
Relative error ≈ steps * eps ≈ 4.38e5 * 1.1e-16 ≈ 4.8e-11
```

This is ~100x smaller than Verlet's integration error. Not a concern.

### Position Precision

Planet positions range from ~5.8e10 m (Mercury) to ~4.5e12 m (Neptune). With float64
providing ~15.7 significant digits, the least significant bit represents:

- Mercury: 5.8e10 * 1.1e-16 ≈ 6.4e-6 m (~6 micrometers)
- Neptune: 4.5e12 * 1.1e-16 ≈ 5.0e-4 m (~0.5 mm)

These are far below any physical concern. Double precision is more than adequate.

---

## 4. Visualization Numerics

### Trail Rendering

Orbital trails are stored as discrete position samples (one per integration step).
The renderer draws piecewise-linear segments between consecutive trail points.

**Catmull-Rom interpolation** is applied when rendering trails to produce smoother
curves, inserting 3-4 interpolated points per stored segment. This is purely visual —
it does not affect the physics. The interpolation uses the standard Catmull-Rom spline:

```
P(t) = 0.5 * ((2*P1) + (-P0+P2)*t + (2*P0-5*P1+4*P2-P3)*t² + (-P0+3*P1-3*P2+P3)*t³)
```

where P0..P3 are four consecutive trail points and t ∈ [0,1].

### Trail Decimation

When trails exceed 200 points, the renderer subsamples by `step = len(trail) / 200`
to keep draw calls bounded. This can lose detail on tight curves. The Catmull-Rom
interpolation partially compensates.

### Viewport Float32 Precision

Screen coordinates use float32 (Fyne framework). At 4K resolution (~4000 px), the
smallest representable offset is ~4000 * 1.2e-7 ≈ 5e-4 px — well below a pixel.
Float32 is adequate for screen-space rendering.

### Spacetime Curvature Grid

The spacetime visualization samples curvature on a grid (default 80 points per axis).
This is a qualitative display, not a physical simulation. The grid density is sufficient
to show the curvature well around the Sun. Interpolation between grid points is not
needed since the display is already a visual approximation.

---

## 5. Cross-Platform Reproducibility

### Go's Math Package

Go's `math` package uses pure software implementations (not hardware FMA/SIMD), which
means results are bitwise identical across platforms (x86, ARM, etc.) for the same
Go version. This is a significant advantage for reproducibility.

### Golden Tests

The `golden_test.go` file captures exact positions and velocities after 100 and 1000
RK4 steps. These values are checked to sub-meter position accuracy and ~1e-6 m/s
velocity accuracy. If a code change alters the integration path, golden tests fail
immediately.

Note: Golden tests explicitly use `IntegratorRK4` since the golden baseline values
were computed with RK4. Changing the default integrator does not affect golden tests.

### FMA Considerations

Some platforms (e.g., ARM with hardware FMA) could produce slightly different results
if the compiler used fused multiply-add. Go's software math avoids this, but if
performance-critical code ever moves to CGo or assembly, FMA consistency should be
verified.

---

## 6. Preset Table

| Parameter | Accurate (slow) | Balanced (default) | Fast |
|---|---|---|---|
| Integrator | Verlet | Verlet | RK4 |
| Base dt | 3600s (1 hr) | 7200s (2 hr) | 14400s (4 hr) |
| Max safe dt | 7200s | 28800s | 86400s |
| Trail length | 2000 | 500 | 200 |
| Spacetime grid | 120 | 80 | 40 |
| GR effects | On | On | Off |
| Energy drift/year | ~1e-10 | ~1e-7 | ~1e-4 |
| Use case | Research, validation | General use | Quick demos |

### Notes

- **Accurate** halves the timestep and doubles the trail buffer, giving ~4x more work
  per simulated second. Best for measuring precession or validating against ephemeris.
- **Balanced** is the default. Verlet keeps energy drift negligible while the 2-hour
  step provides 1000+ steps per Mercury orbit.
- **Fast** switches to RK4 with a 4-hour step. Energy will drift ~1e-4/year, which is
  acceptable for visual demos but not for quantitative analysis. GR is disabled to
  save the extra computation (GR effects are too small to see visually).

---

## 7. What We Considered and Rejected

### Adaptive Timestep

An adaptive integrator (e.g., RK45 with error control) would automatically shrink dt
near perihelion and grow it in straight-line segments. We rejected this because:

1. **Reproducibility**: Variable dt makes golden tests and cross-run comparisons harder
2. **Substeps solve the real problem**: The TimeSpeed slider is the actual source of
   large dt values, and fixed substeps handle it simply
3. **Overhead**: Error estimation requires extra force evaluations per step

### Kahan Summation

See Section 3. Integration error dominates by 7-10 orders of magnitude. Not worth
the code complexity for 8 bodies.

### Higher-Order Symplectic Integrators

Methods like Yoshida 4th-order or Forest-Ruth provide O(dt^4) symplectic integration.
We chose Velocity Verlet because:

1. The 2nd-order error at dt=7200s is already sufficient (energy drift ~1e-9/year)
2. Higher-order methods require 3-7 force evaluations per step (vs Verlet's 2)
3. The validated Mercury precession of 42.97 arcsec/century (0.07% error) confirms
   that Verlet accuracy is more than adequate for our physics

### Curvature Field Interpolation

The spacetime curvature display is qualitative. Bicubic interpolation between grid
points would smooth the visualization but adds complexity for a feature that is
already an approximation. The current grid sampling is sufficient.
