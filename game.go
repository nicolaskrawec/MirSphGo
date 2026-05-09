package main

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Game implémente l'interface ebiten.Game.
type Game struct {
	width  int // Largeur de rendu interne
	height int // Hauteur de rendu interne

	pixels []byte // Tampon de pixels pour l'affichage

	camPos   Vec3    // Position de la caméra
	camYaw   float64 // Rotation horizontale de la caméra
	camPitch float64 // Rotation verticale de la caméra

	prevMouseX int
	prevMouseY int
	hasMouse   bool

	scene Scene // La scène à rendre
}

// Update met à jour l'état du jeu (entrées clavier/souris, animations).
func (g *Game) Update() error {
	// Quitter si la touche Echap est pressée
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	// Recentrer la caméra si la touche R est pressée
	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.camPos = V3(0, 1.7, -2)
		g.camYaw = 0
		g.camPitch = 0
	}

	const moveSpeed = 0.1
	const mouseSens = 0.005
	const invertMouseY = true

	// Gestion de la rotation de la caméra à la souris
	mx, my := ebiten.CursorPosition()

	if g.hasMouse {
		dx := mx - g.prevMouseX
		dy := my - g.prevMouseY

		g.camYaw += float64(dx) * mouseSens

		if invertMouseY {
			g.camPitch += float64(dy) * mouseSens
		} else {
			g.camPitch -= float64(dy) * mouseSens
		}

		// Limiter le pitch pour éviter de retourner la caméra
		if g.camPitch > 1.5 {
			g.camPitch = 1.5
		}
		if g.camPitch < -1.5 {
			g.camPitch = -1.5
		}
	}

	g.prevMouseX = mx
	g.prevMouseY = my
	g.hasMouse = true

	// Calcul des vecteurs avant et droite pour le déplacement
	forward := V3(math.Sin(g.camYaw), 0, math.Cos(g.camYaw))
	right := V3(math.Cos(g.camYaw), 0, -math.Sin(g.camYaw))

	// Déplacement au clavier (Z/W, S, Q/A, D)
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.camPos = g.camPos.Add(forward.Mul(moveSpeed))
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camPos = g.camPos.Sub(forward.Mul(moveSpeed))
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camPos = g.camPos.Add(right.Mul(moveSpeed))
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.camPos = g.camPos.Sub(right.Mul(moveSpeed))
	}

	// Mise à jour des animations de la scène
	t := float64(time.Now().UnixNano()) / 1e9
	for i := range g.scene.Spheres {
		g.scene.Spheres[i].UpdateAnimation(t)
	}

	return nil
}

// Draw affiche le résultat du rendu sur l'écran.
func (g *Game) Draw(screen *ebiten.Image) {
	if g.pixels == nil {
		g.pixels = make([]byte, g.width*g.height*4)
	}

	// Lancer le ray tracing
	g.render()

	// Écrire les pixels calculés dans l'image ebiten
	screen.WritePixels(g.pixels)

	// Afficher des informations de débogage
	objectCount := len(g.scene.Spheres) + len(g.scene.Planes)
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"FPS: %.2f\nRender: %dx%d\nSpheres: %d\nPlanes: %d\nObjects: %d",
			ebiten.ActualFPS(),
			g.width,
			g.height,
			len(g.scene.Spheres),
			len(g.scene.Planes),
			objectCount,
		),
	)
}

// render gère le rendu multi-threadé de la scène.
func (g *Game) render() {
	numWorkers := runtime.NumCPU()
	rows := make(chan int, g.height)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for worker := 0; worker < numWorkers; worker++ {
		go func() {
			defer wg.Done()

			for y := range rows {
				for x := 0; x < g.width; x++ {
					// Créer le rayon pour ce pixel
					ray := g.cameraRay(x, y)
					// Tracer le rayon et obtenir la couleur
					col := g.trace(ray, 0)

					// Convertir en RGBA et stocker dans le tampon
					rgba := ToRGBA(col)
					i := (y*g.width + x) * 4

					g.pixels[i] = rgba.R
					g.pixels[i+1] = rgba.G
					g.pixels[i+2] = rgba.B
					g.pixels[i+3] = rgba.A
				}
			}
		}()
	}

	// Distribuer les lignes aux travailleurs
	for y := 0; y < g.height; y++ {
		rows <- y
	}

	close(rows)
	wg.Wait()
}

