package main

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	
	paused       bool    // État de pause
	animTime     float64 // Temps cumulé pour l'animation
	lastRealTime float64 // Dernier temps réel enregistré
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
	
	// Régénérer la scène si la touche N est pressée
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.scene = createScene()
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

	// Gestion du temps pour l'animation (indépendant du temps réel si mis en pause)
	now := float64(time.Now().UnixNano()) / 1e9
	if g.lastRealTime == 0 {
		g.lastRealTime = now
	}
	dt := now - g.lastRealTime
	g.lastRealTime = now

	// Toggle pause avec la touche P ou Espace
	if inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.paused = !g.paused
	}

	if !g.paused {
		g.animTime += dt
		for i := range g.scene.Spheres {
			g.scene.Spheres[i].UpdateAnimation(g.animTime)
		}
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
	status := "Running"
	if g.paused {
		status = "PAUSED"
	}

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"FPS: %.2f\nRender: %dx%d\nSpheres: %d\nPlanes: %d\nObjects: %d\nStatus: %s\n[N] New Scene [P] Pause",
			ebiten.ActualFPS(),
			g.width,
			g.height,
			len(g.scene.Spheres),
			len(g.scene.Planes),
			objectCount,
			status,
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

	// Gestion de la réfraction et transparence
	if material.Transparency > 0 && depth < MaxDepth {
		outNormal := hit.Normal
		refIdx := material.RefractionIndex
		if refIdx <= 0 {
			refIdx = 1.0
		}
		refRatio := 1.0 / refIdx
		if ray.Dir.Dot(hit.Normal) > 0 {
			outNormal = hit.Normal.Mul(-1)
			refRatio = refIdx
		}

		cosTheta := math.Min(ray.Dir.Mul(-1).Dot(outNormal), 1.0)
		reflectance := Schlick(cosTheta, refRatio)
		// On prend le maximum entre la réflexion spéculaire (miroir) et l'effet Fresnel
		actualReflectance := math.Max(reflectance, material.Reflectivity)

		// Calcul de la réflexion
		reflectDir := ray.Dir.Reflect(outNormal).Normalize()
		reflectRay := Ray{
			Origin: hit.Position.Add(outNormal.Mul(Epsilon)),
			Dir:    reflectDir,
		}
		reflectedColor := g.trace(reflectRay, depth+1)

		// Calcul de la réfraction
		refractedColor := V3(0, 0, 0)
		refractDir, canRefract := ray.Dir.Refract(outNormal, refRatio)
		if canRefract {
			refractRay := Ray{
				Origin: hit.Position.Sub(outNormal.Mul(Epsilon)),
				Dir:    refractDir.Normalize(),
			}
			refractedColor = g.trace(refractRay, depth+1)
		} else {
			// Réflexion totale interne
			actualReflectance = 1.0
		}

		// Mélange entre réflexion et réfraction
		transmissionColor := reflectedColor.Mul(actualReflectance).
			Add(refractedColor.Mul(1 - actualReflectance))

		// On mélange la couleur de l'objet (opacité) avec la couleur transmise
		return localColor.Mul(1 - material.Transparency).
			Add(transmissionColor.Mul(material.Transparency))
	}

	// Gestion des réflexions (récursion classique pour métaux/miroirs)
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
