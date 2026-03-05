package main

import (
	"math"
	"testing"
)

func TestVec3Add(t *testing.T) {
	result := Vec3{1, 2, 3}.Add(Vec3{4, 5, 6})
	assertVec3Near(t, result, Vec3{5, 7, 9}, testEpsilon)
}

func TestVec3Sub(t *testing.T) {
	result := Vec3{5, 7, 9}.Sub(Vec3{4, 5, 6})
	assertVec3Near(t, result, Vec3{1, 2, 3}, testEpsilon)
}

func TestVec3Mul(t *testing.T) {
	result := Vec3{1, 2, 3}.Mul(2)
	assertVec3Near(t, result, Vec3{2, 4, 6}, testEpsilon)
}

func TestVec3MulZero(t *testing.T) {
	result := Vec3{1, 2, 3}.Mul(0)
	assertVec3Near(t, result, Vec3{0, 0, 0}, testEpsilon)
}

func TestVec3Magnitude(t *testing.T) {
	assertFloat64Near(t, Vec3{3, 4, 0}.Magnitude(), 5.0, testEpsilon, "3-4-5 triangle")
	assertFloat64Near(t, Vec3{0, 0, 0}.Magnitude(), 0.0, testEpsilon, "zero vector")
	assertFloat64Near(t, Vec3{1, 1, 1}.Magnitude(), math.Sqrt(3), testEpsilon, "unit diagonal")
}

func TestVec3Normalize(t *testing.T) {
	result := Vec3{3, 0, 0}.Normalize()
	assertVec3Near(t, result, Vec3{1, 0, 0}, testEpsilon)

	result = Vec3{0, 5, 0}.Normalize()
	assertVec3Near(t, result, Vec3{0, 1, 0}, testEpsilon)

	// Normalized vector should have magnitude 1
	v := Vec3{3, 4, 5}.Normalize()
	assertFloat64Near(t, v.Magnitude(), 1.0, testEpsilon, "normalized magnitude")
}

func TestVec3NormalizeZero(t *testing.T) {
	result := Vec3{0, 0, 0}.Normalize()
	assertVec3Near(t, result, Vec3{0, 0, 0}, testEpsilon)
}

func TestVec3Dot(t *testing.T) {
	// Orthogonal vectors
	assertFloat64Near(t, Vec3{1, 0, 0}.Dot(Vec3{0, 1, 0}), 0, testEpsilon, "orthogonal")
	// Parallel vectors
	assertFloat64Near(t, Vec3{1, 0, 0}.Dot(Vec3{3, 0, 0}), 3, testEpsilon, "parallel")
	// General case
	assertFloat64Near(t, Vec3{1, 2, 3}.Dot(Vec3{4, 5, 6}), 32, testEpsilon, "general")
}

func TestVec3Cross(t *testing.T) {
	// Right-hand rule: x × y = z
	result := Vec3{1, 0, 0}.Cross(Vec3{0, 1, 0})
	assertVec3Near(t, result, Vec3{0, 0, 1}, testEpsilon)

	// y × x = -z
	result = Vec3{0, 1, 0}.Cross(Vec3{1, 0, 0})
	assertVec3Near(t, result, Vec3{0, 0, -1}, testEpsilon)

	// Parallel vectors: cross product = zero
	result = Vec3{2, 4, 6}.Cross(Vec3{1, 2, 3})
	assertVec3Near(t, result, Vec3{0, 0, 0}, testEpsilon)
}

func TestVec3CrossAnticommutative(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}
	ab := a.Cross(b)
	ba := b.Cross(a)
	// a × b = -(b × a)
	assertVec3Near(t, ab, ba.Mul(-1), testEpsilon)
}

func TestVec3Operations_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		op   func() Vec3
		want Vec3
	}{
		{"add negatives", func() Vec3 { return Vec3{-1, -2, -3}.Add(Vec3{1, 2, 3}) }, Vec3{0, 0, 0}},
		{"sub self", func() Vec3 { return Vec3{5, 5, 5}.Sub(Vec3{5, 5, 5}) }, Vec3{0, 0, 0}},
		{"mul negative", func() Vec3 { return Vec3{1, 2, 3}.Mul(-1) }, Vec3{-1, -2, -3}},
		{"cross z×x=y", func() Vec3 { return Vec3{0, 0, 1}.Cross(Vec3{1, 0, 0}) }, Vec3{0, 1, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertVec3Near(t, tt.op(), tt.want, testEpsilon)
		})
	}
}
