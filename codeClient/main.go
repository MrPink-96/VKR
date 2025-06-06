package main

import (
	"codeClient/connection"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
)

const (
	// Базовые размеры окна и текста относительно которых масштабируется. Их менять нельзя, иначе всё поедет
	baseWidth             = 1280
	baseHeight            = 720
	fontPath              = "\\resources\\fonts\\PressStart2P-Regular.ttf"
	stateMenu             = "Menu"
	stateListFriends      = "ListFriends"
	stateShop             = "Shop"
	stateBattle           = "Battle"
	stateListBattles      = "ListBattles"
	stateCharacterControl = "CharacterControl"
)

// Глобальные переменные
var (
	menuUI           *MenuUI
	friendsUI        *FriendsUI
	friendlyFightUI  *FriendlyFightUI
	battleUI         *BattleUI
	shopUI           *ShopUI
	listBattlesUI    *ListBattlesUI
	messageServerCh  = make(chan connection.Message)
	isConnected      bool
	currentDirectory string
	font             rl.Font
)

// Структура для управления автоматическим обработчиком сообщений
type AutoMessageHandler struct {
	ctx       context.Context
	cancel    context.CancelFunc
	pauseChan chan struct{}
	isPaused  atomic.Bool
}

func main() {
	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(baseWidth, baseHeight, "Дуэль клинков")
	rl.SetWindowMinSize(640, 360)
	rl.SetExitKey(rl.KeyNull) // Клавиша завершения работы
	defer rl.CloseWindow()
	// Путь к текущей директории
	var err error
	currentDirectory, err = os.Getwd()
	if err != nil {
		fmt.Println("Ошибка при получении текущей директории: ", err)
		return
	}

	// УБРАТЬ ЕСЛИ EXE ТАМ ЖЕ ГДЕ И ПАПКА RESOURCES
	currentDirectory = filepath.Dir(currentDirectory)

	// Путь к шрифту
	runes := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдеёжзийклмнопрстуфхцчшщъыьэюя0123456789…!@#№$%`~^&*()-=_+[]{};:'\",.<>?/|\\ ")
	font = rl.LoadFontEx(currentDirectory+fontPath, 32, runes, int32(len(runes)))
	if font.Texture.ID == 0 {
		fmt.Println("Ошибка загрузки шрифта: ", err)
		return
	}
	defer rl.UnloadFont(font)

	// Выполнение авторизации до запуска главного цикла
	conn, data, err := authorization.Authorization()
	if err != nil {
		fmt.Println("Авторизация не удалась. Завершение программы. Ошибка: ", err)
		return
	}
	isConnected = true
	defer connection.CloseConnection(conn)
	rl.SetWindowTitle("Дуэль клинков")

	player := CreatePlayer(data)
	defer player.UnloadBackground()

	player.character.StartPhysics(conn)
	defer player.character.UnloadTextures()
	defer player.character.StopPhysics()
	defer player.character.StopAnimation()

	// Меню
	menuUI = CreateMenuUI(player.publicID)
	defer menuUI.Unload()

	// Список друзей
	friendsUI = CreateFriendsUI()
	defer friendsUI.Unload()

	// Приглашение на бой
	friendlyFightUI = CreateFriendlyFightUI()
	defer friendlyFightUI.Unload()

	// Бой
	battleUI = CreateBattleUI()
	defer battleUI.Unload()

	// Магазин
	shopUI = CreateShopUI()
	defer shopUI.Unload()

	// Список сражений
	listBattlesUI = CreateListBattlesUI()
	defer listBattlesUI.Unload()

	currentWidth := float32(baseWidth)
	currentHeight := float32(baseHeight)

	gameState := stateMenu

	// Автоматическая обработка входящих сообщений, необходима в случае управления персонажем
	go listenServer(conn)
	autoMesHandle := &AutoMessageHandler{}
	autoMesHandle.Start(player)
	defer autoMesHandle.Stop()

	// Основной цикл игры
	rl.SetTargetFPS(360)

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

		// Рассчитываем коэффициенты масштабирования относительно базовых размеров
		scaleX := currentWidth / baseWidth
		scaleY := currentHeight / baseHeight

		// Центрирование окна
		offsetX := (currentWidth - baseWidth*scaleX) / 2
		offsetY := (currentHeight - baseHeight*scaleY) / 2

		// Обработка нажатий кнопок
		if friendlyFightUI.state == waitingInvitation {
			switch gameState {
			case stateMenu:
				exit := menuUI.HandleInput(conn, &gameState)
				if exit {
					return
				}
			case stateCharacterControl:
				player.character.Control(conn, &gameState)
			case stateListFriends:
				friendsUI.HandleInput(conn, player, &gameState)
			case stateBattle:
				battleUI.HandelInput(conn, player, &gameState)
			case stateShop:
				shopUI.HandleInput(conn, player, &gameState)
			case stateListBattles:
				listBattlesUI.HandelInput(&gameState)
			}
		}

		friendlyFightUI.HandleInput(conn, player, &gameState)

		// Отрисовка
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)
		rl.BeginScissorMode(int32(offsetX), int32(offsetY), int32(baseWidth*scaleX), int32(baseHeight*scaleY))

		DrawBackground(player.background, currentWidth, currentHeight) // Фон

		switch gameState {
		case stateMenu:
			menuUI.Draw(scaleX, scaleY)
		case stateListFriends:
			friendsUI.Draw(scaleX, scaleY)
		case stateBattle:
			battleUI.Draw(player, scaleX, scaleY)
		case stateShop:
			shopUI.Draw(player.money, scaleX, scaleY)
		case stateListBattles:
			listBattlesUI.Draw(scaleX, scaleY)
		}

		friendlyFightUI.Draw(scaleX, scaleY)

		player.character.Draw(scaleX, scaleY)

		rl.EndScissorMode()
		rl.EndDrawing()
	}
}

