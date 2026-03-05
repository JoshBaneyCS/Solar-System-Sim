# Solar System Simulator CLI

The unified CLI (`solar-sim`) provides headless simulation, physics validation,
launch planning, and optional GUI access from a single binary.

## Build

```bash
# With GUI support (requires graphics libraries)
make build-solar-sim

# Headless only (no graphics dependencies)
make build-solar-sim-headless
```

## Commands

### `solar-sim gui`

Launch the graphical user interface.

```bash
solar-sim gui
```

Not available in headless builds (`-tags nogui`).

---

### `solar-sim run`

Run a headless N-body simulation and export planet ephemeris.

```bash
solar-sim run --years 10 --export ephemeris.csv
solar-sim run --years 1 --dt 3600 --format json --export out.json
solar-sim run --years 100 --sample-interval 100 --export century.csv
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--years` | 1.0 | Simulation duration in years |
| `--dt` | 7200 | Integration timestep in seconds |
| `--export` | (required) | Output file path |
| `--format` | csv | Output format: `csv` or `json` |
| `--integrator` | verlet | Integrator: `verlet` or `rk4` |
| `--sample-interval` | 1 | Record every Nth step |

**CSV columns:**
```
time_s, body, pos_x_m, pos_y_m, pos_z_m, vel_x_ms, vel_y_ms, vel_z_ms, distance_from_sun_m
```

**JSON schema:**
```json
{
  "config": {"years": 1, "dt": 7200, "integrator": "verlet"},
  "snapshots": [
    {
      "time_s": 0,
      "bodies": [
        {"name": "Mercury", "pos": [x, y, z], "vel": [vx, vy, vz], "distance_au": 0.387}
      ]
    }
  ]
}
```

**Performance notes:** For long simulations, use `--sample-interval` to control
output size. A 100-year run at dt=7200s produces ~438,000 steps. With 8 planets,
that's ~3.5M CSV rows. Use `--sample-interval 100` to reduce to ~35K rows.
JSON buffers all data in memory; for very large runs, prefer CSV.

Progress is reported to stderr every 10%.

---

### `solar-sim validate`

Run physics validation scenarios to verify simulation accuracy.

```bash
solar-sim validate                                    # all scenarios, 10 years
solar-sim validate --scenario mercury-precession --years 100
solar-sim validate --scenario energy --years 5
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--scenario` | all | Scenario name or `all` |
| `--years` | 10 | Simulation duration |

**Available scenarios:**
- `energy` — Energy conservation (relative drift)
- `angular-momentum` — Angular momentum conservation
- `kepler-earth` — Earth orbital period vs 365.25 days
- `kepler-mercury` — Mercury orbital period vs 87.97 days
- `mercury-precession` — GR perihelion precession vs 43 arcsec/century

Exit code 1 if any scenario fails.

---

### `solar-sim launch`

Compute launch delta-v budget and optionally export trajectory.

```bash
solar-sim launch --dest mars --vehicle saturnv
solar-sim launch --dest moon --vehicle falcon --export moon.csv
solar-sim launch --list-destinations
solar-sim launch --list-vehicles
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--dest` | leo | Target: `leo`, `iss`, `gto`, `moon`, `mars` |
| `--vehicle` | generic | Vehicle: `generic`, `falcon`, `saturnv` |
| `--export` | | CSV trajectory output file |
| `--list-destinations` | | Show available destinations |
| `--list-vehicles` | | Show available vehicles with delta-v |

---

### `solar-sim assets verify`

Validate the asset directory structure (textures, models, meshes).

```bash
solar-sim assets verify
solar-sim assets verify --dir /path/to/assets
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | assets | Asset directory path |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error or validation failure |

## Cross-Platform

Works on macOS, Linux, and Windows. The headless build (`-tags nogui`) has no
graphics dependencies and runs on CI servers without display.
