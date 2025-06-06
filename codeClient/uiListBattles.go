package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

type listBattlesState uint8

const (
	battlesWaitingForResponse listBattlesState = iota + 1
	rankedList
	standardList
)

type BattleEntry struct {
	StartTime        time.Time `msgpack:"st"`
	EndTime          time.Time `msgpack:"et"`
	BattleResult     string    `msgpack:"br"`
	PlayerName       string    `msgpack:"pn"`
	PlayerPublicID   string    `msgpack:"pi"`
	OpponentName     string    `msgpack:"on"`
	OpponentPublicID string    `msgpack:"oi"`
}

type BattleStats struct {
	NumberWins   int `msgpack:"w"`
	NumberLosses int `msgpack:"l"`
	NumberDraws  int `msgpack:"d"`
}

type BattleData struct {
	RankedStats     *BattleStats  `msgpack:"rs"`
	StandardStats   *BattleStats  `msgpack:"ss"`
	RankedBattles   []BattleEntry `msgpack:"rb"`
	StandardBattles []BattleEntry `msgpack:"sb"`
}

type ListBattlesUI struct {
	numberBattles int
	percentWins   float32
	state         listBattlesState
	page          int
	battleDataCh  chan BattleData

	battleData       BattleData
	currentPageItems []BattleEntry

	fieldBG               rl.Texture2D
	rankedListBattlesBG   rl.Texture2D
	standardListBattlesBG rl.Texture2D

	backgroundRect   rl.Rectangle
	fieldPos         rl.Rectangle
	fieldSize        rl.Rectangle
	numberBattlesPos rl.Rectangle
	percentWinsPos   rl.Rectangle
	player1FieldPos  rl.Rectangle
	player2FieldPos  rl.Rectangle
	resultFieldPos   rl.Rectangle
	dateFieldPos     rl.Rectangle
	pagePos          rl.Rectangle

	rankedListBounds   *ClickBounds
	standardListBounds *ClickBounds

	leftBtn  *Button
	rightBtn *Button
}

func CreateListBattlesUI() *ListBattlesUI {
	const (
		fieldBGPath               = "\\resources\\UI\\ListBattles\\Background\\Field.png"
		rankedListBattlesBGPath   = "\\resources\\UI\\ListBattles\\Background\\RankedListBattle.png"
		standardListBattlesBGPath = "\\resources\\UI\\ListBattles\\Background\\StandardListBattle.png"
		btnLeftPressedPath        = "\\resources\\UI\\ListBattles\\Button\\Left\\LeftPressed.png"
		btnLeftReleasedPath       = "\\resources\\UI\\ListBattles\\Button\\Left\\LeftReleased.png"
		btnRightPressedPath       = "\\resources\\UI\\ListBattles\\Button\\Right\\RightPressed.png"
		btnRightReleasedPath      = "\\resources\\UI\\ListBattles\\Button\\Right\\RightReleased.png"
	)
	var (
		textureBounds      = rl.Rectangle{0, 0, baseWidth, baseHeight}
		rankedListBounds   = rl.Rectangle{X: 481, Y: 176, Width: 149, Height: 27}
		standardListBounds = rl.Rectangle{304, 176, 147, 27}
	)

	fieldBG := rl.LoadTexture(currentDirectory + fieldBGPath)
	rankedListBattlesBG := rl.LoadTexture(currentDirectory + rankedListBattlesBGPath)
	standardListBattlesBG := rl.LoadTexture(currentDirectory + standardListBattlesBGPath)

	leftBtn := CreateButton(currentDirectory+btnLeftPressedPath, currentDirectory+btnLeftReleasedPath, textureBounds, rl.Rectangle{554, 484, 21, 31})
	rightBtn := CreateButton(currentDirectory+btnRightPressedPath, currentDirectory+btnRightReleasedPath, textureBounds, rl.Rectangle{707, 484, 21, 31})

	return &ListBattlesUI{
		numberBattles: 0,
		percentWins:   0,
		state:         battlesWaitingForResponse,
		page:          0,
		battleDataCh:  make(chan BattleData, 1),

		fieldBG:               fieldBG,
		rankedListBattlesBG:   rankedListBattlesBG,
		standardListBattlesBG: standardListBattlesBG,

		backgroundRect:   textureBounds,
		fieldPos:         rl.Rectangle{283, 284, 714, 39},
		fieldSize:        rl.Rectangle{X: 0, Y: 0, Width: float32(fieldBG.Width), Height: float32(fieldBG.Height)},
		numberBattlesPos: rl.Rectangle{631, 242, 145, 33},
		percentWinsPos:   rl.Rectangle{889, 242, 102, 33},
		player1FieldPos:  rl.Rectangle{288, 290, 171, 27},
		player2FieldPos:  rl.Rectangle{496, 290, 173, 27},
		resultFieldPos:   rl.Rectangle{679, 290, 152, 27},
		dateFieldPos:     rl.Rectangle{840, 290, 152, 27},
		pagePos:          rl.Rectangle{X: 575, Y: 485, Width: 133, Height: 29},

		rankedListBounds:   createClickBounds(rankedListBounds),
		standardListBounds: createClickBounds(standardListBounds),

		leftBtn:  leftBtn,
		rightBtn: rightBtn,
	}
}

