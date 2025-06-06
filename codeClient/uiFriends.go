package main

import (
	"codeClient/connection"
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"net"
	"strings"
	"time"
)

type friendsState uint8

const (
	friendsWaitingForResponse friendsState = iota + 1
	friendsList
	friendRequests
	searchFriend
)

type FriendEntry struct {
	Name     string
	PublicID string
}

type FriendsData struct {
	Friends  []FriendEntry `msgpack:"f"`
	Incoming []FriendEntry `msgpack:"i"`
	Outgoing []FriendEntry `msgpack:"o"`
}

type FieldsFriendsUI struct {
	fieldBG rl.Texture2D

	fieldSize      rl.Rectangle
	fieldPos       rl.Rectangle
	outputFieldPos rl.Rectangle

	setBtnRect *ClickBounds

	actionBtnList []*Button
	cancelBtnList []*Button
}

type FriendsUI struct {
	fieldsRequests    *FieldsFriendsUI
	fieldsFriends     *FieldsFriendsUI
	state             friendsState
	page              int
	input             string
	addFriendResponse string

	friendsDataCh chan FriendsData

	friendsData      FriendsData
	currentPageItems []FriendEntry
	friendsList      []FriendEntry

	friendsListBG    rl.Texture2D
	friendRequestsBG rl.Texture2D
	searchFriendBG   rl.Texture2D

	backgroundRect        rl.Rectangle
	searchFriendInputRect rl.Rectangle
	addFriendInputRect    rl.Rectangle
	responseServerRect    rl.Rectangle
	pageRect              rl.Rectangle

	friendsListBounds    *ClickBounds
	addFriendBounds      *ClickBounds
	friendRequestsBounds *ClickBounds
	searchFriendBounds   *ClickBounds

	addBtn    *Button
	battleBtn *Button
	deleteBtn *Button
	loupeBtn  *Button
	leftBtn   *Button
	rightBtn  *Button

	// Таймеры для контроля стирания
	backspaceTimer        time.Time
	backspaceInitialDelay time.Duration // Первая задержка
	backspaceRepeatDelay  time.Duration // Текущая задержка
	backspaceMinDelay     time.Duration // Минимальная задержка
}

func createFieldsRequests() *FieldsFriendsUI {
	const (
		fieldBGPath            = "\\resources\\UI\\Friends\\Background\\Field.png"
		btnAcceptPressedPath   = "\\resources\\UI\\Friends\\Button\\Accept\\AcceptPressed.png"
		btnAcceptReleasedPath  = "\\resources\\UI\\Friends\\Button\\Accept\\AcceptReleased.png"
		btnDeclinePressedPath  = "\\resources\\UI\\Friends\\Button\\Decline\\DeclinePressed.png"
		btnDeclineReleasedPath = "\\resources\\UI\\Friends\\Button\\Decline\\DeclineReleased.png"

		countFields = 5
		offset      = 14
	)
	var (
		acceptBounds   = rl.Rectangle{731, 283, 25, 25}
		declineBounds  = rl.Rectangle{760, 283, 25, 25}
		acceptBtnList  = make([]*Button, countFields)
		declineBtnList = make([]*Button, countFields)
	)
	fieldBG := rl.LoadTexture(currentDirectory + fieldBGPath)

	for i := 0; i < countFields; i++ {
		acceptBtnList[i] = CreateButton(currentDirectory+btnAcceptPressedPath, currentDirectory+btnAcceptReleasedPath, OffsetRectY(acceptBounds, float32(i), offset), OffsetRectY(acceptBounds, float32(i), offset))
		declineBtnList[i] = CreateButton(currentDirectory+btnDeclinePressedPath, currentDirectory+btnDeclineReleasedPath, OffsetRectY(declineBounds, float32(i), offset), OffsetRectY(declineBounds, float32(i), offset))
	}

	return &FieldsFriendsUI{
		fieldBG:        fieldBG,
		fieldSize:      rl.Rectangle{X: 0, Y: 0, Width: float32(fieldBG.Width), Height: float32(fieldBG.Height)},
		fieldPos:       rl.Rectangle{X: 489, Y: 276, Width: 303, Height: 39},
		outputFieldPos: rl.Rectangle{X: 494, Y: 282, Width: 235, Height: 27},
		setBtnRect:     createClickBounds(rl.Rectangle{X: 725, Y: 277, Width: 66, Height: 193}),
		actionBtnList:  acceptBtnList,
		cancelBtnList:  declineBtnList,
	}
}

