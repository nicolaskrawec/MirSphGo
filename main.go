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
	prevMouseX    int
}

func (g *Game) Update() error {
	const moveSpeed = 0.1
	const mouseSens = 0.005

	// Mouse rotation
	mx, _ := ebiten.CursorPosition()
	if g.prevMouseX != 0 {
		deltaX := mx - g.prevMouseX
		g.camYaw += float64(deltaX) * mouseSens
	}
	g.prevMouseX = mx

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
		// Right vector is {cos(yaw), 0, -sin(yaw)}
		// Left is -Right
		g.camPos.X -= math.Cos(g.camYaw) * moveSpeed
		g.camPos.Z += math.Sin(g.camYaw) * moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camPos.X += math.Cos(g.camYaw) * moveSpeed
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
			{Center: Vec3{0, 2, 4}, Radius: 1.0, Color: color.RGBA{255, 255, 255, 255}, Reflectivity: 0.5},
			{Center: Vec3{-2.5, 1.5, 5}, Radius: 0.5, Color: color.RGBA{255, 0, 0, 255}, Reflectivity: 0.3},
			{Center: Vec3{2.5, 2.5, 5}, Radius: 0.5, Color: color.RGBA{0, 255, 0, 255}, Reflectivity: 0.8},
		},
	}

	cosYaw := math.Cos(g.camYaw)
	sinYaw := math.Sin(g.camYaw)

	var wg sync.WaitGroup
	wg.Add(g.height)

	for y := 0; y < g.height; y++ {
		go func(y int) {
			defer wg.Done()
			for x := 0; x < g.width; x++ {
				// Local ray direction
				px := (2*(float64(x)+0.5)/float64(g.width) - 1) * aspectRatio * scale
				py := (1 - 2*(float64(y)+0.5)/float64(g.height)) * scale

				// Rotate direction around Y axis
				dirX := px*cosYaw + sinYaw
				dirY := py
				dirZ := -px*sinYaw + cosYaw

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

	// Determine closest hit
	if closestSphere != nil && (!hitFloor || tMin < tFloor) {
		hitPos := ray.Origin.Add(ray.Dir.Mul(tMin))
		normal := hitPos.Sub(closestSphere.Center).Normalize()

		reflectDir := ray.Dir.Sub(normal.Mul(2 * ray.Dir.Dot(normal))).Normalize()
		reflectRay := Ray{
			Origin: hitPos.Add(normal.Mul(0.001)),
			Dir:    reflectDir,
		}

		reflectedCol := g.trace(reflectRay, scene, depth+1)

		// Lighting at hit point
		lightDir := Vec3{0.5, 1, 0.5}.Normalize()
		ambient := 0.2
		diffuse := math.Max(0, normal.Dot(lightDir))
		
		// Shadows
		shadowRay := Ray{
			Origin: hitPos.Add(normal.Mul(0.001)),
			Dir:    lightDir,
		}
		inShadow := false
		for i := range scene.Spheres {
			if _, ok := scene.Spheres[i].Intersect(shadowRay); ok {
				inShadow = true
				break
			}
		}
		
		lightIntensity := ambient
		if !inShadow {
			lightIntensity += diffuse
		}
		if lightIntensity > 1.0 {
			lightIntensity = 1.0
		}

		// Combine sphere color and reflection
		r := (float64(closestSphere.Color.R)*(1-closestSphere.Reflectivity) + float64(reflectedCol.R)*closestSphere.Reflectivity) * lightIntensity
		g_col := (float64(closestSphere.Color.G)*(1-closestSphere.Reflectivity) + float64(reflectedCol.G)*closestSphere.Reflectivity) * lightIntensity
		b := (float64(closestSphere.Color.B)*(1-closestSphere.Reflectivity) + float64(reflectedCol.B)*closestSphere.Reflectivity) * lightIntensity

		return color.RGBA{uint8(r), uint8(g_col), uint8(b), 255}
	}

	if hitFloor {
		hitPos := ray.Origin.Add(ray.Dir.Mul(tFloor))
		floorColor := g.getFloorColor(ray, tFloor)
		
		normal := Vec3{0, 1, 0}
		lightDir := Vec3{0.5, 1, 0.5}.Normalize()
		ambient := 0.2
		diffuse := math.Max(0, normal.Dot(lightDir))
		
		// Shadows on floor
		shadowRay := Ray{
			Origin: hitPos.Add(normal.Mul(0.001)),
			Dir:    lightDir,
		}
		inShadow := false
		for i := range scene.Spheres {
			if _, ok := scene.Spheres[i].Intersect(shadowRay); ok {
				inShadow = true
				break
			}
		}
		
		lightIntensity := ambient
		if !inShadow {
			lightIntensity += diffuse
		}
		if lightIntensity > 1.0 {
			lightIntensity = 1.0
		}
		
		return color.RGBA{
			R: uint8(float64(floorColor.R) * lightIntensity),
			G: uint8(float64(floorColor.G) * lightIntensity),
			B: uint8(float64(floorColor.B) * lightIntensity),
			A: 255,
		}
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