func (lb *ListBattlesUI) Unload() {
	rl.UnloadTexture(lb.fieldBG)
	rl.UnloadTexture(lb.rankedListBattlesBG)
	rl.UnloadTexture(lb.standardListBattlesBG)
	lb.leftBtn.Unload()
	lb.rightBtn.Unload()
}

func (lb *ListBattlesUI) Draw(scaleX, scaleY float32) {
	switch lb.state {
	case battlesWaitingForResponse:
		lb.drawListBattlesLoading(scaleX, scaleY)
	case standardList:
		lb.drawStandardList(scaleX, scaleY)
	case rankedList:
		lb.drawRankedList(scaleX, scaleY)
	}
}

func (lb *ListBattlesUI) drawListBattlesLoading(scaleX, scaleY float32) {
	const (
		fontSize    = 16
		fontSpacing = 0.5
	)
	waiting := rl.Rectangle{X: 489, Y: 372, Width: 303, Height: 47}
	drawTexture(lb.standardListBattlesBG, lb.backgroundRect, lb.backgroundRect, scaleX, scaleY)
	drawFieldText("Загрузка...", waiting, scaleX, scaleY, fontSize, fontSpacing, rl.Red)
}

func (lb *ListBattlesUI) drawPlayerName(name, playerID string, posText rl.Rectangle, scaleX, scaleY float32) {
	const (
		nameFontSize     = 14
		PublicIDFontSize = 8
		fontSpacing      = 0.5
	)
	color := rl.Color{R: 55, G: 190, B: 203, A: 255}

	textPublicID := playerID
	textPublicIDPos := rl.Vector2{
		X: scaleX * (posText.X + 0.3*nameFontSize),
		Y: scaleY * (posText.Y + 0.2*PublicIDFontSize),
	}
	textName := TrimTextWithEllipsis(nameFontSize, fontSpacing, name, posText.Width)
	textNameSize := rl.MeasureTextEx(font, textName, nameFontSize, fontSpacing)
	textNamePos := rl.Vector2{
		X: scaleX * (posText.X + 0.3*nameFontSize),
		Y: scaleY * (posText.Y + posText.Height - textNameSize.Y),
	}

	rl.DrawTextEx(font, textPublicID, textPublicIDPos, scaleX*PublicIDFontSize, scaleX*fontSpacing, color)
	rl.DrawTextEx(font, textName, textNamePos, scaleX*nameFontSize, scaleX*fontSpacing, color)
}

