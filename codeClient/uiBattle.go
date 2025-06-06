package main

import (
	"codeClient/connection"
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"log"
	"math"
	"net"
	"time"
)

type friendlyFightState uint8

type BattleResult int8

type battleState uint8

const (
	waitingInvitation friendlyFightState = iota
	showInvitation
)

const (
	NoBattle BattleResult = -2 // Боя не было
	Defeat   BattleResult = -1 // Поражение
	Draw     BattleResult = 0  // Ничья
	Victory  BattleResult = 1  // Победа
)

const (
	BattleMode battleState = iota + 1
	BattleSearch
	InBattle
	PostBattle
)

type StartBattleInfo struct {
	Timestamp         int64                    `msgpack:"tt"` // Время на сервере, когда событие было обработано
	StartTime         int64                    `msgpack:"st"`
	EndTime           int64                    `msgpack:"et"`
	OpponentPublicID  string                   `msgpack:"oi"`
	OpponentName      string                   `msgpack:"on"`
	OpponentRank      int                      `msgpack:"or"`
	OpponentLevel     int                      `msgpack:"ol"`
	OpponentCharacter connection.CharacterData `msgpack:"oc"`
}

type EndBattleInfo struct {
	Result       BattleResult `msgpack:"w,omitempty"`
	TotalMoney   int          `msgpack:"m"` // Итоговый баланс после боя
	CurrentRank  int          `msgpack:"r"` // Текущий ранг после боя
	CurrentLevel int          `msgpack:"l"` // Текущий уровень после боя
	UpdatedStats ActionResult `msgpack:"u"`
}

type resultBattleUI struct {
	result           BattleResult
	levelDelta       int
	rankDelta        int
	moneyDelta       int
	opponentPublicID string
	opponentName     string
	opponentLevel    int
	opponentRank     int
}

type FriendlyFightUI struct {
	state    friendlyFightState
	friend   *FriendEntry
	friendCh chan *FriendEntry

	friendlyFightBG rl.Texture2D
	textRect        rl.Rectangle
	backgroundRect  rl.Rectangle

	acceptBtn *Button
	refuseBtn *Button
}

type Battle struct {
	timeToStart time.Time
	timeToEnd   time.Time
	opponent    *Opponent

	waiting chan struct{}
	start   chan StartBattleInfo
	end     chan EndBattleInfo
	exit    chan string //struct{}
}

type BattleUI struct {
	currentBattle *Battle

	state          battleState
	friendID       string
	pagePostBattle int
	isExiting      bool

	battleModeBG         rl.Texture2D
	battleSearchBG       rl.Texture2D
	inBattleBG           rl.Texture2D
	postBattlePlayerBG   rl.Texture2D
	postBattleOpponentBG rl.Texture2D
	exitDialogBG         rl.Texture2D

	backgroundRect  rl.Rectangle
	timeSearchRect  rl.Rectangle
	timeBattleRect  rl.Rectangle
	healthBarRect   rl.Rectangle
	nameRect        rl.Rectangle
	medallionRect   rl.Rectangle
	postBattle1Rect rl.Rectangle
	postBattle2Rect rl.Rectangle
	postBattle3Rect rl.Rectangle
	postBattle4Rect rl.Rectangle

	levelMatchBtn  *Button
	rankedMatchBtn *Button
	leftBtn        *Button
	rightBtn       *Button
	okBtn          *Button
	yesBtn         *Button
	noBtn          *Button

	timeSearch time.Time

	resultBattle resultBattleUI
}

func CreateFriendlyFightUI() *FriendlyFightUI {
	const (
		friendlyFightBGPath = "\\resources\\UI\\Battle\\Background\\FriendlyFight.png"

		btnAcceptPressedPath  = "\\resources\\UI\\Battle\\Button\\Accept\\AcceptPressed.png"
		btnAcceptReleasedPath = "\\resources\\UI\\Battle\\Button\\Accept\\AcceptReleased.png"
		btnRefusePressedPath  = "\\resources\\UI\\Battle\\Button\\Refuse\\RefusePressed.png"
		btnRefuseReleasedPath = "\\resources\\UI\\Battle\\Button\\Refuse\\RefuseReleased.png"
	)
	backgroundRect := rl.Rectangle{0, 0, baseWidth, baseHeight}

	friendlyFightBG := rl.LoadTexture(currentDirectory + friendlyFightBGPath)
	acceptBtn := CreateButton(currentDirectory+btnAcceptPressedPath, currentDirectory+btnAcceptReleasedPath, backgroundRect, rl.Rectangle{666, 333, 137, 36})
	refuseBtn := CreateButton(currentDirectory+btnRefusePressedPath, currentDirectory+btnRefuseReleasedPath, backgroundRect, rl.Rectangle{477, 333, 137, 36})

	return &FriendlyFightUI{
		state:           waitingInvitation,
		friend:          nil, //connection.FriendEntry{},
		friendCh:        make(chan *FriendEntry, 10),
		friendlyFightBG: friendlyFightBG,
		backgroundRect:  backgroundRect,
		textRect:        rl.Rectangle{490, 286, 300, 30},
		acceptBtn:       acceptBtn,
		refuseBtn:       refuseBtn,
	}
}