func createFieldsFriends() *FieldsFriendsUI {
	const (
		fieldBGPath           = "\\resources\\UI\\Friends\\Background\\Field.png"
		btnBattlePressedPath  = "\\resources\\UI\\Friends\\Button\\Battle\\BattlePressed.png"
		btnBattleReleasedPath = "\\resources\\UI\\Friends\\Button\\Battle\\BattleReleased.png"
		btnRemovePressedPath  = "\\resources\\UI\\Friends\\Button\\Remove\\RemovePressed.png"
		btnRemoveReleasedPath = "\\resources\\UI\\Friends\\Button\\Remove\\RemoveReleased.png"

		countFields = 5
		offset      = 14
	)
	var (
		battleBounds  = rl.Rectangle{731, 291, 25, 25}
		removeBounds  = rl.Rectangle{760, 291, 25, 25}
		battleBtnList = make([]*Button, countFields)
		removeBtnList = make([]*Button, countFields)
	)
	fieldBG := rl.LoadTexture(currentDirectory + fieldBGPath)
	for i := 0; i < countFields; i++ {
		battleBtnList[i] = CreateButton(currentDirectory+btnBattlePressedPath, currentDirectory+btnBattleReleasedPath, OffsetRectY(battleBounds, float32(i), offset), OffsetRectY(battleBounds, float32(i), offset))
		removeBtnList[i] = CreateButton(currentDirectory+btnRemovePressedPath, currentDirectory+btnRemoveReleasedPath, OffsetRectY(removeBounds, float32(i), offset), OffsetRectY(removeBounds, float32(i), offset))
	}

	return &FieldsFriendsUI{
		fieldBG:        fieldBG,
		fieldSize:      rl.Rectangle{X: 0, Y: 0, Width: float32(fieldBG.Width), Height: float32(fieldBG.Height)},
		fieldPos:       rl.Rectangle{X: 489, Y: 284, Width: 303, Height: 39},
		outputFieldPos: rl.Rectangle{X: 494, Y: 290, Width: 235, Height: 27},
		setBtnRect:     createClickBounds(rl.Rectangle{X: 725, Y: 286, Width: 66, Height: 193}),
		actionBtnList:  battleBtnList,
		cancelBtnList:  removeBtnList,
	}
}

