package main

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/vmihailenco/msgpack/v5"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

// Содержит данные авторизованного пользователя
type UserData struct {
	PlayerID             int          `db:"PlayerID" msgpack:"-"`
	Login                string       `db:"Login" msgpack:"l"`
	PublicID             string       `db:"PlayerPublicID" msgpack:"id"`
	Name                 string       `db:"PlayerName" msgpack:"n"`
	Level                int          `db:"PlayerLevel" msgpack:"lv"`
	Money                int          `db:"PlayerMoney" msgpack:"m"`
	Rank                 int          `db:"PlayerRank" msgpack:"r"`
	ActiveBackgroundPath string       `db:"ActiveBackgroundPath" msgpack:"b"`
	ActiveCharacter      *CharacterDB `msgpack:"c"`
}

type FriendEntry struct {
	Name     string `db:"Name"`
	PublicID string `db:"PublicID"`
}

type FriendsData struct {
	Friends  []FriendEntry `msgpack:"f"`
	Incoming []FriendEntry `msgpack:"i"`
	Outgoing []FriendEntry `msgpack:"o"`
}

type BattleEntry struct {
	StartTime        time.Time `db:"StartTime" msgpack:"st"`
	EndTime          time.Time `db:"EndTime" msgpack:"et"`
	BattleResult     string    `db:"BattleResult" msgpack:"br"`
	PlayerName       string    `db:"PlayerName" msgpack:"pn"`
	PlayerPublicID   string    `db:"PlayerPublicID" msgpack:"pi"`
	OpponentName     string    `db:"OpponentName" msgpack:"on"`
	OpponentPublicID string    `db:"OpponentPublicID" msgpack:"oi"`
}

type BattleStats struct {
	NumberWins   int `db:"Wins" msgpack:"w"`
	NumberLosses int `db:"Losses" msgpack:"l"`
	NumberDraws  int `db:"Draws" msgpack:"d"`
}

type BattleData struct {
	RankedStats     *BattleStats  `msgpack:"rs"`
	StandardStats   *BattleStats  `msgpack:"ss"`
	RankedBattles   []BattleEntry `msgpack:"rb"`
	StandardBattles []BattleEntry `msgpack:"sb"`
}

type ShopBackgroundItem struct {
	ID          int    `db:"id_Background"`
	Name        string `db:"Name"`
	Description string `db:"Description"`
	Cost        int    `db:"Cost"`
	AssetPath   string `db:"AssetPath"`
}

type ShopCharacterItem struct {
	ID          int    `db:"id_Character"`
	Name        string `db:"Name"`
	Description string `db:"Description"`
	Health      int    `db:"Health"`
	Damage      int    `db:"Damage"`
	Cost        int    `db:"Cost"`
	AssetPath   string `db:"AssetPath"`
}

type ShopData struct {
	PurchasedBackgrounds []ShopBackgroundItem `msgpack:"pb"`
	AvailableBackgrounds []ShopBackgroundItem `msgpack:"ab"`
	PurchasedCharacters  []ShopCharacterItem  `msgpack:"pc"`
	AvailableCharacters  []ShopCharacterItem  `msgpack:"ac"`
}

type UserDB struct {
	IdUser       int    `db:"id_User"`
	Login        string `db:"Login"`
	PasswordHash string `db:"PasswordHash"`
	IsActive     bool   `db:"isActive"`
}

type CharacterDB struct {
	Name        string                        `db:"Name" msgpack:"n"`
	Description string                        `db:"Description" msgpack:"d"`
	Health      int                           `db:"Health" msgpack:"h"`
	Damage      int                           `db:"Damage" msgpack:"dm"`
	Cost        int                           `db:"Cost" msgpack:"c"`
	HCharacter  int                           `msgpack:"hc"` // Высота персонажа без оружия
	XStart      int                           `msgpack:"xs"` //Координаты верхнего левого угла кадра
	YStart      int                           `msgpack:"ys"` //Координаты верхнего левого угла кадра
	Assets      map[string]*AssetsCharacterDB `msgpack:"as"`
}

