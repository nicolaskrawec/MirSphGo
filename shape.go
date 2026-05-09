package main

import "math"

// =====================
// Sphere
// =====================

type Sphere struct {
	Center   Vec3
	Radius   float64
	Material Material

	// Animation orbitale optionnelle
	Orbiting    bool
	OrbitCenter Vec3
	OrbitRadius float64
	OrbitSpeed  float64
	OrbitPhase  float64
	OrbitHeight float64
}

func (s *Sphere) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	oc := ray.Origin.Sub(s.Center)

	a := ray.Dir.Dot(ray.Dir)
	halfB := oc.Dot(ray.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius

	discriminant := halfB*halfB - a*c
	if discriminant < 0 {
		return HitRecord{}, false
	}

	sqrtD := math.Sqrt(discriminant)

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

// Méthode spécialisée pour les ombres.
// Elle évite de construire un HitRecord complet.
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

func (s *Sphere) UpdateAnimation(t float64) {
	if !s.Orbiting {
		return
	}

	angle := s.OrbitPhase + t*s.OrbitSpeed

	s.Center.X = s.OrbitCenter.X + math.Cos(angle)*s.OrbitRadius
	s.Center.Z = s.OrbitCenter.Z + math.Sin(angle)*s.OrbitRadius
	s.Center.Y = s.OrbitCenter.Y + s.OrbitHeight
}

// =====================
// Plane
// =====================

type Plane struct {
	Point    Vec3
	Normal   Vec3
	Material Material
}

func (p *Plane) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	n := p.Normal.Normalize()
	denom := n.Dot(ray.Dir)

	if math.Abs(denom) < 1e-6 {
		return HitRecord{}, false
	}

	t := p.Point.Sub(ray.Origin).Dot(n) / denom
	if t < tMin || t > tMax {
		return HitRecord{}, false
	}

	pos := ray.At(t)

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
