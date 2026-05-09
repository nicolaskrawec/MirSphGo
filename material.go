package main

import "math"

// =====================
// Material
// =====================

type CheckerPattern struct {
	ColorA Vec3
	ColorB Vec3
	Scale  float64
}

type Material struct {
	Albedo       Vec3
	Reflectivity float64

	Specular  float64
	Shininess float64

	Checker *CheckerPattern
}

func (m *Material) ColorAt(pos Vec3) Vec3 {
	if m == nil {
		return V3(1, 0, 1)
	}

	if m.Checker == nil {
		return m.Albedo
	}

	scale := m.Checker.Scale
	if scale <= 0 {
		scale = 1
	}

	x := int(math.Floor(pos.X * scale))
	z := int(math.Floor(pos.Z * scale))

	if (x+z)%2 == 0 {
		return m.Checker.ColorA
	}

	return m.Checker.ColorB
}
