package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"log"
	"time"
)

type Cmd uint8

const (
	ScreenWidth  = 1280 // Базовый размер окна
	ScreenHeight = 720  // Базовый размер окна
)

const (
	// Физические параметры
	GroundLevel  = 25 // Расстояние от низа экрана, до ног персонажа (низа кадра)
	Right        = float32(1)
	Left         = float32(-1)
	Speed        = float32(269)
	JumpVelocity = float32(-550)
	Gravity      = float32(1000)
)

const (
	// Команды действий
	cmdRunRight Cmd = iota + 1
	cmdRunLeft
	cmdStopRun
	cmdStartJump
	cmdStopJump
	cmdAttack
	cmdHeavyAttack
)

// Хранит данные персонажа клиента
type CharacterState struct {
	Health               int
	XStart, YStart       float32
	X, Y                 float32 // Координаты верхнего левого угла кадра с персонажем относительно экрана размером 1280 * 720
	Direction            float32 // Направление движения (лево, право)
	VelocityY            float32
	isJumping            bool
	isRunning            bool
	isAttacking          bool
	typeAttack           int8
	isDying              bool
	Died                 chan struct{}
	inBattle             bool
	LastRunTime          int64 // Время последнего передвижения
	LastJumpTime         int64 // Время начала прыжка
	DirectionAfterAttack float32
	isJumpingAfterAttack bool
	isRunningAfterAttack bool
}

// Ответ клиенту на действия персонажа
type ActionResult struct {
	Timestamp   int64   `msgpack:"t"` // Время на сервере, когда событие было обработано
	CommandID   int     `msgpack:"id"`
	Health      int     `msgpack:"h"`
	Direction   float32 `msgpack:"di"`
	X           float32 `msgpack:"x"`
	Y           float32 `msgpack:"y"`
	IsDying     bool    `msgpack:"dy"`
	IsAttacking bool    `msgpack:"a"`
	TypeAttack  int8    `msgpack:"ta"`
	IsJumping   bool    `msgpack:"j"`
	IsRunning   bool    `msgpack:"r"`
}

// Обновление здоровья персонажа
type healthUpdate struct {
	Who    MessageType `msgpack:"w"`
	Health int         `msgpack:"h"`
}

// Действие персонажа
type Action struct {
	Id      int `msgpack:"i"`
	Command Cmd `msgpack:"c"`
}

// Обновляет активного персонажа
func (ch *CharacterState) Update(idActiveCharacter int) {
	ch.Health = activeCharacters[idActiveCharacter].Health
	ch.XStart = float32(-activeCharacters[idActiveCharacter].XBoundary)                                 // Верхний левый угл
	ch.YStart = float32(ScreenHeight - (GroundLevel + activeCharacters[idActiveCharacter].FrameHeight)) // Верхний левый угл
	ch.X = ch.XStart
	ch.Y = ch.YStart
	ch.Direction = Right
	ch.isJumping = false
	ch.isAttacking = false
	ch.isRunning = false
	ch.isDying = false
	ch.Died = make(chan struct{})
	ch.inBattle = false
}

