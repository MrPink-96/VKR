package main

import (
	"codeClient/connection"
	rl "github.com/gen2brain/raylib-go/raylib"
	"net"
)

type MenuUI struct {
	publicID string

	menuBG         rl.Texture2D
	backgroundRect rl.Rectangle
	publicIdRect   rl.Rectangle

	exitBtn       *Button
	friendsBtn    *Button
	battleBtn     *Button
	shopBtn       *Button
	listBattleBtn *Button
}

func CreateMenuUI(code string) *MenuUI {
	const (
		menuBGPath = "\\resources\\UI\\Menu\\Background\\Menu.png"

		btnExitPressedPath         = "\\resources\\UI\\Menu\\Button\\Exit\\ExitPressed.png"
		btnExitReleasedPath        = "\\resources\\UI\\Menu\\Button\\Exit\\ExitReleased.png"
		btnFriendsPressedPath      = "\\resources\\UI\\Menu\\Button\\Friends\\FriendsPressed.png"
		btnFriendsReleasedPath     = "\\resources\\UI\\Menu\\Button\\Friends\\FriendsReleased.png"
		btnBattlePressedPath       = "\\resources\\UI\\Menu\\Button\\Battle\\BattlePressed.png"
		btnBattleReleasedPath      = "\\resources\\UI\\Menu\\Button\\Battle\\BattleReleased.png"
		btnShopPressedPath         = "\\resources\\UI\\Menu\\Button\\Shop\\ShopPressed.png"
		btnShopReleasedPath        = "\\resources\\UI\\Menu\\Button\\Shop\\ShopReleased.png"
		btnListBattlesPressedPath  = "\\resources\\UI\\Menu\\Button\\ListBattles\\ListBattlesPressed.png"
		btnListBattlesReleasedPath = "\\resources\\UI\\Menu\\Button\\ListBattles\\ListBattlesReleased.png"
	)
	textureBounds := rl.Rectangle{0, 0, baseWidth, baseHeight}

	backgroundTexture := rl.LoadTexture(currentDirectory + menuBGPath)
	exitBtn := CreateButton(currentDirectory+btnExitPressedPath, currentDirectory+btnExitReleasedPath, textureBounds, rl.Rectangle{26, 25, 53, 54})
	friendsBtn := CreateButton(currentDirectory+btnFriendsPressedPath, currentDirectory+btnFriendsReleasedPath, textureBounds, rl.Rectangle{208, 25, 107, 54})
	battleBtn := CreateButton(currentDirectory+btnBattlePressedPath, currentDirectory+btnBattleReleasedPath, textureBounds, rl.Rectangle{379, 25, 148, 54})
	shopBtn := CreateButton(currentDirectory+btnShopPressedPath, currentDirectory+btnShopReleasedPath, textureBounds, rl.Rectangle{591, 25, 140, 54})
	listBattleBtn := CreateButton(currentDirectory+btnListBattlesPressedPath, currentDirectory+btnListBattlesReleasedPath, textureBounds, rl.Rectangle{795, 25, 109, 54})

	return &MenuUI{
		publicID:       "Код игрока: " + code,
		menuBG:         backgroundTexture,
		backgroundRect: rl.Rectangle{X: 0, Y: 0, Width: float32(backgroundTexture.Width), Height: float32(backgroundTexture.Height)},
		publicIdRect:   rl.Rectangle{X: 1050, Y: 106, Width: 226, Height: 19},
		exitBtn:        exitBtn,
		friendsBtn:     friendsBtn,
		battleBtn:      battleBtn,
		shopBtn:        shopBtn,
		listBattleBtn:  listBattleBtn,
	}
}

func (m *MenuUI) SetTexture(backgroundPath string) {
	newBackground := rl.LoadTexture(backgroundPath)
	if m.menuBG.ID != 0 {
		rl.UnloadTexture(m.menuBG)
	}

	m.menuBG = newBackground
}

func (m *MenuUI) Unload() {
	if m.menuBG.ID != 0 {
		rl.UnloadTexture(m.menuBG)
	}
	m.exitBtn.Unload()
	m.friendsBtn.Unload()
	m.battleBtn.Unload()
	m.shopBtn.Unload()
	m.listBattleBtn.Unload()
}

func (m *MenuUI) Draw(scaleX, scaleY float32) {

	const (
		fontSize    = float32(11)
		fontSpacing = 1
	)

	drawTexture(m.menuBG, m.backgroundRect, m.backgroundRect, scaleX, scaleY)
	drawFieldText(m.publicID, m.publicIdRect, scaleX, scaleY, fontSize, fontSpacing, rl.Red)
	m.exitBtn.Draw(scaleX, scaleY)
	m.friendsBtn.Draw(scaleX, scaleY)
	m.battleBtn.Draw(scaleX, scaleY)
	m.shopBtn.Draw(scaleX, scaleY)
	m.listBattleBtn.Draw(scaleX, scaleY)
}

// Функция обработки ввода
func (m *MenuUI) HandleInput(conn net.Conn, gameState *string) bool {

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

	// Переключение на управление персонажем
	if rl.IsKeyPressed(rl.KeyF9) {
		*gameState = stateCharacterControl
		//autoMesHandle.Resume()
	}

	// Выход из игры
	if m.exitBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			m.exitBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			m.exitBtn.Released()
			return true
		}
	} else if !m.exitBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		m.exitBtn.Released()
	}

	// Просмотр списка друзей
	if m.friendsBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			m.friendsBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			clearChannel(friendsUI.friendsDataCh)

			sendInput(conn, connection.MsgFriendsData, nil)
			m.friendsBtn.Released()
			*gameState = stateListFriends
		}
	} else if !m.friendsBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		m.friendsBtn.Released()
	}

	// Начать сражение
	if m.battleBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			m.battleBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			m.battleBtn.Released()
			*gameState = stateBattle
		}
	} else if !m.battleBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		m.battleBtn.Released()
	}

	// Открыть магазин
	if m.shopBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			m.shopBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			clearChannel(shopUI.shopDataCh)
			clearChannel(shopUI.purchaseReceiptCh)

			sendInput(conn, connection.MsgShopData, nil)
			m.shopBtn.Released()
			*gameState = stateShop
		}
	} else if !m.shopBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		m.shopBtn.Released()
	}

	// Просмотр списка сражений
	if m.listBattleBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			m.listBattleBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			clearChannel(listBattlesUI.battleDataCh)

			sendInput(conn, connection.MsgListBattles, nil)
			m.listBattleBtn.Released()
			*gameState = stateListBattles
		}
	} else if !m.listBattleBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		m.listBattleBtn.Released()
	}

	return false
}
