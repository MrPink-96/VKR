package main

import (
	"context"
	"errors"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/jmoiron/sqlx"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/xtaci/kcp-go/v5"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	timeoutDuration = 10 * time.Second // Время отключёния клиента при не активности
)

type MessageType uint8

const (
	MsgNone MessageType = iota
	MsgPing
	MsgPong
	MsgExit
	MsgSuccess
	MsgError
	MsgAuthorization   // Authorization
	MsgRegistration    // Registration
	MsgActionCharacter // ActionCharacter
	MsgActionOpponent
	MsgHealthUpdate
	MsgFriendsData
	MsgAddFriend
	MsgAcceptFriendship
	MsgDeclineFriendship
	MsgChallengeToFight
	MsgAcceptChallengeToFight
	MsgRefuseChallengeToFight
	MsgRemoveFriend
	MsgBattle       // Battle
	MsgBattleRanked // BattleRanked
	MsgExitBattle
	MsgWaitingBattle
	MsgStartBattleInfo
	MsgReadyBattle
	MsgEndBattle
	MsgListBattles
	MsgShopData
	MsgShopAction
	MsgMoneyUpdate
	MsgSelectBackground
	MsgSelectCharacter
	MsgPurchaseReceipt
)

var (
	db                  *sqlx.DB
	authorizedClients   = make(map[string]*Client) // Список подключённых клиентов
	clientsMutex        sync.Mutex                 // Ограничиваем доступ к списку подключённых клиентов
	activeCharacters    = make(map[int]*Character)
	listCharactersMutex sync.Mutex
)

// Структура клиента
type Client struct {
	UserID   int    // ID из таблицы Users
	PlayerID int    // ID из таблицы Players
	PublicID string // Публичный ID для приглашений

	Login           string
	Name            string
	Level           int
	Money           int
	Rank            int
	ActiveCharacter int
	State           *CharacterState
	friendID        string // ID друга для дружеского сражения

	Conn      net.Conn
	connMutex sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc

	lastPingTime   int64
	Authorized     bool
	Ping           int64
	waitingForPong int32

	ReceivedMess chan *Message
	BattleInfo   chan *Battle
}

type Character struct {
	Health            int
	Damage            int
	FrameWidth        int
	FrameHeight       int
	XCharacter        int // Левая верхняя точка
	YCharacter        int // Левая верхняя точка
	WCharacter        int // Ширина персонажа без оружия
	HCharacter        int // Высота персонажа без оружия
	XBoundary         int // Наибольший отступ с двух сторон фрейма, чтобы персонаж без оружия поместился
	BitMask           map[string][]uint64
	BitMaskWithWeapon map[string][]uint64
	TimeAnimation     map[string]time.Duration
	Assets            map[string]*AssetsCharacterDB
	Name              string
	Description       string
	Cost              int
}

// Структура для обмена сообщениями
type Message struct {
	Type MessageType `msgpack:"t"`
	Data []byte      `msgpack:"d"`
}

// Структура для авторизации
type LoginData struct {
	Login    string `msgpack:"l"`
	Password string `msgpack:"p"`
}

// Структура для регистрации
type RegisterData struct {
	Login           string `msgpack:"l"`
	Name            string `msgpack:"n"`
	Password        string `msgpack:"p"`
	ConfirmPassword string `msgpack:"cp"`
}

// Создание битовой маски для боя
func createBitMask(path string, countFrame, FrameWidth, FrameHeight int, ch *Character, sizeCharacter bool) []uint64 {
	// Путь к текущей директории
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	currentDir = filepath.Dir(currentDir) // УБРАТЬ ЕСЛИ EXE ТАМ ЖЕ ГДЕ И ПАПКА RESOURCES
	assetCharacter := rl.LoadImage(currentDir + path)
	if assetCharacter == nil || assetCharacter.Width == 0 || assetCharacter.Height == 0 {
		panic("Ошибка загрузки изображения: файл не найден или повреждён")
	}
	defer rl.UnloadImage(assetCharacter)

	rl.ImageResizeNN(assetCharacter, int32(FrameWidth*countFrame), int32(FrameHeight)) // Приводим к базовым размерам

	// Создаём битовую маску общую для всех кадров
	mask := make([]uint64, FrameHeight*(FrameWidth/64+1))

	// Проходим по всем кадрам
	for frame := 0; frame < countFrame; frame++ {
		frameXOffset := frame * FrameWidth

		for y := 0; y < FrameHeight; y++ {
			for x := 0; x < FrameWidth; x++ {
				color := rl.GetImageColor(*assetCharacter, int32(x+frameXOffset), int32(y))

				// Если хотя бы в одном кадре пиксель непрозрачный, он добавляется в маску
				if color.A > 0 {
					index := y*(FrameWidth/64+1) + x/64
					mask[index] |= 1 << (x % 64)
					if sizeCharacter {
						if x < ch.XCharacter {
							ch.XCharacter = x
						}
						if y < ch.YCharacter {
							ch.YCharacter = y
						}
						if x > ch.WCharacter {
							ch.WCharacter = x
						}
						if y > ch.HCharacter {
							ch.HCharacter = y
						}
					}
				}
			}
		}
	}

	return mask
}

