# Physics Deep Dive: Solar System Simulator

This document provides a detailed mathematical and physical explanation of the simulation.

## Table of Contents

1. [3D Orbital Mechanics](#3d-orbital-mechanics)
2. [N-Body Gravitational Dynamics](#n-body-gravitational-dynamics)
3. [General Relativity Corrections](#general-relativity-corrections)
4. [Numerical Integration Methods](#numerical-integration-methods)
5. [Observable Effects](#observable-effects)

---

## 3D Orbital Mechanics

### Keplerian Orbital Elements

Each planet's orbit is defined by six classical orbital elements:

1. **Semi-major axis (a)**: The average orbital radius
    - Determines orbital period via Kepler's 3rd law: T² ∝ a³

2. **Eccentricity (e)**: Shape of the ellipse
    - e = 0: perfect circle
    - 0 < e < 1: ellipse
    - e = 1: parabola
    - e > 1: hyperbola

3. **Inclination (i)**: Tilt of orbital plane relative to ecliptic
    - Measured in degrees
    - Earth's orbit defines i = 0° (reference plane)
    - Mercury has highest inclination: i = 7.005°

4. **Longitude of Ascending Node (Ω)**:
    - Where orbit crosses ecliptic going "upward" (south to north)
    - Measured from vernal equinox direction

5. **Argument of Perihelion (ω)**:
    - Angle from ascending node to perihelion (closest approach)
    - Defines orientation within orbital plane

6. **True Anomaly (ν)**:
    - Current angular position along orbit
    - Measured from perihelion

### From Orbital Elements to Cartesian Coordinates

**Step 1: Position in orbital plane**

The distance from the Sun at true anomaly ν:

```
r = a(1 - e²) / (1 + e·cos(ν))
```

In the orbital plane (2D):
```
x_orb = r·cos(ν)
y_orb = r·sin(ν)
z_orb = 0
```

**Step 2: Rotation to 3D space**

We apply three successive rotations (Euler angles):

1. **Rotation by ω** (argument of perihelion) around z-axis:
```
x₁ = x_orb·cos(ω) - y_orb·sin(ω)
y₁ = x_orb·sin(ω) + y_orb·cos(ω)
z₁ = 0
```

2. **Rotation by i** (inclination) around x-axis:
```
x₂ = x₁
y₂ = y₁·cos(i) - z₁·sin(i)
z₂ = y₁·sin(i) + z₁·cos(i)
```

3. **Rotation by Ω** (longitude of ascending node) around z-axis:
```
x = x₂·cos(Ω) - y₂·sin(Ω)
y = x₂·sin(Ω) + y₂·cos(Ω)
z = z₂
```

**Result:** Position vector **r⃗** = (x, y, z) in heliocentric coordinates.

### Velocity Calculation

The specific angular momentum (per unit mass):
```
h = √(GM·a·(1 - e²))
```

Velocity in orbital plane:
```
v_x,orb = -(h/r)·sin(ν)
v_y,orb = (h/r)·(e + cos(ν))
```

Apply the same three rotations to get 3D velocity **v⃗**.

---

## N-Body Gravitational Dynamics

### The N-Body Problem

Instead of just considering Sun-planet interactions, we solve the full N-body problem where every body affects every other body.

**Total gravitational force on body i:**

```
F⃗ᵢ = Σⱼ₌₁ⁿ (j≠i) F⃗ᵢⱼ
```

Where the force from body j on body i is:

```
F⃗ᵢⱼ = -G·mᵢ·mⱼ/|r⃗ᵢⱼ|² · r̂ᵢⱼ
```

- **r⃗ᵢⱼ** = r⃗ⱼ - r⃗ᵢ (vector from i to j)
- **|r⃗ᵢⱼ|** = distance between bodies
- **r̂ᵢⱼ** = unit vector pointing from i to j

**Total acceleration of body i:**

```
a⃗ᵢ = F⃗ᵢ/mᵢ = Σⱼ₌₁ⁿ (j≠i) -G·mⱼ/|r⃗ᵢⱼ|² · r̂ᵢⱼ
```

### Physical Effects

The N-body interactions cause:

1. **Orbital Perturbations**:
    - Jupiter (mass = 318 Earth masses) significantly affects other planets
    - Mars and Earth's orbits are perturbed by Jupiter

2. **Orbital Resonances**:
    - Jupiter-Saturn 5:2 resonance (5 Jupiter orbits ≈ 2 Saturn orbits)
    - These create periodic perturbations

3. **Long-term Stability**:
    - Chaotic behavior on million-year timescales
    - Short-term (centuries) orbits are stable

### Computational Complexity

- **2-body problem**: O(n) - just Sun and each planet
- **N-body problem**: O(n²) - all pairs of bodies

For 8 planets + Sun = 9 bodies:
- 2-body: 8 force calculations per step
- N-body: 36 force calculations per step

---

## General Relativity Corrections

### The Problem with Newtonian Gravity

Newton's law of gravitation predicts that elliptical orbits should remain fixed in space. However, Mercury's perihelion (point of closest approach to the Sun) was observed to advance by ~574 arcseconds per century.

**Known effects:**
- Perturbations from other planets: ~531"/century
- **Unexplained discrepancy: ~43"/century**

Einstein's General Relativity explained this perfectly!

### Schwarzschild Metric

In GR, the Sun curves spacetime according to the Schwarzschild metric. For a planet orbiting, this adds extra terms to the acceleration.

**Schwarzschild radius of the Sun:**
```
rₛ = 2GM☉/c² ≈ 2.95 km
```

This is tiny compared to the Sun's actual radius (~696,000 km), so we use the **post-Newtonian approximation**.

### Post-Newtonian Correction

The relativistic correction to acceleration:

```
a⃗_GR = (3G²M²)/(c²r³L) · (L⃗ × r⃗)
```

Where:
- **L⃗** = r⃗ × (m·v⃗) = angular momentum vector
- **L** = |L⃗| = magnitude of angular momentum
- **c** = 299,792,458 m/s (speed of light)

This formula comes from the Einstein field equations in the weak-field, slow-motion limit.

### Physical Interpretation

The GR correction adds a **tangential acceleration** perpendicular to the radial direction. This causes the orbit to slowly rotate (precess) in the orbital plane.

**Effect on perihelion:**

The perihelion advances by:
```
Δφ = 6πGM/(c²a(1-e²))  [per orbit]
```

For Mercury:
- a = 0.387 AU = 5.79×10¹⁰ m
- e = 0.206
- M = M☉ = 1.989×10³⁰ kg

Plugging in:
```
Δφ ≈ 5.04×10⁻⁷ radians per orbit
    ≈ 0.104 arcseconds per orbit
    ≈ 43 arcseconds per century ✓
```

This was one of the first experimental confirmations of General Relativity!

### Why Only Mercury?

The GR effect scales as:
```
Δφ ∝ 1/(c²·a·(1-e²))
```

Mercury is closest to the Sun (smallest a) and has high eccentricity (large e), making the effect ~10× larger than for other planets.

**Perihelion precession rates:**
- Mercury: 43"/century (easily observable)
- Venus: 8.6"/century
- Earth: 3.8"/century
- Mars: 1.4"/century
- Other planets: < 0.5"/century

---

## Numerical Integration Methods

### The Differential Equations

For each planet, we have:
```
d²r⃗/dt² = a⃗(r⃗, v⃗, t)
```

This is a second-order ODE. We convert to first-order by defining:
```
dr⃗/dt = v⃗
dv⃗/dt = a⃗
```

### Runge-Kutta 4th Order (RK4)

RK4 is a high-accuracy numerical integration method. For a time step Δt:

**Stage 1:**
```
k₁v = a⃗(r⃗ₙ, v⃗ₙ)
k₁r = v⃗ₙ
```

**Stage 2:**
```
k₂v = a⃗(r⃗ₙ + k₁r·Δt/2, v⃗ₙ + k₁v·Δt/2)
k₂r = v⃗ₙ + k₁v·Δt/2
```

**Stage 3:**
```
k₃v = a⃗(r⃗ₙ + k₂r·Δt/2, v⃗ₙ + k₂v·Δt/2)
k₃r = v⃗ₙ + k₂v·Δt/2
```

**Stage 4:**
```
k₄v = a⃗(r⃗ₙ + k₃r·Δt, v⃗ₙ + k₃v·Δt)
k₄r = v⃗ₙ + k₃v·Δt
```

**Weighted average:**
```
r⃗ₙ₊₁ = r⃗ₙ + (Δt/6)·(k₁r + 2k₂r + 2k₃r + k₄r)
v⃗ₙ₊₁ = v⃗ₙ + (Δt/6)·(k₁v + 2k₂v + 2k₃v + k₄v)
```

### Why RK4?

**Error per step:** O(Δt⁵)  
**Global error:** O(Δt⁴)

This is much better than:
- Euler method: O(Δt) global error
- Verlet method: O(Δt²) global error

For orbital mechanics, RK4 provides excellent long-term stability and accuracy.

---

## Observable Effects

### Comparing Scenarios

#### Scenario 1: Classical 2-Body (Disabled Features)
- Only Sun-planet gravity
- No GR corrections
- Perfect elliptical orbits
- Orbits repeat exactly

#### Scenario 2: N-Body Only (GR disabled)
- Sun + planet-planet gravity
- Subtle orbital perturbations
- Near-elliptical orbits with wobbles
- Mercury's perihelion doesn't precess

#### Scenario 3: Full Physics (All enabled)
- N-body gravity
- GR corrections
- Mercury's perihelion precesses
- Most realistic simulation

### Measurable Quantities

**For Mercury over 100 years:**

| Effect | Perihelion Advance | Observable |
|--------|-------------------|-----------|
| Newtonian (Sun only) | 0"/century | No |
| N-body perturbations | ~531"/century | Yes |
| General Relativity | ~43"/century | Yes |
| **Total (observed)** | **~574"/century** | **Yes** ✓ |

**For other planets:**
- Venus: N-body dominates, GR contributes ~8.6"/century
- Earth: N-body effects visible, GR ~3.8"/century
- Outer planets: N-body effects negligible on human timescales

### Running the Experiments

1. **Enable all physics** and fast-forward 100 simulation years
2. **Disable GR** and reset - compare Mercury's orbital evolution
3. **Disable N-body** - see perfectly repeating ellipses
4. **Change Sun mass** to 2× - all effects scale proportionally

---

## Mathematical Summary

**Complete equation of motion for planet i:**

```
mᵢ·(d²r⃗ᵢ/dt²) = F⃗_Sun + Σⱼ≠ᵢ F⃗_planets,j + F⃗_GR

           = -G·mᵢ·M☉/r²·r̂
             + Σⱼ≠ᵢ [-G·mᵢ·mⱼ/rᵢⱼ²·r̂ᵢⱼ]
             + (3G²M☉²mᵢ)/(c²r³L)·(L⃗ × r⃗)
```

Dividing by mass:

```
d²r⃗ᵢ/dt² = -GM☉/r²·r̂
          + Σⱼ≠ᵢ [-G·mⱼ/rᵢⱼ²·r̂ᵢⱼ]
          + (3G²M☉²)/(c²r³L)·(L⃗ × r⃗)
          
          = a⃗_Sun + a⃗_N-body + a⃗_GR
```

This is what the simulator computes at each time step!

---

## References

1. **Orbital Mechanics**: Curtis, H.D. "Orbital Mechanics for Engineering Students"
2. **N-Body Problem**: Murray, C.D. & Dermott, S.F. "Solar System Dynamics"
3. **General Relativity**: Misner, Thorne & Wheeler "Gravitation"
4. **Numerical Methods**: Press, W.H. et al. "Numerical Recipes"
5. **Mercury's Precession**: Einstein, A. (1915) "Explanation of the Perihelion Motion of Mercury"

---

## Verification

The simulation can be verified by:

1. **Energy conservation**: Total energy should remain constant (within numerical error)
2. **Angular momentum conservation**: For 2-body, L should be constant
3. **Orbital periods**: Compare to known values (Kepler's 3rd law)
4. **Mercury precession**: Measure perihelion advance rate
5. **Long-term stability**: Orbits should remain bounded over centuries

Try these experiments in the simulator to verify the physics!