func CreateBattle() *Battle {
	var battle Battle
	battle.waiting = make(chan struct{})
	battle.start = make(chan StartBattleInfo)
	battle.end = make(chan EndBattleInfo, 1)
	battle.exit = make(chan string, 1) //struct{})
	return &battle
}

func CreateBattleUI() *BattleUI {
	const (
		battleModeBGPath         = "\\resources\\UI\\Battle\\Background\\BattleMode.png"
		battleSearchBGPath       = "\\resources\\UI\\Battle\\Background\\BattleSearch.png"
		inBattleBGPath           = "\\resources\\UI\\Battle\\Background\\InBattleGold.png"
		postBattlePlayerBGPath   = "\\resources\\UI\\Battle\\Background\\PostBattlePlayer.png"
		postBattleOpponentBGPath = "\\resources\\UI\\Battle\\Background\\PostBattleOpponent.png"
		exitDialogBGPath         = "\\resources\\UI\\Battle\\Background\\Exit.png"

		btnLevelPressedPath   = "\\resources\\UI\\Battle\\Button\\LevelMatch\\LevelMatchPressed.png"
		btnLevelReleasedPath  = "\\resources\\UI\\Battle\\Button\\LevelMatch\\LevelMatchReleased.png"
		btnRankedPressedPath  = "\\resources\\UI\\Battle\\Button\\RankedMatch\\RankedMatchPressed.png"
		btnRankedReleasedPath = "\\resources\\UI\\Battle\\Button\\RankedMatch\\RankedMatchReleased.png"
		btnLeftPressedPath    = "\\resources\\UI\\Battle\\Button\\Left\\LeftPressed.png"
		btnLeftReleasedPath   = "\\resources\\UI\\Battle\\Button\\Left\\LeftReleased.png"
		btnRightPressedPath   = "\\resources\\UI\\Battle\\Button\\Right\\RightPressed.png"
		btnRightReleasedPath  = "\\resources\\UI\\Battle\\Button\\Right\\RightReleased.png"
		btnOkPressedPath      = "\\resources\\UI\\Battle\\Button\\Ok\\OkPressed.png"
		btnOkReleasedPath     = "\\resources\\UI\\Battle\\Button\\Ok\\OkReleased.png"
		btnYesPressedPath     = "\\resources\\UI\\Battle\\Button\\Yes\\YesPressed.png"
		btnYesReleasedPath    = "\\resources\\UI\\Battle\\Button\\Yes\\YesReleased.png"
		btnNoPressedPath      = "\\resources\\UI\\Battle\\Button\\No\\NoPressed.png"
		btnNoReleasedPath     = "\\resources\\UI\\Battle\\Button\\No\\NoReleased.png"
	)
	battleModeBG := rl.LoadTexture(currentDirectory + battleModeBGPath)
	battleSearchBG := rl.LoadTexture(currentDirectory + battleSearchBGPath)
	inBattleBG := rl.LoadTexture(currentDirectory + inBattleBGPath)
	postBattlePlayerBG := rl.LoadTexture(currentDirectory + postBattlePlayerBGPath)
	postBattleOpponentBG := rl.LoadTexture(currentDirectory + postBattleOpponentBGPath)
	exitDialogBG := rl.LoadTexture(currentDirectory + exitDialogBGPath)

	textureBounds := rl.Rectangle{0, 0, baseWidth, baseHeight}

	levelMatchBtn := CreateButton(currentDirectory+btnLevelPressedPath, currentDirectory+btnLevelReleasedPath, textureBounds, rl.Rectangle{373, 25, 215, 54})
	rankedMatchBtn := CreateButton(currentDirectory+btnRankedPressedPath, currentDirectory+btnRankedReleasedPath, textureBounds, rl.Rectangle{692, 25, 215, 54})
	leftBtn := CreateButton(currentDirectory+btnLeftPressedPath, currentDirectory+btnLeftReleasedPath, textureBounds, rl.Rectangle{518, 463, 21, 31})
	rightBtn := CreateButton(currentDirectory+btnRightPressedPath, currentDirectory+btnRightReleasedPath, textureBounds, rl.Rectangle{741, 463, 21, 31})
	okBtn := CreateButton(currentDirectory+btnOkPressedPath, currentDirectory+btnOkReleasedPath, textureBounds, rl.Rectangle{573, 461, 134, 36})
	yesBtn := CreateButton(currentDirectory+btnYesPressedPath, currentDirectory+btnYesReleasedPath, textureBounds, rl.Rectangle{655, 324, 103, 36})
	noBtn := CreateButton(currentDirectory+btnNoPressedPath, currentDirectory+btnNoReleasedPath, textureBounds, rl.Rectangle{522, 324, 103, 36})

	return &BattleUI{
		currentBattle:        CreateBattle(),
		state:                BattleMode,
		pagePostBattle:       0,
		friendID:             "",
		isExiting:            false,
		battleModeBG:         battleModeBG,
		battleSearchBG:       battleSearchBG,
		inBattleBG:           inBattleBG,
		postBattlePlayerBG:   postBattlePlayerBG,
		postBattleOpponentBG: postBattleOpponentBG,
		exitDialogBG:         exitDialogBG,
		backgroundRect:       rl.Rectangle{X: 0, Y: 0, Width: float32(baseWidth), Height: float32(baseHeight)},
		timeSearchRect:       rl.Rectangle{X: 990, Y: 426, Width: 71, Height: 14},
		timeBattleRect:       rl.Rectangle{X: 572, Y: 44, Width: 136, Height: 35},
		healthBarRect:        rl.Rectangle{X: 120, Y: 66, Width: 206, Height: 15},
		nameRect:             rl.Rectangle{X: 120, Y: 36, Width: 206, Height: 22},
		medallionRect:        rl.Rectangle{X: 28, Y: 23, Width: 77, Height: 77},
		postBattle1Rect:      rl.Rectangle{X: 493, Y: 277, Width: 290, Height: 30},
		postBattle2Rect:      rl.Rectangle{X: 610, Y: 333, Width: 173, Height: 20},
		postBattle3Rect:      rl.Rectangle{X: 610, Y: 373, Width: 173, Height: 20},
		postBattle4Rect:      rl.Rectangle{X: 610, Y: 414, Width: 173, Height: 20},
		levelMatchBtn:        levelMatchBtn,
		rankedMatchBtn:       rankedMatchBtn,
		leftBtn:              leftBtn,
		rightBtn:             rightBtn,
		okBtn:                okBtn,
		yesBtn:               yesBtn,
		noBtn:                noBtn,
		resultBattle:         resultBattleUI{},
	}
}

