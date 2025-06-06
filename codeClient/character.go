package main

import (
	"codeClient/connection"
	rl "github.com/gen2brain/raylib-go/raylib"
	"net"
	"time"
)

// type State // Тип состояния персонажа
type Cmd uint8 // Тип для отправки команд управления персонажем на сервер

const (
	// Состояния персонажа
	Idle        = "Idle" //State = iota
	Run         = "Run"
	Attack      = "Attack"
	HeavyAttack = "HeavyAttack"
	Death       = "Death"
	Jump        = "Jump"
	Fall        = "Fall"
	TakeHit     = "TakeHit"
	Medallion   = "Medallion"
)

const (
	// Физические параметры
	Right             = float32(1)
	Left              = float32(-1)
	Speed             = float32(269)
	JumpVelocity      = float32(-550)
	Gravity           = float32(1000)
	PhysicsUpdateRate = 3
)

const (
	// Команды для отправки действий на сервер
	cmdRunRight Cmd = iota + 1
	cmdRunLeft
	cmdStopRun
	cmdStartJump
	cmdStopJump
	cmdAttack
	cmdHeavyAttack
)

// Ожидающие подтверждения от сервера команды
type PendingCommand struct {
	command                                              Cmd
	X, Y                                                 float32
	yVelocity                                            float32
	direction                                            float32
	isRunning, isJumping, isAttacking, isDying, isBattle bool
	timeSent                                             int64
}

// Команды персонажа отправленные на сервер
type Action struct {
	Id      int `msgpack:"i"`
	Command Cmd `msgpack:"c"`
}

// Подверждение команды от сервера
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

type Character struct {
	name                                                 string
	description                                          string
	health                                               int
	damage                                               int
	cost                                                 int
	defaultHealth, height                                int // Высота персонажа без оружия
	xStart, yStart                                       float32
	xFrame, yFrame, yVelocity                            float32
	isRunning, isJumping, isAttacking, isDying, isBattle bool
	afterAttack                                          chan struct{}
	currentState                                         string
	currentFrame                                         int
	direction, directionRun                              float32
	animationTicker                                      *time.Ticker
	physicsTicker                                        *time.Ticker
	assets                                               map[string]*connection.AssetsCharacter
	totalNumberCommands                                  int
	pendingCommands                                      map[int]*PendingCommand
}

func CreateCharacter(data connection.CharacterData, direction float32) *Character {
	var character Character
	character.name = data.Name
	character.description = data.Description
	character.defaultHealth, character.health = data.Health, data.Health
	character.damage = data.Damage
	character.cost = data.Cost
	character.height = data.HCharacter
	character.xStart, character.yStart = float32(data.XStart), float32(data.YStart)
	character.xFrame, character.yFrame, character.yVelocity = float32(data.XStart), float32(data.YStart), 0
	character.isRunning, character.isJumping, character.isAttacking, character.isDying, character.isBattle = false, false, false, false, false
	character.afterAttack = make(chan struct{})
	character.currentState = Idle
	character.currentFrame = 0
	character.direction, character.directionRun = direction, direction
	character.assets = make(map[string]*connection.AssetsCharacter)
	character.LoadTextures(data.Assets)
	character.StartAnimation()
	character.totalNumberCommands = 0
	character.pendingCommands = make(map[int]*PendingCommand)

	return &character
}

func (ch *Character) UpdateCharacter(data connection.CharacterData, direction float32) {
	ch.UnloadTextures()
	ch.StopAnimation()

	ch.name = data.Name
	ch.description = data.Description
	ch.defaultHealth, ch.health = data.Health, data.Health
	ch.damage = data.Damage
	ch.cost = data.Cost
	ch.height = data.HCharacter
	ch.xStart, ch.yStart = float32(data.XStart), float32(data.YStart)
	ch.xFrame, ch.yFrame, ch.yVelocity = float32(data.XStart), float32(data.YStart), 0
	ch.isRunning, ch.isJumping, ch.isAttacking, ch.isDying, ch.isBattle = false, false, false, false, false
	ch.currentState = Idle
	ch.currentFrame = 0
	ch.direction, ch.directionRun = direction, direction
	ch.assets = make(map[string]*connection.AssetsCharacter)
	ch.LoadTextures(data.Assets)

	ch.StartAnimation()
}

