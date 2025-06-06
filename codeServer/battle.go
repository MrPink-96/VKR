package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

const (
	MaxRankRange  = 100 // Максимальный диапазон по рангу
	MaxLevelRange = 200 // Максимальный диапазон по уровню
	MatchInterval = 1000 * time.Millisecond
	ExpandTime    = 500 * time.Millisecond // Интервал увеличения диапазона
	BattleTime    = 55 * 2 * time.Second   //time.Minute
)

type BattleResult int8

const (
	NoBattle BattleResult = -2 // Боя не было
	Defeat   BattleResult = -1 // Поражение
	Draw     BattleResult = 0  // Ничья
	Victory  BattleResult = 1  // Победа
)

var (
	matchmakingQueue *MatchmakingQueue // Очереди для боя
)

// Ожидающий клиент очереди на бой
type WaitingClient struct {
	Client       *Client
	EnqueuedTime time.Time // Время добавления в очередь, чтобы расширять диапазон
	RankOrLevel  int
	Range        int // Текущий диапазон поиска
	IsRanked     bool
}

// Очереди на бой
type MatchmakingQueue struct {
	rankMu      sync.Mutex // Для ранговой очереди
	levelMu     sync.Mutex // Для уровневой очереди
	rankedQueue []*WaitingClient
	levelQueue  []*WaitingClient
	stopChan    chan struct{}
}

// Информация о бое для сервера
type Battle struct {
	Player1   *Client
	Player2   *Client
	IsRanked  bool
	Winner    *Client
	StartTime time.Time
	EndTime   time.Time

	Readiness chan struct{}
	EndBattle chan struct{}
}

// Информация о бое для клиента
type StartBattleInfo struct {
	Timestamp         int64        `msgpack:"tt"` // Время на сервере, когда событие было обработано
	StartTime         int64        `msgpack:"st"`
	EndTime           int64        `msgpack:"et"`
	OpponentPublicID  string       `msgpack:"oi"`
	OpponentName      string       `msgpack:"on"`
	OpponentRank      int          `msgpack:"or"`
	OpponentLevel     int          `msgpack:"ol"`
	OpponentCharacter *CharacterDB `msgpack:"oc"`
}

// Информация о результате боя
type EndBattleInfo struct {
	Result       BattleResult  `msgpack:"w,omitempty"`
	TotalMoney   int           `msgpack:"m"` // Итоговый баланс после боя
	CurrentRank  int           `msgpack:"r"` // Текущий ранг после боя
	CurrentLevel int           `msgpack:"l"` // Текущий уровень после боя
	UpdatedStats *ActionResult `msgpack:"u"`
}

// Функция для ожидания противника или запроса на выход из поиска боя
func waitingBattle(client *Client, isRanked bool, friendID string) {
	client.State.inBattle = true
	client.friendID = friendID
	restoreState(client, MsgWaitingBattle)
	if friendID == "" {
		matchmakingQueue.addToQueue(client, isRanked)
	}

	for {
		select {
		case msg, ok := <-client.ReceivedMess:
			if !ok { // Если канал закрыт, то завершить обработку
				log.Printf("Канал сообщений для клиента %d закрыт, завершаем обработку поиска боя", client.UserID)
				matchmakingQueue.removeFromQueue(client, isRanked)
				return
			}

			if msg.Type == MsgExitBattle {
				if friendID == "" {
					matchmakingQueue.removeFromQueue(client, isRanked)
				}
				client.State.inBattle = false
				createAndSendMessage(client, MsgExitBattle, nil)
				return
			} else {
				createAndSendMessage(client, MsgError, "Вы в ожидание боя")
			}

		case battleInfo := <-client.BattleInfo:
			startBattle(client, battleInfo)
			fmt.Println("END ")
			return
		}
	}
}

