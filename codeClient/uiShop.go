package main

import (
	"codeClient/connection"
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"net"
)

type shopState uint8

type ShopActionType int

type ProductType int

const (
	shopWaitingForResponse shopState = iota
	shopBackgrounds
	shopCharacters
)

const (
	actionBuy ShopActionType = iota
	actionSelect
)

const (
	productBackground ProductType = iota
	productCharacter
)

type ShopBackgroundItem struct {
	ID          int
	Name        string
	Description string
	Cost        int
	AssetPath   string
	Preview     rl.Texture2D
}

type ShopCharacterItem struct {
	ID          int
	Name        string
	Description string
	Health      int
	Damage      int
	Cost        int
	AssetPath   string
	Preview     rl.Texture2D
}

type ShopData struct {
	PurchasedBackgrounds []ShopBackgroundItem `msgpack:"pb"`
	AvailableBackgrounds []ShopBackgroundItem `msgpack:"ab"`
	PurchasedCharacters  []ShopCharacterItem  `msgpack:"pc"`
	AvailableCharacters  []ShopCharacterItem  `msgpack:"ac"`
}

type ShopAction struct {
	Action      ShopActionType `msgpack:"a"`
	ProductType ProductType    `msgpack:"t"`
	ProductID   int            `msgpack:"p"`
}

type PurchaseReceipt struct {
	ProductType    ProductType `msgpack:"t"`
	ProductID      int         `msgpack:"p"`
	RemainingMoney int         `msgpack:"r"`
}

type productCard struct {
	size                 rl.Rectangle
	pos                  rl.Rectangle
	namePos              rl.Rectangle
	descriptionField1Pos rl.Rectangle
	descriptionField2Pos rl.Rectangle
	costPos              rl.Rectangle
	imageClickBounds     *ClickBounds
	buyBtn               *Button
	selectBtn            *Button
}

type shopGridUI struct {
	descCardIndex int
	isPurchases   bool
	primaryColor  rl.Color

	backgroundCardBG   rl.Texture2D
	characterCardBG    rl.Texture2D
	backgroundCardList []*productCard
	characterCardList  []*productCard
}

type ShopUI struct {
	state        shopState
	page         int
	primaryColor rl.Color

	shopDataCh         chan ShopData
	purchaseReceiptCh  chan PurchaseReceipt
	updateCharacterCh  chan connection.CharacterData
	updateBackgroundCh chan string

	shopData               ShopData
	currentPageBackgrounds []ShopBackgroundItem
	currentPageCharacters  []ShopCharacterItem

	shopGrid         *shopGridUI
	backgroundShopBG rl.Texture2D
	characterShopBG  rl.Texture2D
	backgroundCardBG rl.Texture2D
	characterCardBG  rl.Texture2D

	backgroundRect rl.Rectangle

	pagePos  rl.Rectangle
	moneyPos rl.Rectangle

	backgroundsBounds *ClickBounds
	charactersBounds  *ClickBounds

	purchasesBtn *Button
	leftBtn      *Button
	rightBtn     *Button
}

// ----------------------------------------- productCard

func createBackgroundProductCard(x, y float32) *productCard {
	const (
		btnBuyPressedPath     = "\\resources\\UI\\Shop\\Button\\Buy\\BuyPressed.png"
		btnBuyReleasedPath    = "\\resources\\UI\\Shop\\Button\\Buy\\BuyReleased.png"
		btnSelectPressedPath  = "\\resources\\UI\\Shop\\Button\\Select\\SelectPressed.png"
		btnSelectReleasedPath = "\\resources\\UI\\Shop\\Button\\Select\\SelectReleased.png"
		widthProductCard      = 312
		heightProductCard     = 233
	)

	var (
		size            = rl.Rectangle{0, 0, widthProductCard, heightProductCard}
		pos             = rl.Rectangle{x, y, widthProductCard, heightProductCard}
		costPos         = rl.Rectangle{pos.X + 60, pos.Y + 201, 104, 25}
		buyVisualBounds = rl.Rectangle{pos.X + 167, pos.Y + 200, 72, 26}
		buyClickBounds  = rl.Rectangle{pos.X + 209, pos.Y + 200, 30, 26}
		selectBounds    = rl.Rectangle{pos.X + 60, pos.Y + 199, 192, 29}
		imagePos        = rl.Rectangle{pos.X + 12, pos.Y + 9, 288, 162}
	)
	buyBtn := CreateButton(currentDirectory+btnBuyPressedPath, currentDirectory+btnBuyReleasedPath, buyVisualBounds, buyClickBounds)
	selectBtn := CreateButton(currentDirectory+btnSelectPressedPath, currentDirectory+btnSelectReleasedPath, selectBounds, selectBounds)

	return &productCard{
		size:             size,
		pos:              pos,
		costPos:          costPos,
		imageClickBounds: createClickBounds(imagePos),
		buyBtn:           buyBtn,
		selectBtn:        selectBtn,
	}
}

