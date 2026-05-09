package main

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	Epsilon  = 0.001
	MaxDepth = 4
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Résolution de rendu interne.
	// Augmente pour plus de qualité, baisse pour plus de FPS.
	renderW, renderH := 640, 480

	// Taille de la fenêtre.
	windowW, windowH := 640, 480

	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Optimized Scalable Ray Tracer")

	game := &Game{
		width:  renderW,
		height: renderH,
		camPos: V3(0, 1.7, -2),
		scene:  createScene(),
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// =====================
// Scene creation
// =====================

func randomMaterial() Material {
	return Material{
		Albedo: V3(
			rand.Float64()*0.8+0.2,
			rand.Float64()*0.8+0.2,
			rand.Float64()*0.8+0.2,
		),
		Reflectivity: rand.Float64() * 0.35,
		Specular:     rand.Float64() * 0.7,
		Shininess:    16 + rand.Float64()*80,
	}
}

func createScene() Scene {
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

	planes := []Plane{
		{
			Point:    V3(0, 0, 0),
			Normal:   V3(0, 1, 0),
			Material: floorMaterial,
		},
	}

	spheres := make([]Sphere, 0, 11)

	centralPosition := V3(0, 1, 4)
	spheres = append(spheres, Sphere{
		Center: centralPosition,
		Radius: 1,
		Material: Material{
			Albedo:       V3(0.9, 0.12, 0.08),
			Reflectivity: 0.22,
			Specular:     0.8,
			Shininess:    64,
		},
	})

	for i := 0; i < 6; i++ {
		radius := rand.Float64()*0.35 + 0.15

		sphere := Sphere{
			Center: V3(
				rand.Float64()*6-3,
				rand.Float64()*3,
				rand.Float64()*5+2,
			),
			Radius:   radius,
			Material: randomMaterial(),
			Orbiting: true,
			OrbitCenter: V3(
				centralPosition.X,
				rand.Float64()*3,
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

		Light: PointLight{
			Position:  V3(5, 8, 2),
			Color:     V3(1.0, 0.95, 0.82),
			Intensity: 1.2,
		},

		Ambient: V3(0.08, 0.08, 0.1),
	}
}
