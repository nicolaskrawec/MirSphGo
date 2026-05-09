package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	Epsilon  = 0.001
	MaxDepth = 4
)

// =====================
// Math / Vector
// =====================

type Vec3 struct {
	X, Y, Z float64
}

func V3(x, y, z float64) Vec3 {
	return Vec3{x, y, z}
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z}
}

func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z}
}

func (v Vec3) Mul(s float64) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

func (v Vec3) Hadamard(o Vec3) Vec3 {
	return Vec3{v.X * o.X, v.Y * o.Y, v.Z * o.Z}
}

func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

func (v Vec3) Normalize() Vec3 {
	l := v.Length()
	if l == 0 {
		return Vec3{}
	}
	return v.Mul(1 / l)
}

func (v Vec3) Reflect(normal Vec3) Vec3 {
	return v.Sub(normal.Mul(2 * v.Dot(normal)))
}

func Lerp(a, b Vec3, t float64) Vec3 {
	return a.Mul(1 - t).Add(b.Mul(t))
}

func Clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func ToRGBA(c Vec3) color.RGBA {
	return color.RGBA{
		R: uint8(Clamp01(c.X) * 255),
		G: uint8(Clamp01(c.Y) * 255),
		B: uint8(Clamp01(c.Z) * 255),
		A: 255,
	}
}

// =====================
// Ray
// =====================

type Ray struct {
	Origin Vec3
	Dir    Vec3
}

func (r Ray) At(t float64) Vec3 {
	return r.Origin.Add(r.Dir.Mul(t))
}

// =====================
// Material
// =====================

type CheckerPattern struct {
	ColorA Vec3
	ColorB Vec3
	Scale  float64
}

type Material struct {
	Albedo       Vec3
	Reflectivity float64

	Specular  float64
	Shininess float64

	Checker *CheckerPattern
}

func (m Material) ColorAt(pos Vec3) Vec3 {
	if m.Checker == nil {
		return m.Albedo
	}

	scale := m.Checker.Scale
	if scale <= 0 {
		scale = 1
	}

	x := int(math.Floor(pos.X * scale))
	z := int(math.Floor(pos.Z * scale))

	if (x+z)%2 == 0 {
		return m.Checker.ColorA
	}

	return m.Checker.ColorB
}

// =====================
// Geometry
// =====================

type HitRecord struct {
	T        float64
	Position Vec3
	Normal   Vec3
	Material Material
}

type Object interface {
	Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool)
}

type Sphere struct {
	Center   Vec3
	Radius   float64
	Material Material
}

func (s Sphere) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	oc := ray.Origin.Sub(s.Center)

	a := ray.Dir.Dot(ray.Dir)
	b := 2.0 * oc.Dot(ray.Dir)
	c := oc.Dot(oc) - s.Radius*s.Radius

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return HitRecord{}, false
	}

	sqrtD := math.Sqrt(discriminant)

	t := (-b - sqrtD) / (2 * a)
	if t < tMin || t > tMax {
		t = (-b + sqrtD) / (2 * a)
		if t < tMin || t > tMax {
			return HitRecord{}, false
		}
	}

	pos := ray.At(t)
	normal := pos.Sub(s.Center).Normalize()

	return HitRecord{
		T:        t,
		Position: pos,
		Normal:   normal,
		Material: s.Material,
	}, true
}

type Plane struct {
	Point    Vec3
	Normal   Vec3
	Material Material
}

func (p Plane) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	n := p.Normal.Normalize()
	denom := n.Dot(ray.Dir)

	if math.Abs(denom) < 1e-6 {
		return HitRecord{}, false
	}

	t := p.Point.Sub(ray.Origin).Dot(n) / denom

	if t < tMin || t > tMax {
		return HitRecord{}, false
	}

	pos := ray.At(t)

	// On s'assure que la normale regarde contre le rayon.
	if n.Dot(ray.Dir) > 0 {
		n = n.Mul(-1)
	}

	return HitRecord{
		T:        t,
		Position: pos,
		Normal:   n,
		Material: p.Material,
	}, true
}

// =====================
// Scene / Light
// =====================

type PointLight struct {
	Position  Vec3
	Color     Vec3
	Intensity float64
}

type Scene struct {
	Objects []Object
	Light   PointLight
	Ambient Vec3
}

func (s Scene) Intersect(ray Ray, tMin, tMax float64) (HitRecord, bool) {
	closest := tMax
	var bestHit HitRecord
	hitAnything := false

	for _, obj := range s.Objects {
		if hit, ok := obj.Intersect(ray, tMin, closest); ok {
			closest = hit.T
			bestHit = hit
			hitAnything = true
		}
	}

	return bestHit, hitAnything
}