func createCharacterProductCard(x, y float32) *productCard {
	const (
		btnBuyPressedPath     = "\\resources\\UI\\Shop\\Button\\Buy\\BuyPressed.png"
		btnBuyReleasedPath    = "\\resources\\UI\\Shop\\Button\\Buy\\BuyReleased.png"
		btnSelectPressedPath  = "\\resources\\UI\\Shop\\Button\\Select\\SelectPressed.png"
		btnSelectReleasedPath = "\\resources\\UI\\Shop\\Button\\Select\\SelectReleased.png"
		widthProductCard      = 416
		heightProductCard     = 343
	)

	var (
		size                 = rl.Rectangle{0, 0, widthProductCard, heightProductCard}
		pos                  = rl.Rectangle{x, y, widthProductCard, heightProductCard}
		namePos              = rl.Rectangle{pos.X + 13, pos.Y + 6, 390, 27}
		descriptionField1Pos = rl.Rectangle{pos.X + 71, pos.Y + 70, 326, 41}
		descriptionField2Pos = rl.Rectangle{pos.X + 71, pos.Y + 118, 326, 41}
		costPos              = rl.Rectangle{pos.X + 214, pos.Y + 311, 104, 25}
		buyVisualBounds      = rl.Rectangle{pos.X + 321, pos.Y + 310, 72, 26}
		buyClickBounds       = rl.Rectangle{pos.X + 363, pos.Y + 310, 30, 26}
		selectBounds         = rl.Rectangle{pos.X + 214, pos.Y + 309, 192, 29}
		imagePos             = rl.Rectangle{pos.X + 16, pos.Y + 65, 384, 210}
	)
	buyBtn := CreateButton(currentDirectory+btnBuyPressedPath, currentDirectory+btnBuyReleasedPath, buyVisualBounds, buyClickBounds)
	selectBtn := CreateButton(currentDirectory+btnSelectPressedPath, currentDirectory+btnSelectReleasedPath, selectBounds, selectBounds)

	return &productCard{
		size:                 size,
		pos:                  pos,
		namePos:              namePos,
		descriptionField1Pos: descriptionField1Pos,
		descriptionField2Pos: descriptionField2Pos,
		costPos:              costPos,
		imageClickBounds:     createClickBounds(imagePos),
		buyBtn:               buyBtn,
		selectBtn:            selectBtn,
	}
}

func (p *productCard) unload() {
	p.buyBtn.Unload()
	p.selectBtn.Unload()
}

func (p *productCard) draw(scaleX, scaleY float32) {
	p.imageClickBounds.Scale(scaleX, scaleY)
	p.buyBtn.Draw(scaleX, scaleY)
	p.selectBtn.Draw(scaleX, scaleY)
}