// ----------------------------------------- FriendlyFightUI

func (f *FriendlyFightUI) Unload() {
	rl.UnloadTexture(f.friendlyFightBG)
	f.acceptBtn.Unload()
	f.refuseBtn.Unload()
}

func (f *FriendlyFightUI) Draw(scaleX, scaleY float32) {
	switch f.state {
	case showInvitation:
		const (
			codeFontSize = float32(8)
			nameFontSize = float32(14)
			fontSpacing  = 0
		)
		color := rl.Color{R: 159, G: 164, B: 197, A: 255}

		textCode := f.friend.PublicID
		textCodePos := rl.Vector2{
			X: scaleX * (f.textRect.X + 0.3*nameFontSize),
			Y: scaleY * (f.textRect.Y + 0.2*codeFontSize),
		}
		textName := TrimTextWithEllipsis(nameFontSize, fontSpacing, f.friend.Name, f.textRect.Width)
		textNameSize := rl.MeasureTextEx(font, textName, nameFontSize, fontSpacing)
		textNamePos := rl.Vector2{
			X: scaleX * (f.textRect.X + 0.3*nameFontSize),
			Y: scaleY * (f.textRect.Y + f.textRect.Height - textNameSize.Y),
		}

		drawTexture(f.friendlyFightBG, f.backgroundRect, f.backgroundRect, scaleX, scaleY)
		rl.DrawTextEx(font, textCode, textCodePos, scaleX*codeFontSize, scaleX*fontSpacing, color)
		rl.DrawTextEx(font, textName, textNamePos, scaleX*nameFontSize, scaleX*fontSpacing, color)

		f.acceptBtn.Draw(scaleX, scaleY)
		f.refuseBtn.Draw(scaleX, scaleY)

	case waitingInvitation:

	}

}

func (f *FriendlyFightUI) HandleInput(conn net.Conn, player *Player, gameState *string) {

	switch {
	case (*gameState != stateBattle || battleUI.state == BattleMode) && len(f.friendCh) > 0 && f.state == waitingInvitation:
		f.friend = <-f.friendCh
		f.state = showInvitation
	case f.state == showInvitation:
		if f.acceptBtn.IsHovered() {
			if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				f.acceptBtn.Pressed()
			} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
				f.acceptBtn.Released()
				f.state = waitingInvitation
				f.resetToDefaultState(gameState)
				*gameState = stateBattle
				go battleUI.waitingBattleSearch(conn, player, connection.MsgAcceptChallengeToFight, f.friend.PublicID, gameState)
				return
			}
		} else if !f.acceptBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			f.acceptBtn.Released()
		}

		if f.refuseBtn.IsHovered() {
			if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
				f.refuseBtn.Pressed()
			} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
				f.refuseBtn.Released()
				f.state = waitingInvitation
				sendInput(conn, connection.MsgRefuseChallengeToFight, f.friend.PublicID)
				fmt.Println(gameState, friendsUI.state)
				return
			}
		} else if !f.refuseBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			f.refuseBtn.Released()
		}
	default:
	}
}

