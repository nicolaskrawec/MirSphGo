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
	MaxDepth = 4     // Nombre maximum de rebonds pour la réflexion
)

func main() {
	// Initialiser le générateur de nombres aléatoires
	rand.Seed(time.Now().UnixNano())

	// Résolution de rendu interne (Viewport)
	// Plus c'est élevé, plus c'est beau, mais plus c'est lourd pour le CPU.
	renderW, renderH := 1024, 768

	// Taille initiale de la fenêtre
	windowW, windowH := 1024, 768

	// Configuration de la fenêtre Ebitengine
	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Optimized Scalable Ray Tracer")

	shader, err := ebiten.NewShader([]byte(rayShaderSource))
	if err != nil {
		log.Fatal(err)
	}

	// Initialisation de l'objet Game
	game := &Game{
		width:  renderW,
		height: renderH,
		shader: shader,
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
	transparency := 0.0
	refIdx := 1.0
	// if rand.Float64() > 0.8 {
	// 	transparency = 0 // 0.6 + rand.Float64()*0.4
	// 	refIdx = 1 // 1.3 + rand.Float64()*0.4
	// }

	return Material{
		Albedo: V3(
			rand.Float64()*0.8+0.2,
			rand.Float64()*0.8+0.2,
			rand.Float64()*0.8+0.2,
		),
		Reflectivity:    0.0 + rand.Float64()*0.35,
		Specular:        rand.Float64() * 0.7,
		Shininess:       16 + rand.Float64()*80,
		Transparency:    transparency,
		RefractionIndex: refIdx,
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
			ColorA: V3(0.985, 0.985, 0.985),
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
	spheres := make([]Sphere, 0, 32)

	// Sphère centrale
	centralPosition := V3(0, 1.5, 4)
	spheres = append(spheres, Sphere{
		Center: centralPosition,
		Radius: 1,
		Material: Material{
			Albedo:          V3(1.0, 0.75, 0.1), // Jaune/Or
			Reflectivity:    0.05,               //0.22,
			Specular:        5,
			Shininess:       90,
			RefractionIndex: 1.2,
			Transparency:    0.8,
		},
	})

	// Sphère de verre (transparente)
	// spheres = append(spheres, Sphere{
	// 	Center: V3(2, 1, 3),
	// 	Radius: 0.8,
	// 	Material: Material{
	// 		Albedo:          V3(1, 1, 1),
	// 		Transparency:    0.95,
	// 		RefractionIndex: 1.5,
	// 		Specular:        1.0,
	// 		Shininess:       100,
	// 	},
	// })

	// Génération de petites sphères orbitales
	for i := 0; i < 31; i++ {
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
		Ambient: V3(0.08, 0.08, 0.01),
	}
}
