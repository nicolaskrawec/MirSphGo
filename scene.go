package main

// PointLight représente une source de lumière ponctuelle.
type PointLight struct {
	Position  Vec3
	Color     Vec3
	Intensity float64
}

// Scene regroupe tous les objets et les lumières de l'environnement.
type Scene struct {
	Spheres []Sphere
	Planes  []Plane

	Light   PointLight
	Ambient Vec3
}

// Intersect parcourt tous les objets de la scène pour trouver l'intersection la plus proche.
func (s *Scene) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	closest := tMax
	var bestHit HitRecord
	hitAnything := false

	// Test d'intersection avec toutes les sphères
	for i := range s.Spheres {
		if hit, ok := s.Spheres[i].Intersect(ray, tMin, closest); ok {
			closest = hit.T
			bestHit = hit
			hitAnything = true
		}
	}

	// Test d'intersection avec tous les plans
	for i := range s.Planes {
		if hit, ok := s.Planes[i].Intersect(ray, tMin, closest); ok {
			closest = hit.T
			bestHit = hit
			hitAnything = true
		}
	}

	return bestHit, hitAnything
}

// IsInShadow vérifie si un point est à l'ombre d'une source lumineuse.
func (s *Scene) IsInShadow(point, normal Vec3) bool {
	toLight := s.Light.Position.Sub(point)
	lightDistance := toLight.Length()
	lightDir := toLight.Normalize()

	// On lance un rayon du point vers la lumière
	shadowRay := Ray{
		Origin: point.Add(normal.Mul(Epsilon)), // On décale légèrement pour éviter l'auto-intersection
		Dir:    lightDir,
	}

	// Si un objet intercepte ce rayon avant d'atteindre la lumière, le point est à l'ombre
	for i := range s.Spheres {
		if s.Spheres[i].IntersectAny(shadowRay, Epsilon, lightDistance-Epsilon) {
			return true
		}
	}

	return false
}

// skyColor retourne la couleur du ciel (dégradé) en fonction de la direction du rayon.
func skyColor(ray Ray) Vec3 {
	t := Clamp01(ray.Dir.Y*0.5 + 0.5)

	horizon := V3(0.53, 0.80, 0.98)
	zenith := V3(0.02, 0.20, 0.55)

	return Lerp(horizon, zenith, t)
}