func (f *FriendlyFightUI) resetToDefaultState(gameState *string) {
	switch *gameState {
	case stateListFriends:
		friendsUI.resetState()
	default:
	}
}

// ----------------------------------------- Battle

func (b *Battle) ClearOpponentResources() {
	if b.opponent != nil {
		b.opponent.character.UnloadTextures()
		b.opponent.character.StopPhysics()
		b.opponent.character.StopAnimation()
	}
}
func (b *Battle) ResetBattleState() {
	b.timeToStart = time.Time{}
	b.timeToEnd = time.Time{}
	b.opponent = nil
}

// ----------------------------------------- BattleUI

func (b *BattleUI) Unload() {
	rl.UnloadTexture(b.battleModeBG)
	rl.UnloadTexture(b.battleSearchBG)
	rl.UnloadTexture(b.inBattleBG)
	rl.UnloadTexture(b.postBattlePlayerBG)
	rl.UnloadTexture(b.postBattleOpponentBG)

	b.levelMatchBtn.Unload()
	b.rankedMatchBtn.Unload()
	b.leftBtn.Unload()
	b.rightBtn.Unload()
	b.okBtn.Unload()
	b.yesBtn.Unload()
	b.noBtn.Unload()

}

func (b *BattleUI) Draw(player *Player, scaleX, scaleY float32) {
	switch b.state {
	case BattleMode:
		b.drawBattleMode(scaleX, scaleY)
	case BattleSearch:
		b.drawBattleSearch(scaleX, scaleY)
	case InBattle:
		b.drawInBattle(player, scaleX, scaleY)
	case PostBattle:
		b.drawPostBattle(player, scaleX, scaleY)
	}
}

func (b *BattleUI) drawBattleMode(scaleX, scaleY float32) {
	drawTexture(b.battleModeBG, b.backgroundRect, b.backgroundRect, scaleX, scaleY)
	b.levelMatchBtn.Draw(scaleX, scaleY)
	b.rankedMatchBtn.Draw(scaleX, scaleY)
}

func (b *BattleUI) drawBattleSearch(scaleX, scaleY float32) {
	const (
		fontSize    = float32(12)
		fontSpacing = 0.5
	)
	blackoutColor := rl.Color{R: 0, G: 0, B: 0, A: 200}

	// Время поиска боя
	timeSearch := time.Now().Sub(b.timeSearch).Truncate(time.Second)
	textTimeSearch := formatDuration(timeSearch)
	textSize := rl.MeasureTextEx(font, textTimeSearch, fontSize, fontSpacing)
	textPos := rl.Vector2{
		X: scaleX * (b.timeSearchRect.X + (b.timeSearchRect.Width-textSize.X)/2 + 0.2*fontSize),
		Y: scaleY * (b.timeSearchRect.Y + (b.timeSearchRect.Height-textSize.Y)/2 + 0.1*fontSize),
	}

	rl.DrawRectangle(int32(b.backgroundRect.X*scaleX), int32(b.backgroundRect.Y*scaleY), int32(b.backgroundRect.Width*scaleX), int32(b.backgroundRect.Height*scaleY), blackoutColor)
	drawTexture(b.battleSearchBG, b.backgroundRect, b.backgroundRect, scaleX, scaleY)
	rl.DrawTextEx(font, textTimeSearch, textPos, fontSize*scaleY, fontSpacing, rl.Red)
}