func (s Scene) IsInShadow(point, normal Vec3) bool {
	toLight := s.Light.Position.Sub(point)
	lightDistance := toLight.Length()
	lightDir := toLight.Normalize()

	shadowRay := Ray{
		Origin: point.Add(normal.Mul(Epsilon)),
		Dir:    lightDir,
	}

	_, hit := s.Intersect(shadowRay, Epsilon, lightDistance-Epsilon)
	return hit
}

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

	mx, my := ebiten.CursorPosition()

	if g.hasMouse {
		dx := mx - g.prevMouseX
		dy := my - g.prevMouseY

		g.camYaw += float64(dx) * mouseSens
		g.camPitch += float64(dy) * mouseSens

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

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.pixels == nil {
		g.pixels = make([]byte, g.width*g.height*4)
	}

	g.render()

	screen.WritePixels(g.pixels)

	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf(
			"FPS: %.2f\nRender: %dx%d\nObjects: %d",
			ebiten.ActualFPS(),
			g.width,
			g.height,
			len(g.scene.Objects),
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

	// Direction locale avant rotation.
	local := V3(px, py, 1)

	// Rotation pitch autour de X.
	y1 := local.Y*cosPitch - local.Z*sinPitch
	z1 := local.Y*sinPitch + local.Z*cosPitch

	// Rotation yaw autour de Y.
	dirX := local.X*cosYaw + z1*sinYaw
	dirY := y1
	dirZ := -local.X*sinYaw + z1*cosYaw

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

	light := g.scene.Light
	ambient := baseColor.Hadamard(g.scene.Ambient)

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

	if material.Reflectivity <= 0 {
		return localColor
	}

	reflectDir := ray.Dir.Reflect(hit.Normal).Normalize()
	reflectRay := Ray{
		Origin: hit.Position.Add(hit.Normal.Mul(Epsilon)),
		Dir:    reflectDir,
	}

	reflected := g.trace(reflectRay, depth+1)

	return localColor.Mul(1 - material.Reflectivity).
		Add(reflected.Mul(material.Reflectivity))
}

func skyColor(ray Ray) Vec3 {
	t := Clamp01(ray.Dir.Y*0.5 + 0.5)

	horizon := V3(0.53, 0.80, 0.98)
	zenith := V3(0.02, 0.20, 0.55)

	return Lerp(horizon, zenith, t)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.width, g.height
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
		Reflectivity: rand.Float64() * 0.45,
		Specular:     rand.Float64() * 0.8,
		Shininess:    16 + rand.Float64()*96,
	}
}

func createScene() Scene {
	objects := make([]Object, 0)

	floorMaterial := Material{
		Albedo:       V3(0.8, 0.8, 0.8),
		Reflectivity: 0.05,
		Specular:     0.1,
		Shininess:    16,
		Checker: &CheckerPattern{
			ColorA: V3(0.85, 0.85, 0.85),
			ColorB: V3(0.12, 0.12, 0.12),
			Scale:  1,
		},
	}

	objects = append(objects, Plane{
		Point:    V3(0, 0, 0),
		Normal:   V3(0, 1, 0),
		Material: floorMaterial,
	})

	for i := 0; i < 10; i++ {
		radius := rand.Float64()*0.35 + 0.15

		sphere := Sphere{
			Center: V3(
				rand.Float64()*6-3,
				radius,
				rand.Float64()*5+2,
			),
			Radius:   radius,
			Material: randomMaterial(),
		}

		objects = append(objects, sphere)
	}

	// Une grosse sphère plus visible.
	objects = append(objects, Sphere{
		Center: V3(0, 1, 4),
		Radius: 1,
		Material: Material{
			Albedo:       V3(0.9, 0.15, 0.1),
			Reflectivity: 0.25,
			Specular:     0.8,
			Shininess:    64,
		},
	})

	return Scene{
		Objects: objects,
		Light: PointLight{
			Position:  V3(5, 8, 2),
			Color:     V3(1.0, 0.95, 0.82),
			Intensity: 1.2,
		},
		Ambient: V3(0.08, 0.08, 0.1),
	}
}

// =====================
// Main
// =====================

func main() {
	rand.Seed(time.Now().UnixNano())

	// Résolution réelle de rendu.
	// Tu peux la baisser pour accélérer le raytracer.
	renderW, renderH := 512, 384

	// Taille de fenêtre.
	// Ebiten va scaler la résolution logique vers cette taille.
	windowW, windowH := 1024, 768

	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowTitle("Scalable Ray Tracer")

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