func CreateFriendsUI() *FriendsUI {
	const (
		friendsListBGPath    = "\\resources\\UI\\Friends\\Background\\FriendsList.png"
		friendRequestsBGPath = "\\resources\\UI\\Friends\\Background\\FriendRequests.png"
		searchFriendBGPath   = "\\resources\\UI\\Friends\\Background\\AddFriend.png"

		btnAddPressedPath     = "\\resources\\UI\\Friends\\Button\\Add\\AddPressed.png"
		btnAddReleasedPath    = "\\resources\\UI\\Friends\\Button\\Add\\AddReleased.png"
		btnBattlePressedPath  = "\\resources\\UI\\Friends\\Button\\Battle\\BattlePressed.png"
		btnBattleReleasedPath = "\\resources\\UI\\Friends\\Button\\Battle\\BattleReleased.png"

		btnLoupePressedPath  = "\\resources\\UI\\Friends\\Button\\Loupe\\LoupePressed.png"
		btnLoupeReleasedPath = "\\resources\\UI\\Friends\\Button\\Loupe\\LoupeReleased.png"
		btnLeftPressedPath   = "\\resources\\UI\\Friends\\Button\\Left\\LeftPressed.png"
		btnLeftReleasedPath  = "\\resources\\UI\\Friends\\Button\\Left\\LeftReleased.png"
		btnRightPressedPath  = "\\resources\\UI\\Friends\\Button\\Right\\RightPressed.png"
		btnRightReleasedPath = "\\resources\\UI\\Friends\\Button\\Right\\RightReleased.png"
	)
	var (
		textureBounds          = rl.Rectangle{0, 0, baseWidth, baseHeight}
		friendManagementBounds = rl.Rectangle{0, 0, 25, 25}
		friendsListBounds      = rl.Rectangle{479, 176, 120, 27}
		addFriendBounds        = rl.Rectangle{623, 176, 152, 27}
		friendRequestsBounds   = rl.Rectangle{492, 238, 120, 27}
		searchFriendBounds     = rl.Rectangle{620, 238, 120, 27}
		pageRect               = rl.Rectangle{X: 575, Y: 485, Width: 133, Height: 29}
	)

	friendsListBG := rl.LoadTexture(currentDirectory + friendsListBGPath)
	friendRequestsBG := rl.LoadTexture(currentDirectory + friendRequestsBGPath)
	searchFriendBG := rl.LoadTexture(currentDirectory + searchFriendBGPath)

	addBtn := CreateButton(currentDirectory+btnAddPressedPath, currentDirectory+btnAddReleasedPath, textureBounds, rl.Rectangle{546, 328, 187, 24})
	battleBtn := CreateButton(currentDirectory+btnBattlePressedPath, currentDirectory+btnBattleReleasedPath, friendManagementBounds, friendManagementBounds)
	loupeBtn := CreateButton(currentDirectory+btnLoupePressedPath, currentDirectory+btnLoupeReleasedPath, textureBounds, rl.Rectangle{754, 240, 33, 33})
	leftBtn := CreateButton(currentDirectory+btnLeftPressedPath, currentDirectory+btnLeftReleasedPath, textureBounds, rl.Rectangle{554, 484, 21, 31})
	rightBtn := CreateButton(currentDirectory+btnRightPressedPath, currentDirectory+btnRightReleasedPath, textureBounds, rl.Rectangle{707, 484, 21, 31})

	return &FriendsUI{
		fieldsRequests: createFieldsRequests(),
		fieldsFriends:  createFieldsFriends(),

		state:             friendsWaitingForResponse,
		page:              0,
		input:             "",
		addFriendResponse: "",

		friendsDataCh: make(chan FriendsData, 1),

		friendsListBG:    friendsListBG,
		friendRequestsBG: friendRequestsBG,
		searchFriendBG:   searchFriendBG,

		backgroundRect:        textureBounds,
		searchFriendInputRect: rl.Rectangle{X: 495, Y: 240, Width: 258, Height: 33},
		addFriendInputRect:    rl.Rectangle{X: 500, Y: 285, Width: 280, Height: 27},
		responseServerRect:    rl.Rectangle{X: 495, Y: 358, Width: 290, Height: 26},
		pageRect:              pageRect,
		friendsListBounds:     createClickBounds(friendsListBounds),
		addFriendBounds:       createClickBounds(addFriendBounds),
		friendRequestsBounds:  createClickBounds(friendRequestsBounds),
		searchFriendBounds:    createClickBounds(searchFriendBounds),

		addBtn:    addBtn,
		battleBtn: battleBtn,
		//deleteBtn: deleteBtn,
		loupeBtn: loupeBtn,
		leftBtn:  leftBtn,
		rightBtn: rightBtn,

		backspaceTimer:        time.Now(),
		backspaceInitialDelay: 300 * time.Millisecond,
		backspaceRepeatDelay:  300 * time.Millisecond,
		backspaceMinDelay:     40 * time.Millisecond,
	}
}

// ----------------------------------------- FieldsUI