func (b *BattleUI) drawInBattle(player *Player, scaleX, scaleY float32) {
	const fontSpacing = 0.5
	textColor := rl.Color{R: 159, G: 164, B: 197, A: 255}
	timerColor := rl.Color{R: 85, G: 220, B: 233, A: 255}

	// Время боя
	fontSizeTime := float32(20)
	timeNow := time.Now()
	var timeBattle time.Duration
	if timeNow.After(b.currentBattle.timeToStart) {
		timeBattle = b.currentBattle.timeToEnd.Sub(timeNow).Truncate(time.Second)
	} else {
		timeBattle = b.currentBattle.timeToStart.Sub(timeNow).Truncate(time.Second)
	}
	if timeBattle < 0 {
		timeBattle = 0
	}
	textTimeBattle := formatDuration(timeBattle)
	textTimeSize := rl.MeasureTextEx(font, textTimeBattle, fontSizeTime, fontSpacing)
	textTimePos := rl.Vector2{
		X: scaleX * (b.timeBattleRect.X + (b.timeBattleRect.Width-textTimeSize.X)/2 + 0.2*fontSizeTime),
		Y: scaleY * (b.timeBattleRect.Y + (b.timeBattleRect.Height-textTimeSize.Y)/2 + 0.1*fontSizeTime),
	}

	// Имя игрока и оппонента
	fontNameSize := float32(16)
	textPlName := TrimTextWithEllipsis(fontNameSize, fontSpacing, player.name, b.nameRect.Width)
	textPlNameSize := rl.MeasureTextEx(font, textPlName, fontNameSize, fontSpacing)
	textPlNamePos := rl.Vector2{
		X: scaleX * b.nameRect.X,
		Y: scaleY * (b.nameRect.Y + (b.nameRect.Height - textPlNameSize.Y) - 0.2*fontNameSize),
	}
	textOpName := TrimTextWithEllipsis(fontNameSize, fontSpacing, b.currentBattle.opponent.name, b.nameRect.Width)
	textOpNameSize := rl.MeasureTextEx(font, textOpName, fontNameSize, fontSpacing)
	textOpNamePos := rl.Vector2{
		X: scaleX * (baseWidth - b.nameRect.X - textOpNameSize.X),
		Y: scaleY * (b.nameRect.Y + (b.nameRect.Height - textOpNameSize.Y) - 0.2*fontNameSize),
	}

	//Количество здоровья игрока и оппонента
	fontHealthSize := float32(10)
	plHealth := float32(player.character.health)
	plDfHealth := float32(player.character.defaultHealth)
	textPlHealth := fmt.Sprintf("%.0f/%.0fHP", plHealth, plDfHealth)
	textPlHealthSize := rl.MeasureTextEx(font, textPlHealth, fontHealthSize, fontSpacing)
	textPlHealthPos := rl.Vector2{
		X: scaleX * (b.healthBarRect.X + b.healthBarRect.Width - textPlHealthSize.X),
		Y: scaleY*(b.healthBarRect.Y+(b.healthBarRect.Height-textPlHealthSize.Y)) - 0.2*fontHealthSize,
	}
	healthBarPl := rl.Rectangle{
		X:      scaleX * (b.healthBarRect.X + b.healthBarRect.Width - plHealth/plDfHealth*b.healthBarRect.Width),
		Y:      scaleY * b.healthBarRect.Y,
		Width:  scaleX * (plHealth / plDfHealth * b.healthBarRect.Width),
		Height: scaleY * b.healthBarRect.Height,
	}
	opHealth := float32(b.currentBattle.opponent.character.health)
	opDfHealth := float32(b.currentBattle.opponent.character.defaultHealth)
	textOpHealth := fmt.Sprintf("%.0f/%.0fHP", opHealth, opDfHealth)
	textOpHealthSize := rl.MeasureTextEx(font, textOpHealth, fontHealthSize, fontSpacing)
	textOpHealthPos := rl.Vector2{
		X: scaleX * (baseWidth - b.healthBarRect.X - textOpHealthSize.X),
		Y: scaleY*(b.healthBarRect.Y+(b.healthBarRect.Height-textOpHealthSize.Y)) - 0.2*fontHealthSize,
	}
	healthBarOp := rl.Rectangle{
		X:      scaleX * (baseWidth - b.healthBarRect.X - opHealth/opDfHealth*b.healthBarRect.Width), //b.healthBarRect.X + b.healthBarRect.Width - opHealth/opDfHealth*b.healthBarRect.Width),
		Y:      scaleY * b.healthBarRect.Y,
		Width:  scaleX * (opHealth / opDfHealth * b.healthBarRect.Width),
		Height: scaleY * b.healthBarRect.Height,
	}

	// Медальоны
	plMedallion := player.character.assets[Medallion]
	plMedallionPos := b.medallionRect
	opMedallion := b.currentBattle.opponent.character.assets[Medallion]
	opMedallionPos := rl.Rectangle{
		X:      baseWidth - b.medallionRect.X - b.medallionRect.Width,
		Y:      b.medallionRect.Y,
		Width:  b.medallionRect.Width,
		Height: b.medallionRect.Height,
	}

	drawTexture(b.inBattleBG, b.backgroundRect, b.backgroundRect, scaleX, scaleY)               // Фон UI для битвы
	rl.DrawTextEx(font, textPlName, textPlNamePos, fontNameSize*scaleY, fontSpacing, textColor) // Имена игроков
	rl.DrawTextEx(font, textOpName, textOpNamePos, fontNameSize*scaleY, fontSpacing, textColor)
	rl.DrawTextEx(font, textTimeBattle, textTimePos, fontSizeTime*scaleY, fontSpacing, timerColor)                            // Время боя
	rl.DrawRectangle(int32(healthBarPl.X), int32(healthBarPl.Y), int32(healthBarPl.Width), int32(healthBarPl.Height), rl.Red) // Полосы здоровья
	rl.DrawRectangle(int32(healthBarOp.X), int32(healthBarOp.Y), int32(healthBarOp.Width), int32(healthBarOp.Height), rl.Red)
	rl.DrawTextEx(font, textPlHealth, textPlHealthPos, fontHealthSize*scaleY, fontSpacing, textColor) // Значение здоровья
	rl.DrawTextEx(font, textOpHealth, textOpHealthPos, fontHealthSize*scaleY, fontSpacing, textColor)
	drawTexture(plMedallion.Texture, rl.Rectangle{X: 0, Y: 0, Width: float32(plMedallion.BaseWidth), Height: float32(plMedallion.BaseHeight)}, plMedallionPos, scaleX, scaleY) // Медальоны игроков
	drawTexture(opMedallion.Texture, rl.Rectangle{X: 0, Y: 0, Width: -float32(opMedallion.BaseWidth), Height: float32(opMedallion.BaseHeight)}, opMedallionPos, scaleX, scaleY)
	if b.currentBattle.opponent != nil {
		b.currentBattle.opponent.character.Draw(scaleX, scaleY)
	}

	if b.isExiting {
		drawTexture(b.exitDialogBG, b.backgroundRect, b.backgroundRect, scaleX, scaleY)
		b.yesBtn.Draw(scaleX, scaleY)
		b.noBtn.Draw(scaleX, scaleY)
	}
}