type AssetsCharacterDB struct {
	AnimationType string  `db:"AnimationType" msgpack:"at"`
	FrameCount    int     `db:"FrameCount" msgpack:"fc"`
	BaseHeight    int     `db:"BaseHeight" msgpack:"bh"`
	BaseWidth     int     `db:"BaseWidth" msgpack:"bw"`
	FrameRate     float32 `db:"FrameRate" msgpack:"fr"`
	AssetPath     string  `db:"AssetPath" msgpack:"ap"`
}

func getCharacterData(idActiveCharacter int) *CharacterDB {
	var ch CharacterDB
	if _, exists := activeCharacters[idActiveCharacter]; !exists {
		err := db.Get(&ch, queryGetCharacter, sql.Named("Id_Character", idActiveCharacter))
		if err != nil {
			panic(err.Error())
		}
		var arrAsCh []AssetsCharacterDB
		err = db.Select(&arrAsCh, queryGetAssetsCharacter, sql.Named("Id_Character", idActiveCharacter))
		if err != nil {
			panic(err.Error())
		}
		mapAsCh := make(map[string]*AssetsCharacterDB)
		for _, as := range arrAsCh {
			if as.AnimationType != "Preview" { // Только для магазина
				mapAsCh[as.AnimationType] = &as
			}

		}
		ch.Assets = mapAsCh

		addCharacterBitMask(idActiveCharacter, ch)
	} else {
		ch.Name = activeCharacters[idActiveCharacter].Name
		ch.Description = activeCharacters[idActiveCharacter].Description
		ch.Health = activeCharacters[idActiveCharacter].Health
		ch.Damage = activeCharacters[idActiveCharacter].Damage
		ch.Cost = activeCharacters[idActiveCharacter].Cost
		ch.Assets = activeCharacters[idActiveCharacter].Assets
	}

	ch.HCharacter = activeCharacters[idActiveCharacter].HCharacter
	ch.XStart = -activeCharacters[idActiveCharacter].XBoundary                                 // Верхний левый угл
	ch.YStart = ScreenHeight - (GroundLevel + activeCharacters[idActiveCharacter].FrameHeight) // Верхний левый угл
	return &ch
}

func getUserData(idUser, idActiveCharacter int, client *Client) *UserData {
	var usDt UserData

	err := db.Get(&usDt, queryGetUserData, sql.Named("Id_User", idUser))
	if err != nil {
		panic(err.Error())
	}

	usDt.ActiveCharacter = getCharacterData(idActiveCharacter)

	client.UserID = idUser
	client.PlayerID = usDt.PlayerID
	client.Authorized = true
	client.BattleInfo = make(chan *Battle)
	client.Login = usDt.Login
	client.PublicID = usDt.PublicID
	client.Name = usDt.Name
	client.Level = usDt.Level
	client.Money = usDt.Money
	client.Rank = usDt.Rank
	client.friendID = ""
	client.ActiveCharacter = idActiveCharacter
	client.State = &CharacterState{}
	client.State.Update(idActiveCharacter)

	addClient(client)

	return &usDt
}

