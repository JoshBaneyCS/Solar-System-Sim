package main

import (
	"math"
	"testing"
)

const testEpsilon = 1e-9

func assertFloat64Near(t *testing.T, got, want, tol float64, msg string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %e, want %e (diff %e, tol %e)", msg, got, want, math.Abs(got-want), tol)
	}
}

func assertVec3Near(t *testing.T, got, want Vec3, tol float64) {
	t.Helper()
	assertFloat64Near(t, got.X, want.X, tol, "X")
	assertFloat64Near(t, got.Y, want.Y, tol, "Y")
	assertFloat64Near(t, got.Z, want.Z, tol, "Z")
}

func assertRelativeError(t *testing.T, got, want, tol float64, msg string) {
	t.Helper()
	if want == 0 {
		assertFloat64Near(t, got, want, tol, msg)
		return
	}
	relErr := math.Abs((got - want) / want)
	if relErr > tol {
		t.Errorf("%s: got %e, want %e (relative error %e, tol %e)", msg, got, want, relErr, tol)
	}
}