// Добавить все битовые маски персонажу
func addCharacterBitMask(id int, chDB CharacterDB) {
	listCharactersMutex.Lock()
	defer listCharactersMutex.Unlock()

	ch := Character{Health: chDB.Health, Damage: chDB.Damage, XCharacter: math.MaxInt, YCharacter: math.MaxInt, WCharacter: 0, HCharacter: 0, Assets: chDB.Assets, Name: chDB.Name, Description: chDB.Description, Cost: chDB.Cost}

	//Считаем, что ширина и высота кадра для всех анимаций одного персонажа одинаковая
	ch.FrameWidth = chDB.Assets["Attack"].BaseWidth
	ch.FrameHeight = chDB.Assets["Attack"].BaseHeight

	BitMask := make(map[string][]uint64)
	BitMaskWithWeapon := make(map[string][]uint64)
	TimeAnimation := make(map[string]time.Duration)
	for _, asset := range chDB.Assets {
		if asset.AnimationType == "Medallion" {
			continue
		}
		lastSlash := strings.LastIndex(asset.AssetPath, "\\")
		if asset.AnimationType == "Attack" || asset.AnimationType == "HeavyAttack" {
			assPthWthWep := asset.AssetPath[:lastSlash] + "\\Weapon" + asset.AssetPath[lastSlash:]
			BitMaskWithWeapon[asset.AnimationType] = createBitMask(assPthWthWep, asset.FrameCount, asset.BaseWidth, asset.BaseHeight, &ch, false)
		}
		frameDuration := time.Second / time.Duration(asset.FrameRate)                        // Время одного кадра
		TimeAnimation[asset.AnimationType] = frameDuration * time.Duration(asset.FrameCount) // Общее время анимации
		assPth := asset.AssetPath[:lastSlash] + "\\Character" + asset.AssetPath[lastSlash:]
		BitMask[asset.AnimationType] = createBitMask(assPth, asset.FrameCount, asset.BaseWidth, asset.BaseHeight, &ch, true)
	}
	ch.BitMask = BitMask
	ch.BitMaskWithWeapon = BitMaskWithWeapon
	ch.TimeAnimation = TimeAnimation

	ch.WCharacter = ch.WCharacter - ch.XCharacter + 1 // Нужно включить максимальный индекс х пикселя
	ch.HCharacter = ch.HCharacter - ch.YCharacter + 1 // Нужно включить максимальный индекс у пикселя
	if ch.XCharacter < ch.FrameWidth-(ch.XCharacter+ch.WCharacter) {
		ch.XBoundary = ch.XCharacter
	} else {
		ch.XBoundary = ch.FrameWidth - (ch.XCharacter + ch.WCharacter)
	}

	activeCharacters[id] = &ch
}

// Добавление клиента
func addClient(client *Client) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	authorizedClients[client.PublicID] = client
}

// Удаление клиента
func removeClient(publicID string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	if _, ok := authorizedClients[publicID]; ok {
		delete(authorizedClients, publicID)
	}
}

// Создание байтового сообщения
func createMessage(mesType MessageType, data interface{}) ([]byte, error) {
	dataBytes, err := msgpack.Marshal(data)
	if err != nil {
		return nil, err
	}

	message := Message{Type: mesType, Data: dataBytes}
	return msgpack.Marshal(message)
}

// Отправка сообщения
func sendMessage(client *Client, data []byte) error {
	client.connMutex.Lock()
	defer client.connMutex.Unlock()

	if client.Conn == nil {
		log.Printf("Попытка отправки сообщения отключённому клиенту %d", client.UserID)
		return errors.New("соединение закрыто")
	}

	_, err := client.Conn.Write(data)
	return err
}

// Создание и отправка байтового сообщения
func createAndSendMessage(client *Client, mesType MessageType, data interface{}) error {
	resp, err := createMessage(mesType, data)
	if err != nil {
		return err
	}
	return sendMessage(client, resp)
}