// ----------------------------------------- shopGridUI
func createGridUI() *shopGridUI {
	const (
		backgroundCardBGPath  = "\\resources\\UI\\Shop\\Background\\BackgroundCard.png"
		characterCardBGPath   = "\\resources\\UI\\Shop\\Background\\CharacterCard.png"
		backgroundCardOffsetX = 312 + 0
		backgroundCardOffsetY = 233 + 5
		characterCardOffsetX  = 416 + 0

		backgroundCardRows = 2
		backgroundCardCols = 4
		characterCardCols  = 3
	)

	var (
		firstBackgroundCardPos = rl.Vector2{X: 16, Y: 63}
		firstCharacterCardPos  = rl.Vector2{X: 15, Y: 109}
		backgroundCardList     = make([]*productCard, backgroundCardRows*backgroundCardCols)
		characterCardList      = make([]*productCard, characterCardCols)
	)

	backgroundCardBG := rl.LoadTexture(currentDirectory + backgroundCardBGPath)
	characterCardBG := rl.LoadTexture(currentDirectory + characterCardBGPath)

	for row := 0; row < backgroundCardRows; row++ {
		for col := 0; col < backgroundCardCols; col++ {
			index := row*backgroundCardCols + col
			x := firstBackgroundCardPos.X + float32(col)*backgroundCardOffsetX
			y := firstBackgroundCardPos.Y + float32(row)*backgroundCardOffsetY
			backgroundCardList[index] = createBackgroundProductCard(x, y)
		}
	}

	for col := 0; col < characterCardCols; col++ {
		x := firstCharacterCardPos.X + float32(col)*characterCardOffsetX
		y := firstCharacterCardPos.Y
		characterCardList[col] = createCharacterProductCard(x, y)
	}

	return &shopGridUI{
		descCardIndex:      -1,
		isPurchases:        false,
		primaryColor:       rl.Color{156, 170, 189, 255},
		backgroundCardBG:   backgroundCardBG,
		characterCardBG:    characterCardBG,
		backgroundCardList: backgroundCardList,
		characterCardList:  characterCardList,
	}
}

func (s *shopGridUI) unload() {
	rl.UnloadTexture(s.backgroundCardBG)
	rl.UnloadTexture(s.characterCardBG)
	for i := 0; i < len(s.backgroundCardList); i++ {
		s.backgroundCardList[i].unload()
	}
	for i := 0; i < len(s.characterCardList); i++ {
		s.characterCardList[i].unload()
	}
}

func (s *shopGridUI) drawBackgroundGrid(pageBackgrounds []ShopBackgroundItem, scaleX, scaleY float32) {
	const (
		fontSize    = float32(18)
		fontSpacing = 1
	)
	for i := 0; i < len(pageBackgrounds); i++ {
		backgroundCard := s.backgroundCardList[i]

		drawTexture(s.backgroundCardBG, backgroundCard.size, backgroundCard.pos, scaleX, scaleY)

		if s.isPurchases {
			backgroundCard.selectBtn.Draw(scaleX, scaleY)
		} else {
			textMoney := TrimTextWithEllipsis(fontSize, fontSpacing, fmt.Sprintf("%d", pageBackgrounds[i].Cost), backgroundCard.costPos.Width)
			textMoneySize := rl.MeasureTextEx(font, textMoney, fontSize, fontSpacing)
			textMoneyPos := rl.Vector2{
				X: scaleX * (backgroundCard.costPos.X + (backgroundCard.costPos.Width - textMoneySize.X)),
				Y: scaleY * (backgroundCard.costPos.Y + (backgroundCard.costPos.Height-textMoneySize.Y)/2),
			}

			rl.DrawTextEx(font, fmt.Sprintf("%d", pageBackgrounds[i].Cost), textMoneyPos, fontSize*scaleY, fontSpacing*scaleX, s.primaryColor)
			backgroundCard.buyBtn.Draw(scaleX, scaleY)
		}
		drawTexture(pageBackgrounds[i].Preview, rl.Rectangle{0, 0, baseWidth, baseHeight}, backgroundCard.imageClickBounds.baseBounds, scaleX, scaleY)

	}
}