func (ch *Character) StartPhysics(conn net.Conn) {
	if ch.physicsTicker != nil {
		ch.physicsTicker.Stop()
	}

	ch.physicsTicker = time.NewTicker(time.Millisecond * PhysicsUpdateRate)
	go func() {
		for range ch.physicsTicker.C {
			ch.Update(conn)
		}
	}()
}

func (ch *Character) StopPhysics() {
	if ch.physicsTicker != nil {
		ch.physicsTicker.Stop()
	}
}

func (ch *Character) AddCommand(c Cmd) {
	cmd := PendingCommand{
		command:     c,
		X:           ch.xFrame,
		Y:           ch.yFrame,
		yVelocity:   ch.yVelocity,
		direction:   ch.direction,
		isRunning:   ch.isRunning,
		isJumping:   ch.isJumping,
		isAttacking: ch.isAttacking,
		isDying:     ch.isDying,
		isBattle:    ch.isBattle,
		timeSent:    time.Now().UnixMilli(),
	}
	ch.pendingCommands[ch.totalNumberCommands] = &cmd
	ch.totalNumberCommands++
}

// Функция обработки управления персонажем
func (ch *Character) Control(conn net.Conn, gameState *string) {
	if rl.IsKeyPressed(rl.KeyF9) && *gameState != stateBattle && ch.currentState == Idle {
		*gameState = stateMenu
	}

	// Действия персонажа
	if ch.isDying {
		return
	}

	if rl.IsKeyPressed(rl.KeyD) {
		ch.directionRun = Right
	} else if rl.IsKeyPressed(rl.KeyA) {
		ch.directionRun = Left
	}

	if ch.isAttacking {
		return
	}

	if rl.IsKeyPressed(rl.KeyD) || rl.IsKeyPressed(rl.KeyA) {
		if ch.directionRun != ch.direction && ch.isRunning { // отправлять стоп при переключении направления
			sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdStopRun})
			ch.AddCommand(cmdStopRun)
		}
		if rl.IsKeyPressed(rl.KeyD) {
			sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdRunRight})
			ch.AddCommand(cmdRunRight)
		} else {
			sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdRunLeft})
			ch.AddCommand(cmdRunLeft)
		}
		ch.isRunning = true
		ch.direction = ch.directionRun
		if ch.currentState != Run && !ch.isJumping {
			ch.currentState = Run
			ch.StartAnimation()
		}
	}

	if rl.IsKeyReleased(rl.KeyD) && ch.direction == Right || rl.IsKeyReleased(rl.KeyA) && ch.direction == Left {
		sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdStopRun})
		ch.AddCommand(cmdStopRun)
		if !ch.isJumping && ch.currentState != Idle {
			if ch.currentState != Idle {
				ch.currentState = Idle
				ch.StartAnimation()
			}
		}
		ch.isRunning = false
	}

	if !ch.isJumping && !ch.isRunning {
		if ch.currentState != Idle {
			ch.currentState = Idle
			ch.StartAnimation()
		}
	}

	if rl.IsKeyPressed(rl.KeySpace) && !ch.isJumping {
		if ch.currentState != Jump {
			sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdStartJump})
			ch.AddCommand(cmdStartJump)
			ch.isJumping = true
			ch.currentState = Jump
			ch.yVelocity = JumpVelocity
			ch.StartAnimation()
		}
	}

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if ch.currentState != Attack {
			if ch.isRunning {
				sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdStopRun})
				ch.AddCommand(cmdStopRun)
				ch.isRunning = false
			}
			sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdAttack})
			ch.AddCommand(cmdAttack)
			ch.isAttacking = true
			ch.currentState = Attack
			ch.StartAnimation()
		}
	} else if rl.IsMouseButtonPressed(rl.MouseRightButton) {
		if ch.currentState != HeavyAttack {
			if ch.isRunning {
				sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdStopRun})
				ch.AddCommand(cmdStopRun)
				ch.isRunning = false
			}
			sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdHeavyAttack})
			ch.AddCommand(cmdHeavyAttack)
			ch.isAttacking = true
			ch.currentState = HeavyAttack
			ch.StartAnimation()
		}
	}
}

