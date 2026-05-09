package main

import (
	"image/color"
	"log"
	"math"

	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Vec3 represents a 3D vector.
type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(o Vec3) Vec3    { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3    { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Mul(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Dot(o Vec3) float64 { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }
func (v Vec3) Length() float64    { return math.Sqrt(v.Dot(v)) }
func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return v.Mul(1 / l)
}

type Ray struct {
	Origin, Dir Vec3
}

type Sphere struct {
	Center       Vec3
	Radius       float64
	Color        color.RGBA
	Reflectivity float64
}

func (s Sphere) Intersect(r Ray) (float64, bool) {
	oc := r.Origin.Sub(s.Center)
	b := oc.Dot(r.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius
	h := b*b - c
	if h < 0 {
		return 0, false
	}
	h = math.Sqrt(h)
	t := -b - h
	if t < 0 {
		t = -b + h
	}
	return t, t >= 0.001 // Use a small epsilon to avoid self-intersection
}

type Game struct {
	width, height int
	pixels        []byte
	camPos        Vec3
	camYaw        float64
	camPitch      float64
	prevMouseX    int
	prevMouseY    int
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}

	const moveSpeed = 0.1
	const mouseSens = 0.005

	// Mouse rotation
	mx, my := ebiten.CursorPosition()
	if g.prevMouseX != 0 || g.prevMouseY != 0 {
		deltaX := mx - g.prevMouseX
		deltaY := my - g.prevMouseY
		g.camYaw += float64(deltaX) * mouseSens
		g.camPitch -= float64(deltaY) * mouseSens

		// Clamp pitch to avoid flipping
		if g.camPitch > 1.5 {
			g.camPitch = 1.5
		}
		if g.camPitch < -1.5 {
			g.camPitch = -1.5
		}
	}
	g.prevMouseX = mx
	g.prevMouseY = my

	// Movement (Forward/Backward)
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.camPos.X += math.Sin(g.camYaw) * moveSpeed
		g.camPos.Z += math.Cos(g.camYaw) * moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camPos.X -= math.Sin(g.camYaw) * moveSpeed
		g.camPos.Z -= math.Cos(g.camYaw) * moveSpeed
	}

	// Strafing (Left/Right)
	if ebiten.IsKeyPressed(ebiten.KeyQ) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camPos.X -= math.Cos(g.camYaw) * moveSpeed
		g.camPos.Z += math.Sin(g.camYaw) * moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camPos.X += math.Cos(g.camYaw) * moveSpeed
		g.camPos.Z -= math.Cos(g.camYaw) * 0 // Placeholder to fix index
		g.camPos.Z -= math.Sin(g.camYaw) * moveSpeed
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.pixels == nil {
		g.pixels = make([]byte, g.width*g.height*4)
	}

	fov := math.Pi / 3 // 60 degrees
	aspectRatio := float64(g.width) / float64(g.height)
	scale := math.Tan(fov / 2)

	scene := Scene{
		Spheres: []Sphere{
			{Center: Vec3{0, 2, 4}, Radius: 1.0, Color: color.RGBA{255, 255, 255, 255}, Reflectivity: 0.2},
			{Center: Vec3{-2.5, 1.5, 5}, Radius: 0.5, Color: color.RGBA{255, 0, 0, 255}, Reflectivity: 0.2},
			{Center: Vec3{2.5, 2.5, 5}, Radius: 0.5, Color: color.RGBA{0, 255, 0, 255}, Reflectivity: 0.2},
		},
	}

	cosYaw := math.Cos(g.camYaw)
	sinYaw := math.Sin(g.camYaw)
	cosPitch := math.Cos(g.camPitch)
	sinPitch := math.Sin(g.camPitch)

	var wg sync.WaitGroup
	wg.Add(g.height)

	for y := 0; y < g.height; y++ {
		go func(y int) {
			defer wg.Done()
			for x := 0; x < g.width; x++ {
				// Local ray direction
				px := (2*(float64(x)+0.5)/float64(g.width) - 1) * aspectRatio * scale
				py := (1 - 2*(float64(y)+0.5)/float64(g.height)) * scale

				// 1. Rotate around X axis (Pitch)
				y1 := py*cosPitch + sinPitch
				z1 := -py*sinPitch + cosPitch

				// 2. Rotate around Y axis (Yaw)
				dirX := px*cosYaw + z1*sinYaw
				dirY := y1
				dirZ := -px*sinYaw + z1*cosYaw

				ray := Ray{
					Origin: g.camPos,
					Dir:    Vec3{dirX, dirY, dirZ}.Normalize(),
				}

				col := g.trace(ray, scene, 0)

				i := (y*g.width + x) * 4
				g.pixels[i] = col.R
				g.pixels[i+1] = col.G
				g.pixels[i+2] = col.B
				g.pixels[i+3] = col.A
			}
		}(y)
	}
	wg.Wait()

	screen.WritePixels(g.pixels)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))
}

type Scene struct {
	Spheres []Sphere
}

