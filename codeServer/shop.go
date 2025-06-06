package main

import "fmt"

type ShopActionType int

type ProductType int

const (
	actionBuy ShopActionType = iota
	actionSelect
)

const (
	productBackground ProductType = iota
	productCharacter
)

// действие в магазине
type ShopAction struct {
	Action      ShopActionType `msgpack:"a"`
	ProductType ProductType    `msgpack:"t"`
	ProductID   int            `msgpack:"p"`
}

// Подверждение покупки
type PurchaseReceipt struct {
	ProductType    ProductType `msgpack:"t"`
	ProductID      int         `msgpack:"p"`
	RemainingMoney int         `msgpack:"r"`
}

// Определяет тип события в магазине
func (a *ShopAction) Apply(client *Client) {
	switch a.Action {
	case actionBuy:
		a.actionBuy(client)
	case actionSelect:
		a.actionSelect(client)
	default:
		createAndSendMessage(client, MsgError, fmt.Sprintf("неизвестный тип действия магазина: %d", a.Action))
		return
	}
}

// Обрабатывает покупку
func (a *ShopAction) actionBuy(client *Client) {
	switch a.ProductType {
	case productBackground:
		remainingMoney, err := buyBackgroundDB(client.PlayerID, a.ProductID)
		if err != nil {
			createAndSendMessage(client, MsgError, err.Error())
			return
		}
		client.Money = remainingMoney
		a.sendPurchaseReceipt(client, remainingMoney)

		a.selectBackground(client)

	case productCharacter:
		remainingMoney, err := buyCharacterDB(client.PlayerID, a.ProductID)
		if err != nil {
			createAndSendMessage(client, MsgError, err.Error())
			return
		}
		client.Money = remainingMoney
		a.sendPurchaseReceipt(client, remainingMoney)

		a.selectCharacter(client)
	default:
		createAndSendMessage(client, MsgError, fmt.Sprintf("неизвестный тип продукта магазина: %d", a.ProductType))
	}
}

// Обрабатывает выбор и установку
func (a *ShopAction) actionSelect(client *Client) {
	switch a.ProductType {
	case productBackground:
		a.selectBackground(client)
	case productCharacter:
		a.selectCharacter(client)
	default:
		createAndSendMessage(client, MsgError, fmt.Sprintf("неизвестный тип продукта магазина: %d", a.ProductType))
	}
}

// Обрабатывает выбор активного фона
func (a *ShopAction) selectBackground(client *Client) {
	assetPath, err := selectBackgroundDB(client.PlayerID, a.ProductID)
	if err != nil {
		createAndSendMessage(client, MsgError, err.Error())
		return
	}
	createAndSendMessage(client, MsgSelectBackground, assetPath)
}

// Обрабатывает выбор активного персонажа
func (a *ShopAction) selectCharacter(client *Client) {
	character, err := selectCharacterDB(client.PlayerID, a.ProductID)
	if err != nil {
		createAndSendMessage(client, MsgError, err.Error())
		return
	}
	client.ActiveCharacter = a.ProductID
	client.State.Update(client.ActiveCharacter)

	createAndSendMessage(client, MsgSelectCharacter, character)
}

// Отправляет подтверждение покупки
func (a *ShopAction) sendPurchaseReceipt(client *Client, remainingMoney int) {
	purchaseReceipt := PurchaseReceipt{
		ProductType:    a.ProductType,
		ProductID:      a.ProductID,
		RemainingMoney: remainingMoney,
	}
	createAndSendMessage(client, MsgPurchaseReceipt, purchaseReceipt)
}
