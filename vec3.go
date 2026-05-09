package main

import (
	"image/color"
	"math"
)

// Vec3 représente un vecteur à 3 dimensions (X, Y, Z).
// Utilisé pour les positions, les directions et les couleurs.
type Vec3 struct {
	X, Y, Z float64
}

// V3 est un helper pour créer un nouveau Vec3.
func V3(x, y, z float64) Vec3 {
	return Vec3{x, y, z}
}

// Add additionne deux vecteurs.
func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z}
}

// Sub soustrait le vecteur 'o' du vecteur 'v'.
func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z}
}

// Mul multiplie un vecteur par un scalaire.
func (v Vec3) Mul(s float64) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

// Hadamard effectue un produit de Hadamard (multiplication composante par composante).
// Très utile pour mélanger des couleurs.
func (v Vec3) Hadamard(o Vec3) Vec3 {
	return Vec3{v.X * o.X, v.Y * o.Y, v.Z * o.Z}
}

// Dot calcule le produit scalaire entre deux vecteurs.
func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

// Length retourne la longueur (norme) du vecteur.
func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

// Normalize retourne un vecteur de même direction mais de longueur 1.
func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return v.Mul(1 / l)
}

// Reflect calcule le vecteur de réflexion par rapport à une normale.
func (v Vec3) Reflect(normal Vec3) Vec3 {
	return v.Sub(normal.Mul(2 * v.Dot(normal)))
}

// Refract calcule le vecteur de réfraction selon la loi de Snell-Descartes.
func (v Vec3) Refract(n Vec3, etaiOverEtat float64) (Vec3, bool) {
	cosTheta := math.Min(v.Mul(-1).Dot(n), 1.0)
	rOutPerp := v.Add(n.Mul(cosTheta)).Mul(etaiOverEtat)
	discriminant := 1.0 - rOutPerp.Dot(rOutPerp)
	if discriminant < 0 {
		return Vec3{}, false // Réflexion totale interne
	}
	rOutParallel := n.Mul(-math.Sqrt(math.Abs(discriminant)))
	return rOutPerp.Add(rOutParallel), true
}

// Schlick calcule l'approximation de Schlick pour le coefficient de réflexion.
func Schlick(cosine, refIdx float64) float64 {
	r0 := (1 - refIdx) / (1 + refIdx)
	r0 = r0 * r0
	return r0 + (1-r0)*math.Pow((1-cosine), 5)
}

// Lerp effectue une interpolation linéaire entre les vecteurs 'a' et 'b'.
func Lerp(a, b Vec3, t float64) Vec3 {
	return a.Mul(1 - t).Add(b.Mul(t))
}

// Clamp01 restreint une valeur dans l'intervalle [0, 1].
func Clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// ToRGBA convertit un Vec3 (valeurs entre 0 et 1) en color.RGBA.
func ToRGBA(c Vec3) color.RGBA {
	return color.RGBA{
		R: uint8(Clamp01(c.X) * 255),
		G: uint8(Clamp01(c.Y) * 255),
		B: uint8(Clamp01(c.Z) * 255),
		A: 255,
	}
}