func (b *BattleUI) drawPostBattle(player *Player, scaleX, scaleY float32) {
	const (
		fontSpacing  = 0.05
		pagePlayer   = 0
		pageOpponent = 1
	)
	color := rl.Color{R: 179, G: 164, B: 152, A: 255}
	fontSize := float32(16)
	switch b.pagePostBattle {
	case pagePlayer:
		b.drawPagePlayer(player, fontSize, fontSpacing, scaleX, scaleY, color)
	case pageOpponent:
		b.drawPageOpponent(fontSize, fontSpacing, scaleX, scaleY, color)
	}

	b.leftBtn.Draw(scaleX, scaleY)
	b.rightBtn.Draw(scaleX, scaleY)
	b.okBtn.Draw(scaleX, scaleY)
}

func (b *BattleUI) drawPagePlayer(player *Player, fontSize, fontSpacing, scaleX, scaleY float32, color rl.Color) {
	var textResult string
	switch b.resultBattle.result {
	case NoBattle:
		textResult = "Бой отменён"
	case Victory:
		textResult = "Победа"
	case Draw:
		textResult = "Ничья"
	case Defeat:
		textResult = "Поражение"
	}
	// Результат боя
	fontResultSize := float32(20)
	textResultSize := rl.MeasureTextEx(font, textResult, fontResultSize, fontSpacing)
	textResultPos := rl.Vector2{
		X: scaleX * (b.postBattle1Rect.X + (b.postBattle1Rect.Width-textResultSize.X)/2),
		Y: scaleY * (b.postBattle1Rect.Y + (b.postBattle1Rect.Height - textResultSize.Y) - 0.2*fontResultSize),
	}
	// Изменение уровня
	textLevel := fmt.Sprintf("%+d(%d)", b.resultBattle.levelDelta, player.level)
	textLevel = TrimTextWithEllipsis(fontSize, fontSpacing, textLevel, b.postBattle2Rect.Width)
	textLevelSize := rl.MeasureTextEx(font, textLevel, fontSize, fontSpacing)
	textLevelPos := rl.Vector2{
		X: scaleX * (b.postBattle2Rect.X + (b.postBattle2Rect.Width - textLevelSize.X) + 0.3*fontSize),
		Y: scaleY * (b.postBattle2Rect.Y + (b.postBattle2Rect.Height - textLevelSize.Y) - 0.4*fontSize),
	}
	// Изменение ранга
	textRank := fmt.Sprintf("%+d(%d)", b.resultBattle.rankDelta, player.rank)
	textRank = TrimTextWithEllipsis(fontSize, fontSpacing, textRank, b.postBattle3Rect.Width)
	textRankSize := rl.MeasureTextEx(font, textRank, fontSize, fontSpacing)
	textRankPos := rl.Vector2{
		X: scaleX * (b.postBattle3Rect.X + (b.postBattle3Rect.Width - textRankSize.X) + 0.3*fontSize),
		Y: scaleY * (b.postBattle3Rect.Y + (b.postBattle3Rect.Height - textRankSize.Y)),
	}
	// Изменение монет
	textMoney := fmt.Sprintf("%+d(%d)", b.resultBattle.moneyDelta, player.money)
	textMoney = TrimTextWithEllipsis(fontSize, fontSpacing, textMoney, b.postBattle4Rect.Width)
	textMoneySize := rl.MeasureTextEx(font, textMoney, fontSize, fontSpacing)
	textMoneyPos := rl.Vector2{
		X: scaleX * (b.postBattle4Rect.X + (b.postBattle4Rect.Width - textMoneySize.X) + 0.3*fontSize),
		Y: scaleY * (b.postBattle4Rect.Y + (b.postBattle4Rect.Height - textMoneySize.Y)),
	}

	drawTexture(b.postBattlePlayerBG, b.backgroundRect, b.backgroundRect, scaleX, scaleY)
	rl.DrawTextEx(font, textResult, textResultPos, fontResultSize*scaleY, fontSpacing, color)
	rl.DrawTextEx(font, textLevel, textLevelPos, fontSize*scaleY, fontSpacing, color)
	rl.DrawTextEx(font, textRank, textRankPos, fontSize*scaleY, fontSpacing, color)
	rl.DrawTextEx(font, textMoney, textMoneyPos, fontSize*scaleY, fontSpacing, color)
}