func (g *Game) trace(ray Ray, scene Scene, depth int) color.RGBA {
	if depth > 4 {
		return color.RGBA{0, 0, 0, 255} // Max depth
	}

	var closestSphere *Sphere
	tMin := math.MaxFloat64

	for i := range scene.Spheres {
		if t, ok := scene.Spheres[i].Intersect(ray); ok {
			if t < tMin {
				tMin = t
				closestSphere = &scene.Spheres[i]
			}
		}
	}

	tFloor := -1.0
	hitFloor := false
	if ray.Dir.Y < 0 {
		tFloor = -ray.Origin.Y / ray.Dir.Y
		if tFloor > 0 {
			hitFloor = true
		}
	}

	// Light parameters (Point light)
	lightPos := Vec3{5, 20, 5}
	lightColor := Vec3{1.0, 0.95, 0.8} // Warm solar color
	ambientIntensity := 0.1

	if closestSphere != nil && (!hitFloor || tMin < tFloor) {
		hitPos := ray.Origin.Add(ray.Dir.Mul(tMin))
		normal := hitPos.Sub(closestSphere.Center).Normalize()

		// Direction to light
		L := lightPos.Sub(hitPos).Normalize()

		// Shadows
		shadowRay := Ray{
			Origin: hitPos.Add(normal.Mul(0.001)),
			Dir:    L,
		}
		inShadow := false
		for i := range scene.Spheres {
			if _, ok := scene.Spheres[i].Intersect(shadowRay); ok {
				inShadow = true
				break
			}
		}

		diffuse := math.Max(0, normal.Dot(L))
		intensity := ambientIntensity
		if !inShadow {
			intensity += diffuse
		}

		reflectDir := ray.Dir.Sub(normal.Mul(2 * ray.Dir.Dot(normal))).Normalize()
		reflectRay := Ray{
			Origin: hitPos.Add(normal.Mul(0.001)),
			Dir:    reflectDir,
		}

		reflectedCol := g.trace(reflectRay, scene, depth+1)

		// Specular highlights (Phong)
		specular := 0.0
		if !inShadow {
			viewDir := ray.Dir.Mul(-1).Normalize()
			halfDir := L.Add(viewDir).Normalize()
			specular = math.Pow(math.Max(0, normal.Dot(halfDir)), 64)
		}

		// Combine sphere color (lit) and reflection (unaffected by local light)
		r := float64(closestSphere.Color.R)*(1-closestSphere.Reflectivity)*intensity*lightColor.X +
			float64(reflectedCol.R)*closestSphere.Reflectivity + specular*255
		g_col := float64(closestSphere.Color.G)*(1-closestSphere.Reflectivity)*intensity*lightColor.Y +
			float64(reflectedCol.G)*closestSphere.Reflectivity + specular*255
		b := float64(closestSphere.Color.B)*(1-closestSphere.Reflectivity)*intensity*lightColor.Z +
			float64(reflectedCol.B)*closestSphere.Reflectivity + specular*255

		return color.RGBA{uint8(math.Min(255, r)), uint8(math.Min(255, g_col)), uint8(math.Min(255, b)), 255}
	}

	if hitFloor {
		hitPos := ray.Origin.Add(ray.Dir.Mul(tFloor))
		floorColor := g.getFloorColor(ray, tFloor)

		normal := Vec3{0, 1, 0}
		L := lightPos.Sub(hitPos).Normalize()

		// Shadows on floor
		shadowRay := Ray{
			Origin: hitPos.Add(normal.Mul(0.001)),
			Dir:    L,
		}
		inShadow := false
		for i := range scene.Spheres {
			if _, ok := scene.Spheres[i].Intersect(shadowRay); ok {
				inShadow = true
				break
			}
		}

		diffuse := math.Max(0, normal.Dot(L))
		intensity := ambientIntensity
		if !inShadow {
			intensity += diffuse
		}

		r := float64(floorColor.R) * intensity * lightColor.X
		g_col := float64(floorColor.G) * intensity * lightColor.Y
		b := float64(floorColor.B) * intensity * lightColor.Z

		return color.RGBA{uint8(math.Min(255, r)), uint8(math.Min(255, g_col)), uint8(math.Min(255, b)), 255}
	}

	// Sky Gradient
	t := ray.Dir.Y
	if t < 0 {
		t = 0
	}
	r := uint8(float64(135)*(1-t) + float64(0)*t)
	g_col := uint8(float64(206)*(1-t) + float64(119)*t)
	b := uint8(float64(250)*(1-t) + float64(190)*t)

	return color.RGBA{r, g_col, b, 255}
}

func (g *Game) getFloorColor(ray Ray, t float64) color.RGBA {
	hitPos := ray.Origin.Add(ray.Dir.Mul(t))
	// Checkerboard pattern 1m squares
	check := (int(math.Floor(hitPos.X)) + int(math.Floor(hitPos.Z))) % 2
	if check == 0 {
		return color.RGBA{255, 255, 255, 255} // White
	}
	return color.RGBA{0, 0, 0, 255} // Black
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.width, g.height
}

func main() {
	w, h := 640, 480
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("Simple Ray Tracer")
	game := &Game{
		width:  w,
		height: h,
		camPos: Vec3{0, 1.7, 0},
	}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