// Отправляет инфу о бое клиентам
func sendStartBattleInfo(client, opponent *Client, startTime, endTime time.Time) {
	var battleInfo StartBattleInfo
	battleInfo.OpponentPublicID = opponent.PublicID
	battleInfo.OpponentName = opponent.Name
	battleInfo.OpponentRank = opponent.Rank
	battleInfo.OpponentLevel = opponent.Level
	var ch CharacterDB
	ac := activeCharacters[opponent.ActiveCharacter]
	ch.Name = ac.Name
	ch.Description = ac.Description
	ch.Health = ac.Health
	ch.Damage = ac.Damage
	ch.Cost = ac.Cost
	ch.HCharacter = ac.HCharacter
	ch.XStart = -ac.XBoundary
	ch.YStart = ScreenHeight - (GroundLevel + ac.FrameHeight)
	ch.Assets = ac.Assets
	battleInfo.OpponentCharacter = &ch
	battleInfo.StartTime = startTime.UnixMilli()
	battleInfo.EndTime = endTime.UnixMilli()
	battleInfo.Timestamp = time.Now().UnixMilli()

	createAndSendMessage(client, MsgStartBattleInfo, battleInfo)
}

// Старт боя
func startBattle(client *Client, battleInfo *Battle) {
	defer func() {
		client.State.inBattle = false
	}()

	var opponent *Client
	if client == battleInfo.Player1 {
		opponent = battleInfo.Player2
	} else {
		opponent = battleInfo.Player1
	}

	sendStartBattleInfo(client, opponent, battleInfo.StartTime, battleInfo.EndTime)

	select {
	case msg, ok := <-client.ReceivedMess:
		if !ok { // Если канал закрыт, то завершить обработку
			log.Printf("Бой не начнется, т.к канал сообщений для клиента %d закрыт, завершаем обработку", client.UserID)
			close(battleInfo.Readiness)
			break
		}
		if msg.Type == MsgReadyBattle {
			break
		}
	case <-time.After(3 * time.Second):
		close(battleInfo.Readiness)
		break
	}

	var skillDiff int
	if battleInfo.IsRanked {
		skillDiff = client.Rank - opponent.Rank
	} else {
		skillDiff = client.Level - opponent.Level
	}

	for {
		select {
		case msg, ok := <-client.ReceivedMess:
			timeNow := time.Now().UTC()
			switch {
			case !ok:
				client.State.Died <- struct{}{}
				log.Printf("Канал сообщений для клиента %d закрыт, завершаем обработку боя", client.UserID)
				break
			case msg.Type == MsgExitBattle:
				client.State.Died <- struct{}{}
				log.Printf("Клиент %d решил выйти из боя!", client.UserID)
				break

			case msg.Type == MsgActionCharacter && timeNow.After(battleInfo.StartTime) && timeNow.Before(battleInfo.EndTime):
				var action Action
				err := msgpack.Unmarshal(msg.Data, &action)
				if err != nil {
					log.Printf("Ошибка десериализации в startBattle: %v", err)
					createAndSendMessage(client, MsgError, "десериализации данных Action на сервере")
					break
				}
				actionCharacter(action, client, opponent)
			default:
				createAndSendMessage(client, MsgError, "Бой еще не начался или уже закончился!")
			}
		case <-battleInfo.EndBattle:
			finalizeBattleOutcome(client, skillDiff, battleInfo)
			return
		}
	}
}

// Подводит результаты боя
func finalizeBattleOutcome(client *Client, skillDiff int, battleInfo *Battle) {
	restoreState(client, MsgNone)
	endBattleInfo := EndBattleInfo{
		TotalMoney:   client.Money,
		CurrentRank:  client.Rank,
		CurrentLevel: client.Level,
	}
	state := getCharacterState(client, -1)
	endBattleInfo.UpdatedStats = &state

	if battleInfo.Readiness == nil { // Проверка, что канал подтверждения готовности был закрыт, а значит боя не было
		endBattleInfo.Result = NoBattle
		createAndSendMessage(client, MsgEndBattle, endBattleInfo)
		return
	}

	if battleInfo.Winner == nil {
		endBattleInfo.Result = Draw
		grantRewards(client, skillDiff, int(Draw), battleInfo.IsRanked)
	} else if client == battleInfo.Winner {
		endBattleInfo.Result = Victory
		grantRewards(client, skillDiff, int(Victory), battleInfo.IsRanked)
	} else {
		endBattleInfo.Result = Defeat
		grantRewards(client, skillDiff, int(Defeat), battleInfo.IsRanked)
	}
	endBattleInfo.TotalMoney = client.Money
	endBattleInfo.CurrentRank = client.Rank
	endBattleInfo.CurrentLevel = client.Level

	updatePlayerStats(client.PlayerID, client.Level, client.Rank, client.Money)

	createAndSendMessage(client, MsgEndBattle, endBattleInfo)

}