func (lb *ListBattlesUI) drawBattleRecords(battleRecords []BattleEntry, scaleX, scaleY float32) {
	const (
		statsFontSize  = 16
		pageFontSize   = 19
		resultFontSize = 15
		dateFontSize   = 10
		fontSpacing    = 0.5
	)
	color := rl.Color{R: 55, G: 190, B: 203, A: 255}

	for i := 0; i < len(battleRecords); i++ {
		rec := battleRecords[i]

		fieldPos := OffsetRectY(lb.fieldPos, float32(i), 0)

		player1FieldPos := OffsetRectY(lb.player1FieldPos, float32(i), 12)
		player2FieldPos := OffsetRectY(lb.player2FieldPos, float32(i), 12)
		resultFieldPos := OffsetRectY(lb.resultFieldPos, float32(i), 12)
		dateFieldPos := OffsetRectY(lb.dateFieldPos, float32(i), 12)
		dateFieldPos.Height /= 2

		drawTexture(lb.fieldBG, lb.fieldSize, fieldPos, scaleX, scaleY)
		lb.drawPlayerName(rec.PlayerName, rec.PlayerPublicID, player1FieldPos, scaleX, scaleY)
		lb.drawPlayerName(rec.OpponentName, rec.OpponentPublicID, player2FieldPos, scaleX, scaleY)
		drawFieldText(rec.BattleResult, resultFieldPos, scaleX, scaleY, resultFontSize, fontSpacing, color)
		drawFieldText(rec.StartTime.Format("15:04:05 ")+rec.EndTime.Format("- 15:04:05"), dateFieldPos, scaleX, scaleY, dateFontSize, 0, color)
		dateFieldPos.Y += dateFieldPos.Height
		drawFieldText(rec.StartTime.Format("02.01.2006"), dateFieldPos, scaleX, scaleY, dateFontSize, 0, color)
	}

	drawFieldText(fmt.Sprintf("%d", lb.numberBattles), lb.numberBattlesPos, scaleX, scaleY, statsFontSize, 0, color)
	drawFieldText(fmt.Sprintf("%.2f%%", lb.percentWins), lb.percentWinsPos, scaleX, scaleY, statsFontSize, 0, color)
	drawFieldText(fmt.Sprintf("%d", lb.page+1), lb.pagePos, scaleX, scaleY, pageFontSize, fontSpacing, color)
	lb.leftBtn.Draw(scaleX, scaleY)
	lb.rightBtn.Draw(scaleX, scaleY)
}

func (lb *ListBattlesUI) drawStandardList(scaleX, scaleY float32) {
	lb.rankedListBounds.Scale(scaleX, scaleY)

	drawTexture(lb.standardListBattlesBG, lb.backgroundRect, lb.backgroundRect, scaleX, scaleY)
	lb.drawBattleRecords(lb.currentPageItems, scaleX, scaleY)
}

func (lb *ListBattlesUI) drawRankedList(scaleX, scaleY float32) {
	lb.standardListBounds.Scale(scaleX, scaleY)

	drawTexture(lb.rankedListBattlesBG, lb.backgroundRect, lb.backgroundRect, scaleX, scaleY)
	lb.drawBattleRecords(lb.currentPageItems, scaleX, scaleY)
}

func (lb *ListBattlesUI) resetState() {
	lb.state = battlesWaitingForResponse
	lb.numberBattles = 0
	lb.percentWins = 0
	lb.page = 0
}

func (lb *ListBattlesUI) HandelInput(gameState *string) {
	switch {
	case rl.IsKeyReleased(rl.KeyEscape):
		lb.resetState()
		*gameState = stateMenu
	case lb.state == battlesWaitingForResponse:
		lb.stateWaitingForResponse()
	case lb.state == standardList:
		lb.stateStandardList()
	case lb.state == rankedList:
		lb.stateRankedList()
	}
}