func (f *FieldsFriendsUI) Draw(list []FriendEntry, scaleX, scaleY float32) {
	const (
		publicIDFontSize = float32(8)
		nameFontSize     = float32(14)
		fontSpacing      = 0
	)
	color := rl.Color{R: 55, G: 190, B: 203, A: 255}
	f.setBtnRect.Scale(scaleX, scaleY)
	for i := 0; i < len(list) && i < len(f.actionBtnList); i++ {
		req := list[i]
		fieldPos := OffsetRectY(f.fieldPos, float32(i), 0)
		posText := OffsetRectY(f.outputFieldPos, float32(i), 12)

		textPublicID := req.PublicID
		textPublicIDPos := rl.Vector2{
			X: scaleX * (posText.X + 0.3*nameFontSize),
			Y: scaleY * (posText.Y + 0.2*publicIDFontSize),
		}
		textName := TrimTextWithEllipsis(nameFontSize, fontSpacing, req.Name, f.outputFieldPos.Width)
		textNameSize := rl.MeasureTextEx(font, textName, nameFontSize, fontSpacing)
		textNamePos := rl.Vector2{
			X: scaleX * (posText.X + 0.3*nameFontSize),
			Y: scaleY * (posText.Y + posText.Height - textNameSize.Y),
		}

		drawTexture(f.fieldBG, f.fieldSize, fieldPos, scaleX, scaleY)
		rl.DrawTextEx(font, textPublicID, textPublicIDPos, scaleX*publicIDFontSize, scaleX*fontSpacing, color)
		rl.DrawTextEx(font, textName, textNamePos, scaleX*nameFontSize, scaleX*fontSpacing, color)
		f.actionBtnList[i].Draw(scaleX, scaleY)
		f.cancelBtnList[i].Draw(scaleX, scaleY)
	}

}

func (f *FieldsFriendsUI) Unload() {
	rl.UnloadTexture(f.fieldBG)

	for i := 0; i < len(f.actionBtnList); i++ {
		f.actionBtnList[i].Unload()
		f.cancelBtnList[i].Unload()
	}
}

func (f *FieldsFriendsUI) FindClickedButton(list []FriendEntry) (codeBtn, number int) {
	const (
		acceptCode  = 1
		declineCode = 0
		noneCode    = -1
	)
	if f.setBtnRect.IsHovered() {
		for i := 0; i < len(list); i++ {
			switch {
			case f.actionBtnList[i].IsHovered():
				if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
					f.actionBtnList[i].Pressed()
				} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && f.actionBtnList[i].isPressed {
					f.actionBtnList[i].Released()
					return acceptCode, i
				}
			case f.cancelBtnList[i].IsHovered():
				if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
					f.cancelBtnList[i].Pressed()
				} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) && f.cancelBtnList[i].isPressed {
					f.cancelBtnList[i].Released()
					return declineCode, i
				}
			default:
				f.actionBtnList[i].Released()
				f.cancelBtnList[i].Released()
			}
		}
	}

	return noneCode, -1
}

// ----------------------------------------- uiFriends

func (f *FriendsUI) Unload() {
	f.fieldsRequests.Unload()
	f.fieldsFriends.Unload()

	rl.UnloadTexture(f.friendsListBG)
	rl.UnloadTexture(f.friendRequestsBG)
	rl.UnloadTexture(f.searchFriendBG)

	f.addBtn.Unload()
	f.loupeBtn.Unload()
	f.battleBtn.Unload()
	//f.deleteBtn.Unload()
	f.leftBtn.Unload()
	f.rightBtn.Unload()
	f.leftBtn.Unload()
	f.rightBtn.Unload()
}

func (f *FriendsUI) Draw(scaleX, scaleY float32) {
	f.addFriendBounds.Scale(scaleX, scaleY)

	switch f.state {
	case friendsWaitingForResponse:
		f.drawFriendsListLoading(scaleX, scaleY)
	case friendsList:
		f.drawFriendsList(scaleX, scaleY)
	case friendRequests:
		f.drawFriendsRequests(scaleX, scaleY)
	case searchFriend:
		f.drawSearchFriends(scaleX, scaleY)
	}
}

