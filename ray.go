package main

// =====================
// Ray
// =====================

type Ray struct {
	Origin Vec3
	Dir    Vec3
}

func (r Ray) At(t float64) Vec3 {
	return r.Origin.Add(r.Dir.Mul(t))
}

// =====================
// Hit
// =====================

type HitRecord struct {
	T        float64
	Position Vec3
	Normal   Vec3
	Material *Material
}
