package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRayShaderCompiles(t *testing.T) {
	if _, err := ebiten.NewShader([]byte(rayShaderSource)); err != nil {
		t.Fatal(err)
	}
}
