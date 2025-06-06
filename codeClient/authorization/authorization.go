package authorization

import (
	"codeClient/connection"
	"codeClient/registration"
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Данные для запроса на авторизацию пользователя
type authData struct {
	Login    string `msgpack:"l"`
	Password string `msgpack:"p"`
}

// Authorization выполняет процесс авторизации и возвращает результат
func Authorization() (net.Conn, *connection.UserData, error) {
	rl.SetWindowTitle("Авторизация")

	// Базовые размеры окна относительно которых масштабируется
	const baseWidth = 960
	const baseHeight = 540
	const baseFontSize = 20

	// Путь к текущей директории
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Ошибка при получении текущей директории:", err)
		return nil, nil, fmt.Errorf("Ошибка при получении текущей директории:", err)
	}

	// УБРАТЬ ЕСЛИ EXE ТАМ ЖЕ ГДЕ И ПАПКА RESOURCES
	currentDir = filepath.Dir(currentDir)

	// Загрузка текстур
	backgroundTexture := rl.LoadTexture(currentDir + "\\resources\\UI\\Authorization\\Background\\LoginRegBack.png")
	backgroundLoginTexture := rl.LoadTexture(currentDir + "\\resources\\UI\\Authorization\\Background\\LoginTexture.png")
	entryReleasedTexture := rl.LoadTexture(currentDir + "\\resources\\UI\\Authorization\\Button\\Entry\\EntryReleased.png")
	entryPressedTexture := rl.LoadTexture(currentDir + "\\resources\\UI\\Authorization\\Button\\Entry\\EntryPressed.png")
	regPressedTexture := rl.LoadTexture(currentDir + "\\resources\\UI\\Authorization\\Button\\Registration\\RegistrationPressed.png")
	defer rl.UnloadTexture(backgroundTexture)
	defer rl.UnloadTexture(backgroundLoginTexture)
	defer rl.UnloadTexture(entryReleasedTexture)
	defer rl.UnloadTexture(entryPressedTexture)
	defer rl.UnloadTexture(regPressedTexture)

	// Путь к шрифту
	fontPath := currentDir + "\\resources\\fonts\\PressStart2P-Regular.ttf"
	runes := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдеёжзийклмнопрстуфхцчшщъыьэюя0123456789!@#№$%`~^&*()-=_+[]{};:'\",.<>?/|\\ ")
	font := rl.LoadFontEx(fontPath, 32, runes, int32(len(runes)))
	if font.Texture.ID == 0 {
		fmt.Println("Ошибка загрузки шрифта!")
		return nil, nil, fmt.Errorf("Ошибка загрузки шрифта!")
	}
	defer rl.UnloadFont(font)

	authResultChan := make(chan connection.AuthRegResult, 1) // Канал для получения результата авторизации

	// Поля
	activeField := 0
	loginText := ""
	passwordText := ""
	errorText := ""

	// Базовые размеры UI-элементов
	loginBoxBounds := rl.Rectangle{X: 353, Y: 158, Width: 251, Height: 38}
	passwordBoxBounds := rl.Rectangle{X: 352, Y: 284, Width: 252, Height: 38}
	entryButtonBounds := rl.Rectangle{X: 352, Y: 351, Width: 252, Height: 38}
	registrationButtonBounds := rl.Rectangle{X: 370, Y: 417, Width: 216, Height: 31}

	// Таймеры для контроля стирания
	backspaceTimer := time.Now()
	backspaceInitialDelay := 300 * time.Millisecond // Первая задержка
	backspaceRepeatDelay := backspaceInitialDelay   // Текущая задержка
	backspaceMinDelay := 10 * time.Millisecond      // Минимальная задержка

	// Флаг для отслеживания состояния кнопки
	entryBtPressed := false
	registrationBtPressed := false

	isProcessing := false // Флаг ожидания получения ответа от сервера

	currentWidth := float32(baseWidth)
	currentHeight := float32(baseHeight)

	rl.SetTargetFPS(60)

	// Подключение к серверу
	for !rl.WindowShouldClose() {
		// Пропорциональное масштабирование
		if currentWidth != float32(rl.GetScreenWidth()) || currentHeight != float32(rl.GetScreenHeight()) {
			changeX := math.Abs(float64(float32(rl.GetScreenWidth()) - currentWidth))
			changeY := math.Abs(float64(float32(rl.GetScreenHeight()) - currentHeight))
			var scale float32
			if changeX >= changeY {
				scale = float32(rl.GetScreenWidth()) / currentWidth
			} else {
				scale = float32(rl.GetScreenHeight()) / currentHeight
			}
			rl.SetWindowSize(int(currentWidth*scale), int(currentHeight*scale))
		}

		// Вычисляем текущие размеры окна
		currentWidth = float32(rl.GetScreenWidth())
		currentHeight = float32(rl.GetScreenHeight())

		// Рассчитываем коэффициенты масштабирования
		scaleX := currentWidth / baseWidth
		scaleY := currentHeight / baseHeight

		// Центрирование окна
		offsetX := (currentWidth - baseWidth*scaleX) / 2
		offsetY := (currentHeight - baseHeight*scaleY) / 2

		mousePos := rl.GetMousePosition()

		// Переключение на полноэкранный режим по нажатию клавиши F11
		if rl.IsKeyPressed(rl.KeyF11) {
			if rl.IsWindowFullscreen() {
				rl.ToggleFullscreen()
			} else {
				rl.ToggleFullscreen()

				// Устанавливаем размеры окна и позицию после переключения в полноэкранный режим, чтобы не было белых полос по краям
				screenWidth := rl.GetMonitorWidth(0)
				screenHeight := rl.GetMonitorHeight(0)
				rl.SetWindowSize(screenWidth, screenHeight)
				rl.SetWindowPosition(0, 0) // Устанавливаем окно в верхний левый угол
			}
			rl.ClearBackground(rl.RayWhite)
		}

		// Переключение активного поля
		if rl.IsKeyReleased(rl.KeyTab) {
			activeField = (activeField + 1) % 2
		}
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if rl.CheckCollisionPointRec(mousePos, scaleRectangle(loginBoxBounds, scaleX, scaleY)) {
				activeField = 0
			} else if rl.CheckCollisionPointRec(mousePos, scaleRectangle(passwordBoxBounds, scaleX, scaleY)) {
				activeField = 1
			}
		}

		// Обработка нажатия кнопки войти
		if rl.IsKeyPressed(rl.KeyEnter) || rl.CheckCollisionPointRec(mousePos, scaleRectangle(entryButtonBounds, scaleX, scaleY)) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			entryBtPressed = true
		}
		if entryBtPressed && (rl.IsKeyReleased(rl.KeyEnter) || rl.IsMouseButtonReleased(rl.MouseLeftButton)) {
			entryBtPressed = false
			if !isProcessing {
				isProcessing = true
				authMes, err := connection.CreateMessage(connection.MsgAuthorization, authData{loginText, passwordText})
				if err == nil {
					errorText = "Получение ответа..."
					go func(authMes []byte) {
						conn, data, err := connection.AuthorizationRegistrationToServer(authMes)
						authResultChan <- connection.AuthRegResult{Conn: conn, Data: data, Err: err}
					}(authMes)
				} else {
					isProcessing = false
					errorText = "Ошибка при преобразовании данных для отправки"
				}
			}
		}
		if !(rl.CheckCollisionPointRec(mousePos, scaleRectangle(entryButtonBounds, scaleX, scaleY)) || rl.IsKeyDown(rl.KeyEnter)) {
			entryBtPressed = false
		}

		// Обработка нажатия кнопки зарегистрироваться
		if rl.CheckCollisionPointRec(mousePos, scaleRectangle(registrationButtonBounds, scaleX, scaleY)) {
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				registrationBtPressed = true
			} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) && registrationBtPressed {
				registrationBtPressed = false
				errorText = ""
				conn, data, err := registration.Registration()
				if err == nil {
					return conn, data, nil
				} else if err.Error() == "Пользователь закрыл окно, регистрация не произошла!" {
					return nil, nil, fmt.Errorf("Пользователь закрыл окно, авторизация не произошла!")
				}
				rl.SetWindowTitle("Окно авторизации")
			}
		} else {
			registrationBtPressed = false
		}

		// Обработка ввода текста
		if activeField == 0 {
			loginText, backspaceRepeatDelay = handleTextInput(loginText, &backspaceTimer, backspaceInitialDelay, backspaceRepeatDelay, backspaceMinDelay)
		} else if activeField == 1 {
			passwordText, backspaceRepeatDelay = handleTextInput(passwordText, &backspaceTimer, backspaceInitialDelay, backspaceRepeatDelay, backspaceMinDelay)
		}

		// Проверяем канал на наличие результата
		select {
		case result := <-authResultChan:
			isProcessing = false
			if result.Err != nil {
				errorText = "Ошибка: " + result.Err.Error()
			} else {
				return result.Conn, result.Data, nil
			}
		default:
			// Ничего не делаем, если результата нет
		}

		// Отрисовка
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		// Применяем масштабирование
		rl.BeginScissorMode(int32(offsetX), int32(offsetY), int32(baseWidth*scaleX), int32(baseHeight*scaleY))

		// Рисуем фон
		DrawBackground(backgroundTexture, rl.GetScreenWidth(), rl.GetScreenHeight())      // Фон
		DrawBackground(backgroundLoginTexture, rl.GetScreenWidth(), rl.GetScreenHeight()) // Фон для элементов интерфейса

		// Поле логина
		drawFieldText(font, baseFontSize, scaleRectangle(loginBoxBounds, scaleX, scaleY), scaleX, loginText, loginBoxBounds.Width, activeField == 0)

		// Поле пароля
		hiddenPassword := maskPassword(passwordText)
		drawFieldText(font, baseFontSize, scaleRectangle(passwordBoxBounds, scaleX, scaleY), scaleX, hiddenPassword, passwordBoxBounds.Width, activeField == 1)

		// Отображаем текстуру
		if entryBtPressed {
			rl.DrawTexturePro(entryPressedTexture,
				rl.Rectangle{X: 0, Y: 0, Width: float32(entryPressedTexture.Width), Height: float32(entryPressedTexture.Height)},
				scaleRectangle(entryButtonBounds, scaleX, scaleY),
				rl.Vector2{X: 0, Y: 0}, 0, rl.White)
		} else {
			rl.DrawTexturePro(entryReleasedTexture,
				rl.Rectangle{X: 0, Y: 0, Width: float32(entryReleasedTexture.Width), Height: float32(entryReleasedTexture.Height)},
				scaleRectangle(entryButtonBounds, scaleX, scaleY),
				rl.Vector2{X: 0, Y: 0}, 0, rl.White)
		}

		if registrationBtPressed {
			rl.DrawTexturePro(regPressedTexture,
				rl.Rectangle{X: 0, Y: 0, Width: float32(regPressedTexture.Width), Height: float32(regPressedTexture.Height)},
				scaleRectangle(registrationButtonBounds, scaleX, scaleY),
				rl.Vector2{X: 0, Y: 0}, 0, rl.White)
		}

		// Вывод ошибок
		errorFontSize := 13 * scaleX
		errorTextSize := rl.MeasureTextEx(font, errorText, errorFontSize, 1)
		errorTextX := currentWidth/2 - errorTextSize.X/2
		errorTextY := float32(460) * scaleY

		// Рисуйте текст
		rl.DrawTextEx(font, errorText, rl.Vector2{X: errorTextX, Y: errorTextY}, errorFontSize, 1, rl.Red)
		rl.EndScissorMode()
		rl.EndDrawing()
	}
	return nil, nil, fmt.Errorf("Пользователь закрыл окно, авторизация не произошла!")
}