func (s *shopGridUI) drawCharacterGrid(pageCharacters []ShopCharacterItem, scaleX, scaleY float32) {
	const (
		fontSize      = float32(18)
		fontHandDSize = float32(22)
		fontSpacing   = 1
		previewWidth  = 384
		previewHeight = 210
	)
	for i := 0; i < len(pageCharacters); i++ {
		characterCard := s.characterCardList[i]
		characterCard.imageClickBounds.Scale(scaleX, scaleY)

		drawTexture(s.characterCardBG, characterCard.size, characterCard.pos, scaleX, scaleY)

		if s.isPurchases {
			characterCard.selectBtn.Draw(scaleX, scaleY)
		} else {
			textMoney := TrimTextWithEllipsis(fontSize, fontSpacing, fmt.Sprintf("%d", pageCharacters[i].Cost), characterCard.costPos.Width)
			textMoneySize := rl.MeasureTextEx(font, textMoney, fontSize, fontSpacing)
			textMoneyPos := rl.Vector2{
				X: scaleX * (characterCard.costPos.X + (characterCard.costPos.Width - textMoneySize.X)),
				Y: scaleY * (characterCard.costPos.Y + (characterCard.costPos.Height-textMoneySize.Y)/2),
			}

			rl.DrawTextEx(font, fmt.Sprintf("%d", pageCharacters[i].Cost), textMoneyPos, fontSize*scaleY, fontSpacing*scaleX, s.primaryColor)
			characterCard.buyBtn.Draw(scaleX, scaleY)
		}
		drawFieldText(pageCharacters[i].Name, characterCard.namePos, scaleX, scaleY, fontSize, fontSpacing, s.primaryColor)
		drawFieldText(fmt.Sprintf("%d", pageCharacters[i].Health), characterCard.descriptionField1Pos, scaleX, scaleY, fontHandDSize, fontSpacing, s.primaryColor)
		drawFieldText(fmt.Sprintf("%d", pageCharacters[i].Damage), characterCard.descriptionField2Pos, scaleX, scaleY, fontHandDSize, fontSpacing, s.primaryColor)
		if s.descCardIndex != i {
			drawTexture(pageCharacters[i].Preview, rl.Rectangle{0, 0, previewWidth, previewHeight}, characterCard.imageClickBounds.baseBounds, scaleX, scaleY)
		}
	}
}

func (s *shopGridUI) handelBackgroundGrid(lenCardList int) *ShopAction {
	for i := 0; i < lenCardList; i++ {
		backgroundCard := s.backgroundCardList[i]
		if s.isPurchases {
			if backgroundCard.selectBtn.IsHovered() {
				if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
					backgroundCard.selectBtn.Pressed()
				} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
					backgroundCard.selectBtn.Released()
					return &ShopAction{Action: actionSelect, ProductType: productBackground, ProductID: i}
				}
			} else if !backgroundCard.selectBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
				backgroundCard.selectBtn.Released()
			}
		} else {
			if backgroundCard.buyBtn.IsHovered() {
				if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
					backgroundCard.buyBtn.Pressed()
				} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
					backgroundCard.buyBtn.Released()
					return &ShopAction{Action: actionBuy, ProductType: productBackground, ProductID: i}
				}
			} else if !backgroundCard.buyBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
				backgroundCard.buyBtn.Released()
			}
		}
	}
	return nil
}

func (s *shopGridUI) handelCharacterGrid(lenCardList int) *ShopAction {
	for i := 0; i < lenCardList; i++ {
		characterCard := s.characterCardList[i]
		if characterCard.imageClickBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			if s.descCardIndex == i {
				s.descCardIndex = -1
			} else {
				s.descCardIndex = i
			}
		}

		if s.isPurchases {
			if characterCard.selectBtn.IsHovered() {
				if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
					characterCard.selectBtn.Pressed()
				} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
					characterCard.selectBtn.Released()
					return &ShopAction{Action: actionSelect, ProductType: productCharacter, ProductID: i}
				}
			} else if !characterCard.selectBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
				characterCard.selectBtn.Released()
			}
		} else {
			if characterCard.buyBtn.IsHovered() {
				if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
					characterCard.buyBtn.Pressed()
				} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
					characterCard.buyBtn.Released()
					return &ShopAction{Action: actionBuy, ProductType: productCharacter, ProductID: i}
				}
			} else if !characterCard.buyBtn.IsHovered() || !rl.IsMouseButtonDown(rl.MouseButtonLeft) {
				characterCard.buyBtn.Released()
			}
		}
	}
	return nil
}

// ----------------------------------------- ShopUI