// Обрабатывает команды персонажа
func actionCharacter(action Action, client *Client, opponent *Client) {
	command := action.Command
	idCmd := action.Id
	state := client.State

	if state.isDying {
		return
	}

	if command != cmdStopJump && state.isAttacking { // Если команда на движение пришла до того, как атака закончилась на сервере
		if command == cmdRunRight {
			state.isRunningAfterAttack = true
			state.DirectionAfterAttack = Right
		} else if command == cmdRunLeft {
			state.isRunningAfterAttack = true
			state.DirectionAfterAttack = Left
		} else if command == cmdStartJump {
			state.isJumpingAfterAttack = true
		}
		return
	}

	switch command {
	case cmdRunRight:
		state.LastRunTime = time.Now().UnixMilli()
		state.isRunning = true
		state.Direction = Right
		currentPositionCharacter(client)
	case cmdRunLeft:
		state.LastRunTime = time.Now().UnixMilli()
		state.isRunning = true
		state.Direction = Left
		currentPositionCharacter(client)
	case cmdStopRun:
		currentPositionCharacter(client)
		if state.isRunning {
			state.isRunning = false
		}
	case cmdStartJump:
		currentPositionCharacter(client)
		if !state.isJumping {
			state.LastJumpTime = time.Now().UnixMilli()
			state.isJumping = true
			state.VelocityY = JumpVelocity
		}
	case cmdStopJump:
		currentPositionCharacter(client)
	case cmdAttack:
		if state.isRunning {
			actionCharacter(Action{idCmd - 1, cmdStopRun}, client, opponent)
		}
		state.isAttacking = true
		state.typeAttack = int8(cmdAttack)
		go attackCharacter(client, opponent, "Attack")
		currentPositionCharacter(client)
	case cmdHeavyAttack:
		if state.isRunning {
			actionCharacter(Action{idCmd - 1, cmdStopRun}, client, opponent)
		}
		state.isAttacking = true
		state.typeAttack = int8(cmdHeavyAttack)
		go attackCharacter(client, opponent, "HeavyAttack")
		currentPositionCharacter(client)
	default:
		log.Printf("Неизвестная команда: %d, клиент: %s", command, client.UserID)
		createAndSendMessage(client, MsgError, fmt.Sprintf("Неизвестная команда: %d", command))
		return
	}
	sendCharacterState(client, opponent, idCmd, MsgActionCharacter)
}

// Обновляет позицию игрока
func currentPositionCharacter(client *Client) {
	state := client.State
	ch := activeCharacters[client.ActiveCharacter]
	if state.isRunning {
		elapsedRun := float32(time.Now().UnixMilli()-state.LastRunTime) / 1000.0
		state.LastRunTime = time.Now().UnixMilli()
		state.X += state.Direction * Speed * elapsedRun
		if state.X > float32(ScreenWidth-ch.FrameWidth+ch.XBoundary) {
			state.X = float32(ScreenWidth - ch.FrameWidth + ch.XBoundary)
		} else if state.X < float32(-ch.XBoundary) {
			state.X = float32(-ch.XBoundary)
		}
	}
	if !state.isJumping && state.Y < state.YStart {
		state.LastJumpTime = time.Now().UnixMilli()
		state.VelocityY = 0
		state.isJumping = true
	}

	if state.isJumping {
		elapsedJump := float32(time.Now().UnixMilli()-state.LastJumpTime) / 1000.0
		state.LastJumpTime = time.Now().UnixMilli()
		state.Y += state.VelocityY*elapsedJump + 0.5*Gravity*elapsedJump*elapsedJump
		state.VelocityY += Gravity * elapsedJump
		if state.Y < -float32(ch.FrameHeight-ch.HCharacter) {
			state.Y = -float32(ch.FrameHeight - ch.HCharacter)
			state.VelocityY = 0
		} else if state.Y >= state.YStart-1 { // Величина погрешности
			state.Y = state.YStart
			state.VelocityY = 0
			state.isJumping = false
		}
	}
}