// Масштабирует прямоугольник
func scaleRectangle(rect rl.Rectangle, scaleX, scaleY float32) rl.Rectangle {
	return rl.Rectangle{
		X:      rect.X * scaleX,
		Y:      rect.Y * scaleY,
		Width:  rect.Width * scaleX,
		Height: rect.Height * scaleY,
	}
}

// Допустимые символы
func isValidInput(input string) bool {
	re := regexp.MustCompile(`^[a-zA-Zа-яА-Я0-9!@#№$%` + "`" + `~^&*()\-_+=\[\]{};:'",.<>?/|\\ ]+$`)
	return re.MatchString(input)
}

// Обрабатывает ввод текста
func handleTextInput(currentText string, timer *time.Time, initialDelay, repeatDelay, minDelay time.Duration) (string, time.Duration) {
	if rl.IsKeyDown(rl.KeyBackspace) && len(currentText) > 0 {
		elapsed := time.Since(*timer)
		if elapsed >= repeatDelay {
			// Стирает символ и преобразуем строку в массив рун для корректного обрезания
			runes := []rune(currentText)
			if len(runes) > 0 {
				currentText = string(runes[:len(runes)-1])
			}

			// Ускоряем стирание
			if repeatDelay > minDelay {
				repeatDelay -= 50 * time.Millisecond
			}
			*timer = time.Now()
		}
	} else if rl.IsKeyReleased(rl.KeyBackspace) {
		// Сбрасывает задержку
		repeatDelay = initialDelay
	}

	// Корректность ввода символов
	enteredChar := rl.GetCharPressed()
	for enteredChar > 0 {
		if isValidInput(string(enteredChar)) && len(currentText) < 128 {
			currentText += string(enteredChar)
		}
		enteredChar = rl.GetCharPressed()
	}
	return currentText, repeatDelay
}