func (lb *ListBattlesUI) stateWaitingForResponse() {
	select {
	case data := <-lb.battleDataCh:
		wins := data.StandardStats.NumberWins
		losses := data.StandardStats.NumberLosses
		draws := data.StandardStats.NumberDraws
		lb.numberBattles = wins + losses + draws
		fmt.Println(data.RankedStats.NumberWins, data.RankedStats.NumberLosses, data.RankedStats.NumberDraws)
		//lb.percentWins = float32(roundFloat(float64(wins)/float64(lb.numberBattles)*100, 2))
		if lb.numberBattles > 0 {
			lb.percentWins = float32(wins) / float32(lb.numberBattles) * 100
		} else {
			lb.percentWins = 0
		}
		lb.battleData = data
		lb.GetCurrentPageItems(lb.battleData.StandardBattles)
		lb.state = standardList
	default:
	}
}

func (lb *ListBattlesUI) stateStandardList() {

	switch {
	case lb.rankedListBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		wins := lb.battleData.RankedStats.NumberWins
		losses := lb.battleData.RankedStats.NumberLosses
		draws := lb.battleData.RankedStats.NumberDraws
		lb.numberBattles = wins + losses + draws
		if lb.numberBattles > 0 {
			lb.percentWins = float32(wins) / float32(lb.numberBattles) * 100
		} else {
			lb.percentWins = 0
		}
		lb.page = 0
		lb.GetCurrentPageItems(lb.battleData.RankedBattles)
		lb.state = rankedList
	case lb.leftBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			lb.leftBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			lb.leftBtn.Released()
			lb.prevPage(lb.battleData.StandardBattles)
			lb.GetCurrentPageItems(lb.battleData.StandardBattles)
		}
	case lb.rightBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			lb.rightBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			lb.rightBtn.Released()
			lb.nextPage(lb.battleData.StandardBattles)
			lb.GetCurrentPageItems(lb.battleData.StandardBattles)
		}
	default:
		lb.rightBtn.Released()
		lb.leftBtn.Released()
	}
}

func (lb *ListBattlesUI) stateRankedList() {
	switch {
	case lb.standardListBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		wins := lb.battleData.StandardStats.NumberWins
		losses := lb.battleData.StandardStats.NumberLosses
		draws := lb.battleData.StandardStats.NumberDraws
		lb.numberBattles = wins + losses + draws
		if lb.numberBattles > 0 {
			lb.percentWins = float32(wins) / float32(lb.numberBattles) * 100
		} else {
			lb.percentWins = 0
		}
		lb.page = 0
		lb.GetCurrentPageItems(lb.battleData.StandardBattles)
		lb.state = standardList
	case lb.leftBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			lb.leftBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			lb.leftBtn.Released()
			lb.prevPage(lb.battleData.RankedBattles)
			lb.GetCurrentPageItems(lb.battleData.RankedBattles)
		}
	case lb.rightBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			lb.rightBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			lb.rightBtn.Released()
			lb.nextPage(lb.battleData.RankedBattles)
			lb.GetCurrentPageItems(lb.battleData.RankedBattles)
		}
	default:
		lb.rightBtn.Released()
		lb.leftBtn.Released()
	}
}

func (lb *ListBattlesUI) nextPage(list []BattleEntry) {
	totalPages := (len(list) + 4) / 5
	if totalPages == 0 {
		totalPages = 1
	}
	lb.page = (lb.page + 1) % totalPages
}

func (lb *ListBattlesUI) prevPage(list []BattleEntry) {
	totalPages := (len(list) + 4) / 5
	if totalPages == 0 {
		totalPages = 1
	}
	lb.page = (lb.page - 1 + totalPages) % totalPages
}

func (lb *ListBattlesUI) GetCurrentPageItems(list []BattleEntry) {
	start := lb.page * 5
	end := start + 5
	if end > len(list) {
		end = len(list)
	}
	lb.currentPageItems = list[start:end]
}