// Начисляет награды за бой
func grantRewards(client *Client, skillDiff, IsWinner int, IsRanked bool) {
	baseProgress := 25.0 // Средний прогресс ранга или уровня
	baseCoins := 50.0    // Награда для ничьи не рангового боя
	effectScale := 0.03  // Крутизна кривой
	effect := math.Tanh(float64(-skillDiff) * effectScale)
	rankMod := 1.0 + float64(IsWinner)*effect
	rankMod = math.Max(0.2, rankMod)

	if IsRanked {
		baseCoins = 2 * baseCoins
		client.Rank += IsWinner * int(rankMod*baseProgress)
		if client.Rank < 0 {
			client.Rank = 0
		}
	}

	levelMod := math.Pow(2, float64(IsWinner)) * (1.0 + effect) * 0.5
	levelMod = math.Max(0.2, levelMod)

	client.Level += int(levelMod * baseProgress)

	moneyMod := math.Pow(2, float64(IsWinner)) * (1.0 + effect)
	moneyMod = math.Max(0.2, moneyMod)

	client.Money += int(moneyMod * baseCoins)
}

// Горутина управления и синхронизации боя
func manageBattle(clientA, clientB *Client, isRanked bool) {
	var battle Battle

	startTime := time.Now().UTC().Add(5 * time.Second).Truncate(time.Second) // Задержка начала боя
	endTime := startTime.Add(BattleTime).Truncate(time.Second)
	Readiness := make(chan struct{})
	chanBattleEnd := make(chan struct{})

	battle.Player1 = clientA
	battle.Player2 = clientB
	battle.IsRanked = isRanked
	battle.Winner = nil
	battle.StartTime = startTime
	battle.EndTime = endTime

	battle.Readiness = Readiness
	battle.EndBattle = chanBattleEnd

	log.Println("БОЙ НАЧИНАЕТСЯ!", clientA.Name, "VS", clientB.Name)
	log.Println("Начало: ", startTime)
	log.Println("Конец: ", endTime)

	// Оповещаем игроков, что бой начался
	clientA.BattleInfo <- &battle
	clientB.BattleInfo <- &battle

	select {
	case <-battle.Readiness: // Один из пользователей не подтвердил готовность, бой завершен досрочно
		fmt.Println("Один из пользователей не подтвердил готовность к бою")
		battle.Readiness = nil
		close(battle.EndBattle)
		return
	case <-clientA.State.Died:
		battle.Winner = clientB

	case <-clientB.State.Died:
		battle.Winner = clientA

	case <-time.After(time.Until(endTime)):
		battle.Winner = nil
	}

	battle.EndTime = time.Now().UTC() // реальное время окончания боя
	saveBattleResultsDB(battle)

	log.Println("Бой ", clientA.Name, " VS ", clientB.Name, " завершился, победитель: ", battle.Winner)
	close(battle.EndBattle)
}