func (f *FriendsUI) drawFriendsListLoading(scaleX, scaleY float32) {
	const (
		fontSize    = float32(16)
		fontSpacing = 0.5
	)
	waiting := rl.Rectangle{X: 489, Y: 372, Width: 303, Height: 47}

	drawTexture(f.friendsListBG, f.backgroundRect, f.backgroundRect, scaleX, scaleY)
	f.loupeBtn.Draw(scaleX, scaleY)
	drawFieldText("Загрузка...", waiting, scaleX, scaleY, fontSize, fontSpacing, rl.Red)
}

func (f *FriendsUI) drawFriendsList(scaleX, scaleY float32) {
	const (
		fontSize    = float32(18)
		fontSpacing = 0.5
	)
	color := rl.Color{R: 55, G: 190, B: 203, A: 255}

	drawTexture(f.friendsListBG, f.backgroundRect, f.backgroundRect, scaleX, scaleY)
	drawFieldBorder(f.searchFriendInputRect, scaleX, scaleY, true)
	f.loupeBtn.Draw(scaleX, scaleY)
	drawFieldText(f.input, f.searchFriendInputRect, scaleX, scaleY, fontSize, fontSpacing, color)

	f.fieldsFriends.Draw(f.currentPageItems, scaleX, scaleY)
	f.leftBtn.Draw(scaleX, scaleY)
	f.rightBtn.Draw(scaleX, scaleY)
	drawFieldText(fmt.Sprintf("%d", f.page+1), f.pageRect, scaleX, scaleY, fontSize, fontSpacing, color)

}

func (f *FriendsUI) drawFriendsRequests(scaleX, scaleY float32) {
	const (
		fontSize    = float32(18)
		fontSpacing = 0.5
	)
	color := rl.Color{R: 55, G: 190, B: 203, A: 255}

	f.friendsListBounds.Scale(scaleX, scaleY)
	f.searchFriendBounds.Scale(scaleX, scaleY)

	drawTexture(f.friendRequestsBG, f.backgroundRect, f.backgroundRect, scaleX, scaleY)

	f.fieldsRequests.Draw(f.currentPageItems, scaleX, scaleY)
	f.leftBtn.Draw(scaleX, scaleY)
	f.rightBtn.Draw(scaleX, scaleY)
	drawFieldText(fmt.Sprintf("%d", f.page+1), f.pageRect, scaleX, scaleY, fontSize, fontSpacing, color)
}

func (f *FriendsUI) drawSearchFriends(scaleX, scaleY float32) {
	const (
		fontSize    = float32(16)
		fontSpacing = 0.5
	)
	color := rl.Color{R: 55, G: 190, B: 203, A: 255}

	f.friendsListBounds.Scale(scaleX, scaleY)
	f.friendRequestsBounds.Scale(scaleX, scaleY)

	drawTexture(f.searchFriendBG, f.backgroundRect, f.backgroundRect, scaleX, scaleY)
	drawFieldBorder(f.addFriendInputRect, scaleX, scaleY, true)
	f.addBtn.Draw(scaleX, scaleY)
	drawFieldText(f.input, f.addFriendInputRect, scaleX, scaleY, fontSize, fontSpacing, color)
	drawFieldText(f.addFriendResponse, f.responseServerRect, scaleX, scaleY, fontSize-5, fontSpacing, rl.Red)
}

func (f *FriendsUI) HandleInput(conn net.Conn, player *Player, gameState *string) {

	switch {
	case rl.IsKeyReleased(rl.KeyEscape):
		f.resetState()
		*gameState = stateMenu
	case f.state == friendsWaitingForResponse:
		f.stateWaitingForResponse()
	case f.state == friendsList:
		f.stateFriendsList(conn, player, gameState)
	case f.state == friendRequests:
		f.stateFriendRequests(conn)
	case f.state == searchFriend:
		f.stateSearchFriend(conn)

	}
}