func CreateShopUI() *ShopUI {
	const (
		backgroundShopBGPath     = "\\resources\\UI\\Shop\\Background\\BackgroundShop .png"
		characterShopBGPath      = "\\resources\\UI\\Shop\\Background\\CharacterShop .png"
		backgroundCardBGPath     = "\\resources\\UI\\Shop\\Background\\BackgroundCard.png"
		characterCardBGPath      = "\\resources\\UI\\Shop\\Background\\CharacterCard.png"
		btnPurchasesPressedPath  = "\\resources\\UI\\Shop\\Button\\Purchases\\PurchasesPressed.png"
		btnPurchasesReleasedPath = "\\resources\\UI\\Shop\\Button\\Purchases\\PurchasesReleased.png"
		btnLeftPressedPath       = "\\resources\\UI\\Shop\\Button\\Left\\LeftPressed.png"
		btnLeftReleasedPath      = "\\resources\\UI\\Shop\\Button\\Left\\LeftReleased.png"
		btnRightPressedPath      = "\\resources\\UI\\Shop\\Button\\Right\\RightPressed.png"
		btnRightReleasedPath     = "\\resources\\UI\\Shop\\Button\\Right\\RightReleased.png"
	)
	var (
		textureBounds             = rl.Rectangle{0, 0, baseWidth, baseHeight}
		purchasesManagementBounds = rl.Rectangle{318, 23, 25, 25}
		backgroundsBounds         = rl.Rectangle{21, 19, 99, 33}
		charactersBounds          = rl.Rectangle{132, 19, 169, 33}
	)

	backgroundShopBG := rl.LoadTexture(currentDirectory + backgroundShopBGPath)
	characterShopBG := rl.LoadTexture(currentDirectory + characterShopBGPath)
	backgroundCardBG := rl.LoadTexture(currentDirectory + backgroundCardBGPath)
	characterCardBG := rl.LoadTexture(currentDirectory + characterCardBGPath)

	purchasesBtn := CreateButton(currentDirectory+btnPurchasesPressedPath, currentDirectory+btnPurchasesReleasedPath, purchasesManagementBounds, purchasesManagementBounds)
	leftBtn := CreateButton(currentDirectory+btnLeftPressedPath, currentDirectory+btnLeftReleasedPath, textureBounds, rl.Rectangle{554, 544, 21, 31})
	rightBtn := CreateButton(currentDirectory+btnRightPressedPath, currentDirectory+btnRightReleasedPath, textureBounds, rl.Rectangle{706, 544, 21, 31})

	return &ShopUI{
		state:        shopWaitingForResponse,
		page:         0,
		primaryColor: rl.Color{156, 170, 189, 255},

		shopDataCh:         make(chan ShopData, 1),
		purchaseReceiptCh:  make(chan PurchaseReceipt, 1),
		updateCharacterCh:  make(chan connection.CharacterData, 1),
		updateBackgroundCh: make(chan string, 1),

		shopGrid: createGridUI(),

		backgroundShopBG: backgroundShopBG,
		characterShopBG:  characterShopBG,
		backgroundCardBG: backgroundCardBG,
		characterCardBG:  characterCardBG,

		backgroundRect: textureBounds,

		pagePos:  rl.Rectangle{580, 544, 121, 31},
		moneyPos: rl.Rectangle{518, 19, 156, 33},

		backgroundsBounds: createClickBounds(backgroundsBounds),
		charactersBounds:  createClickBounds(charactersBounds),

		purchasesBtn: purchasesBtn,
		leftBtn:      leftBtn,
		rightBtn:     rightBtn,
	}
}

func (s *ShopUI) Unload() {
	s.shopGrid.unload()
	rl.UnloadTexture(s.backgroundShopBG)
	rl.UnloadTexture(s.characterShopBG)
	rl.UnloadTexture(s.backgroundCardBG)
	rl.UnloadTexture(s.characterCardBG)
	s.unloadPreview()
	s.purchasesBtn.Unload()
	s.leftBtn.Unload()
	s.rightBtn.Unload()
}

