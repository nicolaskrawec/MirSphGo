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

// =====================
// Game
// =====================

type Game struct {
	width  int
	height int

	pixels []byte

	camPos   Vec3
	camYaw   float64
	camPitch float64

	prevMouseX int
	prevMouseY int
	hasMouse   bool

	scene Scene
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	const moveSpeed = 0.1
	const mouseSens = 0.005
	const invertMouseY = true

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

	forward := V3(math.Sin(g.camYaw), 0, math.Cos(g.camYaw))
	right := V3(math.Cos(g.camYaw), 0, -math.Sin(g.camYaw))

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

	t := float64(time.Now().UnixNano()) / 1e9

	for i := range g.scene.Spheres {
		g.scene.Spheres[i].UpdateAnimation(t)
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.pixels == nil {
		g.pixels = make([]byte, g.width*g.height*4)
	}

	g.render()

	screen.WritePixels(g.pixels)

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
					ray := g.cameraRay(x, y)
					col := g.trace(ray, 0)

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

	for y := 0; y < g.height; y++ {
		rows <- y
	}

	close(rows)
	wg.Wait()
}

func (g *Game) cameraRay(x, y int) Ray {
	fov := math.Pi / 3
	aspectRatio := float64(g.width) / float64(g.height)
	scale := math.Tan(fov / 2)

	px := (2*(float64(x)+0.5)/float64(g.width) - 1) * aspectRatio * scale
	py := (1 - 2*(float64(y)+0.5)/float64(g.height)) * scale

	cosYaw := math.Cos(g.camYaw)
	sinYaw := math.Sin(g.camYaw)
	cosPitch := math.Cos(g.camPitch)
	sinPitch := math.Sin(g.camPitch)

	// Direction locale : caméra qui regarde vers +Z.
	localX := px
	localY := py
	localZ := 1.0

	// Pitch autour de X.
	y1 := localY*cosPitch - localZ*sinPitch
	z1 := localY*sinPitch + localZ*cosPitch

	// Yaw autour de Y.
	dirX := localX*cosYaw + z1*sinYaw
	dirY := y1
	dirZ := -localX*sinYaw + z1*cosYaw

	return Ray{
		Origin: g.camPos,
		Dir:    V3(dirX, dirY, dirZ).Normalize(),
	}
}

func (g *Game) trace(ray Ray, depth int) Vec3 {
	if depth > MaxDepth {
		return V3(0, 0, 0)
	}

	hit, ok := g.scene.Intersect(ray, Epsilon, math.MaxFloat64)
	if !ok {
		return skyColor(ray)
	}

	material := hit.Material
	baseColor := material.ColorAt(hit.Position)

	ambient := baseColor.Hadamard(g.scene.Ambient)

	light := g.scene.Light
	toLight := light.Position.Sub(hit.Position)
	lightDir := toLight.Normalize()

	inShadow := g.scene.IsInShadow(hit.Position, hit.Normal)

	diffuse := V3(0, 0, 0)
	specular := V3(0, 0, 0)

	if !inShadow {
		ndotl := math.Max(0, hit.Normal.Dot(lightDir))

		diffuse = baseColor.
			Hadamard(light.Color).
			Mul(ndotl * light.Intensity)

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

	if material.Reflectivity <= 0.01 || depth >= MaxDepth {
		return localColor
	}

	reflectDir := ray.Dir.Reflect(hit.Normal).Normalize()
	reflectRay := Ray{
		Origin: hit.Position.Add(hit.Normal.Mul(Epsilon)),
		Dir:    reflectDir,
	}

	reflectedColor := g.trace(reflectRay, depth+1)

	return localColor.Mul(1 - material.Reflectivity).
		Add(reflectedColor.Mul(material.Reflectivity))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.width, g.height
}