func (f *FriendsUI) resetState() {
	f.input = ""
	f.state = friendsWaitingForResponse
	f.page = 0
}

func (f *FriendsUI) stateWaitingForResponse() {
	select {
	case data := <-f.friendsDataCh:
		f.friendsData = data
		f.addFriendResponse = ""
		f.friendsList = data.Friends
		f.UpdateCurrentPageItems(f.friendsList)
		f.state = friendsList
	default:
	}
}

func (f *FriendsUI) stateFriendsList(conn net.Conn, player *Player, gameState *string) {
	switch {
	case f.addFriendBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		f.input = ""
		f.page = 0
		f.UpdateCurrentPageItems(f.friendsData.Incoming)
		f.state = friendRequests
	case f.loupeBtn.IsHovered() || rl.IsKeyDown(rl.KeyEnter) || rl.IsKeyReleased(rl.KeyEnter):
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) || rl.IsKeyDown(rl.KeyEnter) {
			f.loupeBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) || rl.IsKeyReleased(rl.KeyEnter) {
			f.loupeBtn.Released()
			/// Обрабока поиска
			f.searchFriend(f.input)
			f.page = 0
			f.UpdateCurrentPageItems(f.friendsList)
		}
	case f.leftBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			f.leftBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			f.leftBtn.Released()
			f.prevPage(f.friendsList)
			f.UpdateCurrentPageItems(f.friendsList)
		}
	case f.rightBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			f.rightBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			f.rightBtn.Released()
			f.nextPage(f.friendsList)
			f.UpdateCurrentPageItems(f.friendsList)
		}
	case f.fieldsFriends.setBtnRect.IsHovered() && (rl.IsMouseButtonReleased(rl.MouseLeftButton) || !rl.IsMouseButtonUp(rl.MouseLeftButton)):
		codeBtn, number := f.fieldsFriends.FindClickedButton(f.currentPageItems)
		if codeBtn != -1 {
			msgType, friendID := f.friendsProcessing(codeBtn, number)
			if msgType == connection.MsgChallengeToFight {
				f.resetState()
				*gameState = stateBattle
				go battleUI.waitingBattleSearch(conn, player, msgType, friendID, gameState)
				return
			}
			sendInput(conn, msgType, friendID)
			f.UpdateCurrentPageItems(f.friendsList)

		}
	default:
		f.loupeBtn.Released()
	}

	handleTextInput(&f.input, &f.backspaceTimer, &f.backspaceInitialDelay, f.backspaceRepeatDelay, f.backspaceMinDelay)
}

func (f *FriendsUI) searchFriend(search string) {
	search = strings.ToLower(search)
	var exactMatches []FriendEntry
	var partialMatches []FriendEntry

	all := append([]FriendEntry{}, f.friendsData.Friends...)

	for _, friend := range all {
		nameLower := strings.ToLower(friend.Name)
		codeLower := strings.ToLower(friend.PublicID)

		if nameLower == search || codeLower == search {
			exactMatches = append(exactMatches, friend)
		} else if strings.Contains(nameLower, search) || strings.Contains(codeLower, search) {
			partialMatches = append(partialMatches, friend)
		}
	}

	f.friendsList = append(exactMatches, partialMatches...)
}

func (f *FriendsUI) friendsProcessing(codeBtn, number int) (connection.MessageType, string) {
	const (
		battleCode = 1
		RemoveCode = 0
	)
	id := f.page*5 + number
	request := f.friendsList[id]

	switch codeBtn {
	case battleCode:
		return connection.MsgChallengeToFight, request.PublicID
	default:
		for i := 0; i < len(f.friendsData.Friends); i++ {
			if f.friendsData.Friends[i].PublicID == request.PublicID {
				f.friendsData.Friends = append(f.friendsData.Friends[:i], f.friendsData.Friends[i+1:]...)
				break
			}
		}
		f.friendsList = append(f.friendsList[:id], f.friendsList[id+1:]...)
		return connection.MsgRemoveFriend, request.PublicID
	}
}