func (s *ShopUI) Draw(money int, scaleX, scaleY float32) {
	switch s.state {
	case shopWaitingForResponse:
		s.drawShopLoading(scaleX, scaleY)
	case shopBackgrounds:
		s.drawShopBackgrounds(scaleX, scaleY)
	case shopCharacters:
		s.drawShopCharacters(scaleX, scaleY)
	}

	s.purchasesBtn.Draw(scaleX, scaleY)
	s.drawPlayerMoney(money, scaleX, scaleY)
	drawFieldText(fmt.Sprintf("%d", s.page+1), s.pagePos, scaleX, scaleY, 18, 1, s.primaryColor)
	s.leftBtn.Draw(scaleX, scaleY)
	s.rightBtn.Draw(scaleX, scaleY)
}

func (s *ShopUI) drawPlayerMoney(money int, scaleX, scaleY float32) {
	const (
		fontSize    = float32(18)
		fontSpacing = 1
	)
	textMoney := TrimTextWithEllipsis(fontSize, fontSpacing, fmt.Sprintf("%d", money), s.moneyPos.Width)
	textMoneySize := rl.MeasureTextEx(font, textMoney, fontSize, fontSpacing)
	textMoneyPos := rl.Vector2{
		X: scaleX * (s.moneyPos.X + (s.moneyPos.Width - textMoneySize.X)),
		Y: scaleY * (s.moneyPos.Y + (s.moneyPos.Height-textMoneySize.Y)/2),
	}

	rl.DrawTextEx(font, fmt.Sprintf("%d", money), textMoneyPos, fontSize*scaleY, fontSpacing*scaleX, s.primaryColor)
}

func (s *ShopUI) drawShopLoading(scaleX, scaleY float32) {
	const (
		fontSize    = float32(19)
		fontSpacing = 1
	)
	waiting := rl.Rectangle{X: 11, Y: 62, Width: 1259, Height: 521}

	drawTexture(s.backgroundShopBG, s.backgroundRect, s.backgroundRect, scaleX, scaleY)
	drawFieldText("Загрузка...", waiting, scaleX, scaleY, fontSize, fontSpacing, rl.Red)
}

func (s *ShopUI) drawShopBackgrounds(scaleX, scaleY float32) {
	s.charactersBounds.Scale(scaleX, scaleY)

	drawTexture(s.backgroundShopBG, s.backgroundRect, s.backgroundRect, scaleX, scaleY)
	s.shopGrid.drawBackgroundGrid(s.currentPageBackgrounds, scaleX, scaleY)

}

func (s *ShopUI) drawShopCharacters(scaleX, scaleY float32) {
	s.backgroundsBounds.Scale(scaleX, scaleY)

	drawTexture(s.characterShopBG, s.backgroundRect, s.backgroundRect, scaleX, scaleY)
	s.shopGrid.drawCharacterGrid(s.currentPageCharacters, scaleX, scaleY)
}

func (s *ShopUI) HandleInput(conn net.Conn, player *Player, gameState *string) {
	select {
	case purchaseReceipt := <-s.purchaseReceiptCh:
		s.updateShopData(player, purchaseReceipt)
	case assetPath := <-s.updateBackgroundCh:
		player.LoadBackground(assetPath)
	case characterData := <-s.updateCharacterCh:
		player.character.UpdateCharacter(characterData, Right)
	default:
	}

	switch {
	case rl.IsKeyReleased(rl.KeyEscape):
		s.resetState()
		s.unloadPreview()
		s.state = shopWaitingForResponse
		*gameState = stateMenu
	case s.state == shopWaitingForResponse:
		s.stateWaitingForResponse()
	case s.state == shopBackgrounds:
		s.stateShopBackgrounds(conn)
	case s.state == shopCharacters:
		s.stateShopCharacters(conn)

	}
}

func (s *ShopUI) resetState() {
	s.purchasesBtn.Released()
	s.shopGrid.isPurchases = false
	s.shopGrid.descCardIndex = -1
	s.page = 0
}

func (s *ShopUI) stateWaitingForResponse() {
	select {
	case data := <-s.shopDataCh:
		s.shopData = data
		//s.friendsList = data.Friends
		s.loadPreview()
		s.updateCurrentBackgrounds()
		s.state = shopBackgrounds
	default:
	}
}