func validateRegistrationData(rgDt RegisterData) error {
	const (
		maxLen              = 24
		loginPattern        = `^[a-zA-Z0-9_@.]+$`
		namePasswordPattern = `^[a-zA-Zа-яА-Я0-9!@#№$%` + "`" + `~^&*()\-_+=\[\]{};:'",.<>?/|\\ ]+$`
	)
	if strings.TrimSpace(rgDt.Login) == "" || strings.TrimSpace(rgDt.Name) == "" || strings.TrimSpace(rgDt.Password) == "" {
		log.Printf("Ошибка регистрации: поля не могут быть пустыми")
		return fmt.Errorf("поля не могут быть пустыми")
	}

	if len(rgDt.Login) >= maxLen {
		log.Printf("Ошибка регистрации: длина логина '%s' превышает допустимую", rgDt.Login)
		return fmt.Errorf("превышена длина логина")
	}
	if len(rgDt.Name) >= maxLen {
		log.Printf("Ошибка регистрации: длина имени '%s' превышает допустимую", rgDt.Name)
		return fmt.Errorf("превышена длина имени")
	}
	if len(rgDt.Password) >= maxLen {
		log.Printf("Ошибка регистрации: длина пароля '%s' превышает допустимую", rgDt.Password)
		return fmt.Errorf("превышена длина пароля")
	}

	loginRegexp := regexp.MustCompile(loginPattern)
	if !loginRegexp.MatchString(rgDt.Login) {
		log.Printf("Ошибка регистрации: логин '%s' должен содержать символы: '%s'", rgDt.Login, loginPattern)
		return fmt.Errorf("логин должен содержать символы: 'a-zA-Z0-9_@.'", loginPattern)
	}
	namePasswordRegexp := regexp.MustCompile(namePasswordPattern)
	for _, str := range [...]string{rgDt.Name, rgDt.Password} {
		if !namePasswordRegexp.MatchString(str) {
			log.Printf("Ошибка регистрации: имя или пароль '%s' должны содержать символы:  %s", str, namePasswordPattern)
			return fmt.Errorf("допустимые символы для имени и пароля: %s", `a-zA-Zа-яА-Я0-9 !@#№$%`+"`"+`~^&*()-_+=[]{};:'",.<>?/|\`)
		}
	}

	if rgDt.Password != rgDt.ConfirmPassword {
		log.Println("Ошибка регистрации: пароли не совпадают")
		return fmt.Errorf("пароли не совпадают")
	}
	return nil
}

// Функция для генерации случайного 6-значного публичного кода
func generateRandomCode(chars string, length int) string {
	rand.Seed(time.Now().UnixNano())
	code := make([]byte, length)
	for i := 0; i < length; i++ {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}

func actionRegistration(client *Client, data []byte) (*UserData, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	const codeLength = 6

	var rgDt RegisterData
	err := msgpack.Unmarshal(data, &rgDt)
	if err != nil {
		log.Printf("Ошибка при десериализации данных: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}

	err = validateRegistrationData(rgDt)
	if err != nil {
		return nil, err
	}

	// login уникальный
	var exists bool
	err = db.QueryRow(queryUniqueLogin, sql.Named("login", rgDt.Login)).Scan(&exists)
	if err != nil {
		panic(err.Error())
	}
	if exists {
		log.Println("Ошибка регистрации: логин уже существует")
		return nil, fmt.Errorf("логин уже существует")
	}

	tx, err := db.Beginx()
	if err != nil {
		log.Printf("Ошибка при старте транзакции: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		}
	}()

	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rgDt.Password)))
	var UserID int
	err = tx.QueryRow(queryInsertUser, sql.Named("Login", rgDt.Login), sql.Named("PasswordHash", passwordHash), sql.Named("isActive", true)).Scan(&UserID)
	if err != nil {
		log.Printf("Ошибка при добавление нового пользователя: %v", err)
		return nil, fmt.Errorf("не удалось создать пользователя")
	}
	var ActiveBackgroundID, ActiveCharacterID int
	err = tx.QueryRow(queryGetDefaultBackground).Scan(&ActiveBackgroundID)
	if err != nil {
		log.Printf("Ошибка при получении фона: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}
	err = tx.QueryRow(queryGetDefaultCharacter).Scan(&ActiveCharacterID)
	if err != nil {
		log.Printf("Ошибка при получении персонажа: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}

	for attempts := 0; attempts < 100; attempts++ {
		publicCode := generateRandomCode(chars, codeLength)
		_, err = tx.Exec(queryInsertPlayer, sql.Named("id_User", UserID), sql.Named("PublicCode", publicCode), sql.Named("Name", rgDt.Name), sql.Named("Level", 0), sql.Named("Money", 100), sql.Named("Rank", 0), sql.Named("id_ActiveCharacter", ActiveCharacterID), sql.Named("id_ActiveBackground", ActiveBackgroundID))
		fmt.Println(publicCode)
		if err == nil {
			break
		} else if err.Error() == "UNIQUE constraint failed: Players.PublicCode" && attempts < 99 {
			continue
		} else {
			log.Printf("Ошибка при создании игрока: %v", err)
			return nil, fmt.Errorf("не удалось создать профиль игрока")
		}
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Ошибка коммита транзакции: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}

	usDt := getUserData(UserID, ActiveCharacterID, client)
	return usDt, nil
}

func actionAuthorization(client *Client, data []byte) (*UserData, error) {
	// Десериализация данных внутри Data в структуру
	var lgDt LoginData
	err := msgpack.Unmarshal(data, &lgDt)
	if err != nil {
		log.Printf("Ошибка при десериализации данных: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}
	var us UserDB
	err = db.Get(&us, queryAuthenticateUser, sql.Named("login", lgDt.Login))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("Ошибка: пользователь с Логином '%s' не найден", lgDt.Login)
			return nil, fmt.Errorf(fmt.Sprintf("пользователь '%s' не найден", lgDt.Login))
		}
		panic(err.Error())
	}
	if !us.IsActive {
		log.Printf("Ошибка: пользователь с Id=%d, Логином '%s' заблокирован", us.IdUser, us.Login)
		return nil, fmt.Errorf(fmt.Sprintf("пользователь '%s' заблокирован", lgDt.Login))
	}
	passwordHash := fmt.Sprintf("%x", sha256.Sum256([]byte(lgDt.Password)))
	if us.PasswordHash != passwordHash {
		log.Printf("Ошибка: неверно введен пароль")
		return nil, fmt.Errorf("неверно введен пароль")
	}
	var publicID string
	err = db.Get(&publicID, queryGetPublicIDPlayer, sql.Named("id_User", us.IdUser))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("Ошибка: игровые данные '%s' не найдены", lgDt.Login)
			return nil, fmt.Errorf(fmt.Sprintf("игровые данные '%s' не найдены", lgDt.Login))
		}
		panic(err.Error())
	}

	if authClient, exists := authorizedClients[publicID]; exists {
		now := time.Now().UnixMilli()
		elapsed := time.Duration(now-atomic.LoadInt64(&authClient.lastPingTime)) * time.Millisecond
		if atomic.LoadInt32(&authClient.waitingForPong) == 1 && elapsed > 3*time.Second {
			log.Printf("Удалено неактивное соединение для пользователя %d", us.IdUser)
			removeClient(authClient.PublicID)
		} else {
			log.Printf("Ошибка: пользователь %d уже авторизован!", us.IdUser)
			return nil, fmt.Errorf("вы уже авторизованы")
		}
	}

	var idActiveCharacter int

	err = db.Get(&idActiveCharacter, queryGetActiveCharacter, sql.Named("id_User", us.IdUser))
	if err != nil {
		log.Printf("Ошибка при получении идентификатора: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}

	usDt := getUserData(us.IdUser, idActiveCharacter, client)

	return usDt, nil
}

func GetFriendsAndRequestsDB(playerID int) (*FriendsData, error) {

	rows, err := db.Queryx(queryGetFriendsData, sql.Named("PlayerID", playerID))
	if err != nil {
		log.Printf("ошибка при выполнении процедуры: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}
	defer rows.Close()

	result := &FriendsData{}

	var friends []FriendEntry
	if err = sqlx.StructScan(rows, &friends); err != nil {
		log.Printf("Ошибка при чтении друзей: %v", err)
		return nil, fmt.Errorf("внутренняя ошибка сервера")
	}
	result.Friends = friends

	if rows.NextResultSet() {
		var incoming []FriendEntry
		if err = sqlx.StructScan(rows, &incoming); err != nil {
			log.Printf("Ошибка при чтении входящих заявок: %v", err)
			return nil, fmt.Errorf("внутренняя ошибка сервера")
		}
		result.Incoming = incoming
	}

	if rows.NextResultSet() {
		var outgoing []FriendEntry
		if err = sqlx.StructScan(rows, &outgoing); err != nil {
			log.Printf("ошибка при чтении исходящих заявок: %v", err)
			return nil, fmt.Errorf("внутренняя ошибка сервера")
		}
		result.Outgoing = outgoing
	}

	return result, nil

}

func addFriendDB(requesterPlayerID int, friendPublicID string) error {
	const (
		successAccept   = 1
		successRequest  = 0
		invalidFriendId = -1
		repeatRequest   = -2
	)

	tx, err := db.Beginx()
	if err != nil {
		log.Printf("Ошибка при старте транзакции: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("Ошибка при коммите транзакции: %v", err)
			}
		}

	}()

	var result int

	err = tx.QueryRow(queryRequestFriendship, sql.Named("RequesterPlayerID", requesterPlayerID), sql.Named("FriendPublicID", friendPublicID)).Scan(&result)
	if err != nil {
		log.Printf("Ошибка при запросе на дружбу: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}

	switch result {
	case successAccept:
		return fmt.Errorf("Заявка в друзья принята")
	case successRequest:
		return fmt.Errorf("Заявка отправлена")
	case invalidFriendId:
		return fmt.Errorf("Неверный код")
	case repeatRequest:
		return fmt.Errorf("Повторная заявка")
	default:
		return fmt.Errorf("Неизвестный статус заявки")
	}

}

func acceptFriendshipDB(playerID int, requesterPublicID string) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Printf("Ошибка при старте транзакции: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("Ошибка при коммите транзакции: %v", err)
			}
		}

	}()

	_, err = tx.Exec(queryAcceptFriendship, sql.Named("PlayerID", playerID), sql.Named("RequesterPublicID", requesterPublicID))
	if err != nil {
		log.Printf("Ошибка при принятии заявки: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}

	return nil
}

func declineFriendshipDB(playerID int, requesterPublicID string) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Printf("Ошибка при старте транзакции: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("Ошибка при коммите транзакции: %v", err)
			}
		}
	}()

	_, err = tx.Exec(queryDeclineFriendship, sql.Named("PlayerID", playerID), sql.Named("RequesterPublicID", requesterPublicID))
	if err != nil {
		log.Printf("Ошибка при отклонении заявки: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}

	return nil
}

func removeFriendDB(playerID int, friendPublicID string) error {
	tx, err := db.Beginx()
	if err != nil {
		log.Printf("Ошибка при старте транзакции: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("Ошибка при коммите транзакции: %v", err)
			}
		}
	}()

	_, err = tx.Exec(queryRemoveFriendship, sql.Named("PlayerID", playerID), sql.Named("FriendPublicID", friendPublicID))
	if err != nil {
		log.Printf("Ошибка при удалении друга: %v", err)
		return fmt.Errorf("внутренняя ошибка сервера")
	}

	return nil
}

func saveBattleResultsDB(battle Battle) error {
	winnerID := sql.NullInt32{Valid: false}
	if battle.Winner != nil {
		winnerID = sql.NullInt32{Int32: int32(battle.Winner.PlayerID), Valid: true}
	}

	_, err := db.Exec(querySaveBattleResults, sql.Named("Player1ID", battle.Player1.PlayerID), sql.Named("Player2ID", battle.Player2.PlayerID), sql.Named("WinnerID", winnerID), sql.Named("StartTime", battle.StartTime), sql.Named("EndTime", battle.EndTime), sql.Named("isRanked", battle.IsRanked))
	if err != nil {
		log.Printf("Ошибка при сохранения результатов боя: %v", err)
	}
	return err
}

func updatePlayerStats(playerID, newLevel, newRank, newMoney int) error {
	_, err := db.Exec(queryUpdatePlayerStats, sql.Named("PlayerID", playerID), sql.Named("newLevel", newLevel), sql.Named("newRank", newRank), sql.Named("newMoney", newMoney))
	if err != nil {
		log.Printf("Ошибка при обновлении уровня, ранга, монет после боя: %v", err)
	}
	return err
}

func GetBattleEntryAndStatsDB(playerID int, isRanked bool) ([]BattleEntry, *BattleStats, error) {
	var battleEntry []BattleEntry
	var battleStats BattleStats
	rows, err := db.Queryx(queryGetPlayerBattleStats, sql.Named("PlayerID", playerID), sql.Named("isRanked", isRanked))
	if err != nil {
		log.Printf("Ошибка при получении из бд данных о сражениях (ранговые %t): %v ", isRanked, err)
		return battleEntry, &battleStats, fmt.Errorf("внутренняя ошибка сервера")
	}
	defer rows.Close()

	err = sqlx.StructScan(rows, &battleEntry)
	if err != nil {
		log.Printf("Ошибка записи в структуру списка сражений (ранговые %t): %v ", isRanked, err)
		return battleEntry, &battleStats, fmt.Errorf("внутренняя ошибка сервера")
	}
	if !(rows.NextResultSet() && rows.Next()) {
		log.Printf("Ошибка при переходе к статистике сражений (ранговые %t): %v ", isRanked, err)
		return battleEntry, &battleStats, fmt.Errorf("внутренняя ошибка сервера")
	}
	err = rows.StructScan(&battleStats)
	if err != nil {
		log.Printf("Ошибка записи статистики сражений (ранговые %t): %v ", isRanked, err)
		return battleEntry, &battleStats, fmt.Errorf("внутренняя ошибка сервера")
	}

	return battleEntry, &battleStats, nil
}

func GetListBattles(playerID int) (*BattleData, error) {
	rankedBattles, rankedStats, err := GetBattleEntryAndStatsDB(playerID, true)
	if err != nil {
		return nil, err
	}
	standardBattles, standardStats, err := GetBattleEntryAndStatsDB(playerID, false)
	if err != nil {
		return nil, err
	}
	return &BattleData{
		RankedStats:     rankedStats,
		RankedBattles:   rankedBattles,
		StandardStats:   standardStats,
		StandardBattles: standardBattles,
	}, nil
}

func GetShopBackgroundsDB(playerID int) ([]ShopBackgroundItem, []ShopBackgroundItem, error) {
	var purchased, available []ShopBackgroundItem

	rows, err := db.Queryx(queryGetShopBackgrounds, sql.Named("playerID", playerID))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	err = sqlx.StructScan(rows, &purchased)
	if err != nil {
		log.Printf("Ошибка записи в структуру списка купленных фонов: %v", err)
		return purchased, available, fmt.Errorf("внутренняя ошибка сервера")
	}
	if !rows.NextResultSet() {
		log.Printf("Ошибка при переходе к списку доступных для покупки фонов: %v ", err)
		return purchased, available, fmt.Errorf("внутренняя ошибка сервера")
	}
	err = sqlx.StructScan(rows, &available)
	if err != nil {
		log.Printf("Ошибка записи в структуру  доступных для покупки фонов: %v", err)
		return purchased, available, fmt.Errorf("внутренняя ошибка сервера")
	}

	return purchased, available, nil
}

func GetShopCharactersDB(playerID int) ([]ShopCharacterItem, []ShopCharacterItem, error) {
	var purchased, available []ShopCharacterItem

	rows, err := db.Queryx(queryGetShopCharacters, sql.Named("PlayerID", playerID))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	err = sqlx.StructScan(rows, &purchased)
	if err != nil {
		log.Printf("Ошибка записи в структуру купленных персонажей: %v", err)
		return purchased, available, fmt.Errorf("внутренняя ошибка сервера")
	}

	if !rows.NextResultSet() {
		log.Printf("Ошибка при переходе к списку доступных для покупки персонажей: %v ", err)
		return purchased, available, fmt.Errorf("внутренняя ошибка сервера")
	}

	err = sqlx.StructScan(rows, &available)
	if err != nil {
		log.Printf("Ошибка записи в структуру  доступных для покупки персонажей: %v", err)
		return purchased, available, fmt.Errorf("внутренняя ошибка сервера")
	}

	return purchased, available, nil
}

func GetShopData(playerID int) (*ShopData, error) {
	purchasedBackgrounds, availableBackgrounds, err := GetShopBackgroundsDB(playerID)
	if err != nil {
		return nil, err
	}
	purchasedCharacters, availableCharacters, err := GetShopCharactersDB(playerID)
	if err != nil {
		return nil, err
	}
	return &ShopData{
		PurchasedBackgrounds: purchasedBackgrounds,
		AvailableBackgrounds: availableBackgrounds,
		PurchasedCharacters:  purchasedCharacters,
		AvailableCharacters:  availableCharacters,
	}, nil
}

func buyBackgroundDB(playerID, backgroundID int) (remainingMoney int, err error) {
	const (
		resultSuccess       = 0
		resultNotFound      = 1
		resultAlreadyBought = 2
		resultNoMoney       = 3
		resultError         = 4
	)
	var resultCode int

	err = db.QueryRow(queryBuyBackground,
		sql.Named("PlayerID", playerID),
		sql.Named("BackgroundID", backgroundID),
		sql.Named("RemainingMoney", sql.Out{Dest: &remainingMoney}),
		sql.Named("ResultCode", sql.Out{Dest: &resultCode}),
	).Err()

	if err != nil {
		log.Printf("Ошибка выполнения запроса BuyBackground: %v", err)
		return 0, fmt.Errorf("внутренняя ошибка сервера")
	}

	switch resultCode {
	case resultSuccess:
		return remainingMoney, nil
	case resultNotFound:
		return 0, fmt.Errorf("фон не найден")
	case resultAlreadyBought:
		return 0, fmt.Errorf("фон уже куплен")
	case resultNoMoney:
		return 0, fmt.Errorf("недостаточно денег для покупки")
	case resultError:
		return 0, fmt.Errorf("внутренняя ошибка при обработке запроса")
	default:
		return 0, fmt.Errorf("неизвестная ошибка при покупке фона")
	}
}

func selectBackgroundDB(playerID, backgroundID int) (assetPath string, err error) {
	const (
		resultSuccess   = 0
		resultNotFound  = 1
		resultNotBought = 2
		resultError     = 3
	)

	var resultCode int

	err = db.QueryRow(querySelectBackground,
		sql.Named("PlayerID", playerID),
		sql.Named("BackgroundID", backgroundID),
		sql.Named("AssetPath", sql.Out{Dest: &assetPath}),
		sql.Named("ResultCode", sql.Out{Dest: &resultCode}),
	).Err()

	if err != nil {
		log.Printf("Ошибка SelectBackground: %v", err)
		return "", fmt.Errorf("внутренняя ошибка сервера")
	}

	switch resultCode {
	case resultSuccess:
		return assetPath, nil
	case resultNotFound:
		return "", fmt.Errorf("фон не найден")
	case resultNotBought:
		return "", fmt.Errorf("фон не куплен")
	case resultError:
		return "", fmt.Errorf("внутренняя ошибка при обработке запроса")
	default:
		return "", fmt.Errorf("неизвестная ошибка при выборе фона")
	}
}

func buyCharacterDB(playerID, characterID int) (remainingMoney int, err error) {
	const (
		resultSuccess       = 0
		resultNotFound      = 1
		resultAlreadyBought = 2
		resultNoMoney       = 3
		resultError         = 4
	)

	var resultCode int

	err = db.QueryRow(queryBuyCharacter,
		sql.Named("PlayerID", playerID),
		sql.Named("CharacterID", characterID),
		sql.Named("RemainingMoney", sql.Out{Dest: &remainingMoney}),
		sql.Named("ResultCode", sql.Out{Dest: &resultCode}),
	).Err()

	if err != nil {
		log.Printf("Ошибка выполнения запроса BuyCharacter: %v", err)
		return 0, fmt.Errorf("внутренняя ошибка сервера")
	}

	switch resultCode {
	case resultSuccess:
		return remainingMoney, nil
	case resultNotFound:
		return 0, fmt.Errorf("персонаж не найден")
	case resultAlreadyBought:
		return 0, fmt.Errorf("персонаж уже куплен")
	case resultNoMoney:
		return 0, fmt.Errorf("недостаточно денег для покупки")
	case resultError:
		return 0, fmt.Errorf("внутренняя ошибка при обработке запроса")
	default:
		return 0, fmt.Errorf("неизвестная ошибка при покупке персонажа")
	}
}

func selectCharacterDB(playerID, characterID int) (ch CharacterDB, err error) {
	const (
		resultSuccess      = 0
		resultNotFound     = 1
		resultNotPurchased = 2
		resultError        = 3
	)

	var resultCode int

	err = db.QueryRow(querySelectCharacter,
		sql.Named("PlayerID", playerID),
		sql.Named("CharacterID", characterID),
		sql.Named("ResultCode", sql.Out{Dest: &resultCode}),
	).Err()

	if err != nil {
		log.Printf("Ошибка SelectCharacter: %v", err)
		return ch, fmt.Errorf("внутренняя ошибка сервера")
	}

	switch resultCode {
	case resultSuccess:
		chPtr := getCharacterData(characterID)
		if chPtr == nil {
			return ch, fmt.Errorf("ошибка при загрузке данных персонажа")
		}
		return *chPtr, nil
	case resultNotFound:
		return ch, fmt.Errorf("персонаж не найден")
	case resultNotPurchased:
		return ch, fmt.Errorf("персонаж не куплен")
	case resultError:
		return ch, fmt.Errorf("внутренняя ошибка при обработке запроса")
	default:
		return ch, fmt.Errorf("неизвестная ошибка при выборе персонажа")
	}
}