// Функция безопасной отправки на сервер при параллельной обработке
func sendInput(conn net.Conn, dataType connection.MessageType, data interface{}) {
	if isConnected {
		var netErr net.Error
		msg, err := connection.CreateMessage(dataType, data)
		err = connection.SendMessage(conn, msg)
		if errors.Is(err, io.EOF) || errors.As(err, &netErr) {
			log.Println("Соединение неожиданно закрыто:", err)
			isConnected = false
		} else if err != nil {
			log.Println("Ошибка отправки:", err)
		}

	}
}

// Непрерывное получение сообщений от сервера
func listenServer(conn net.Conn) {
	var netErr net.Error
	for {
		msg, err := connection.GetMessage(conn, nil)
		if err != nil {
			// EOF: клиент явно закрыл соединение
			if errors.Is(err, io.EOF) {
				log.Println("Соединение закрыто клиентом:", err)
				isConnected = false
				return
			}

			// Оборачиваемая сетевая ошибка
			if errors.As(err, &netErr) {
				if netErr.Timeout() {
					log.Println("Таймаут соединения, разрыв:", err)
					isConnected = false
					return
				}
				if !netErr.Temporary() {
					log.Println("Постоянная сетевая ошибка:", err)
					isConnected = false
					return
				}

				log.Println("Ошибка получения сообщения:", err)
				continue
			}
		}

		if msg.Type == connection.MsgPing || msg.Type == connection.MsgPong {
			continue
		}

		messageServerCh <- msg
	}
}

// Функция для запуска горутины автоматической обработки сообщений
func (h *AutoMessageHandler) Start(player *Player) {
	if h.ctx != nil {
		fmt.Println("Горутина уже запущена")
		return
	}
	h.ctx, h.cancel = context.WithCancel(context.Background())
	h.pauseChan = make(chan struct{})
	h.isPaused.Store(false)

	go h.handleServerMessages(player)
	log.Println("Горутина запущена")
}

// Функция для остановки горутины автоматической обработки сообщений
func (h *AutoMessageHandler) Stop() {
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
		h.ctx = nil
	}
}

// Функци для паузы и возобновления автоматической обработки сообщений
func (h *AutoMessageHandler) Pause() {
	if !h.isPaused.Load() {
		h.isPaused.Store(true)
		h.pauseChan <- struct{}{} // Отправляем сигнал паузы
	}
}
func (h *AutoMessageHandler) Resume() {
	if h.isPaused.Load() {
		h.isPaused.Store(false)
		h.pauseChan <- struct{}{} // Отправляем сигнал возобновления
	}
}