// Добавление игрока в очередь
func (m *MatchmakingQueue) addToQueue(client *Client, isRanked bool) {
	player := &WaitingClient{
		Client:       client,
		EnqueuedTime: time.Now().UTC(),
		Range:        0,
		IsRanked:     isRanked,
	}

	// Вставляем в очередь по возрастанию ранга
	if isRanked {
		m.rankMu.Lock()
		player.RankOrLevel = client.Rank
		m.rankedQueue = append(m.rankedQueue, player)
		sort.Slice(m.rankedQueue, func(i, j int) bool {
			return m.rankedQueue[i].RankOrLevel < m.rankedQueue[j].RankOrLevel
		})
		m.rankMu.Unlock()
	} else {
		m.levelMu.Lock()
		player.RankOrLevel = client.Level
		m.levelQueue = append(m.levelQueue, player)
		sort.Slice(m.levelQueue, func(i, j int) bool {
			return m.levelQueue[i].RankOrLevel < m.levelQueue[j].RankOrLevel
		})
		m.levelMu.Unlock()
	}
}

// Запуск поиска пар в очередях
func (m *MatchmakingQueue) RunSearch() {
	if m.stopChan == nil {
		m.stopChan = make(chan struct{})
	}

	ticker := time.NewTicker(MatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.findMatch(&m.rankedQueue, &m.rankMu)
			m.findMatch(&m.levelQueue, &m.levelMu)
		}
	}
}

// Остановка поиска пар на бой
func (m *MatchmakingQueue) StopSearch() {
	close(m.stopChan)
}

// Функция поиска пары в конкретной очереди
func (m *MatchmakingQueue) findMatch(queue *[]*WaitingClient, mu *sync.Mutex) {
	mu.Lock()
	defer mu.Unlock()

	if len(*queue) < 2 {
		return
	}

	for i := 0; i < len(*queue)-1; i++ {
		player1 := (*queue)[i]
		m.updateMatchmakingRange(player1)

		for j := i + 1; j < len(*queue); j++ {
			player2 := (*queue)[j]
			m.updateMatchmakingRange(player2)

			// Проверяем, подходят ли игроки друг другу
			if player1.RankOrLevel+player1.Range >= player2.RankOrLevel &&
				player2.RankOrLevel+player2.Range >= player1.RankOrLevel {
				go manageBattle(player1.Client, player2.Client, player1.IsRanked)
				removePairFromQueue(queue, i, j)

				return
			}

			// Если player1 уже не может никого найти с таким уровнем, нет смысла проверять дальше
			if player1.RankOrLevel+player1.Range < player2.RankOrLevel {
				break
			}
		}
	}
}

// Удаление пары из очереди
func removePairFromQueue(queue *[]*WaitingClient, i, j int) {
	if i > j {
		i, j = j, i
	}
	last := len(*queue) - 1
	if j != last {
		(*queue)[j] = (*queue)[last]
	}
	*queue = (*queue)[:last]
	last = len(*queue) - 1
	if i != last {
		(*queue)[i] = (*queue)[last]
	}
	*queue = (*queue)[:last]
}

// Удаление игрока из очереди
func (m *MatchmakingQueue) removeFromQueue(client *Client, isRanked bool) {
	if isRanked {
		m.rankMu.Lock()
		defer m.rankMu.Unlock()
		removeClientFromQueue(&m.rankedQueue, client)
	} else {
		m.levelMu.Lock()
		defer m.levelMu.Unlock()
		removeClientFromQueue(&m.levelQueue, client)
	}
}

// Удаление игрока из очереди
func removeClientFromQueue(queue *[]*WaitingClient, client *Client) {
	for i := 0; i < len(*queue); i++ {
		if (*queue)[i].Client == client {
			*queue = append((*queue)[:i], (*queue)[i+1:]...)
			return
		}
	}
}

// Увеличиваем диапазон, если игрок долго в очереди
func (m *MatchmakingQueue) updateMatchmakingRange(client *WaitingClient) {
	elapsed := time.Since(client.EnqueuedTime)

	if elapsed > ExpandTime {
		client.Range += int(elapsed / ExpandTime)
		if client.IsRanked && client.Range > MaxRankRange {
			client.Range = MaxRankRange
		} else if !client.IsRanked && client.Range > MaxLevelRange {
			client.Range = MaxLevelRange
		}
	}
}