// Обрабатывает атаку по оппоненту
func attackCharacter(client, opponent *Client, typeAttack string) {
	damageMultiplier := float32(1)
	if typeAttack == "HeavyAttack" {
		damageMultiplier = 1.75
		if client.State.isJumping {
			damageMultiplier = 2.5
		}
	}

	ch := activeCharacters[client.ActiveCharacter]

	// Задержка на время атаки
	timer := time.NewTimer(ch.TimeAnimation[typeAttack])
	defer timer.Stop()
	<-timer.C
	if client.State.isDying {
		return
	}

	client.State.isAttacking = false
	if client.State.isRunningAfterAttack {
		client.State.LastRunTime = time.Now().UnixMilli()
		client.State.isRunning = true
		client.State.Direction = client.State.DirectionAfterAttack
		client.State.isRunningAfterAttack = false
	}
	if client.State.isJumpingAfterAttack {
		client.State.LastJumpTime = time.Now().UnixMilli()
		client.State.isJumping = true
		client.State.isJumpingAfterAttack = false
	}
	if opponent != nil {
		if checkBitMaskCollision(client, opponent, typeAttack) {
			damage := int(damageMultiplier * float32(ch.Damage))
			takeHit(opponent, client, damage)

			log.Printf(client.Name, " попал и нанес урон ", opponent.Name, " здровоье оппонента: ", opponent.State.Health)
		} else {
			log.Println(client.Name, " не попал по ", opponent.Name)
		}
	}

	fmt.Println(time.Now())
	currentPositionCharacter(client)
	sendCharacterState(client, opponent, -1, MsgActionCharacter)
}

// Обрабатывает получаемый урон
func takeHit(target, attacker *Client, damage int) {
	target.State.Health -= damage
	if target.State.Health <= 0 {
		target.State.Health = 0
		dead(target, attacker)
	} else {
		sendHealthUpdate(target, attacker)
	}
}

// Инициирует смерть персонажа
func dead(target, attacker *Client) {
	target.State.isDying = true
	target.State.isAttacking = false
	target.State.isRunning = false
	target.State.isRunningAfterAttack = false
	target.State.isJumpingAfterAttack = false
	sendCharacterState(target, attacker, -1, MsgActionCharacter)
	target.State.Died <- struct{}{}
}

// Отправляет обновленное здоровье
func sendHealthUpdate(target, attacker *Client) {
	upHp := healthUpdate{
		Who:    MsgActionOpponent,
		Health: target.State.Health,
	}
	createAndSendMessage(attacker, MsgHealthUpdate, upHp)
	upHp.Who = MsgActionCharacter
	createAndSendMessage(target, MsgHealthUpdate, upHp)
}

// Проверка пересечения кадров
func checkBoundingBoxCollision(client, opponent *Client) (bool, rl.Rectangle) {
	chClient := activeCharacters[client.ActiveCharacter]
	chOpponent := activeCharacters[opponent.ActiveCharacter]

	rectClient := rl.Rectangle{X: client.State.X, Y: client.State.Y, Width: float32(chClient.FrameWidth), Height: float32(chClient.FrameHeight)}

	invertedX := ScreenWidth - opponent.State.X - float32(chOpponent.FrameWidth)
	rectOpponent := rl.Rectangle{X: invertedX, Y: opponent.State.Y, Width: float32(chOpponent.FrameWidth), Height: float32(chOpponent.FrameHeight)}

	if rl.CheckCollisionRecs(rectClient, rectOpponent) {
		return true, rl.GetCollisionRec(rectClient, rectOpponent)
	}
	return false, rl.Rectangle{}
}

