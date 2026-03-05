# Launch Planner (Kennedy Space Center)

Simulate a launch from Kennedy Space Center to selectable destinations using patched-conic trajectory modeling and Hohmann transfers.

## Overview

The Launch Planner computes delta-v budgets and generates numerical trajectories for launches from KSC (28.57 N latitude) to various orbital destinations. It uses classical orbital mechanics with the following simplifications:

- **Patched-conics** for multi-body transfers (Earth SOI then heliocentric)
- **Hohmann transfers** for orbit-to-orbit maneuvers (optimal two-impulse)
- **Simplified ascent model** using Tsiolkovsky rocket equation with a constant gravity/drag loss term (~1500 m/s)
- **No atmospheric drag model** in the first implementation

## Formulas

### Tsiolkovsky Rocket Equation
```
dv = Isp * g0 * ln(m0 / mf)
```
Where `Isp` is specific impulse (s), `g0 = 9.80665 m/s^2`, `m0` is initial mass, `mf` is final mass (dry).

### Circular Orbital Velocity
```
v = sqrt(mu / r)
```

### Hohmann Transfer
```
dv1 = sqrt(mu/r1) * (sqrt(2*r2/(r1+r2)) - 1)    [departure burn]
dv2 = sqrt(mu/r2) * (1 - sqrt(2*r1/(r1+r2)))      [arrival burn]
t   = pi * sqrt(a_t^3 / mu)                         [transfer time]
```
Where `a_t = (r1 + r2) / 2` is the transfer orbit semi-major axis.

### Plane Change
```
dv = 2 * v * sin(delta_i / 2)
```
Applied only for Earth-centered targets with inclination differing from KSC latitude.

### Hyperbolic Excess (Interplanetary Departure)
```
v_depart = sqrt(v_inf^2 + 2*mu/r)
dv = |v_depart - v_circular|
```

## Vehicle Presets

| Vehicle | Stages | Total dv | Notes |
|---------|--------|----------|-------|
| Generic | 2 | ~13.2 km/s | Two-stage, Isp 290/340s |
| Falcon-like | 2 | ~9.2 km/s | Based on Falcon 9, Isp 282/348s |
| Saturn V-like | 3 | ~18.3 km/s | Based on Saturn V, Isp 263/421/421s |

## Destinations

| Destination | Altitude | Inclination | Frame | Typical Total dv |
|-------------|----------|-------------|-------|-----------------|
| LEO (200 km) | 200 km | 28.57 deg | Earth-centered | ~8.9 km/s |
| ISS | 408 km | 51.6 deg | Earth-centered | ~12.0 km/s (incl. plane change) |
| GTO | 200 x 35,786 km | 28.57 deg | Earth-centered | ~12.8 km/s |
| Moon (TLI) | 200 km parking | 28.57 deg | Earth-centered | ~12.8 km/s |
| Mars (Hohmann) | 200 km parking | ecliptic | Heliocentric | ~14.6 km/s |

## Delta-V Budget Breakdown

Each launch plan breaks down the total dv into four components:

1. **Ascent** — from KSC surface to parking orbit. Includes orbital velocity, gravity losses (~1500 m/s), minus Earth rotation assist (~407 m/s at KSC).
2. **Plane Change** — inclination adjustment at parking orbit (only for Earth-centered targets with non-KSC inclination, e.g., ISS at 51.6 deg).
3. **Transfer Burn** — Hohmann departure burn or TLI/hyperbolic excess burn.
4. **Arrival Burn** — circularization at target orbit or capture burn.

## Trajectory Propagation

After computing the dv budget, the planner generates a numerical trajectory using RK4 integration:

- **Near-Earth** (LEO/ISS/GTO/Moon): 2-body problem with Earth, 60s timestep
- **Interplanetary** (Mars): 2-body problem with Sun, 3600s timestep
- Outputs ~1000 points for rendering

## CLI Usage

```bash
# Basic LEO launch
go run ./cmd/cli --dest leo --vehicle generic

# Mars transfer with CSV export
go run ./cmd/cli --dest mars --vehicle saturnv --output mars_launch.csv

# List available destinations
go run ./cmd/cli --list-destinations

# List available vehicles
go run ./cmd/cli --list-vehicles
```

## GUI Usage

1. Open the simulator GUI
2. Click the **Launch Planner** tab in the left panel
3. Select a destination and vehicle from the dropdowns
4. Click **Simulate Launch**
5. View the dv budget breakdown in the results area
6. The trajectory renders as a colored overlay on the simulation canvas

## Known Limitations

- Mars assumes optimal planetary alignment (no launch window computation)
- No atmospheric drag model (ascent losses approximated as constant)
- Lunar capture simplified to ~800 m/s constant
- No gravity assists or multi-body perturbations during transfer
- Plane changes computed as simple single-impulse maneuvers

## Future Enhancements

- Lambert solver for arbitrary transfer geometries
- Launch window optimization based on planetary positions
- Atmospheric drag model for ascent phase
- Multi-body propagation during transfer
- Gravity assist trajectories
- Staging timeline visualization