func (b *BattleUI) drawPageOpponent(fontSize, fontSpacing, scaleX, scaleY float32, color rl.Color) {
	// Имя оппонента
	textOppName := TrimTextWithEllipsis(fontSize, fontSpacing, b.resultBattle.opponentPublicID+": "+b.resultBattle.opponentName, b.postBattle1Rect.Width)
	textOppNameSize := rl.MeasureTextEx(font, textOppName, fontSize, fontSpacing)
	textOppNamePos := rl.Vector2{
		X: scaleX * (b.postBattle1Rect.X + (b.postBattle1Rect.Width-textOppNameSize.X)/2),
		Y: scaleY * (b.postBattle2Rect.Y + (b.postBattle2Rect.Height - textOppNameSize.Y)),
	}
	// Уровень оппонента
	textLevel := fmt.Sprintf("%d", b.resultBattle.opponentLevel)
	textLevelSize := rl.MeasureTextEx(font, textLevel, fontSize, fontSpacing)
	textLevelPos := rl.Vector2{
		X: scaleX * (b.postBattle3Rect.X + (b.postBattle3Rect.Width - textLevelSize.X)),
		Y: scaleY * (b.postBattle3Rect.Y + (b.postBattle3Rect.Height - textLevelSize.Y) - 0.4*fontSize),
	}
	// Ранг оппонента
	textRank := fmt.Sprintf("%d", b.resultBattle.opponentRank)
	textRankSize := rl.MeasureTextEx(font, textRank, fontSize, fontSpacing)
	textRankPos := rl.Vector2{
		X: scaleX * (b.postBattle4Rect.X + (b.postBattle4Rect.Width - textRankSize.X)),
		Y: scaleY * (b.postBattle4Rect.Y + (b.postBattle4Rect.Height - textRankSize.Y)),
	}

	drawTexture(b.postBattleOpponentBG, b.backgroundRect, b.backgroundRect, scaleX, scaleY)
	rl.DrawTextEx(font, textOppName, textOppNamePos, fontSize*scaleY, fontSpacing, color)
	rl.DrawTextEx(font, textLevel, textLevelPos, fontSize*scaleY, fontSpacing, color)
	rl.DrawTextEx(font, textRank, textRankPos, fontSize*scaleY, fontSpacing, color)
}

func (b *BattleUI) HandelInput(conn net.Conn, player *Player, gameState *string) {
	switch b.state {
	case BattleMode:
		b.stateBattleMode(conn, player, gameState)
	case BattleSearch:
		b.stateBattleSearch(conn, player, gameState)
	case InBattle:
		b.stateInBattle(conn, player, gameState)
	case PostBattle:
		b.statePostBattle(gameState)
	}
}

func (b *BattleUI) stateBattleMode(conn net.Conn, player *Player, gameState *string) {
	// Выход
	if rl.IsKeyReleased(rl.KeyEscape) {
		*gameState = stateMenu
		return
	}
	// Обычный бой
	if b.levelMatchBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			b.levelMatchBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			b.levelMatchBtn.Released()

			//b.timeSearch = time.Now()
			//b.state = BattleSearch
			typeBattle := connection.MsgBattle
			go b.waitingBattleSearch(conn, player, typeBattle, "", gameState)
		}
	} else if !b.levelMatchBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b.levelMatchBtn.Released()
	}
	// Ранговый Бой
	if b.rankedMatchBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			b.rankedMatchBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			b.rankedMatchBtn.Released()
			//b.timeSearch = time.Now()
			//b.state = BattleSearch
			typeBattle := connection.MsgBattleRanked
			go b.waitingBattleSearch(conn, player, typeBattle, "", gameState)
		}
	} else if !b.rankedMatchBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b.rankedMatchBtn.Released()
	}
}

func (b *BattleUI) waitingBattleSearch(conn net.Conn, player *Player, typeBattle connection.MessageType, friendID string, gameState *string) {
	select {
	case <-b.currentBattle.exit:
	default:
	}
	b.timeSearch = time.Now()
	b.state = BattleSearch
	b.friendID = friendID
	sendInput(conn, typeBattle, friendID)
	select {
	case <-time.After(5 * time.Second):
		log.Println(time.Now())
		log.Println("Ожидание боя истекло, выход в меню")
		sendInput(conn, connection.MsgExitBattle, nil)
		*gameState = stateMenu
		b.state = BattleMode
		return
	case <-b.currentBattle.waiting:
		return
	}
}

