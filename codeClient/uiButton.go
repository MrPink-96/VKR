package main

import rl "github.com/gen2brain/raylib-go/raylib"

type Button struct {
	isPressed       bool
	texturePressed  rl.Texture2D
	textureReleased rl.Texture2D
	visualBounds    rl.Rectangle
	*ClickBounds
}

type ClickBounds struct {
	baseBounds rl.Rectangle
	bounds     rl.Rectangle
}

func CreateButton(pressedPath, releasedPath string, visualBounds, clickBounds rl.Rectangle) *Button {
	texturePressed := rl.LoadTexture(pressedPath)
	textureReleased := rl.LoadTexture(releasedPath)

	return &Button{
		isPressed:       false,
		texturePressed:  texturePressed,
		textureReleased: textureReleased,
		visualBounds:    visualBounds,
		ClickBounds:     createClickBounds(clickBounds),
		//clickBounds:     clickBounds,
	}
}

func (b *Button) Unload() {
	if b.texturePressed.ID != 0 {
		rl.UnloadTexture(b.texturePressed)
	}
	if b.textureReleased.ID != 0 {
		rl.UnloadTexture(b.textureReleased)
	}
}

func (b *Button) Pressed() {
	b.isPressed = true
}
func (b *Button) Released() {
	b.isPressed = false
}

func (b *Button) Draw(scaleX, scaleY float32) {
	var texture rl.Texture2D
	if b.isPressed {
		texture = b.texturePressed
	} else {
		texture = b.textureReleased
	}

	b.Scale(scaleX, scaleY)
	rl.DrawTexturePro(
		texture,
		rl.Rectangle{X: 0, Y: 0, Width: float32(texture.Width), Height: float32(texture.Height)},
		rl.Rectangle{X: b.visualBounds.X * scaleX, Y: b.visualBounds.Y * scaleY, Width: b.visualBounds.Width * scaleX, Height: b.visualBounds.Height * scaleY},
		rl.Vector2{X: 0, Y: 0},
		0,
		rl.White,
	)
}

// Создает область для клика
func createClickBounds(bounds rl.Rectangle) *ClickBounds {
	return &ClickBounds{
		baseBounds: bounds,
		bounds:     bounds,
	}
}

// Масштабирует прямоугольник
func (b *ClickBounds) Scale(scaleX, scaleY float32) {
	b.bounds = rl.Rectangle{X: b.baseBounds.X * scaleX, Y: b.baseBounds.Y * scaleY, Width: b.baseBounds.Width * scaleX, Height: b.baseBounds.Height * scaleY}
}

// Проверяет, есть ли точка, курсор, в области
func (b *ClickBounds) IsHovered() bool {
	return rl.CheckCollisionPointRec(rl.GetMousePosition(), b.bounds)
}