func (s *ShopUI) stateShopBackgrounds(conn net.Conn) {
	switch {
	case s.charactersBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		s.resetState()
		s.updateCurrentCharacters()
		s.state = shopCharacters
	case s.purchasesBtn.IsHovered() && rl.IsMouseButtonReleased(rl.MouseButtonLeft):
		if s.purchasesBtn.isPressed {
			s.purchasesBtn.Released()
		} else {
			s.purchasesBtn.Pressed()
		}
		s.shopGrid.isPurchases = s.purchasesBtn.isPressed
		s.page = 0
		s.updateCurrentBackgrounds()
	case s.leftBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			s.leftBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			s.leftBtn.Released()
			s.prevPage()
			s.updateCurrentBackgrounds()
		}
	case s.rightBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			s.rightBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			s.rightBtn.Released()
			s.nextPage()
			s.updateCurrentBackgrounds()
		}
	default:
		s.rightBtn.Released()
		s.leftBtn.Released()
		if action := s.shopGrid.handelBackgroundGrid(len(s.currentPageBackgrounds)); action != nil {
			action.ProductID = s.currentPageBackgrounds[action.ProductID].ID
			sendInput(conn, connection.MsgShopAction, action)
		}
	}

}

func (s *ShopUI) updateCurrentBackgrounds() {
	var currentBackgrounds []ShopBackgroundItem
	if s.purchasesBtn.isPressed {
		currentBackgrounds = s.shopData.PurchasedBackgrounds
	} else {
		currentBackgrounds = s.shopData.AvailableBackgrounds
	}
	start := s.page * 8
	end := start + 8
	if end > len(currentBackgrounds) {
		end = len(currentBackgrounds)
	}
	s.currentPageBackgrounds = currentBackgrounds[start:end]
}

func (s *ShopUI) stateShopCharacters(conn net.Conn) {
	switch {
	case s.backgroundsBounds.IsHovered() && rl.IsMouseButtonPressed(rl.MouseButtonLeft):
		s.resetState()
		s.updateCurrentBackgrounds()
		s.state = shopBackgrounds
	case s.purchasesBtn.IsHovered() && rl.IsMouseButtonReleased(rl.MouseButtonLeft):
		if s.purchasesBtn.isPressed {
			s.purchasesBtn.Released()
		} else {
			s.purchasesBtn.Pressed()
		}
		s.shopGrid.isPurchases = s.purchasesBtn.isPressed
		s.page = 0
		s.updateCurrentCharacters()
	case s.leftBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			s.leftBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			s.leftBtn.Released()
			s.prevPage()
			s.updateCurrentCharacters()
		}
	case s.rightBtn.IsHovered():
		if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
			s.rightBtn.Pressed()
		} else if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
			s.rightBtn.Released()
			s.nextPage()
			s.updateCurrentCharacters()
		}
	default:
		s.rightBtn.Released()
		s.leftBtn.Released()
		if action := s.shopGrid.handelCharacterGrid(len(s.currentPageCharacters)); action != nil {
			action.ProductID = s.currentPageCharacters[action.ProductID].ID
			sendInput(conn, connection.MsgShopAction, action)
		}
	}
}

func (s *ShopUI) updateCurrentCharacters() {
	var currentCharacters []ShopCharacterItem
	if s.purchasesBtn.isPressed {
		currentCharacters = s.shopData.PurchasedCharacters
	} else {
		currentCharacters = s.shopData.AvailableCharacters
	}
	start := s.page * 3
	end := start + 3
	if end > len(currentCharacters) {
		end = len(currentCharacters)
	}
	s.currentPageCharacters = currentCharacters[start:end]
}

func (s *ShopUI) updateShopData(player *Player, purchaseReceipt PurchaseReceipt) {
	player.money = purchaseReceipt.RemainingMoney
	switch purchaseReceipt.ProductType {
	case productBackground:
		s.moveBackgroundToPurchase(purchaseReceipt.ProductID)
	case productCharacter:
		s.moveCharacterToPurchase(purchaseReceipt.ProductID)
	default:
	}
}

