package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"regexp"
	"time"
)

// Допустимые символы
func isValidInput(input string) bool {
	re := regexp.MustCompile(`^[a-zA-Zа-яА-Я0-9!@#№$%` + "`" + `~^&*()\-_+=\[\]{};:'",.<>?/|\\ ]+$`)
	return re.MatchString(input)
}

// Рисует текстуру
func drawTexture(texture rl.Texture2D, sourceRec, destRec rl.Rectangle, scaleX, scaleY float32) {
	rl.DrawTexturePro(
		texture,
		sourceRec,
		rl.Rectangle{X: destRec.X * scaleX, Y: destRec.Y * scaleY, Width: destRec.Width * scaleX, Height: destRec.Height * scaleY},
		rl.Vector2{X: 0, Y: 0},
		0,
		rl.White,
	)
}

// Приводит Duration к виду mm:ss
func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// Обрезает текст конец текста, добавляя многоточие, чтобы он помещался в указанную ширину
func TrimTextWithEllipsis(fontSize, spacing float32, text string, maxWidth float32) string {
	textWidth := rl.MeasureTextEx(font, text, fontSize, spacing).X
	if textWidth <= maxWidth {
		return text
	}

	for len(text) > 0 {
		text = text[:len(text)-1] // Убираем последний символ
		textWidth = rl.MeasureTextEx(font, text, fontSize, spacing).X
		if textWidth <= maxWidth {
			break
		}
	}
	text = text[:len(text)-1] + "…"

	return text
}

// Обрезает начало текст, чтобы он помещался в указанную ширину
func TrimTextFromStart(fontSize, spacing float32, text string, maxWidth float32) string {
	textWidth := rl.MeasureTextEx(font, text, fontSize, spacing).X
	if textWidth <= maxWidth {
		return text
	}
	for len(text) > 0 {
		text = text[1:] // Убираем первый символ
		textWidth = rl.MeasureTextEx(font, text, fontSize, spacing).X
		if textWidth <= maxWidth {
			break
		}
	}
	return text
}

// Обрабатывает ввод текста
func handleTextInput(text *string, timer *time.Time, repeatDelay *time.Duration, initialDelay, minDelay time.Duration) {
	if rl.IsKeyDown(rl.KeyBackspace) && len(*text) > 0 {
		elapsed := time.Since(*timer)
		if elapsed >= *repeatDelay {
			runes := []rune(*text)
			if len(runes) > 0 {
				*text = string(runes[:len(runes)-1])
			}
			// Ускоряем стирание
			if *repeatDelay > minDelay {
				*repeatDelay -= 50 * time.Millisecond
			}
			*timer = time.Now()
		}
	} else if rl.IsKeyReleased(rl.KeyBackspace) {
		// Сбрасывает задержку
		*repeatDelay = initialDelay
	}

	// Корректность ввода символов
	enteredChar := rl.GetCharPressed()
	for enteredChar > 0 {
		if isValidInput(string(enteredChar)) && len(*text) < 128 {
			*text += string(enteredChar)
		}
		enteredChar = rl.GetCharPressed()
	}
}

// Рисует текст в поле с учетом масштаба и положения
func drawFieldText(text string, bounds rl.Rectangle, scaleX, scaleY float32, baseFontSize, baseSpacing float32, color rl.Color) {
	bounds = rl.Rectangle{X: scaleX * bounds.X, Y: scaleY * bounds.Y, Width: scaleX * bounds.Width, Height: scaleY * bounds.Height}

	scaledFontSize := baseFontSize * scaleX
	scaledSpacing := baseSpacing * scaleX
	displayText := TrimTextFromStart(scaledFontSize, scaledSpacing, text, bounds.Width)

	textBounds := rl.MeasureTextEx(font, displayText, scaledFontSize, scaledSpacing)
	textX := bounds.X + (bounds.Width-textBounds.X)/2
	textY := bounds.Y + (bounds.Height-textBounds.Y)/2

	rl.DrawTextEx(font, displayText, rl.Vector2{X: textX, Y: textY}, scaledFontSize, scaledSpacing, color)
}

// Рисует рамку для поля ввода
func drawFieldBorder(bounds rl.Rectangle, scaleX, scaleY float32, isActive bool) {
	bounds = rl.Rectangle{X: scaleX * bounds.X, Y: scaleY * bounds.Y, Width: scaleX * bounds.Width, Height: scaleY * bounds.Height}
	color := rl.Gray
	if isActive {
		color = rl.Blue
	}
	rl.DrawRectangleLinesEx(bounds, 2, color)
}

func OffsetRectY(rect rl.Rectangle, shiftY, offset float32) rl.Rectangle {
	rect.Y += (rect.Height + offset) * shiftY
	return rect
}

func OffsetRectX(rect rl.Rectangle, shiftX, offset float32) rl.Rectangle {
	rect.X += (rect.Width + offset) * shiftX
	return rect
}

func OffsetRect(rect rl.Rectangle, shiftX, shiftY, offsetX, offsetY float32) rl.Rectangle {
	rect = OffsetRectX(rect, shiftX, offsetX)
	return OffsetRectY(rect, shiftY, offsetY)
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func clearChannel[T any](ch chan T) {
	select {
	case <-ch:
	default:
	}
}