// Сменить анимацию после последнего кадра атаки
func (ch *Character) NextAnimationAfterAttack(conn net.Conn) {
	if ch.isJumping {
		if rl.IsKeyDown(rl.KeyD) && ch.directionRun == Right || rl.IsKeyDown(rl.KeyA) && ch.directionRun == Left {
			if ch.directionRun == Right {
				sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdRunRight})
				ch.AddCommand(cmdRunRight)
			} else {
				sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdRunLeft})
				ch.AddCommand(cmdRunLeft)
			}
			ch.isRunning = true
			ch.direction = ch.directionRun
		}
		ch.currentState = Jump
	} else {
		if rl.IsKeyDown(rl.KeyD) && ch.directionRun == Right || rl.IsKeyDown(rl.KeyA) && ch.directionRun == Left {
			if ch.directionRun == Right {
				sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdRunRight})
				ch.AddCommand(cmdRunRight)
			} else {
				sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdRunLeft})
				ch.AddCommand(cmdRunLeft)
			}
			ch.currentState = Run
			ch.isRunning = true
			ch.direction = ch.directionRun
		} else {
			ch.currentState = Idle
		}
	}
	ch.StartAnimation()
}

func (ch *Character) Dead() {
	ch.health = 0
	ch.isDying = true
	ch.isAttacking = false
	ch.isRunning = false
	if ch.yFrame == ch.yStart {
		ch.currentState = Death
	}
	ch.StartAnimation()
}

// Обновление состояния персонажа
func (ch *Character) Update(conn net.Conn) {
	dt := float32(PhysicsUpdateRate) / 1000 // Для изменения положения на фиксированную величину вне зависимости от частоты кадров

	select {
	case <-ch.afterAttack:
		if conn != nil {
			ch.NextAnimationAfterAttack(conn)
			ch.isAttacking = false
		}
	default:
		if ch.isRunning {
			if ch.direction == Right && ch.xFrame < float32(baseWidth-ch.assets[Attack].BaseWidth)-ch.xStart {
				ch.xFrame += Speed * ch.direction * dt
			} else if ch.direction == Left && ch.xFrame > ch.xStart {
				ch.xFrame += Speed * ch.direction * dt
			}
		}

		if !ch.isJumping && ch.yFrame < ch.yStart {
			ch.yVelocity = 0
			ch.isJumping = true
		}

		if ch.isJumping {
			ch.yFrame += ch.yVelocity*dt + 0.5*Gravity*dt*dt
			ch.yVelocity += Gravity * dt

			if ch.yFrame < -float32(ch.assets[Attack].BaseHeight-ch.height) {
				ch.yFrame = -float32(ch.assets[Attack].BaseHeight - ch.height)
				ch.yVelocity = 0
			}

			if ch.yFrame >= ch.yStart {
				if conn != nil {
					sendInput(conn, connection.MsgActionCharacter, Action{ch.totalNumberCommands, cmdStopJump})
					ch.AddCommand(cmdStopJump)
				}

				ch.yFrame = ch.yStart
				ch.yVelocity = 0
				ch.isJumping = false

				if !ch.isAttacking {
					if ch.isDying {
						ch.currentState = Death
					} else if ch.isRunning {
						ch.currentState = Run
						ch.direction = ch.directionRun
					} else {
						ch.currentState = Idle
					}
					ch.StartAnimation()
				}
			}

			if !ch.isAttacking {
				if ch.yVelocity < 0 { // Проигрываем анимацию прыжка вверх
					ch.currentState = Jump
				} else if ch.yVelocity > 0 { // Проигрываем анимацию падения
					ch.currentState = Fall
				}
			}
		}
	}
}

