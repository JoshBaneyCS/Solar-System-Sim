# Physics

## Core Model

- **N-body Newtonian gravity**: F = -GMm/r² between all bodies (Sun + 8 planets)
- **General Relativity**: 1PN post-Newtonian correction for Mercury's perihelion precession
- **Two integrators**: RK4 (default) and Velocity Verlet (symplectic)
- **Gravity softening**: Optional ε parameter for close-encounter stability

## Units

- SI internally: meters, kilograms, seconds
- Constants from CODATA 2018 (G, c) and IAU (AU)

## Integrators

### RK4 (4th-order Runge-Kutta)
- Default integrator, 4th-order accuracy
- Not symplectic: energy drifts over long integrations (~1e-5 relative drift per year for N-body)
- Set via `sim.Integrator = IntegratorRK4`

### Velocity Verlet (Symplectic)
- 2nd-order accuracy, but symplectic (conserves phase space volume)
- Much better long-term energy conservation (~1e-9 relative drift over 1000 steps)
- Ideal for long-duration orbit studies
- Set via `sim.Integrator = IntegratorVerlet`

## General Relativity

The 1PN post-Newtonian correction is applied to Mercury only:

```
a_GR = (GM/(c²r³)) × [(4GM/r - v²)r + 4(r·v)v]
```

This produces Mercury's perihelion precession of ~43 arcsec/century, matching the theoretical prediction to < 1%.

## Gravity Softening

For close encounter stability, the gravitational force denominator can be softened:

```
F = GMm / (r² + ε²)
```

Set via `sim.SofteningLength = ε` (default 0, meaning disabled).

## Orbital Elements

Planets are initialized from Keplerian orbital elements (a, e, i, Ω, ω, ν) converted to Cartesian state vectors. Velocity in the perifocal frame uses the standard formulas:

```
vx = -(μ/h) sin(ν)
vy = (μ/h) (e + cos(ν))
```

where h = √(μa(1-e²)) is the specific angular momentum.

## Validation Harness

The `internal/validation/` module provides automated physics validation scenarios accessible via CLI:

```bash
solar-sim validate --scenario all --years 10
```

### Available Scenarios

| Scenario | What it tests | Pass criterion |
|----------|--------------|----------------|
| `energy` | Total energy conservation (N-body) | ΔE/E < 1e-4 × years |
| `angular-momentum` | Angular momentum conservation | ΔL/L < 1e-6 |
| `kepler-earth` | Earth orbital period vs 365.256 days | < 1% error |
| `kepler-mercury` | Mercury orbital period vs 87.969 days | < 1% error |
| `mercury-precession` | GR perihelion precession vs 43 arcsec/century | < 70% error |

### Mercury Precession Measurement

Uses the Laplace-Runge-Lenz (eccentricity) vector to continuously track the perihelion direction. The LRL vector A = (v × L)/GM - r̂ points toward perihelion and rotates at the precession rate. Linear regression on the angle time series gives the rate. The GR component is isolated by subtracting a Newton-only run.

### Typical Results

```
Energy Conservation:     drift ~1.4e-5 per year (RK4)
Angular Momentum:        drift ~6e-15 per year
Kepler Period (Earth):   365.25 days (error 0.002%)
Kepler Period (Mercury): 87.94 days (error 0.04%)
Mercury Precession (GR): 42.97 arcsec/century (error 0.07%)
```