// Получение сообщений клиентом
func receiveClientMessages(client *Client) {
	defer func() {
		client.cancel()
		if client.Authorized {
			client.Authorized = false
			removeClient(client.PublicID)
		}
		close(client.ReceivedMess)
		client.ReceivedMess = nil
		client.Conn.Close()
		client.Conn = nil
		log.Printf("Клиент %d отключён", client.UserID)
	}()
	// Пинг каждые N секунд
	go func() {
		const N = 3
		ticker := time.NewTicker(N * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-client.ctx.Done():
				log.Printf("Пинг понг клиента %d завершил работу по контексту", client.UserID)
				return

			case <-ticker.C:
				if atomic.LoadInt32(&client.waitingForPong) == 1 {
					continue
				}

				now := time.Now().UnixMilli()
				atomic.StoreInt64(&client.lastPingTime, now)
				atomic.StoreInt32(&client.waitingForPong, 1)

				pingMsg := Message{Type: MsgPing}
				data, err := msgpack.Marshal(pingMsg)
				if err != nil {
					log.Printf("Ошибка сериализации Ping: %v", err)
					atomic.StoreInt32(&client.waitingForPong, 0)
					continue
				}

				client.connMutex.Lock()
				if client.Conn == nil {
					client.connMutex.Unlock()
					return
				}
				_, err = client.Conn.Write(data)
				client.connMutex.Unlock()
				if err != nil {
					log.Printf("Ошибка отправки Ping клиенту %d: %v", client.UserID, err)
					return
				}
			}
		}
	}()

	var netErr net.Error
	decoder := msgpack.NewDecoder(client.Conn)

	for {
		select {
		case <-client.ctx.Done():
			log.Printf("Получатель сообщений клиента %d завершил работу по контексту", client.UserID)
			return

		default:
			client.Conn.SetReadDeadline(time.Now().Add(timeoutDuration))

			var msg Message
			err := decoder.Decode(&msg)
			if err != nil {
				if errors.As(err, &netErr) && netErr.Timeout() {
					log.Printf("Клиент %d отключён по таймауту", client.UserID)
					return
				}
				createAndSendMessage(client, MsgError, "ошибка при десериализации сообщения!")
				log.Printf("Ошибка при десериализации сообщения от клиента %d: %v", client.UserID, err)
				return
			}

			switch msg.Type {
			case MsgExit:
				return
			case MsgPing:
				handlePing(client)
			case MsgPong:
				handlePong(client)
			default:
				client.ReceivedMess <- &msg
			}
		}
	}
}

// Обработчик сообщенйи клиента
func handleClientMessages(client *Client) {
	client.ReceivedMess = make(chan *Message)
	client.ctx, client.cancel = context.WithCancel(context.Background())

	go receiveClientMessages(client)
	for {
		select {

		case <-client.ctx.Done():
			log.Printf("Клиент %d завершил работу по контексту", client.UserID)
			return

		case msg, ok := <-client.ReceivedMess:
			// Если канал закрыт, то завершить обработку
			if !ok {
				log.Printf("Канал сообщений для клиента %d закрыт, завершаем обработку", client.UserID)
				return
			}

			// Поиск функции обработки
			if handler, ok := handlers[msg.Type]; ok {
				handler(client, msg.Data)
			} else {
				createAndSendMessage(client, MsgError, fmt.Sprintf("неизвестный тип сообщения: %d", msg.Type))
			}
		}
	}
}

// Обработчик KCP
func kcpHandler() {
	listener, err := kcp.Listen(":7777") // Порт для KCP
	if err != nil {
		log.Fatalf("Ошибка запуска KCP сервера: %v", err)
	}
	log.Println("KCP сервер запущен на :7777")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Ошибка Accept KCP: %v", err)
			continue
		}

		// Приводим тип и настраиваем параметры KCP
		if session, ok := conn.(*kcp.UDPSession); ok {
			session.SetNoDelay(1, 10, 2, 1) // Fast mode
			session.SetWindowSize(128, 128) // Увеличенные окна
			session.SetACKNoDelay(true)     // Без отложенных ACK
		} else {
			log.Printf("соединение не является *kcp.UDPSession")
			continue
		}

		client := &Client{
			UserID:     -1,
			PlayerID:   -1,
			Conn:       conn,
			Authorized: false,
		}
		log.Printf("Новое подключение от %s", conn.RemoteAddr().String())
		go handleClientMessages(client)
	}
}

func main() {
	var err error
	db, err = sqlx.Open("sqlserver", "server=localhost;database=GAME_FQW;trusted_connection=yes")
	if err != nil {
		log.Fatalf("Ошибка открытия соединения базы данных: %v", err)
	}
	// Проверка соединения
	if err = db.Ping(); err != nil {
		panic("Не удалось подключиться к базе данных: " + err.Error())
	}
	defer db.Close()

	// Инициализация очередей для матчей
	matchmakingQueue = &MatchmakingQueue{}

	// Запускаем горутину для поиска матчей
	go matchmakingQueue.RunSearch()
	defer matchmakingQueue.StopSearch()

	go kcpHandler()

	// Ожидание любых системных сигналов (SIGINT, SIGTERM и другие)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
	<-sig

	log.Println("Сервер завершил работу.")
}
