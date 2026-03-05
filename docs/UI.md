# UI Reference

## Layout

```
+---------------------------------------------------------------+
| File | View | Simulation | Settings | About                   |
+---------------+-----------------------------------------------+
| [Simulation]  |                                               |
| [Launch Plan] |          3D Canvas / Renderer                 |
| [Bodies]      |                                               |
|               |                                               |
| Left panel    |                                               |
| (tabbed)      |                                               |
+---------------+-----------------------------------------------+
```

The window opens at 1280x800 with three regions:

- **Left panel** (300px) — tabbed container with Simulation controls, Launch Planner, and Bodies manager
- **Center** — the main 3D rendering canvas
- **Menu bar** — File, View, Simulation, Settings, About

## Menu Bar

### File
| Item              | Action                                         |
|-------------------|------------------------------------------------|
| Export Screenshot  | Captures canvas to PNG via file-save dialog    |
| Quit              | Exits the application                          |

### View
| Item              | Action                                         |
|-------------------|------------------------------------------------|
| Toggle Trails     | Show/hide orbital trails (checkmark)           |
| Toggle Spacetime  | Show/hide spacetime curvature grid (checkmark) |
| Toggle Labels     | Show/hide planet name labels (checkmark)       |
| Maximize          | Maximize window                                |
| Fullscreen        | Toggle fullscreen mode                         |
| Reset Size        | Restore default 1280x800                       |

### Simulation
| Item              | Action                                         |
|-------------------|------------------------------------------------|
| Play / Pause      | Toggle simulation playback                     |
| Reset             | Reset all bodies to initial conditions          |
| Integrator: Verlet| Select Velocity Verlet (checkmark, default)    |
| Integrator: RK4   | Select Runge-Kutta 4th order (checkmark)       |

### Settings
Opens a dialog to configure GPU mode, ray tracing, quality preset, integrator, display toggles (trails, spacetime, labels), and physics toggles (planet-planet gravity, GR corrections). All settings persist across sessions via Fyne Preferences.

### About
Opens a window showing author, repository link, credits, and license.

## Left Panel Tabs

### Simulation Tab
- **Time speed** slider (1x to 100,000x)
- **Play/Pause** and **Reset** buttons
- **Integrator** dropdown (Verlet / RK4)
- **Trails** and **Spacetime** checkboxes
- **Planet-planet gravity** and **Relativity** toggles

### Launch Planner Tab
- **Destination** selector (LEO, ISS, GTO, Moon, Mars)
- **Vehicle** selector (Generic, Falcon, Saturn V)
- **Plan Launch** button with mission summary output

### Bodies Tab
Scrollable list of all celestial bodies (Sun + 8 planets). Each card shows:
- **Name** (bold)
- **Mass** (kg)
- **Distance from Sun** (AU, live-updated)
- **Velocity** (km/s, live-updated)
- **Orbital period** estimate (days, live-updated)
- **Show Trail** checkbox — per-body trail toggle
- **Follow** button — centers viewport on the body

Live stats update every 500ms while the simulation runs.

## Settings Persistence

Settings are stored using the Fyne Preferences API, which uses platform-native storage:
- **macOS**: `~/Library/Preferences/com.joshbaney.solar-sim.plist`
- **Linux**: `~/.config/fyne/com.joshbaney.solar-sim/`
- **Windows**: Registry under `HKCU\Software\com.joshbaney.solar-sim`

Settings are loaded on startup and applied to the simulator and renderer. Changes made via the Settings dialog or menu toggles are saved immediately.
