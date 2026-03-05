package math3d

// CatmullRom computes a point on a Catmull-Rom spline defined by four
// control points p0..p3 at parameter t ∈ [0,1]. The curve passes through
// p1 (at t=0) and p2 (at t=1).
func CatmullRom(p0, p1, p2, p3 Vec3, t float64) Vec3 {
	t2 := t * t
	t3 := t2 * t
	return Vec3{
		X: 0.5 * ((2 * p1.X) + (-p0.X+p2.X)*t + (2*p0.X-5*p1.X+4*p2.X-p3.X)*t2 + (-p0.X+3*p1.X-3*p2.X+p3.X)*t3),
		Y: 0.5 * ((2 * p1.Y) + (-p0.Y+p2.Y)*t + (2*p0.Y-5*p1.Y+4*p2.Y-p3.Y)*t2 + (-p0.Y+3*p1.Y-3*p2.Y+p3.Y)*t3),
		Z: 0.5 * ((2 * p1.Z) + (-p0.Z+p2.Z)*t + (2*p0.Z-5*p1.Z+4*p2.Z-p3.Z)*t2 + (-p0.Z+3*p1.Z-3*p2.Z+p3.Z)*t3),
	}
}