func (b *BattleUI) stateBattleSearch(conn net.Conn, player *Player, gameState *string) {
	if rl.IsKeyReleased(rl.KeyEscape) {
		sendInput(conn, connection.MsgExitBattle, nil)
	}
	select {
	case response := <-b.currentBattle.start:
		b.handleBattleStart(conn, player, response)
		b.state = InBattle
		return
	case friendID := <-b.currentBattle.exit:
		if friendID == "" {
			*gameState = stateMenu
			b.state = BattleMode
			return
		} else if b.friendID == friendID { // в случае отказа от приглашения на бой
			sendInput(conn, connection.MsgExitBattle, nil)
			b.friendID = ""
		}

	default:
	}
}

func (b *BattleUI) handleBattleStart(conn net.Conn, player *Player, response StartBattleInfo) {
	timeNow := time.Now().UnixMilli()
	offset := timeNow + connection.ServerLag - response.Timestamp

	sendInput(conn, connection.MsgReadyBattle, nil)
	opponent := CreateOpponent(response)
	b.currentBattle.opponent = opponent
	b.currentBattle.timeToStart = time.UnixMilli(response.StartTime + offset).Local()
	b.currentBattle.timeToEnd = time.UnixMilli(response.EndTime + offset).Local()
}

func (b *BattleUI) stateInBattle(conn net.Conn, player *Player, gameState *string) {
	timeNow := time.Now()
	switch {
	case len(b.currentBattle.end) > 0:
		if player.character.isDying || b.currentBattle.opponent.character.isDying {
			ch := b.currentBattle.opponent.character
			if player.character.isDying {
				ch = player.character
			}
			if ch.currentFrame < ch.assets[ch.currentState].FrameCount-1 {
				break
			}
		}
		battleResults := <-b.currentBattle.end
		b.handleBattleEnd(player, battleResults)
		b.state = PostBattle
		b.isExiting = false
		return
	case rl.IsKeyReleased(rl.KeyEscape):
		b.isExiting = !b.isExiting
	case b.isExiting:
		b.exit(conn)
	case timeNow.After(b.currentBattle.timeToStart) && timeNow.Before(b.currentBattle.timeToEnd):
		player.character.Control(conn, gameState)
	}
}

func (b *BattleUI) exit(conn net.Conn) {
	if b.yesBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			b.yesBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			b.yesBtn.Released()
			sendInput(conn, connection.MsgExitBattle, nil)
		}
	} else if !b.yesBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b.yesBtn.Released()
	}
	if b.noBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			b.noBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			b.noBtn.Released()
			b.isExiting = false
		}
	} else if !b.noBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b.noBtn.Released()
	}

	if rl.IsKeyDown(rl.KeyEscape) {
		b.noBtn.Pressed()
	} else if rl.IsKeyReleased(rl.KeyEscape) {
		b.noBtn.Released()
		b.isExiting = false
	}

	if rl.IsKeyDown(rl.KeyEnter) {
		b.yesBtn.Pressed()
	} else if rl.IsKeyReleased(rl.KeyEnter) {
		b.yesBtn.Released()
		sendInput(conn, connection.MsgExitBattle, nil)
	}
}

func (b *BattleUI) handleBattleEnd(player *Player, battleResults EndBattleInfo) {
	b.resultBattle = resultBattleUI{
		result:           battleResults.Result,
		levelDelta:       battleResults.CurrentLevel - player.level,
		rankDelta:        battleResults.CurrentRank - player.rank,
		moneyDelta:       battleResults.TotalMoney - player.money,
		opponentPublicID: b.currentBattle.opponent.publicID,
		opponentName:     b.currentBattle.opponent.name,
		opponentLevel:    b.currentBattle.opponent.level,
		opponentRank:     b.currentBattle.opponent.rank,
	}

	player.ApplyBattleResults(battleResults)
	player.character.RestoreState(battleResults.UpdatedStats)
	b.currentBattle.ClearOpponentResources()
	b.currentBattle.ResetBattleState()
}

func (b *BattleUI) statePostBattle(gameState *string) {
	// Выйти
	if b.okBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			b.okBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			b.okBtn.Released()
			b.state = BattleMode
			b.pagePostBattle = 0
			*gameState = stateMenu
			return
		}
	} else if !b.okBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b.okBtn.Released()
	}

	// Пролистнуть влево
	if b.leftBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			b.leftBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			b.leftBtn.Released()
			b.pagePostBattle = int(math.Abs(float64(b.pagePostBattle-1))) % 2
		}
	} else if !b.leftBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b.leftBtn.Released()
	}

	// Пролистнуть вправо
	if b.rightBtn.IsHovered() {
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			b.rightBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			b.rightBtn.Released()
			b.pagePostBattle = int(math.Abs(float64(b.pagePostBattle+1))) % 2
		}
	} else if !b.rightBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
		b.rightBtn.Released()
	}
}
