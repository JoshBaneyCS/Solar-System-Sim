package ui

import (
	"math"

	"solar-system-sim/internal/launch"
	"solar-system-sim/internal/math3d"
)

// Telemetry holds the current mission telemetry data.
type Telemetry struct {
	ElapsedTime    float64 // seconds
	Speed          float64 // m/s
	DistFromEarth  float64 // m
	DistFromTarget float64 // m (estimated)
	ProgressPct    float64 // 0-100
}

// MissionPlayback manages animated playback of a launch trajectory.
type MissionPlayback struct {
	trajectory  *launch.Trajectory
	earthPos    math3d.Vec3 // Earth position at launch time
	currentTime float64     // seconds into mission
	playSpeed   float64     // multiplier (1.0 = realtime, 100.0 = 100x)
	isPlaying   bool
	totalTime   float64 // total mission duration

	// Interpolated current state
	CurrentPos math3d.Vec3
	CurrentVel math3d.Vec3
}

// NewMissionPlayback creates a playback controller for a trajectory.
func NewMissionPlayback(traj *launch.Trajectory, earthPos math3d.Vec3) *MissionPlayback {
	if traj == nil || len(traj.Points) < 2 {
		return nil
	}
	totalTime := traj.Points[len(traj.Points)-1].Time - traj.Points[0].Time
	mp := &MissionPlayback{
		trajectory:  traj,
		earthPos:    earthPos,
		playSpeed:   100.0,
		isPlaying:   false,
		totalTime:   totalTime,
		currentTime: 0,
	}
	mp.interpolate()
	return mp
}

// Tick advances the playback by dt real seconds.
func (mp *MissionPlayback) Tick(dt float64) {
	if !mp.isPlaying || mp.trajectory == nil {
		return
	}
	mp.currentTime += dt * mp.playSpeed
	if mp.currentTime > mp.totalTime {
		mp.currentTime = mp.totalTime
		mp.isPlaying = false
	}
	if mp.currentTime < 0 {
		mp.currentTime = 0
		mp.isPlaying = false
	}
	mp.interpolate()
}

// SetTime sets the playback position directly (for scrubbing).
func (mp *MissionPlayback) SetTime(t float64) {
	if t < 0 {
		t = 0
	}
	if t > mp.totalTime {
		t = mp.totalTime
	}
	mp.currentTime = t
	mp.interpolate()
}

// SetSpeed sets the playback speed multiplier.
func (mp *MissionPlayback) SetSpeed(speed float64) {
	mp.playSpeed = speed
}

// Play starts or resumes playback.
func (mp *MissionPlayback) Play() {
	mp.isPlaying = true
}

// Pause pauses playback.
func (mp *MissionPlayback) Pause() {
	mp.isPlaying = false
}

// IsPlaying returns whether playback is active.
func (mp *MissionPlayback) IsPlaying() bool {
	return mp.isPlaying
}

// TotalTime returns the total mission duration in seconds.
func (mp *MissionPlayback) TotalTime() float64 {
	return mp.totalTime
}

// CurrentTimeSeconds returns the current playback time.
func (mp *MissionPlayback) CurrentTimeSeconds() float64 {
	return mp.currentTime
}

// CurrentTelemetry computes the current telemetry.
func (mp *MissionPlayback) CurrentTelemetry() Telemetry {
	dist := mp.CurrentPos.Magnitude()
	speed := mp.CurrentVel.Magnitude()
	progress := 0.0
	if mp.totalTime > 0 {
		progress = mp.currentTime / mp.totalTime * 100
	}
	return Telemetry{
		ElapsedTime:   mp.currentTime,
		Speed:         speed,
		DistFromEarth: dist,
		ProgressPct:   progress,
	}
}

// WorldPosition returns the current position in heliocentric coordinates.
func (mp *MissionPlayback) WorldPosition() math3d.Vec3 {
	if mp.trajectory.Frame == launch.EarthCentered {
		return mp.CurrentPos.Add(mp.earthPos)
	}
	return mp.CurrentPos
}

// interpolate computes position/velocity at currentTime via linear interpolation.
func (mp *MissionPlayback) interpolate() {
	pts := mp.trajectory.Points
	if len(pts) < 2 {
		return
	}

	startTime := pts[0].Time
	t := startTime + mp.currentTime

	// Binary search for the bracketing segment
	lo, hi := 0, len(pts)-1
	for lo < hi-1 {
		mid := (lo + hi) / 2
		if pts[mid].Time <= t {
			lo = mid
		} else {
			hi = mid
		}
	}

	if lo == hi || pts[hi].Time == pts[lo].Time {
		mp.CurrentPos = pts[lo].Position
		mp.CurrentVel = pts[lo].Velocity
		return
	}

	// Linear interpolation
	frac := (t - pts[lo].Time) / (pts[hi].Time - pts[lo].Time)
	frac = math.Max(0, math.Min(1, frac))

	mp.CurrentPos = pts[lo].Position.Add(pts[hi].Position.Sub(pts[lo].Position).Mul(frac))
	mp.CurrentVel = pts[lo].Velocity.Add(pts[hi].Velocity.Sub(pts[lo].Velocity).Mul(frac))
}