func (ch *Character) DeleteCommands(id int) {
	keysToDelete := []int{}
	for k := range ch.pendingCommands {
		if k <= id {
			keysToDelete = append(keysToDelete, k)
		}
	}
	for _, k := range keysToDelete {
		delete(ch.pendingCommands, k)
	}
}

func (ch *Character) ApplyCommand(curCommand *PendingCommand, elapsedTime float32, xCorrected, yCorrected, yVelocity *float32) {
	if curCommand.isRunning {
		*xCorrected += curCommand.direction * Speed * elapsedTime
		if *xCorrected >= float32(baseWidth-ch.assets[Attack].BaseWidth)-ch.xStart {
			*xCorrected = float32(baseWidth-ch.assets[Attack].BaseWidth) - ch.xStart
		} else if *xCorrected <= ch.xStart {
			*xCorrected = ch.xStart
		}
	}

	if curCommand.isJumping {
		*yCorrected += *yVelocity*elapsedTime + 0.5*Gravity*elapsedTime*elapsedTime
		*yVelocity += Gravity * elapsedTime
		curCommand.yVelocity = *yVelocity
		if *yCorrected < -float32(ch.assets[Attack].BaseHeight-ch.height) {
			*yCorrected = -float32(ch.assets[Attack].BaseHeight - ch.height)
		} else if *yCorrected >= ch.yStart {
			*yCorrected = ch.yStart
		}
	}
}

// Коррекция положения с учетом задержки
func (ch *Character) ReplayCommands(response ActionResult) {

	if response.IsDying {
		ch.Dead()
	}

	startId := response.CommandID
	predCommand, ok := ch.pendingCommands[startId]
	if !ok {
		return
	}

	var yVelocity float32
	xCorrected, yCorrected := response.X, response.Y

	for id := startId + 1; ; id++ {
		curCommand, ok := ch.pendingCommands[id]
		if !ok {
			break
		}
		elapsedTime := float32(curCommand.timeSent-predCommand.timeSent) / 1000.0
		if predCommand.isJumping == false {
			yVelocity = JumpVelocity
		} else {
			yVelocity = predCommand.yVelocity
		}

		ch.ApplyCommand(curCommand, elapsedTime, &xCorrected, &yCorrected, &yVelocity)
		predCommand = curCommand
	}
	curCommand := &PendingCommand{
		command:     cmdStartJump,
		X:           ch.xFrame,
		Y:           ch.yFrame,
		yVelocity:   ch.yVelocity,
		direction:   ch.direction,
		isRunning:   ch.isRunning,
		isJumping:   ch.isJumping,
		isAttacking: ch.isAttacking,
		isDying:     ch.isDying,
		isBattle:    ch.isBattle,
		timeSent:    time.Now().UnixMilli(),
	}

	elapsedTime := float32(curCommand.timeSent-predCommand.timeSent) / 1000.0
	if predCommand.isJumping == false {
		yVelocity = JumpVelocity
	} else {
		yVelocity = predCommand.yVelocity
	}

	ch.ApplyCommand(curCommand, elapsedTime, &xCorrected, &yCorrected, &yVelocity)

	ch.xFrame = xCorrected
	if ch.yFrame == ch.yStart && ch.yFrame-1 <= yCorrected {
		yCorrected = ch.yStart
	}
	ch.yFrame = yCorrected
	ch.yVelocity = yVelocity

	ch.DeleteCommands(response.CommandID)
}

