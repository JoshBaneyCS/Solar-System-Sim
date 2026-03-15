package ui

import (
	"math"
	"testing"

	"solar-system-sim/internal/launch"
	"solar-system-sim/internal/math3d"
)

func makeTestTrajectory() *launch.Trajectory {
	return &launch.Trajectory{
		Frame: launch.EarthCentered,
		Points: []launch.TrajectoryPoint{
			{Time: 0, Position: math3d.Vec3{X: 0, Y: 0, Z: 0}, Velocity: math3d.Vec3{X: 1000, Y: 0, Z: 0}},
			{Time: 100, Position: math3d.Vec3{X: 100000, Y: 0, Z: 0}, Velocity: math3d.Vec3{X: 1000, Y: 0, Z: 0}},
			{Time: 200, Position: math3d.Vec3{X: 200000, Y: 0, Z: 0}, Velocity: math3d.Vec3{X: 1000, Y: 0, Z: 0}},
		},
	}
}

func TestNewMissionPlayback(t *testing.T) {
	traj := makeTestTrajectory()
	earthPos := math3d.Vec3{X: 1.496e11, Y: 0, Z: 0}
	mp := NewMissionPlayback(traj, earthPos)

	if mp == nil {
		t.Fatal("expected non-nil playback")
	}
	if mp.TotalTime() != 200 {
		t.Errorf("expected total time 200, got %f", mp.TotalTime())
	}
}

func TestNewMissionPlayback_NilTrajectory(t *testing.T) {
	mp := NewMissionPlayback(nil, math3d.Vec3{})
	if mp != nil {
		t.Error("expected nil for nil trajectory")
	}
}

func TestMissionPlayback_InterpolationAtStart(t *testing.T) {
	traj := makeTestTrajectory()
	mp := NewMissionPlayback(traj, math3d.Vec3{})

	mp.SetTime(0)
	if math.Abs(mp.CurrentPos.X) > 1 {
		t.Errorf("expected X near 0 at start, got %f", mp.CurrentPos.X)
	}
}

func TestMissionPlayback_InterpolationAtMid(t *testing.T) {
	traj := makeTestTrajectory()
	mp := NewMissionPlayback(traj, math3d.Vec3{})

	mp.SetTime(100)
	if math.Abs(mp.CurrentPos.X-100000) > 1000 {
		t.Errorf("expected X near 100000 at midpoint, got %f", mp.CurrentPos.X)
	}
}

func TestMissionPlayback_InterpolationAtEnd(t *testing.T) {
	traj := makeTestTrajectory()
	mp := NewMissionPlayback(traj, math3d.Vec3{})

	mp.SetTime(200)
	if math.Abs(mp.CurrentPos.X-200000) > 1000 {
		t.Errorf("expected X near 200000 at end, got %f", mp.CurrentPos.X)
	}
}

func TestMissionPlayback_PlayAndTick(t *testing.T) {
	traj := makeTestTrajectory()
	mp := NewMissionPlayback(traj, math3d.Vec3{})

	mp.SetSpeed(1.0)
	mp.Play()
	if !mp.IsPlaying() {
		t.Error("expected playing")
	}

	mp.Tick(50)
	if mp.CurrentTimeSeconds() < 49 || mp.CurrentTimeSeconds() > 51 {
		t.Errorf("expected time near 50, got %f", mp.CurrentTimeSeconds())
	}
}

func TestMissionPlayback_Pause(t *testing.T) {
	traj := makeTestTrajectory()
	mp := NewMissionPlayback(traj, math3d.Vec3{})

	mp.Play()
	mp.Pause()
	if mp.IsPlaying() {
		t.Error("expected paused")
	}

	prevTime := mp.CurrentTimeSeconds()
	mp.Tick(1.0)
	if mp.CurrentTimeSeconds() != prevTime {
		t.Error("time should not advance when paused")
	}
}

func TestMissionPlayback_Telemetry(t *testing.T) {
	traj := makeTestTrajectory()
	mp := NewMissionPlayback(traj, math3d.Vec3{})

	mp.SetTime(100)
	telem := mp.CurrentTelemetry()

	if telem.ElapsedTime != 100 {
		t.Errorf("expected elapsed time 100, got %f", telem.ElapsedTime)
	}
	if telem.ProgressPct < 49 || telem.ProgressPct > 51 {
		t.Errorf("expected progress near 50%%, got %f", telem.ProgressPct)
	}
}

func TestMissionPlayback_WorldPosition_EarthCentered(t *testing.T) {
	traj := makeTestTrajectory()
	earthPos := math3d.Vec3{X: 1e11, Y: 0, Z: 0}
	mp := NewMissionPlayback(traj, earthPos)

	mp.SetTime(0)
	worldPos := mp.WorldPosition()
	// Should be earthPos + currentPos
	if math.Abs(worldPos.X-1e11) > 1e6 {
		t.Errorf("expected world X near 1e11, got %e", worldPos.X)
	}
}

func TestMissionPlayback_SetTimeClamping(t *testing.T) {
	traj := makeTestTrajectory()
	mp := NewMissionPlayback(traj, math3d.Vec3{})

	mp.SetTime(-100)
	if mp.CurrentTimeSeconds() != 0 {
		t.Error("expected clamped to 0")
	}

	mp.SetTime(1000)
	if mp.CurrentTimeSeconds() != 200 {
		t.Error("expected clamped to total time")
	}
}
