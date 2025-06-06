package main

import (
	"codeClient/connection"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Player struct {
	publicID   string
	login      string
	name       string
	level      int
	money      int
	rank       int
	background rl.Texture2D
	character  *Character
}

type Opponent struct {
	publicID  string
	name      string
	level     int
	rank      int
	character *Character
}

func CreatePlayer(data *connection.UserData) *Player {
	var player Player
	player.login = data.Login
	player.publicID = data.PublicID
	player.name = data.Name
	player.level = data.Level
	player.money = data.Money
	player.rank = data.Rank
	player.LoadBackground(data.ActiveBackgroundPath)
	player.character = CreateCharacter(data.ActiveCharacter, Right)
	return &player
}

func (p *Player) LoadBackground(backgroundPath string) {
	if p.background.ID > 0 {
		p.UnloadBackground()
	}
	p.background = rl.LoadTexture(currentDirectory + backgroundPath)
}

func (p *Player) UnloadBackground() {
	rl.UnloadTexture(p.background)
}

func (p *Player) ApplyBattleResults(battleResult EndBattleInfo) {
	p.level = battleResult.CurrentLevel
	p.rank = battleResult.CurrentRank
	p.money = battleResult.TotalMoney
}

func (p *Player) UpdateMoney(money int) {
	p.money = money
}

func CreateOpponent(data StartBattleInfo) *Opponent {
	var opponent Opponent
	opponent.publicID = data.OpponentPublicID
	opponent.name = data.OpponentName
	opponent.level = data.OpponentLevel
	opponent.rank = data.OpponentRank
	opponent.character = CreateCharacter(data.OpponentCharacter, Left)
	opponent.character.xFrame = float32(baseWidth-opponent.character.assets[Attack].BaseWidth) - opponent.character.xStart
	opponent.character.StartPhysics(nil)
	return &opponent
}
