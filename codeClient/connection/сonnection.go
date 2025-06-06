package connection

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const serverAddr = "...:7777"

var (
	connMutex      sync.Mutex
	ServerLag      int64 // Переменная для задержки в миллисекундах
	lastPingTime   int64 // Время отправки последнего Ping
	waitingForPong int32 = 0
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

// Для получения данных о регистрации и авторизации
type AuthRegResult struct {
	Conn net.Conn
	Data *UserData
	Err  error
}

// Содержит данные авторизованного пользователя
type UserData struct {
	Login                string        `msgpack:"l"`
	PublicID             string        `msgpack:"id"`
	Name                 string        `msgpack:"n"`
	Level                int           `msgpack:"lv"`
	Money                int           `msgpack:"m"`
	Rank                 int           `msgpack:"r"`
	ActiveBackgroundPath string        `msgpack:"b"`
	ActiveCharacter      CharacterData `msgpack:"c"`
}

// Содержит данные персонажа
type CharacterData struct {
	Name        string                     `msgpack:"n"`
	Description string                     `msgpack:"d"`
	Health      int                        `msgpack:"h"`
	Damage      int                        `msgpack:"dm"`
	Cost        int                        `msgpack:"c"`
	HCharacter  int                        `msgpack:"hc"` // Высота персонажа без оружия
	XStart      int                        `msgpack:"xs"`
	YStart      int                        `msgpack:"ys"`
	Assets      map[string]AssetsCharacter `msgpack:"as"`
}

// Содержит данные изображений персонажа
type AssetsCharacter struct {
	AnimationType string       `msgpack:"at"`
	FrameCount    int          `msgpack:"fc"`
	BaseHeight    int          `msgpack:"bh"`
	BaseWidth     int          `msgpack:"bw"`
	FrameRate     float32      `msgpack:"fr"`
	AssetPath     string       `msgpack:"ap"`
	Texture       rl.Texture2D `msgpack:"t"`
}

// Содержит данные об изменении здоровья
type HealthUpdate struct {
	Who    MessageType `msgpack:"w"`
	Health int         `msgpack:"h"`
}

// Универсальное сообщения для связи с сервером
type Message struct {
	Type MessageType `msgpack:"t"`
	Data []byte      `msgpack:"d"`
}

// Функция для превращения сообщения в массив байт
func CreateMessage(mesType MessageType, data interface{}) ([]byte, error) {
	dataBytes, err := msgpack.Marshal(data)
	if err != nil {
		return nil, err
	}
	message := Message{Type: mesType, Data: dataBytes}
	return msgpack.Marshal(message)
}

// Функция отправки сообщений на сервер
func SendMessage(conn net.Conn, data []byte) error {
	connMutex.Lock()
	defer connMutex.Unlock()
	_, err := conn.Write(data)
	return err
}

// Функция полученяи сообщений от сервера
func GetMessage(conn net.Conn, data interface{}) (Message, error) {
	decoder := msgpack.NewDecoder(conn)

	var msg Message
	err := decoder.Decode(&msg)
	if err != nil {
		log.Println("GetMessage: ошибка при десериализации сообщения:", err)
		return Message{}, err
	}

	if data != nil && msg.Data != nil {
		err = msgpack.Unmarshal(msg.Data, data)
		if err != nil {
			log.Println("GetMessage: ошибка при десериализации msg.Data:", err)
			return Message{}, err
		}
	}

	if msg.Type == MsgPing {
		pong := Message{Type: MsgPong}
		raw, _ := msgpack.Marshal(pong)
		SendMessage(conn, raw)
	}

	if msg.Type == MsgPong {
		now := time.Now().UnixMilli()
		sentTime := atomic.LoadInt64(&lastPingTime)
		RTT := now - sentTime
		if RTT > 0 {
			atomic.StoreInt64(&ServerLag, RTT/2)
			log.Printf("Обновленная задержка: %d ms\n", RTT/2)
		}
		atomic.StoreInt32(&waitingForPong, 0)
	}

	return msg, nil
}

// Функция подключения к серверу
func ConnectToServer() (net.Conn, error) {
	conn, err := kcp.Dial(serverAddr)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к KCP серверу: %v", err)
	}

	// Приводим тип и настраиваем параметры KCP
	if session, ok := conn.(*kcp.UDPSession); ok {
		session.SetNoDelay(1, 10, 2, 1) // Fast mode
		session.SetWindowSize(128, 128) // Увеличенные окна
		session.SetACKNoDelay(true)     // Без отложенных ACK
	} else {
		return nil, fmt.Errorf("соединение не является *kcp.UDPSession")
	}

	go startPingRoutine(conn)

	return conn, nil
}

// Функция начала ping pong
func startPingRoutine(conn net.Conn) {
	ticker := time.NewTicker(1123 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		if atomic.LoadInt32(&waitingForPong) == 1 {
			continue
		}

		now := time.Now().UnixMilli()
		atomic.StoreInt64(&lastPingTime, now)
		atomic.StoreInt32(&waitingForPong, 1)

		ping := Message{Type: MsgPing}

		data, err := msgpack.Marshal(ping)
		if err != nil {
			log.Println("Ошибка сериализации Ping:", err)
			atomic.StoreInt32(&waitingForPong, 0)
			continue
		}
		err = SendMessage(conn, data)

		if err != nil {
			log.Printf("Ошибка отправки Ping: %v", err)
			return
		}
	}
}

// Функция завершеняи соединения
func CloseConnection(conn net.Conn) {
	exitMsg, _ := CreateMessage(MsgExit, nil)
	SendMessage(conn, exitMsg)

	connMutex.Lock()
	conn.Close()
	connMutex.Unlock()
}

// Функция для устанволеняи соединения при регистрации и авторизации
func AuthorizationRegistrationToServer(data []byte) (net.Conn, *UserData, error) {

	conn, err := ConnectToServer()
	if err != nil {
		log.Println("Не удалось подключиться к серверу: ", err)
		return nil, nil, fmt.Errorf("не удалось подключиться к серверу, попробуйте позже")
	}

	// Установка таймаута для чтения ответа
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Отправка данных
	err = SendMessage(conn, data) //err = conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		CloseConnection(conn)
		fmt.Println("Не удалось отправить данные на сервер: ", err)
		return nil, nil, fmt.Errorf("не удалось отправить данные на сервер")
	}

	for {
		// Установка таймаута для чтения ответа
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		msg, err := GetMessage(conn, nil)
		if err != nil {
			return nil, nil, err
		}
		switch msg.Type {
		case MsgPong:
		case MsgPing:
		case MsgError:
			var erDt string
			err = msgpack.Unmarshal(msg.Data, &erDt)
			CloseConnection(conn)
			log.Println("Ошибка: ", erDt)
			return nil, nil, fmt.Errorf(erDt)
		case MsgSuccess:
			usDt := new(UserData)
			// Десериализация данных внутри Data в структуру
			err = msgpack.Unmarshal(msg.Data, usDt)
			if err != nil {
				fmt.Println("Ошибка при десериализации данных:", err)
				return nil, nil, fmt.Errorf("при десериализации данных от сервера")
			}
			conn.SetReadDeadline(time.Time{})
			log.Printf("Успешный вход пользователя: %s", usDt.Name)
			return conn, usDt, nil
		default:
			CloseConnection(conn)
			return nil, nil, fmt.Errorf("неизвестный ответ от сервера")
		}
	}
}