func (ch *Character) ChangeState(data ActionResult) {
	ch.direction = data.Direction
	ch.xFrame = data.X
	ch.yFrame = data.Y

	if data.IsDying {
		if !ch.isDying {
			ch.Dead()
		}
	} else {
		ch.isDying = false
	}

	if data.IsAttacking {
		if !ch.isAttacking {
			if data.TypeAttack == int8(cmdAttack) {
				ch.currentState = Attack
			} else {
				ch.currentState = HeavyAttack
			}
			ch.StartAnimation()
		}
		ch.isAttacking = true
	} else {
		ch.isAttacking = false
	}

	if data.IsJumping {
		if !ch.isJumping {
			ch.currentState = Jump
			ch.yVelocity = JumpVelocity
			ch.StartAnimation()
		}
		ch.isJumping = true
	} else {
		ch.isJumping = false
	}

	if data.IsRunning {
		if ch.currentState != Run && !ch.isJumping {
			ch.currentState = Run
			ch.StartAnimation()
		}
		ch.isRunning = true
	} else {
		ch.isRunning = false
	}

	if !data.IsDying && !data.IsAttacking && !data.IsJumping && !data.IsRunning {
		if ch.currentState != Idle {
			//log.Println("Idle")
			ch.currentState = Idle
			ch.StartAnimation()
		}
	}
}

func (ch *Character) HealthUpdate(health int) {
	ch.health = health
}

// Сброс персонажа до стартовых параметров
func (ch *Character) RestoreState(data ActionResult) {
	ch.health = data.Health
	ch.xFrame, ch.yFrame = data.X, data.Y
	ch.direction, ch.directionRun = data.Direction, data.Direction
	ch.yVelocity = 0
	ch.isJumping = data.IsJumping
	ch.isRunning = data.IsRunning
	ch.isAttacking = data.IsAttacking
	ch.isDying = data.IsDying

	ch.currentState = Idle
	ch.StartAnimation()
	ch.pendingCommands = make(map[int]*PendingCommand)
}

// Начало анимации персонажа с выбранной частотой смены кадров
func (ch *Character) StartAnimation() {
	ch.currentFrame = 0
	if ch.animationTicker != nil {
		ch.animationTicker.Stop()
	}
	frameDuration := time.Second / time.Duration(ch.assets[ch.currentState].FrameRate)
	ch.animationTicker = time.NewTicker(frameDuration)

	go func() {
		for range ch.animationTicker.C {
			ch.UpdateAnimationFrame()
		}
	}()
}

func (ch *Character) StopAnimation() {
	if ch.animationTicker != nil {
		ch.animationTicker.Stop()
	}
}

// Смена кадров анимации
func (ch *Character) UpdateAnimationFrame() {
	asset := ch.assets[ch.currentState]

	if ch.currentFrame < asset.FrameCount-1 {
		ch.currentFrame++
	} else {
		if ch.isAttacking {
			ch.afterAttack <- struct{}{}
			return
		}
		if !ch.isDying {
			ch.currentFrame = 0
		}
	}

}

func (ch *Character) Draw(scaleX, scaleY float32) {
	asset := ch.assets[ch.currentState]

	rl.DrawTexturePro(
		asset.Texture,
		rl.Rectangle{X: float32(asset.Texture.Width) / float32(asset.FrameCount) * float32(ch.currentFrame), Y: 0, Width: ch.direction * float32(asset.Texture.Width) / float32(asset.FrameCount), Height: float32(asset.Texture.Height)}, // Исходная область
		rl.Rectangle{X: ch.xFrame * scaleX, Y: ch.yFrame * scaleY, Width: float32(asset.BaseWidth) * scaleX, Height: float32(asset.BaseHeight) * scaleY},                                                                                  // Область экрана
		rl.Vector2{X: 0, Y: 0}, // Центр поворота
		0,                      // Угол поворота
		rl.White,               // Цвет
	)
}

func (ch *Character) LoadTextures(data map[string]connection.AssetsCharacter) {
	for typeAs, as := range data {
		as.Texture = rl.LoadTexture(currentDirectory + as.AssetPath)
		ch.assets[typeAs] = &as
	}
}

func (ch *Character) UnloadTextures() {
	for typeAs, asset := range ch.assets {
		rl.UnloadTexture(asset.Texture)
		asset.Texture = rl.Texture2D{}
		ch.assets[typeAs] = asset
	}
}
