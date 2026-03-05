# Advanced Customization Guide

This guide explains how to customize and extend the Solar System Simulator.

## Table of Contents

1. [Adding Custom Planets](#adding-custom-planets)
2. [Modifying Trail Colors](#modifying-trail-colors)
3. [Changing Display Scale](#changing-display-scale)
4. [Adding Planet-Planet Interactions](#adding-planet-planet-interactions)
5. [3D Orbital Inclinations](#3d-orbital-inclinations)
6. [Performance Tuning](#performance-tuning)
7. [Export Orbital Data](#export-orbital-data)

## Adding Custom Planets

To add a custom planet or asteroid, add a new entry to the `planetData` slice:

```go
var planetData = []Planet{
    // ... existing planets ...
    {
        Name:           "Custom Planet",
        Mass:           1e24,              // kg
        SemiMajorAxis:  2.5,               // AU
        Eccentricity:   0.1,               // 0-1 (0=circle)
        OrbitalPeriod:  800,               // Earth days
        Color:          color.RGBA{255, 100, 200, 255},
        DisplayRadius:  10,                // pixels
        InitialAnomaly: 0,                 // radians (starting position)
    },
}
```

### Understanding Orbital Elements

- **Semi-major Axis (a)**: Half the longest diameter of the elliptical orbit
    - Measured in AU (Astronomical Units)
    - 1 AU = 149.6 million km (Earth's average distance)

- **Eccentricity (e)**: Shape of the orbit
    - e = 0: Perfect circle
    - 0 < e < 1: Ellipse
    - e = 1: Parabola (escape trajectory)
    - Most planets have e < 0.1 (nearly circular)

- **Initial Anomaly**: Starting position on orbit (in radians)
    - 0 = perihelion (closest to Sun)
    - π = aphelion (farthest from Sun)

## Modifying Trail Colors

### Individual Planet Colors

Change colors in the `planetData` slice:

```go
Color: color.RGBA{R, G, B, A}, // A=255 for fully opaque
```

### Rainbow Trails

To create rainbow trails that change color over time, modify the trail rendering:

```go
// In createCanvas(), modify the trail rendering section:
for j := 0; j < len(planet.Trail)-1; j++ {
    // Calculate color based on position in trail
    hue := float64(j) / float64(len(planet.Trail)) * 360
    r, g, b := hsvToRGB(hue, 1.0, 1.0)
    
    line := canvas.NewLine(color.RGBA{uint8(r), uint8(g), uint8(b), 200})
    // ... rest of line setup
}

// Add this helper function:
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
    c := v * s
    x := c * (1 - math.Abs(math.Mod(h/60.0, 2) - 1))
    m := v - c
    
    var r, g, b float64
    switch {
    case h < 60:  r, g, b = c, x, 0
    case h < 120: r, g, b = x, c, 0
    case h < 180: r, g, b = 0, c, x
    case h < 240: r, g, b = 0, x, c
    case h < 300: r, g, b = x, 0, c
    default:      r, g, b = c, 0, x
    }
    
    return uint8((r+m)*255), uint8((g+m)*255), uint8((b+m)*255)
}
```

## Changing Display Scale

To zoom in/out or change the viewport:

```go
const (
    displayScale = 150.0  // Increase for zoom in, decrease for zoom out
    canvasWidth  = 1200   // Increase for larger window
    canvasHeight = 900
)
```

### Dynamic Zoom

Add zoom buttons:

```go
var zoomLevel = 1.0

zoomInButton := widget.NewButton("Zoom In", func() {
    zoomLevel *= 1.2
})

zoomOutButton := widget.NewButton("Zoom Out", func() {
    zoomLevel /= 1.2
})

// Modify worldToScreen function:
func (app *SolarSystemApp) worldToScreen(pos Vec2) (float32, float32) {
    x := float32(pos.X/AU*displayScale*zoomLevel + canvasWidth/2)
    y := float32(pos.Y/AU*displayScale*zoomLevel + canvasHeight/2)
    return x, y
}
```

## Adding Planet-Planet Interactions

Currently, only the Sun's gravity affects planets. To add planet-planet gravitational interactions:

```go
func (s *Simulator) calculateAcceleration(body *Body) Vec2 {
    totalAccel := Vec2{0, 0}
    
    // Sun's gravity
    rSun := s.Sun.Position.Sub(body.Position)
    distSun := rSun.Magnitude()
    if distSun > 1e6 {
        rHatSun := rSun.Normalize()
        accelSun := G * s.SunMass / (distSun * distSun)
        totalAccel = totalAccel.Add(rHatSun.Mul(accelSun))
    }
    
    // Other planets' gravity
    for i := range s.Planets {
        other := &s.Planets[i]
        if other == body {
            continue
        }
        
        rPlanet := other.Position.Sub(body.Position)
        distPlanet := rPlanet.Magnitude()
        
        if distPlanet > 1e6 {
            rHatPlanet := rPlanet.Normalize()
            accelPlanet := G * other.Mass / (distPlanet * distPlanet)
            totalAccel = totalAccel.Add(rHatPlanet.Mul(accelPlanet))
        }
    }
    
    return totalAccel
}
```

**Note**: This will make the simulation more computationally expensive (O(n²) instead of O(n)).

## 3D Orbital Inclinations

To add realistic 3D orbits with inclination:

### 1. Extend Vec2 to Vec3

```go
type Vec3 struct {
    X, Y, Z float64
}

// Add all Vec2 operations for Vec3
```

### 2. Add Orbital Inclination to Planet Data

```go
type Planet struct {
    // ... existing fields ...
    Inclination float64 // degrees
    LongitudeOfAscendingNode float64 // degrees
}
```

### 3. Modify Initial Position Calculation

```go
func (s *Simulator) createPlanetFromElements(p Planet) Body {
    // ... calculate r, nu as before ...
    
    // Convert angles to radians
    i := p.Inclination * math.Pi / 180
    omega := p.LongitudeOfAscendingNode * math.Pi / 180
    
    // Position in orbital plane
    xOrbital := r * math.Cos(nu)
    yOrbital := r * math.Sin(nu)
    zOrbital := 0.0
    
    // Rotate to 3D space
    x := (math.Cos(omega)*math.Cos(nu) - math.Sin(omega)*math.Sin(nu)*math.Cos(i)) * r
    y := (math.Sin(omega)*math.Cos(nu) + math.Cos(omega)*math.Sin(nu)*math.Cos(i)) * r
    z := math.Sin(nu) * math.Sin(i) * r
    
    // ... similar for velocity ...
}
```

### 4. Project 3D to 2D for Display

```go
func (app *SolarSystemApp) worldToScreen(pos Vec3) (float32, float32) {
    // Orthographic projection (ignore Z for top-down view)
    // Or isometric: x' = x - z*0.5, y' = y - z*0.5
    x := float32(pos.X/AU*displayScale + canvasWidth/2)
    y := float32(pos.Y/AU*displayScale + canvasHeight/2)
    return x, y
}
```

## Performance Tuning

### Adjust Time Step

Smaller time steps = more accurate but slower:

```go
const baseTimeStep = 3600.0 * 24 // Current: 1 day
const baseTimeStep = 3600.0      // Try: 1 hour (more accurate)
```

### Trail Length

Reduce trail length for better performance:

```go
maxTrailLen: 200, // Default is 500
```

### Frame Rate

Adjust the animation ticker:

```go
ticker := time.NewTicker(16 * time.Millisecond)  // 60 FPS
ticker := time.NewTicker(33 * time.Millisecond)  // 30 FPS (lighter)
```

### Conditional Trail Updates

Only update trails every N frames:

```go
var frameCount int

func (s *Simulator) step(dt float64) {
    frameCount++
    
    for i := range s.Planets {
        // ... physics updates ...
        
        // Only add trail every 5 frames
        if s.ShowTrails && frameCount%5 == 0 {
            planet.Trail = append(planet.Trail, planet.Position)
            // ... trail management ...
        }
    }
}
```

## Export Orbital Data

Add functionality to export position data:

```go
import (
    "encoding/csv"
    "os"
)

func (s *Simulator) ExportData(filename string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    writer := csv.NewWriter(file)
    defer writer.Flush()
    
    // Header
    header := []string{"Time", "Planet", "X", "Y", "VX", "VY", "Distance", "Velocity"}
    writer.Write(header)
    
    // Data for each planet
    for _, planet := range s.Planets {
        r := planet.Position.Magnitude()
        v := planet.Velocity.Magnitude()
        
        row := []string{
            fmt.Sprintf("%.2f", s.CurrentTime),
            planet.Name,
            fmt.Sprintf("%.6e", planet.Position.X),
            fmt.Sprintf("%.6e", planet.Position.Y),
            fmt.Sprintf("%.6e", planet.Velocity.X),
            fmt.Sprintf("%.6e", planet.Velocity.Y),
            fmt.Sprintf("%.6e", r),
            fmt.Sprintf("%.6e", v),
        }
        writer.Write(row)
    }
    
    return nil
}

// Add button to UI:
exportButton := widget.NewButton("Export Data", func() {
    err := app.simulator.ExportData("orbital_data.csv")
    if err != nil {
        fmt.Println("Error exporting:", err)
    } else {
        fmt.Println("Data exported to orbital_data.csv")
    }
})
```

## Advanced Physics Displays

### Energy Conservation

Add total energy calculation:

```go
func (s *Simulator) calculateTotalEnergy() float64 {
    totalEnergy := 0.0
    
    for i := range s.Planets {
        planet := &s.Planets[i]
        
        // Kinetic energy: (1/2)mv²
        v := planet.Velocity.Magnitude()
        ke := 0.5 * planet.Mass * v * v
        
        // Potential energy: -GMm/r
        r := planet.Position.Magnitude()
        pe := -G * s.SunMass * planet.Mass / r
        
        totalEnergy += ke + pe
    }
    
    return totalEnergy
}
```

### Angular Momentum

```go
func (s *Simulator) calculateAngularMomentum(planet *Body) float64 {
    // L = r × mv
    r := planet.Position
    v := planet.Velocity
    
    // In 2D: L = m(x*vy - y*vx)
    L := planet.Mass * (r.X*v.Y - r.Y*v.X)
    
    return L
}
```

## Custom Scenarios

### Binary Star System

```go
func NewBinaryStarSimulator() *Simulator {
    star1Mass := 1.989e30
    star2Mass := 1.5e30
    separation := 2 * AU
    
    sim := &Simulator{
        Sun: Body{
            Name:     "Star 1",
            Mass:     star1Mass,
            Position: Vec2{-separation/2, 0},
            Velocity: Vec2{0, 20000}, // m/s
            // ...
        },
        // Add Star 2 as a "planet" or create separate stars list
    }
    
    return sim
}
```

### Asteroid Belt

```go
func addAsteroidBelt(sim *Simulator, count int) {
    for i := 0; i < count; i++ {
        a := 2.2 + rand.Float64()*1.0 // 2.2-3.2 AU
        e := rand.Float64() * 0.2     // low eccentricity
        angle := rand.Float64() * 2 * math.Pi
        
        asteroid := Planet{
            Name:           fmt.Sprintf("Asteroid_%d", i),
            Mass:           1e15, // small mass
            SemiMajorAxis:  a,
            Eccentricity:   e,
            OrbitalPeriod:  365 * math.Sqrt(a*a*a), // Kepler's 3rd law
            Color:          color.RGBA{128, 128, 128, 100},
            DisplayRadius:  1,
            InitialAnomaly: angle,
        }
        
        sim.Planets = append(sim.Planets, sim.createPlanetFromElements(asteroid))
    }
}
```

## Troubleshooting

### Planets Flying Off

- Check initial velocity calculations
- Ensure time step isn't too large
- Verify G constant and masses are in correct units

### Jittery Motion

- Use smaller time steps
- Check RK4 implementation
- Ensure floating-point precision is adequate

### Performance Issues

- Reduce trail length
- Decrease frame rate
- Simplify physics (use Euler integration instead of RK4)
- Don't add planet-planet interactions unless necessary

## Further Reading

- **Kepler's Laws**: https://en.wikipedia.org/wiki/Kepler%27s_laws_of_planetary_motion
- **Orbital Mechanics**: Curtis, H.D. "Orbital Mechanics for Engineering Students"
- **Numerical Integration**: Press, W.H. et al. "Numerical Recipes"
- **N-Body Problem**: https://en.wikipedia.org/wiki/N-body_problem

## Contributing

Feel free to extend this simulator! Some ideas:
- Add moons for planets
- Implement relativistic corrections
- Add comets with highly eccentric orbits
- Create a solar system "time machine" to view past/future positions
- Add collision detection
- Implement proper 3D rendering with perspective