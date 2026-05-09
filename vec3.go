package main

import (
	"image/color"
	"math"
)

// =====================
// Vec3
// =====================

type Vec3 struct {
	X, Y, Z float64
}

func V3(x, y, z float64) Vec3 {
	return Vec3{x, y, z}
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z}
}

func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z}
}

func (v Vec3) Mul(s float64) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

func (v Vec3) Hadamard(o Vec3) Vec3 {
	return Vec3{v.X * o.X, v.Y * o.Y, v.Z * o.Z}
}

func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return v.Mul(1 / l)
}

func (v Vec3) Reflect(normal Vec3) Vec3 {
	return v.Sub(normal.Mul(2 * v.Dot(normal)))
}

func Lerp(a, b Vec3, t float64) Vec3 {
	return a.Mul(1 - t).Add(b.Mul(t))
}

func Clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func ToRGBA(c Vec3) color.RGBA {
	return color.RGBA{
		R: uint8(Clamp01(c.X) * 255),
		G: uint8(Clamp01(c.Y) * 255),
		B: uint8(Clamp01(c.Z) * 255),
		A: 255,
	}
}
