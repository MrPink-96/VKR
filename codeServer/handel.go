package main

import (
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"log"
	"sync/atomic"
	"time"
)

// Тип функции-обработчика
type messageHandler func(*Client, []byte)

// map обработчиков
var handlers = map[MessageType]messageHandler{
	//MsgPing:                   handlePing,
	//MsgPong:                   handlePong,
	MsgAuthorization:          handleAuthorization,
	MsgRegistration:           handleRegistration,
	MsgActionCharacter:        requireAuth(requiresNoBattle(handleActionCharacter)),
	MsgBattle:                 requireAuth(requiresNoBattle(handleBattle)),
	MsgBattleRanked:           requireAuth(requiresNoBattle(handleBattleRanked)),
	MsgFriendsData:            requireAuth(requiresNoBattle(handelFriendsData)),
	MsgAddFriend:              requireAuth(requiresNoBattle(handelAddFriend)),
	MsgAcceptFriendship:       requireAuth(requiresNoBattle(handelAcceptFriendship)),
	MsgDeclineFriendship:      requireAuth(requiresNoBattle(handelDeclineFriendship)),
	MsgChallengeToFight:       requireAuth(requiresNoBattle(handelChallengeToFight)),
	MsgAcceptChallengeToFight: requireAuth(requiresNoBattle(handelAcceptChallengeToFight)),
	MsgRefuseChallengeToFight: requireAuth(requiresNoBattle(handelRefuseChallengeToFight)),
	MsgRemoveFriend:           requireAuth(requiresNoBattle(handelRemoveFriend)),
	MsgListBattles:            requireAuth(requiresNoBattle(handelListBattles)),
	MsgShopData:               requireAuth(requiresNoBattle(handelShopData)),
	MsgShopAction:             requireAuth(requiresNoBattle(handleShopAction)),
}

// Для команд, требующих авторизации и отсутствия сражения
func requireAuth(handler messageHandler) messageHandler {
	return func(client *Client, data []byte) {
		if !client.Authorized {
			createAndSendMessage(client, MsgError, "Вы не авторизованы!")
			return
		}
		handler(client, data)
	}
}

// Для команд, требующих отсутствия сражения
func requiresNoBattle(handler messageHandler) messageHandler {
	return func(client *Client, data []byte) {
		if client.State.inBattle {
			createAndSendMessage(client, MsgError, "Вы уже в поиске боя!")
			return
		}
		handler(client, data)
	}
}

func handlePing(client *Client) {
	pongMsg, _ := createMessage(MsgPong, nil)
	sendMessage(client, pongMsg)
}

func handlePong(client *Client) {
	now := time.Now().UnixMilli()
	sentTime := atomic.LoadInt64(&client.lastPingTime)
	RTT := now - sentTime
	log.Printf("Обновленная задержка клиента %d: %d RTT\n", client.UserID, RTT)
	client.Ping = RTT / 2
	if client.Ping > 0 {
		log.Printf("Обновленная задержка клиента %d: %d ms\n", client.UserID, client.Ping)
	}
	atomic.StoreInt32(&client.waitingForPong, 0)
}

// Авторизация
func handleAuthorization(client *Client, data []byte) {

	var resp []byte
	usDt, err := actionAuthorization(client, data)
	if err != nil {
		resp, _ = createMessage(MsgError, err.Error())
		defer client.cancel()
	} else {
		resp, _ = createMessage(MsgSuccess, usDt)
	}
	log.Println(client.Name, " авторизован")
	sendMessage(client, resp)
}

// Регистрация
func handleRegistration(client *Client, data []byte) {
	var resp []byte
	if client.Authorized {
		resp, _ = createMessage(MsgError, "Вы уже авторизованы!")
	} else {
		usDt, err := actionRegistration(client, data)
		if err != nil {
			resp, _ = createMessage(MsgError, err.Error())
			defer client.cancel()
		} else {
			resp, _ = createMessage(MsgSuccess, usDt)
		}
	}
	log.Println(client.Name, " зарегистрирован")
	sendMessage(client, resp)
}

// Управление персонажем
func handleActionCharacter(client *Client, data []byte) {
	var action Action

	err := msgpack.Unmarshal(data, &action)
	if err != nil {
		log.Printf("Ошибка десериализации в handleActionCharacter: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации Action")
		return
	}
	actionCharacter(action, client, nil)
}

// Поиск обычного боя
func handleBattle(client *Client, data []byte) {
	waitingBattle(client, false, "")
}

// Поиск рейтингового боя
func handleBattleRanked(client *Client, data []byte) {
	waitingBattle(client, true, "")
}

// Получение списка друзей и заявок в друзья.
func handelFriendsData(client *Client, data []byte) {
	var resp []byte
	frDt, err := GetFriendsAndRequestsDB(client.PlayerID)
	if err != nil {
		resp, _ = createMessage(MsgError, err.Error())
	} else {
		resp, _ = createMessage(MsgFriendsData, frDt)
	}
	sendMessage(client, resp)
}

// Обрабатывает запрос на добавление друга
func handelAddFriend(client *Client, data []byte) {
	var friendID string

	err := msgpack.Unmarshal(data, &friendID)
	if err != nil {
		log.Printf("Ошибка десериализации в handelAddFriend: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации friendID")
		return
	}

	result := addFriendDB(client.PlayerID, friendID).Error()
	createAndSendMessage(client, MsgAddFriend, result)
}

