package main

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	Epsilon  = 0.001 // Petite valeur pour éviter les erreurs de précision et l'auto-intersection
	MaxDepth = 2     // Nombre maximum de rebonds pour la réflexion
)

func main() {
	// Initialiser le générateur de nombres aléatoires
	rand.Seed(time.Now().UnixNano())

	// Résolution de rendu interne (Viewport)
	// Plus c'est élevé, plus c'est beau, mais plus c'est lourd pour le CPU.
	renderW, renderH := 640, 480

	// Taille initiale de la fenêtre
	windowW, windowH := 640, 480

	// Configuration de la fenêtre Ebitengine
	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Optimized Scalable Ray Tracer")

	// Initialisation de l'objet Game
	game := &Game{
		width:  renderW,
		height: renderH,
		camPos: V3(0, 1.7, -2), // Position de départ de la caméra
		scene:  createScene(),  // Génération de la scène
	}

	// Lancement de la boucle de jeu
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// randomMaterial génère un matériau avec des propriétés aléatoires.
func randomMaterial() Material {
	return Material{
		Albedo: V3(
			rand.Float64()*0.8+0.2,
			rand.Float64()*0.8+0.2,
			rand.Float64()*0.8+0.2,
		),
		Reflectivity: 0.2 + rand.Float64()*0.35,
		Specular:     rand.Float64() * 0.7,
		Shininess:    16 + rand.Float64()*80,
	}
}

// createScene construit l'environnement (sol, sphères, lumières).
func createScene() Scene {
	// Matériau pour le sol (damier réfléchissant)
	floorMaterial := Material{
		Albedo:       V3(0.8, 0.8, 0.8),
		Reflectivity: 0.5,
		Specular:     0.05,
		Shininess:    16,
		Checker: &CheckerPattern{
			ColorA: V3(0.85, 0.85, 0.85),
			ColorB: V3(0.15, 0.15, 0.15),
			Scale:  1,
		},
	}

	// Ajout du sol
	planes := []Plane{
		{
			Point:    V3(0, 0, 0),
			Normal:   V3(0, 1, 0),
			Material: floorMaterial,
		},
	}

	// Liste des sphères
	spheres := make([]Sphere, 0, 11)

	// Sphère centrale rouge
	centralPosition := V3(0, 1.5, 4)
	spheres = append(spheres, Sphere{
		Center: centralPosition,
		Radius: 1,
		Material: Material{
			Albedo:       V3(0.9, 0.12, 0.08),
			Reflectivity: 0.8, //0.22,
			Specular:     5,
			Shininess:    90,
		},
	})

	// Génération de petites sphères orbitales
	for i := 0; i < 4; i++ {
		radius := rand.Float64()*0.35 + 0.15

		sphere := Sphere{
			Center: V3(
				rand.Float64()*6-3,
				rand.Float64()*4,
				rand.Float64()*5+2,
			),
			Radius:   radius,
			Material: randomMaterial(),
			Orbiting: true,
			OrbitCenter: V3(
				centralPosition.X,
				rand.Float64()*4,
				centralPosition.Z,
			),
			OrbitRadius: rand.Float64()*2 + 1,
			OrbitSpeed:  rand.Float64()*0.5 + 0.8,
			OrbitPhase:  float64(i) * 2 * math.Pi / 8,
			OrbitHeight: 0.0,
		}

		spheres = append(spheres, sphere)
	}

	return Scene{
		Spheres: spheres,
		Planes:  planes,

		// Source de lumière
		Light: PointLight{
			Position:  V3(5, 8, 2),
			Color:     V3(1.0, 0.95, 0.82),
			Intensity: 1.2,
		},

		// Lumière ambiante pour déboucher les ombres
		Ambient: V3(0.08, 0.08, 0.1),
	}
}