// Проверка битовой маски с учетом области пересечения
func checkBitMaskCollision(client, opponent *Client, typeAttack string) bool {
	currentPositionCharacter(client)
	currentPositionCharacter(opponent)

	collision, intersect := checkBoundingBoxCollision(client, opponent)
	if !collision {
		return false
	}

	chClient := activeCharacters[client.ActiveCharacter]
	chOpponent := activeCharacters[opponent.ActiveCharacter]

	maskCl := chClient.BitMaskWithWeapon[typeAttack]
	var maskOp []uint64
	if opponent.State.isRunning {
		maskOp = chOpponent.BitMask["Run"]
	} else if opponent.State.isJumping {
		if opponent.State.VelocityY < 0 {
			maskOp = chOpponent.BitMask["Jump"]
		} else {
			maskOp = chOpponent.BitMask["Fall"]
		}
	} else if opponent.State.isAttacking {
		if opponent.State.typeAttack == int8(cmdAttack) {
			maskOp = chOpponent.BitMask["Attack"]
		} else {
			maskOp = chOpponent.BitMask["HeavyAttack"]
		}
	} else {
		maskOp = chOpponent.BitMask["Idle"]
	}

	opponentDirection := -opponent.State.Direction
	opponentX := ScreenWidth - opponent.State.X - float32(chOpponent.FrameWidth)

	for y := int(intersect.Y); y < int(intersect.Y+intersect.Height); y++ {
		for x := int(intersect.X); x < int(intersect.X+intersect.Width); x++ {

			// Преобразуем координаты в локальные для каждого персонажа
			localXClient := x - int(client.State.X)
			localYClient := y - int(client.State.Y)

			localXOpponent := x - int(opponentX)
			localYOpponent := y - int(opponent.State.Y)

			// Если оппонент смотрит влево, отражаем X-координату
			if opponentDirection == -1 {
				localXOpponent = chOpponent.FrameWidth - 1 - localXOpponent
			}

			// Если клиент смотрит влево, отражаем X-координату
			if client.State.Direction == -1 {
				localXClient = chClient.FrameWidth - 1 - localXClient
			}

			if localXClient >= 0 && localXClient < chClient.FrameWidth &&
				localYClient >= 0 && localYClient < chClient.FrameHeight &&
				localXOpponent >= 0 && localXOpponent < chOpponent.FrameWidth &&
				localYOpponent >= 0 && localYOpponent < chOpponent.FrameHeight {

				// Проверяем биты в масках
				idxClient := localYClient*(chClient.FrameWidth/64+1) + localXClient/64
				bitClient := (maskCl[idxClient] >> (localXClient % 64)) & 1

				idxOpponent := localYOpponent*(chOpponent.FrameWidth/64+1) + localXOpponent/64
				bitOpponent := (maskOp[idxOpponent] >> (localXOpponent % 64)) & 1

				if bitClient == 1 && bitOpponent == 1 {
					return true
				}
			}
		}
	}

	return false
}

// Возвращает состояние персонажа
func getCharacterState(client *Client, id int) ActionResult {
	state := client.State
	acMs := ActionResult{
		Timestamp:   time.Now().UnixMilli(),
		CommandID:   id,
		Health:      state.Health,
		Direction:   state.Direction,
		X:           state.X,
		Y:           state.Y,
		IsDying:     state.isDying,
		IsAttacking: state.isAttacking,
		TypeAttack:  state.typeAttack,
		IsJumping:   state.isJumping,
		IsRunning:   state.isRunning,
	}
	return acMs
}

// Отправляет состояние персонажа клиенту
func sendCharacterState(client, opponent *Client, id int, mesType MessageType) {
	chSt := getCharacterState(client, id)
	if opponent != nil {
		ch := activeCharacters[client.ActiveCharacter]
		invertedX := ScreenWidth - client.State.X - float32(ch.FrameWidth)
		sendToOpponentCharacterState(opponent, chSt, invertedX, MsgActionOpponent)
	}

	createAndSendMessage(client, mesType, chSt)

}

// Отправляет состояние персонажа оппоненту
func sendToOpponentCharacterState(opponent *Client, data ActionResult, invertedX float32, mesType MessageType) {
	data.X = invertedX
	data.Direction = -data.Direction

	createAndSendMessage(opponent, mesType, data)
}

// Сбрасывает состояние персонажа до начального
func restoreState(client *Client, mesType MessageType) {
	state := client.State
	state.Health = activeCharacters[client.ActiveCharacter].Health
	state.X, state.Y = state.XStart, state.YStart
	state.Direction = Right
	state.VelocityY = 0
	state.isJumping = false
	state.isRunning = false
	state.isAttacking = false
	state.isDying = false
	state.isJumpingAfterAttack = false
	state.isRunningAfterAttack = false

	if mesType != MsgNone {
		sendCharacterState(client, nil, -1, mesType)
	}
}