func (f *FriendsUI) stateFriendRequests(conn net.Conn) {
	switch {
	case f.friendsListBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		f.page = 0
		f.UpdateCurrentPageItems(f.friendsList)
		f.state = friendsList
	case f.searchFriendBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		f.state = searchFriend
	case f.leftBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			f.leftBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			f.leftBtn.Released()
			f.prevPage(f.friendsData.Incoming)
			f.UpdateCurrentPageItems(f.friendsData.Incoming)
		}
	case f.rightBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			f.rightBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			f.rightBtn.Released()
			f.nextPage(f.friendsData.Incoming)
			f.UpdateCurrentPageItems(f.friendsData.Incoming)
		}
	case f.fieldsRequests.setBtnRect.IsHovered() && (rl.IsMouseButtonReleased(rl.MouseLeftButton) || !rl.IsMouseButtonUp(rl.MouseLeftButton)):
		codeBtn, number := f.fieldsRequests.FindClickedButton(f.currentPageItems)
		if codeBtn != -1 {
			msgType, friendID := f.requestProcessing(codeBtn, number)
			sendInput(conn, msgType, friendID)
			f.UpdateCurrentPageItems(f.friendsData.Incoming)
		}
	default:
		f.rightBtn.Released()
		f.leftBtn.Released()

	}
}

func (f *FriendsUI) requestProcessing(codeBtn, number int) (connection.MessageType, string) {
	const (
		acceptCode  = 1
		declineCode = 0
	)
	id := f.page*5 + number
	request := f.friendsData.Incoming[id]
	f.friendsData.Incoming = append(f.friendsData.Incoming[:id], f.friendsData.Incoming[id+1:]...)

	switch codeBtn {
	case acceptCode:
		f.friendsData.Friends = append(f.friendsData.Friends, request)
		f.friendsList = f.friendsData.Friends // append(f.friendsList, request)
		return connection.MsgAcceptFriendship, request.PublicID
	default:
		return connection.MsgDeclineFriendship, request.PublicID
	}
}

func (f *FriendsUI) nextPage(list []FriendEntry) {
	totalPages := (len(list) + 4) / 5
	if totalPages == 0 {
		totalPages = 1
	}
	f.page = (f.page + 1) % totalPages
}

func (f *FriendsUI) prevPage(list []FriendEntry) {
	totalPages := (len(list) + 4) / 5
	if totalPages == 0 {
		totalPages = 1
	}
	f.page = (f.page - 1 + totalPages) % totalPages
}

func (f *FriendsUI) UpdateCurrentPageItems(list []FriendEntry) {
	start := f.page * 5
	end := start + 5
	if end > len(list) {
		end = len(list)
	}
	f.currentPageItems = list[start:end]
}

func (f *FriendsUI) stateSearchFriend(conn net.Conn) {
	switch {
	case f.friendsListBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		f.input = ""
		f.addFriendResponse = ""
		f.page = 0
		f.UpdateCurrentPageItems(f.friendsList)
		f.state = friendsList
	case f.friendRequestsBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		f.input = ""
		f.addFriendResponse = ""
		f.state = friendRequests
	case f.addBtn.IsHovered() || rl.IsKeyDown(rl.KeyEnter) || rl.IsKeyReleased(rl.KeyEnter):
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) || rl.IsKeyDown(rl.KeyEnter) {
			f.addBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) || rl.IsKeyReleased(rl.KeyEnter) {
			f.addBtn.Released()
			sendInput(conn, connection.MsgAddFriend, f.input)
			f.input = ""
			f.addFriendResponse = "Поиск..."
		}
	default:
		f.addBtn.Released()
	}

	handleTextInput(&f.input, &f.backspaceTimer, &f.backspaceInitialDelay, f.backspaceRepeatDelay, f.backspaceMinDelay)
}
