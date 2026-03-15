#[derive(Clone, Copy, Default, Debug, PartialEq)]
pub struct Vec3 {
    pub x: f64,
    pub y: f64,
    pub z: f64,
}

impl Vec3 {
    #[inline]
    pub fn new(x: f64, y: f64, z: f64) -> Self {
        Self { x, y, z }
    }

    #[inline]
    pub fn add(self, other: Self) -> Self {
        Self {
            x: self.x + other.x,
            y: self.y + other.y,
            z: self.z + other.z,
        }
    }

    #[inline]
    pub fn sub(self, other: Self) -> Self {
        Self {
            x: self.x - other.x,
            y: self.y - other.y,
            z: self.z - other.z,
        }
    }

    #[inline]
    pub fn mul(self, scalar: f64) -> Self {
        Self {
            x: self.x * scalar,
            y: self.y * scalar,
            z: self.z * scalar,
        }
    }

    #[inline]
    pub fn magnitude(self) -> f64 {
        (self.x * self.x + self.y * self.y + self.z * self.z).sqrt()
    }

    #[inline]
    pub fn magnitude_sq(self) -> f64 {
        self.x * self.x + self.y * self.y + self.z * self.z
    }

    #[inline]
    pub fn normalize(self) -> Self {
        let mag = self.magnitude();
        if mag == 0.0 {
            return Self::default();
        }
        self.mul(1.0 / mag)
    }

    #[inline]
    pub fn dot(self, other: Self) -> f64 {
        self.x * other.x + self.y * other.y + self.z * other.z
    }

    #[inline]
    pub fn cross(self, other: Self) -> Self {
        Self {
            x: self.y * other.z - self.z * other.y,
            y: self.z * other.x - self.x * other.z,
            z: self.x * other.y - self.y * other.x,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn assert_near(a: f64, b: f64, tol: f64) {
        assert!(
            (a - b).abs() < tol,
            "expected {} to be near {} (tol {})",
            a,
            b,
            tol
        );
    }

    fn assert_vec3_near(a: Vec3, b: Vec3, tol: f64) {
        assert_near(a.x, b.x, tol);
        assert_near(a.y, b.y, tol);
        assert_near(a.z, b.z, tol);
    }

    #[test]
    fn test_add() {
        let a = Vec3::new(1.0, 2.0, 3.0);
        let b = Vec3::new(4.0, 5.0, 6.0);
        assert_eq!(a.add(b), Vec3::new(5.0, 7.0, 9.0));
    }

    #[test]
    fn test_sub() {
        let a = Vec3::new(4.0, 5.0, 6.0);
        let b = Vec3::new(1.0, 2.0, 3.0);
        assert_eq!(a.sub(b), Vec3::new(3.0, 3.0, 3.0));
    }

    #[test]
    fn test_mul() {
        let a = Vec3::new(1.0, 2.0, 3.0);
        assert_eq!(a.mul(2.0), Vec3::new(2.0, 4.0, 6.0));
    }

    #[test]
    fn test_magnitude() {
        let a = Vec3::new(3.0, 4.0, 0.0);
        assert_near(a.magnitude(), 5.0, 1e-15);
    }

    #[test]
    fn test_normalize() {
        let a = Vec3::new(3.0, 4.0, 0.0);
        let n = a.normalize();
        assert_near(n.magnitude(), 1.0, 1e-15);
        assert_vec3_near(n, Vec3::new(0.6, 0.8, 0.0), 1e-15);
    }

    #[test]
    fn test_normalize_zero() {
        let a = Vec3::default();
        assert_eq!(a.normalize(), Vec3::default());
    }

    #[test]
    fn test_dot() {
        let a = Vec3::new(1.0, 2.0, 3.0);
        let b = Vec3::new(4.0, 5.0, 6.0);
        assert_near(a.dot(b), 32.0, 1e-15);
    }

    #[test]
    fn test_cross() {
        let a = Vec3::new(1.0, 0.0, 0.0);
        let b = Vec3::new(0.0, 1.0, 0.0);
        assert_eq!(a.cross(b), Vec3::new(0.0, 0.0, 1.0));
    }

    #[test]
    fn test_cross_anti_commutativity() {
        let a = Vec3::new(1.0, 2.0, 3.0);
        let b = Vec3::new(4.0, 5.0, 6.0);
        let ab = a.cross(b);
        let ba = b.cross(a);
        assert_vec3_near(ab, ba.mul(-1.0), 1e-15);
    }
}
