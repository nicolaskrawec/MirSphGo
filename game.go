package main

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Game implemente l'interface ebiten.Game.
type Game struct {
	width  int
	height int

	shader *ebiten.Shader

	camPos   Vec3
	camYaw   float64
	camPitch float64

	prevMouseX int
	prevMouseY int
	hasMouse   bool

	scene Scene

	paused       bool
	animTime     float64
	lastRealTime float64
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.camPos = V3(0, 1.7, -2)
		g.camYaw = 0
		g.camPitch = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.scene = createScene()
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

	now := float64(time.Now().UnixNano()) / 1e9
	if g.lastRealTime == 0 {
		g.lastRealTime = now
	}
	dt := now - g.lastRealTime
	g.lastRealTime = now

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

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawRectShader(
		g.width,
		g.height,
		g.shader,
		&ebiten.DrawRectShaderOptions{
			Uniforms: g.shaderUniforms(),
		},
	)

	objectCount := len(g.scene.Spheres) + len(g.scene.Planes)
	status := "Running"
	if g.paused {
		status = "PAUSED"
	}

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"FPS: %.2f\nRender: %dx%d\nSpheres: %d\nPlanes: %d\nObjects: %d\nStatus: %s\nGPU shader\n[N] New Scene [P] Pause",
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

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.width, g.height
}

func (g *Game) shaderUniforms() map[string]interface{} {
	sphereCenterRadius := make([]float32, maxShaderSpheres*4)
	sphereAlbedoReflect := make([]float32, maxShaderSpheres*4)
	sphereSpecShineTransIOR := make([]float32, maxShaderSpheres*4)

	sphereCount := len(g.scene.Spheres)
	if sphereCount > maxShaderSpheres {
		sphereCount = maxShaderSpheres
	}

	for i := 0; i < sphereCount; i++ {
		sphere := g.scene.Spheres[i]
		material := sphere.Material
		offset := i * 4

		sphereCenterRadius[offset] = float32(sphere.Center.X)
		sphereCenterRadius[offset+1] = float32(sphere.Center.Y)
		sphereCenterRadius[offset+2] = float32(sphere.Center.Z)
		sphereCenterRadius[offset+3] = float32(sphere.Radius)

		sphereAlbedoReflect[offset] = float32(material.Albedo.X)
		sphereAlbedoReflect[offset+1] = float32(material.Albedo.Y)
		sphereAlbedoReflect[offset+2] = float32(material.Albedo.Z)
		sphereAlbedoReflect[offset+3] = float32(material.Reflectivity)

		sphereSpecShineTransIOR[offset] = float32(material.Specular)
		sphereSpecShineTransIOR[offset+1] = float32(material.Shininess)
		sphereSpecShineTransIOR[offset+2] = float32(material.Transparency)
		sphereSpecShineTransIOR[offset+3] = float32(material.RefractionIndex)
	}

	floor := Plane{
		Point:    V3(0, 0, 0),
		Normal:   V3(0, 1, 0),
		Material: Material{Albedo: V3(0.8, 0.8, 0.8)},
	}
	if len(g.scene.Planes) > 0 {
		floor = g.scene.Planes[0]
	}

	floorColorA := floor.Material.Albedo
	floorColorB := floor.Material.Albedo
	checkerScale := 1.0
	if floor.Material.Checker != nil {
		floorColorA = floor.Material.Checker.ColorA
		floorColorB = floor.Material.Checker.ColorB
		checkerScale = floor.Material.Checker.Scale
	}

	return map[string]interface{}{
		"Resolution":              []float32{float32(g.width), float32(g.height)},
		"CamPos":                  vec3Uniform(g.camPos),
		"CamYaw":                  float32(g.camYaw),
		"CamPitch":                float32(g.camPitch),
		"SphereCount":             float32(sphereCount),
		"SphereCenterRadius":      sphereCenterRadius,
		"SphereAlbedoReflect":     sphereAlbedoReflect,
		"SphereSpecShineTransIOR": sphereSpecShineTransIOR,
		"FloorPoint":              vec3Uniform(floor.Point),
		"FloorNormal":             vec3Uniform(floor.Normal),
		"FloorColorA":             vec3Uniform(floorColorA),
		"FloorColorB":             vec3Uniform(floorColorB),
		"FloorParams": []float32{
			float32(floor.Material.Reflectivity),
			float32(floor.Material.Specular),
			float32(floor.Material.Shininess),
			float32(checkerScale),
		},
		"LightPosition":  vec3Uniform(g.scene.Light.Position),
		"LightColor":     vec3Uniform(g.scene.Light.Color),
		"LightIntensity": float32(g.scene.Light.Intensity),
		"Ambient":        vec3Uniform(g.scene.Ambient),
	}
}

func vec3Uniform(v Vec3) []float32 {
	return []float32{float32(v.X), float32(v.Y), float32(v.Z)}
}