func (s *ShopUI) moveBackgroundToPurchase(productID int) {
	for i, item := range s.shopData.AvailableBackgrounds {
		if item.ID == productID {
			s.shopData.PurchasedBackgrounds = append(s.shopData.PurchasedBackgrounds, item)
			s.shopData.AvailableBackgrounds = append(s.shopData.AvailableBackgrounds[:i], s.shopData.AvailableBackgrounds[i+1:]...)
			s.updateCurrentBackgrounds()
			return
		}
	}
}

func (s *ShopUI) moveCharacterToPurchase(productID int) {
	for i, item := range s.shopData.AvailableCharacters {
		if item.ID == productID {
			s.shopData.PurchasedCharacters = append(s.shopData.PurchasedCharacters, item)
			s.shopData.AvailableCharacters = append(s.shopData.AvailableCharacters[:i], s.shopData.AvailableCharacters[i+1:]...)
			s.updateCurrentCharacters()
			return
		}
	}
}

func (s *ShopUI) totalPages() int {
	var totalPages int
	switch s.state {
	case shopBackgrounds:
		if s.purchasesBtn.isPressed {
			totalPages = (len(s.shopData.PurchasedBackgrounds) + 7) / 8
		} else {
			totalPages = (len(s.shopData.AvailableBackgrounds) + 7) / 8
		}
	case shopCharacters:
		if s.purchasesBtn.isPressed {
			totalPages = (len(s.shopData.PurchasedCharacters) + 1) / 3
		} else {
			totalPages = (len(s.shopData.AvailableCharacters) + 1) / 3
		}
	default:
	}
	if totalPages == 0 {
		totalPages = 1
	}

	return totalPages
}

func (s *ShopUI) nextPage() {
	s.page = (s.page + 1) % s.totalPages()
}

func (s *ShopUI) prevPage() {
	s.page = (s.page - 1 + s.totalPages()) % s.totalPages()
}

func (s *ShopUI) loadPreview() {
	for i := range s.shopData.AvailableBackgrounds {
		item := &s.shopData.AvailableBackgrounds[i]
		if item.AssetPath != "" && item.Preview.ID == 0 {
			item.Preview = rl.LoadTexture(currentDirectory + item.AssetPath)
		}
	}

	for i := range s.shopData.PurchasedBackgrounds {
		item := &s.shopData.PurchasedBackgrounds[i]
		if item.AssetPath != "" && item.Preview.ID == 0 {
			item.Preview = rl.LoadTexture(currentDirectory + item.AssetPath)
		}
	}

	for i := range s.shopData.AvailableCharacters {
		item := &s.shopData.AvailableCharacters[i]
		if item.AssetPath != "" && item.Preview.ID == 0 {
			item.Preview = rl.LoadTexture(currentDirectory + item.AssetPath)
		}
	}

	for i := range s.shopData.PurchasedCharacters {
		item := &s.shopData.PurchasedCharacters[i]
		if item.AssetPath != "" && item.Preview.ID == 0 {
			item.Preview = rl.LoadTexture(currentDirectory + item.AssetPath)
		}
	}
}

func (s *ShopUI) unloadPreview() {
	for i := range s.shopData.AvailableBackgrounds {
		item := &s.shopData.AvailableBackgrounds[i]
		if item.Preview.ID != 0 {
			rl.UnloadTexture(item.Preview)
			item.Preview = rl.Texture2D{}
		}
	}

	for i := range s.shopData.PurchasedBackgrounds {
		item := &s.shopData.PurchasedBackgrounds[i]
		if item.Preview.ID != 0 {
			rl.UnloadTexture(item.Preview)
			item.Preview = rl.Texture2D{}
		}
	}

	for i := range s.shopData.AvailableCharacters {
		item := &s.shopData.AvailableCharacters[i]
		if item.Preview.ID != 0 {
			rl.UnloadTexture(item.Preview)
			item.Preview = rl.Texture2D{}
		}
	}

	for i := range s.shopData.PurchasedCharacters {
		item := &s.shopData.PurchasedCharacters[i]
		if item.Preview.ID != 0 {
			rl.UnloadTexture(item.Preview)
			item.Preview = rl.Texture2D{}
		}
	}
}