// Рисует текст в поле с учетом масштаба и положения
func drawFieldText(font rl.Font, baseFontSize float32, bounds rl.Rectangle, scaleX float32, text string, maxWidth float32, isActive bool) {
	scaledFontSize := baseFontSize * scaleX
	displayText := trimTextToFit(font, baseFontSize, text, maxWidth)

	textBounds := rl.MeasureTextEx(font, displayText, scaledFontSize, 1)
	textX := bounds.X + (bounds.Width-textBounds.X)/2
	textY := bounds.Y + (bounds.Height-textBounds.Y)/2

	rl.DrawTextEx(font, displayText, rl.Vector2{X: textX, Y: textY}, scaledFontSize, 1, rl.Color{177, 181, 193, 255})
	drawFieldBorder(bounds, isActive)
}

// Обрезает текст, чтобы он помещался в указанную ширину
func trimTextToFit(font rl.Font, baseFontSize float32, text string, maxWidth float32) string {
	textWidth := rl.MeasureTextEx(font, text, baseFontSize, 2).X
	if textWidth <= maxWidth {
		return text
	}
	for len(text) > 0 {
		text = text[1:]
		textWidth = rl.MeasureTextEx(font, text, baseFontSize, 2).X
		if textWidth <= maxWidth {
			break
		}
	}
	return text
}

// Скрывает текст пароля звёздочками
func maskPassword(password string) string {
	return strings.Repeat("*", len([]rune(password)))
}

// Рисует рамку для поля ввода
func drawFieldBorder(bounds rl.Rectangle, isActive bool) {
	color := rl.Gray
	if isActive {
		color = rl.Blue
	}
	rl.DrawRectangleLinesEx(bounds, 2, color)
}

func DrawBackground(texture rl.Texture2D, screenWidth, screenHeight int) {
	rl.DrawTexturePro(
		texture,
		rl.Rectangle{X: 0, Y: 0, Width: float32(texture.Width), Height: float32(texture.Height)}, // Исходная область
		rl.Rectangle{X: 0, Y: 0, Width: float32(screenWidth), Height: float32(screenHeight)},     // Область экрана
		rl.Vector2{X: 0, Y: 0}, // Центр поворота
		0,                      // Угол поворота
		rl.White,               // Цвет
	)
}
