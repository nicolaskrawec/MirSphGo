package main

import "math"

// Sphere représente un objet sphérique dans la scène.
type Sphere struct {
	Center   Vec3
	Radius   float64
	Material Material

	// Propriétés pour l'animation orbitale
	Orbiting    bool    // Si vrai, la sphère tourne autour d'un centre
	OrbitCenter Vec3    // Point central de l'orbite
	OrbitRadius float64 // Rayon de l'orbite
	OrbitSpeed  float64 // Vitesse de rotation
	OrbitPhase  float64 // Déphasage initial (en radians)
	OrbitHeight float64 // Hauteur constante ou décalage en Y
}

// Intersect calcule l'intersection entre un rayon et la sphère.
// Retourne un HitRecord et vrai si le rayon touche la sphère dans l'intervalle [tMin, tMax].
func (s *Sphere) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	oc := ray.Origin.Sub(s.Center)

	// Équation quadratique : a*t^2 + b*t + c = 0
	a := ray.Dir.Dot(ray.Dir)
	halfB := oc.Dot(ray.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius

	discriminant := halfB*halfB - a*c
	if discriminant < 0 {
		return HitRecord{}, false
	}

	sqrtD := math.Sqrt(discriminant)

	// Trouve la racine la plus proche qui est dans l'intervalle acceptable
	t := (-halfB - sqrtD) / a
	if t < tMin || t > tMax {
		t = (-halfB + sqrtD) / a
		if t < tMin || t > tMax {
			return HitRecord{}, false
		}
	}

	pos := ray.At(t)
	normal := pos.Sub(s.Center).Normalize()

	return HitRecord{
		T:        t,
		Position: pos,
		Normal:   normal,
		Material: &s.Material,
	}, true
}

// IntersectAny est une version optimisée d'Intersect pour les tests d'ombre.
// Retourne vrai dès qu'une intersection est trouvée dans l'intervalle.
func (s *Sphere) IntersectAny(ray Ray, tMin, tMax float64) bool {
	oc := ray.Origin.Sub(s.Center)

	a := ray.Dir.Dot(ray.Dir)
	halfB := oc.Dot(ray.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius

	discriminant := halfB*halfB - a*c
	if discriminant < 0 {
		return false
	}

	sqrtD := math.Sqrt(discriminant)

	t := (-halfB - sqrtD) / a
	if t >= tMin && t <= tMax {
		return true
	}

	t = (-halfB + sqrtD) / a
	return t >= tMin && t <= tMax
}

// UpdateAnimation met à jour la position de la sphère en fonction du temps 't'.
func (s *Sphere) UpdateAnimation(t float64) {
	if !s.Orbiting {
		return
	}

	angle := s.OrbitPhase + t*s.OrbitSpeed

	s.Center.X = s.OrbitCenter.X + math.Cos(angle)*s.OrbitRadius
	s.Center.Z = s.OrbitCenter.Z + math.Sin(angle)*s.OrbitRadius
	s.Center.Y = s.OrbitCenter.Y + s.OrbitHeight
}

// Plane représente un plan infini défini par un point et une normale.
type Plane struct {
	Point    Vec3
	Normal   Vec3
	Material Material
}

// Intersect calcule l'intersection entre un rayon et le plan.
func (p *Plane) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	n := p.Normal.Normalize()
	denom := n.Dot(ray.Dir)

	// Si le dénominateur est proche de zéro, le rayon est parallèle au plan
	if math.Abs(denom) < 1e-6 {
		return HitRecord{}, false
	}

	t := p.Point.Sub(ray.Origin).Dot(n) / denom
	if t < tMin || t > tMax {
		return HitRecord{}, false
	}

	pos := ray.At(t)

	// S'assure que la normale pointe vers le rayon incident
	if n.Dot(ray.Dir) > 0 {
		n = n.Mul(-1)
	}

	return HitRecord{
		T:        t,
		Position: pos,
		Normal:   n,
		Material: &p.Material,
	}, true
}