// Обрабатывает запрос на подтверждение заявки в друзья
func handelAcceptFriendship(client *Client, data []byte) {
	var friendID string

	err := msgpack.Unmarshal(data, &friendID)
	if err != nil {
		log.Printf("Ошибка десериализации в handelAcceptFriendship: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации friendID")
		return
	}

	err = acceptFriendshipDB(client.PlayerID, friendID)
	if err != nil {
		createAndSendMessage(client, MsgError, err.Error())
		return
	}

}

// Обрабатывает запрос на отклонение заявки в друзья
func handelDeclineFriendship(client *Client, data []byte) {
	var friendID string

	err := msgpack.Unmarshal(data, &friendID)
	if err != nil {
		log.Printf("Ошибка десериализации в handelDeclineFriendship: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации friendID")
		return
	}

	err = declineFriendshipDB(client.PlayerID, friendID)
	if err != nil {
		createAndSendMessage(client, MsgError, err.Error())
		return
	}

}

// Обрабатывает запрос на вызов друга на дуэль
func handelChallengeToFight(client *Client, data []byte) {
	var friendID string

	err := msgpack.Unmarshal(data, &friendID)
	if err != nil {
		log.Printf("Ошибка десериализации в handelRemoveFriend: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации friendID")
		return
	}

	friend, ok := authorizedClients[friendID]
	if !ok {
		log.Printf("Ошибка вызова на дуэль, игрок: %v, не авторизован или ID неверно", friendID)
		createAndSendMessage(client, MsgError, fmt.Sprintf("игрок: %v, не авторизован или ID неверно", friendID))
		return
	}

	if friend.State.inBattle {
		createAndSendMessage(client, MsgError, fmt.Sprintf("игрок: %v уже в бою, попробуйте позже", friendID))
		return
	}

	err = createAndSendMessage(friend, MsgChallengeToFight, FriendEntry{Name: client.Name, PublicID: client.PublicID})
	if err != nil {
		createAndSendMessage(client, MsgError, "не удалось отправить приглашение")
		return
	}
	waitingBattle(client, false, friendID)
}

// Обрабатывает согласие клиента на дуэль
func handelAcceptChallengeToFight(client *Client, data []byte) {
	var friendID string

	err := msgpack.Unmarshal(data, &friendID)
	if err != nil {
		log.Printf("Ошибка десериализации в handelAcceptChallengeToFight: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации friendID")
		return
	}

	friend, ok := authorizedClients[friendID]
	if !ok {
		log.Printf("Ошибка при отправке согласии на дуэль: %v", err)
		createAndSendMessage(client, MsgError, "игрок вышел из игры")
		return
	}

	if !friend.State.inBattle {
		log.Printf("Ошибка при отправке согласии на дуэль, игрок %v больше не ожидает оппонента", friendID)
		createAndSendMessage(client, MsgError, "друг отменил приглашение на бой, попробуйте снова")
		return
	}

	if friend.State.inBattle && friend.friendID != client.PublicID { // friend.State.inBattle && friend.State.friendID != client.PublicID {
		log.Printf("Ошибка при отправке согласии на дуэль, игрок %v уже в бою с другим", friendID)
		createAndSendMessage(client, MsgError, "приглашение на бой больше не активно, игрок уже в бою")
		return
	}

	restoreState(client, MsgWaitingBattle)
	go waitingBattle(client, false, friendID)
	manageBattle(friend, client, false)
}

// Обрабатывает отказ клиента от дуэли
func handelRefuseChallengeToFight(client *Client, data []byte) {
	var friendID string

	err := msgpack.Unmarshal(data, &friendID)
	if err != nil {
		log.Printf("Ошибка десериализации в handelRefuseChallengeToFight: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации friendID")
		return
	}

	friend, ok := authorizedClients[friendID]
	if !ok {
		log.Printf("Ошибка при отправке отказа на дуэль: %v", err)
		createAndSendMessage(client, MsgError, "игрок вышел из игры")
		return
	}

	createAndSendMessage(friend, MsgRefuseChallengeToFight, client.PublicID)
}

// Обрабатывает запрос на удаление друга
func handelRemoveFriend(client *Client, data []byte) {
	var friendID string

	err := msgpack.Unmarshal(data, &friendID)
	if err != nil {
		log.Printf("Ошибка десериализации в handelRemoveFriend: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации friendID")
		return
	}
	err = removeFriendDB(client.PlayerID, friendID)
	if err != nil {
		createAndSendMessage(client, MsgError, err.Error())
		return
	}
}

// Получение списка боёв
func handelListBattles(client *Client, data []byte) {
	listBattles, err := GetListBattles(client.PlayerID)
	if err != nil {
		createAndSendMessage(client, MsgError, err.Error())
		return
	}
	fmt.Println(listBattles)
	createAndSendMessage(client, MsgListBattles, listBattles)
}

// Обрабатывает действия клиента в магазине
func handelShopData(client *Client, data []byte) {
	shopData, err := GetShopData(client.PlayerID)
	if err != nil {
		createAndSendMessage(client, MsgError, err.Error())
		return
	}
	fmt.Println(shopData)

	createAndSendMessage(client, MsgShopData, shopData)
}

// Получение данных магазина
func handleShopAction(client *Client, data []byte) {
	var shopAction ShopAction
	err := msgpack.Unmarshal(data, &shopAction)
	if err != nil {
		log.Printf("Ошибка десериализации в handleShopAction: %v", err)
		createAndSendMessage(client, MsgError, "внутренняя ошибка сервера при десериализации ShopAction")
		return
	}
	shopAction.Apply(client)
}