// cameraRay génère un rayon partant de la caméra pour un pixel (x, y) donné.
func (g *Game) cameraRay(x, y int) Ray {
	fov := math.Pi / 3
	aspectRatio := float64(g.width) / float64(g.height)
	scale := math.Tan(fov / 2)

	// Coordonnées normalisées [-1, 1]
	px := (2*(float64(x)+0.5)/float64(g.width) - 1) * aspectRatio * scale
	py := (1 - 2*(float64(y)+0.5)/float64(g.height)) * scale

	cosYaw := math.Cos(g.camYaw)
	sinYaw := math.Sin(g.camYaw)
	cosPitch := math.Cos(g.camPitch)
	sinPitch := math.Sin(g.camPitch)

	// Direction locale : caméra qui regarde vers +Z
	localX := px
	localY := py
	localZ := 1.0

	// Rotation (Pitch autour de X, puis Yaw autour de Y)
	y1 := localY*cosPitch - localZ*sinPitch
	z1 := localY*sinPitch + localZ*cosPitch

	dirX := localX*cosYaw + z1*sinYaw
	dirY := y1
	dirZ := -localX*sinYaw + z1*cosYaw

	return Ray{
		Origin: g.camPos,
		Dir:    V3(dirX, dirY, dirZ).Normalize(),
	}
}

// trace suit un rayon dans la scène et retourne la couleur finale calculée.
func (g *Game) trace(ray Ray, depth int) Vec3 {
	if depth > MaxDepth {
		return V3(0, 0, 0) // Limite de récursion atteinte
	}

	// Chercher l'objet le plus proche
	hit, ok := g.scene.Intersect(ray, Epsilon, math.MaxFloat64)
	if !ok {
		return skyColor(ray) // Rien touché, afficher le ciel
	}

	material := hit.Material
	baseColor := material.ColorAt(hit.Position)

	// Éclairage ambiant
	ambient := baseColor.Hadamard(g.scene.Ambient)

	light := g.scene.Light
	toLight := light.Position.Sub(hit.Position)
	lightDir := toLight.Normalize()

	// Vérifier si le point est à l'ombre
	inShadow := g.scene.IsInShadow(hit.Position, hit.Normal)

	diffuse := V3(0, 0, 0)
	specular := V3(0, 0, 0)

	if !inShadow {
		// Éclairage diffus (Lambert)
		ndotl := math.Max(0, hit.Normal.Dot(lightDir))
		diffuse = baseColor.
			Hadamard(light.Color).
			Mul(ndotl * light.Intensity)

		// Éclairage spéculaire (Blinn-Phong)
		if material.Specular > 0 {
			viewDir := ray.Dir.Mul(-1).Normalize()
			halfDir := lightDir.Add(viewDir).Normalize()

			spec := math.Pow(
				math.Max(0, hit.Normal.Dot(halfDir)),
				material.Shininess,
			)

			specular = light.Color.Mul(spec * material.Specular * light.Intensity)
		}
	}

	localColor := ambient.Add(diffuse).Add(specular)

	// Gestion des réflexions (récursion)
	if material.Reflectivity <= 0.01 || depth >= MaxDepth {
		return localColor
	}

	reflectDir := ray.Dir.Reflect(hit.Normal).Normalize()
	reflectRay := Ray{
		Origin: hit.Position.Add(hit.Normal.Mul(Epsilon)),
		Dir:    reflectDir,
	}

	reflectedColor := g.trace(reflectRay, depth+1)

	// Mélange de la couleur locale et de la couleur réfléchie
	return localColor.Mul(1 - material.Reflectivity).
		Add(reflectedColor.Mul(material.Reflectivity))
}

// Layout définit la taille logique du jeu indépendamment de la fenêtre.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.width, g.height
}
