package main

// Ray représente un rayon lumineux avec une origine et une direction.
type Ray struct {
	Origin Vec3 // Point de départ du rayon
	Dir    Vec3 // Direction du rayon (généralement normalisée)
}

// At calcule la position du rayon à une distance 't' de son origine.
func (r Ray) At(t float64) Vec3 {
	return r.Origin.Add(r.Dir.Mul(t))
}

// HitRecord stocke les détails d'une intersection entre un rayon et un objet.
type HitRecord struct {
	T        float64   // Distance le long du rayon
	Position Vec3      // Point exact de l'impact
	Normal   Vec3      // Normale à la surface au point d'impact
	Material *Material // Matériau de l'objet touché
}
