package main



// =====================
// Scene
// =====================

type PointLight struct {
	Position  Vec3
	Color     Vec3
	Intensity float64
}

type Scene struct {
	Spheres []Sphere
	Planes  []Plane

	Light   PointLight
	Ambient Vec3
}

func (s *Scene) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	closest := tMax
	var bestHit HitRecord
	hitAnything := false

	for i := range s.Spheres {
		if hit, ok := s.Spheres[i].Intersect(ray, tMin, closest); ok {
			closest = hit.T
			bestHit = hit
			hitAnything = true
		}
	}

	for i := range s.Planes {
		if hit, ok := s.Planes[i].Intersect(ray, tMin, closest); ok {
			closest = hit.T
			bestHit = hit
			hitAnything = true
		}
	}

	return bestHit, hitAnything
}

func (s *Scene) IsInShadow(point, normal Vec3) bool {
	toLight := s.Light.Position.Sub(point)
	lightDistance := toLight.Length()
	lightDir := toLight.Normalize()

	shadowRay := Ray{
		Origin: point.Add(normal.Mul(Epsilon)),
		Dir:    lightDir,
	}

	// Optimisation volontaire :
	// le sol ne bloque pas la lumière dans ce modèle simple.
	// On ne teste donc que les sphères.
	for i := range s.Spheres {
		if s.Spheres[i].IntersectAny(shadowRay, Epsilon, lightDistance-Epsilon) {
			return true
		}
	}

	return false
}

func skyColor(ray Ray) Vec3 {
	t := Clamp01(ray.Dir.Y*0.5 + 0.5)

	horizon := V3(0.53, 0.80, 0.98)
	zenith := V3(0.02, 0.20, 0.55)

	return Lerp(horizon, zenith, t)
}
