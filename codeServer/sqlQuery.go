package main

const (
	// Проверка на уникальность логина
	queryUniqueLogin = "SELECT CASE WHEN EXISTS (SELECT 1 FROM Users WHERE Login = @login) THEN 1 ELSE 0 END;"
	// Добавляет запись о новом пользователе и возвращает его идентификатор
	queryInsertUser = "INSERT INTO Users VALUES (@Login, @PasswordHash, @isActive); SELECT SCOPE_IDENTITY();"
	// Добавляет запись о новом игроке для соответствующего пользователя
	queryInsertPlayer = "INSERT INTO Players VALUES (@id_User,  @PublicCode, @Name, @Level, @Money, @Rank, @id_ActiveCharacter, @id_ActiveBackground)"
	// Получение идентификатора стартового фона
	queryGetDefaultBackground = "SELECT id_Background FROM Backgrounds WHERE Name = 'Рынок'"
	// Получение идентификатора стартового персонажа
	queryGetDefaultCharacter = "SELECT id_Character FROM Characters WHERE Name = 'Девушка рыцарь'"
	// Идентификация пользователя по логину с извлечением информации для аутентификации
	queryAuthenticateUser = "SELECT * FROM Users WHERE Login = @login"
	// Получение публичного идентификатора игрока
	queryGetPublicIDPlayer = "SELECT PublicID FROM Players WHERE id_User = @id_User"
	// Получение идентификатора активного персонажа пользователя
	queryGetActiveCharacter = "SELECT id_ActiveCharacter FROM Players WHERE id_User=@id_User"
	// Получение пользовательских данных
	queryGetUserData = "EXEC getUserData @Id_User"
	// Получение данных персонажа
	queryGetCharacter = "EXEC getCharacter @Id_Character"
	// Получение данных об анимации персонажа
	queryGetAssetsCharacter = "EXEC getAssetsCharacter @Id_Character"
	// Получение информации о друзьях
	queryGetFriendsData = "EXEC GetFriendsAndRequests @PlayerID"
	// Запрос в друзья
	queryRequestFriendship = "EXEC RequestFriendship @RequesterPlayerID, @FriendPublicID"
	// Принять запрос в друзья
	queryAcceptFriendship = "EXEC AcceptFriendship @PlayerID, @RequesterPublicID"
	// Отклонить запрос в друзья
	queryDeclineFriendship = "EXEC DeclineFriendship @PlayerID, @RequesterPublicID"
	// Удалить из друзей
	queryRemoveFriendship = "EXEC RemoveFriendship @PlayerID, @FriendPublicID"
	// Добавляет запись о бое
	querySaveBattleResults = "EXEC SaveBattleResults @Player1ID, @Player2ID, @WinnerID, @StartTime, @EndTime, @isRanked"
	// Обновляет уровень, ранг, число монет после боя
	queryUpdatePlayerStats = "EXEC UpdatePlayerStats @PlayerID, @newLevel, @newRank, @newMoney"
	// Получение информации о сражениях игрока
	queryGetPlayerBattleStats = "EXEC GetPlayerBattleStats @playerID, @isRanked"
	// Получение фонов для магазина
	queryGetShopBackgrounds = "EXEC GetShopBackgrounds @PlayerID"
	// Получение персонажей для магазина
	queryGetShopCharacters = "EXEC GetShopCharacters @PlayerID"
	// Покупка фона
	queryBuyBackground = "EXEC BuyBackground @PlayerID, @BackgroundID, @RemainingMoney OUTPUT, @ResultCode OUTPUT"
	// Сделать фон активным
	querySelectBackground = "EXEC SelectBackground @PlayerID, @BackgroundID, @AssetPath OUTPUT, @ResultCode OUTPUT"
	// Покупка персонажа
	queryBuyCharacter = "EXEC BuyCharacter @PlayerID, @CharacterID, @RemainingMoney OUTPUT, @ResultCode OUTPUT"
	// Сделать персонажа активным
	querySelectCharacter = "EXEC SelectCharacter @PlayerID, @CharacterID, @ResultCode OUTPUT"
)
