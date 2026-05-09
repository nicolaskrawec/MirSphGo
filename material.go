package main

import "math"

// CheckerPattern définit un motif en damier avec deux couleurs et une échelle.
type CheckerPattern struct {
	ColorA Vec3
	ColorB Vec3
	Scale  float64
}

// Material définit les propriétés physiques d'une surface.
type Material struct {
	Albedo       Vec3    // Couleur de base (diffusion)
	Reflectivity float64 // Coefficient de réflexion (0 à 1)

	Specular  float64 // Intensité du reflet spéculaire
	Shininess float64 // Brillance (plus c'est élevé, plus le reflet est petit)

	Checker *CheckerPattern // Optionnel : motif en damier
}

// ColorAt retourne la couleur du matériau à une position donnée.
// Gère le motif en damier si présent.
func (m *Material) ColorAt(pos Vec3) Vec3 {
	if m == nil {
		return V3(1, 0, 1) // Couleur d'erreur (magenta)
	}

	if m.Checker == nil {
		return m.Albedo
	}

	scale := m.Checker.Scale
	if scale <= 0 {
		scale = 1
	}

	// Calcul de la case du damier basé sur les coordonnées X et Z
	x := int(math.Floor(pos.X * scale))
	z := int(math.Floor(pos.Z * scale))

	if (x+z)%2 == 0 {
		return m.Checker.ColorA
	}

	return m.Checker.ColorB
}