// Функция автоматической обработки сообщений
func (h *AutoMessageHandler) handleServerMessages(player *Player) {
	for {
		select {
		case <-h.ctx.Done():
			log.Println("Горутина автоматического прослушивания сервера завершена")
			return
		case <-h.pauseChan:
			log.Println("Горутина на паузе...")
			<-h.pauseChan
			log.Println("Горутина возобновлена")

		case msg, ok := <-messageServerCh: // Проверку на закрытие канала
			if !ok {
				log.Println("Канал сообщений закрыт, завершение горутины")
				return
			}
			switch msg.Type {
			case connection.MsgActionCharacter:
				var response ActionResult
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных во время управленяи персонажем: %v", err)
					break
				}
				// Обновляем положение персонажа
				player.character.ReplayCommands(response)

			case connection.MsgFriendsData:
				var response FriendsData
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных друзей: %v", err)
					break
				}

				select {
				case friendsUI.friendsDataCh <- response:
					log.Println("Данные друзей доставлены.")
				default:
					log.Println("Данные друзей отброшены, нет получателя.")
				}

			case connection.MsgAddFriend:
				var response string
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных на добавление в друзья: %v", err)
					break
				}
				friendsUI.addFriendResponse = response

			case connection.MsgChallengeToFight:
				var response FriendEntry
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных на участие в дружеской схватке: %v", err)
					break
				}

				select {
				case friendlyFightUI.friendCh <- &response:
					log.Println("Данные друга на участие в дружеской схватке доставлены.")
				default:
					log.Println("Данные друга на участие в дружеской схватке отброшены, нет получателя.")
				}

			case connection.MsgRefuseChallengeToFight:
				var response string
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных об отказе в дружеской схватке: %v", err)
					break
				}

				select {
				case battleUI.currentBattle.exit <- response:
					log.Println("Отказ друга на участие в дружеской схватке доставлен.")
				default:
					log.Println("Отказ друга на участие в дружеской схватке отброшен, нет получателя.")
				}

			case connection.MsgWaitingBattle:
				var response ActionResult
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных ожидания боя: %v", err)
					break
				}

				battleUI.currentBattle.waiting <- struct{}{}

			case connection.MsgExitBattle:
				select {
				case battleUI.currentBattle.exit <- "":
					log.Println("Выход из боя доставлен.")
				default:
					log.Println("Выход из боя отброшен, нет получателя.")
				}

			case connection.MsgStartBattleInfo:
				var response StartBattleInfo
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных начала боя и оппонента: %v", err)
					break
				}

				battleUI.currentBattle.start <- response

			case connection.MsgEndBattle:
				var response EndBattleInfo
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных конца боя: %v", err)
					break
				}

				battleUI.currentBattle.end <- response

			case connection.MsgShopData:
				var response ShopData
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных магазина: %v", err)
					break
				}
				select {
				case shopUI.shopDataCh <- response:
					log.Println("Данные магазина доставлены.")
				default:
					log.Println("Данные магазина отброшены, нет получателя.")
				}

			case connection.MsgMoneyUpdate:
				var money int
				err := msgpack.Unmarshal(msg.Data, &money)
				if err != nil {
					log.Printf("Ошибка при десериализации данных обновления монет: %v", err)
					break
				}
				player.money = money

			case connection.MsgPurchaseReceipt:
				var purchaseReceipt PurchaseReceipt
				err := msgpack.Unmarshal(msg.Data, &purchaseReceipt)
				if err != nil {
					log.Printf("Ошибка при десериализации данных чека покупки: %v", err)
					break
				}
				select {
				case shopUI.purchaseReceiptCh <- purchaseReceipt:
					log.Println("Данные чека покупки доставлены.")
				default:
					log.Println("Данные чека покупки отброшены, нет получателя.")
				}

			case connection.MsgSelectBackground:
				var assetPath string
				err := msgpack.Unmarshal(msg.Data, &assetPath)
				if err != nil {
					log.Printf("Ошибка при десериализации данных выбора фона: %v", err)
					break
				}
				select {
				case shopUI.updateBackgroundCh <- assetPath:
					log.Println("Данные обновления фона доставлены.")
				default:
					log.Println("Данные обновления фона отброшены, нет получателя.")
				}

			case connection.MsgSelectCharacter:
				var characterData connection.CharacterData
				err := msgpack.Unmarshal(msg.Data, &characterData)
				if err != nil {
					log.Printf("Ошибка при десериализации данных выбора персонажа: %v", err)
					break
				}
				select {
				case shopUI.updateCharacterCh <- characterData:
					log.Println("Данные обновления персонажа доставлены.")
				default:
					log.Println("Данные обновления персонажа отброшены, нет получателя.")
				}

			case connection.MsgListBattles:
				var response BattleData
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации данных списка сражений: %v", err)
					break
				}
				select {
				case listBattlesUI.battleDataCh <- response:
					log.Println("Данные списка сражений доставлены.")
				default:
					log.Println("Данные списка сражений отброшены, нет получателя.")
				}

			case connection.MsgActionOpponent:
				var response ActionResult
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации действий оппонента: %v", err)
					break
				}
				if battleUI.currentBattle.opponent != nil {
					battleUI.currentBattle.opponent.character.ChangeState(response)
				}

			case connection.MsgHealthUpdate:
				var response connection.HealthUpdate
				err := msgpack.Unmarshal(msg.Data, &response)
				if err != nil {
					log.Printf("Ошибка при десериализации обновления здоровья: %v", err)
					break
				}
				if response.Who == connection.MsgActionCharacter {
					player.character.HealthUpdate(response.Health)
					fmt.Println("player.character ", player.character.health)
				} else {
					battleUI.currentBattle.opponent.character.HealthUpdate(response.Health)
					fmt.Println("opponent.character ", battleUI.currentBattle.opponent.character.health)
				}

			case connection.MsgError:
				var errInf string
				err := msgpack.Unmarshal(msg.Data, &errInf)
				if err != nil {
					log.Printf("Ошибка при десериализации данных ошибки: %v", err)
					break
				}
				log.Println("Ошибка: ", errInf)
			default:
				log.Println("Неизвестный код типа сообщения: ", msg.Type)
			}

		}

	}

}

func DrawBackground(texture rl.Texture2D, screenWidth, screenHeight float32) {
	rl.DrawTexturePro(
		texture,
		rl.Rectangle{X: 0, Y: 0, Width: float32(texture.Width), Height: float32(texture.Height)}, // Исходная область
		rl.Rectangle{X: 0, Y: 0, Width: screenWidth, Height: screenHeight},                       // Область экрана
		rl.Vector2{X: 0, Y: 0}, // Центр поворота
		0,                      // Угол поворота
		rl.White,               // Цвет
	)
}
