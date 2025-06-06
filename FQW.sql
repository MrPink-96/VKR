
CREATE DATABASE GAME_FQW

GO

use GAME_FQW

GO

CREATE TABLE Users
(
	id_User INT IDENTITY(1,1) PRIMARY KEY,
	Login NVARCHAR(50) NOT NULL UNIQUE,
	PasswordHash NVARCHAR(256) NOT NULL,
	isActive BIT NOT NULL DEFAULT 1
)

GO

CREATE TABLE Backgrounds
(
	id_Background INT IDENTITY(1,1) PRIMARY KEY,
	Name NVARCHAR(256) NOT NULL,
	Description NVARCHAR(500) NOT NULL,
	Cost INT DEFAULT 0 CHECK (Cost >= 0),
	AssetPath NVARCHAR(500) NOT NULL
)

GO

CREATE TABLE Characters
(
	id_Character INT IDENTITY(1,1) PRIMARY KEY,
	Name NVARCHAR(256) NOT NULL,
	Description NVARCHAR(500) NOT NULL,
	Health INT DEFAULT 1000 CHECK (Health >= 0),
	Damage INT DEFAULT 10 CHECK (Damage > 0),
	Cost INT DEFAULT 0 CHECK (Cost >= 0)
)

GO

CREATE TABLE Assets_Characters
(
	id_AC INT IDENTITY(1,1) PRIMARY KEY,
	id_Character INT NOT NULL, 
	AnimationType NVARCHAR(256) NOT NULL,
	FrameCount INT NOT NULL CHECK (FrameCount > 0),
	BaseHeight INT NOT NULL,
	BaseWidth INT NOT NULL,
	FrameRate float NOT NULL CHECK (FrameRate >= 0),
	AssetPath NVARCHAR(500) NOT NULL,
	FOREIGN KEY (id_Character) REFERENCES Characters(id_Character) ON DELETE CASCADE
)

GO

CREATE TABLE Players
(
	id_Player INT IDENTITY(1,1) PRIMARY KEY,
	id_User INT NOT NULL,
	PublicID NVARCHAR(10) NOT NULL UNIQUE,
	Name NVARCHAR(50) NOT NULL,
	Level INT DEFAULT 0 CHECK (Level >= 0),
	Money INT DEFAULT 50 CHECK (Money >= 0),
	Rank INT DEFAULT 0 CHECK (Rank >= 0),
	id_ActiveCharacter INT NOT NULL, 
	id_ActiveBackground INT NOT NULL,
	FOREIGN KEY (id_User) REFERENCES Users(id_User)  ON DELETE CASCADE,
	FOREIGN KEY (id_ActiveCharacter) REFERENCES Characters(id_Character),
    FOREIGN KEY (id_ActiveBackground) REFERENCES Backgrounds(id_Background)
)

GO

CREATE TABLE List_Backgrounds
(
	id_LB INT IDENTITY(1,1) PRIMARY KEY,
	id_Player INT NOT NULL,
	id_Background INT NOT NULL,
	FOREIGN KEY (id_Player) REFERENCES Players(id_Player) ON DELETE CASCADE,
	FOREIGN KEY (id_Background) REFERENCES Backgrounds(id_Background) ON DELETE CASCADE
)

GO

CREATE TABLE List_Characters
(
	id_LC INT IDENTITY(1,1) PRIMARY KEY,
	id_Player INT NOT NULL,
	id_Character INT NOT NULL,
	FOREIGN KEY (id_Player) REFERENCES Players(id_Player) ON DELETE CASCADE,
	FOREIGN KEY (id_Character) REFERENCES Characters(id_Character) ON DELETE CASCADE
)

GO


CREATE TABLE Friends
(
    id_Player INT NOT NULL,           
    id_Friend INT NOT NULL,           
    IsConfirmed BIT NOT NULL,         -- 1 — дружба подтверждена, 0 — заявка 
    PRIMARY KEY (id_Player, id_Friend), 
    FOREIGN KEY (id_Player) REFERENCES Players(id_Player) ON DELETE NO ACTION,  
    FOREIGN KEY (id_Friend) REFERENCES Players(id_Player) ON DELETE NO ACTION   
)

GO

CREATE TABLE Battles
(
    id_Battle INT IDENTITY(1,1) PRIMARY KEY,
    id_Player INT NULL,  
    id_Opponent INT NULL,  
    id_Winner INT NULL,  
    StartTime DATETIME NOT NULL,
    EndTime DATETIME NOT NULL,
    isRanked BIT DEFAULT 0 NOT NULL,
    FOREIGN KEY (id_Player) REFERENCES Players(id_Player) ON DELETE NO ACTION,   
    FOREIGN KEY (id_Opponent) REFERENCES Players(id_Player) ON DELETE NO ACTION, 
    FOREIGN KEY (id_Winner) REFERENCES Players(id_Player) ON DELETE NO ACTION    
)

GO


--------------------------------Процедуры
CREATE OR ALTER PROCEDURE getUserData
    @Id_User INT = NULL
AS
BEGIN
    SELECT 
        u.Login AS Login,
		p.id_Player AS PlayerID,
		p.PublicID AS PlayerPublicID,
        p.Name AS PlayerName,
        p.Level AS PlayerLevel,
        p.Money AS PlayerMoney,
        p.Rank AS PlayerRank,
        b.AssetPath AS ActiveBackgroundPath
    FROM Users u
    JOIN Players p ON u.id_User = p.id_User
    LEFT JOIN Backgrounds b ON p.id_ActiveBackground = b.id_Background
    WHERE u.id_User = @Id_User;
END;

GO

CREATE OR ALTER PROCEDURE getAssetsCharacter
	@Id_Character INT = NULL
AS
BEGIN
	SELECT AnimationType, FrameCount, BaseHeight, BaseWidth, FrameRate, AssetPath  FROM Assets_Characters
	WHERE id_Character = @Id_Character
END;

GO

CREATE OR ALTER PROCEDURE getCharacter
	@Id_Character INT = NULL
AS
BEGIN
	SELECT Name, Description, Health, Damage, Cost FROM Characters
	WHERE id_Character = @Id_Character
END;

GO

CREATE OR ALTER PROCEDURE GetFriendsAndRequests
    @PlayerID INT
AS
BEGIN
    SET NOCOUNT ON;

    -- Подтверждённые друзья
    SELECT 
        p.Name AS Name,
        p.PublicID AS PublicID
    FROM Friends f
    JOIN Players p ON p.id_Player = 
        CASE 
            WHEN f.id_Player = @PlayerID THEN f.id_Friend
            ELSE f.id_Player
        END
    WHERE (f.id_Player = @PlayerID OR f.id_Friend = @PlayerID)
        AND f.IsConfirmed = 1;

    -- Входящие заявки
    SELECT 
        p.Name AS Name,
        p.PublicID AS PublicID
    FROM Friends f
    JOIN Players p ON p.id_Player = f.id_Player
    WHERE f.id_Friend = @PlayerID AND f.IsConfirmed = 0;

    -- Исходящие заявки
    SELECT 
        p.Name AS Name,
        p.PublicID AS PublicID
    FROM Friends f
    JOIN Players p ON p.id_Player = f.id_Friend
    WHERE f.id_Player = @PlayerID AND f.IsConfirmed = 0;
END;

GO

CREATE OR ALTER PROCEDURE RequestFriendship
    @RequesterPlayerID INT,
    @FriendPublicID NVARCHAR(10)
AS
BEGIN
    SET NOCOUNT ON;

    DECLARE @FriendPlayerID INT, @Result INT;

    SELECT @FriendPlayerID = id_Player FROM Players WHERE PublicID = @FriendPublicID;

    IF @RequesterPlayerID IS NULL OR @FriendPlayerID IS NULL OR @RequesterPlayerID = @FriendPlayerID
    BEGIN
        SET @Result = -1; -- Неверный игрок или попытка добавить себя
        SELECT @Result AS Result;
        RETURN;
    END

    -- Проверяем, есть ли уже исходящая заявка
    IF EXISTS (SELECT 1 FROM Friends WHERE id_Player = @RequesterPlayerID AND id_Friend = @FriendPlayerID)
    BEGIN
        SET @Result = -2; -- Уже есть исходящая заявка
        SELECT @Result AS Result;
        RETURN;
    END

    -- Автоподтверждение входящей заявки
    IF EXISTS (SELECT 1 FROM Friends WHERE id_Player = @FriendPlayerID AND id_Friend = @RequesterPlayerID AND IsConfirmed = 0)
    BEGIN
        UPDATE Friends 
        SET IsConfirmed = 1 
        WHERE id_Player = @FriendPlayerID AND id_Friend = @RequesterPlayerID;
        
        SET @Result = 1; -- Заявка подтверждена
        SELECT @Result AS Result;
        RETURN;
    END

    -- Новая заявка
    INSERT INTO Friends (id_Player, id_Friend, IsConfirmed) 
    VALUES (@RequesterPlayerID, @FriendPlayerID, 0);

    SET @Result = 0; -- Заявка создана
    SELECT @Result AS Result;
END;

GO

CREATE OR ALTER PROCEDURE AcceptFriendship
    @PlayerID INT,
    @RequesterPublicID NVARCHAR(10)
AS
BEGIN
    SET NOCOUNT ON;

    DECLARE @RequesterPlayerID INT;

    SELECT @RequesterPlayerID = id_Player FROM Players WHERE PublicID = @RequesterPublicID;

    UPDATE Friends
    SET IsConfirmed = 1
    WHERE id_Player = @RequesterPlayerID 
      AND id_Friend = @PlayerID 
      AND IsConfirmed = 0;
END;

GO

CREATE OR ALTER PROCEDURE DeclineFriendship
    @PlayerID INT,
    @RequesterPublicID NVARCHAR(10)
AS
BEGIN
    SET NOCOUNT ON;

    DECLARE @RequesterPlayerID INT;

    SELECT @RequesterPlayerID = id_Player FROM Players WHERE PublicID = @RequesterPublicID;

    DELETE FROM Friends
    WHERE id_Player = @RequesterPlayerID 
      AND id_Friend = @PlayerID 
      AND IsConfirmed = 0;
END;


GO

CREATE OR ALTER PROCEDURE RemoveFriendship
    @PlayerID INT,
    @FriendPublicID NVARCHAR(10)
AS
BEGIN
    SET NOCOUNT ON;

    DECLARE @FriendPlayerID INT;

    SELECT @FriendPlayerID = id_Player FROM Players WHERE PublicID = @FriendPublicID;

    DELETE FROM Friends
    WHERE 
        (id_Player = @PlayerID AND id_Friend = @FriendPlayerID)
        OR
        (id_Player = @FriendPlayerID AND id_Friend = @PlayerID);
END;

GO

CREATE OR ALTER PROCEDURE SaveBattleResults
    @Player1ID INT,
    @Player2ID INT,
    @WinnerID INT = NULL,
    @StartTime DATETIME,
    @EndTime DATETIME,
    @IsRanked BIT
AS
BEGIN
    SET NOCOUNT ON;

    INSERT INTO Battles (id_Player, id_Opponent, id_Winner, StartTime, EndTime, isRanked)
    VALUES (@Player1ID, @Player2ID, @WinnerID, @StartTime, @EndTime, @IsRanked);
END;

GO

CREATE OR ALTER PROCEDURE UpdatePlayerStats
    @PlayerID INT,
    @NewLevel INT,
    @NewRank INT,
    @NewMoney INT
AS
BEGIN
    SET NOCOUNT ON;

    UPDATE Players
    SET
        Level = @NewLevel,
        Rank = @NewRank,
        Money = @NewMoney
    WHERE id_Player = @PlayerID;
END;

GO

CREATE OR ALTER PROCEDURE GetPlayerBattleStats
    @PlayerID INT,
    @IsRanked BIT
AS
BEGIN
    SET NOCOUNT ON;

    -- Последние 30 боёв с результатом игрока
    SELECT TOP 30
        B.StartTime,
        B.EndTime,
        CASE 
            WHEN B.id_Winner IS NULL THEN 'Ничья'
            WHEN B.id_Winner = @PlayerID THEN 'Победа'
            ELSE 'Поражение'
        END AS BattleResult,
        
        P1.Name AS PlayerName,
        P1.PublicID AS PlayerPublicID,

        P2.Name AS OpponentName,
        P2.PublicID AS OpponentPublicID
    FROM Battles B
    LEFT JOIN Players P1 ON B.id_Player = P1.id_Player
    LEFT JOIN Players P2 ON B.id_Opponent = P2.id_Player
    WHERE B.isRanked = @IsRanked AND (B.id_Player = @PlayerID OR B.id_Opponent = @PlayerID)
    ORDER BY B.StartTime DESC;

    -- Статистика
    SELECT
        ISNULL(SUM(CASE WHEN B.id_Winner = @PlayerID THEN 1 ELSE 0 END), 0) AS Wins,
        ISNULL(SUM(CASE WHEN B.id_Winner = CASE WHEN B.id_Player = @PlayerID THEN B.id_Opponent ELSE B.id_Player END THEN 1 ELSE 0 END), 0) AS Losses,
        ISNULL(SUM(CASE WHEN B.id_Winner IS NULL THEN 1 ELSE 0 END), 0) AS Draws
    FROM Battles B
    WHERE B.isRanked = @IsRanked AND (B.id_Player = @PlayerID OR B.id_Opponent = @PlayerID);
END;

GO

CREATE OR ALTER PROCEDURE GetShopCharacters
    @PlayerID INT
AS
BEGIN
    SET NOCOUNT ON;

    -- Получаем персонажей, которые есть у игрока
    SELECT 
        c.id_Character,
        c.Name,
        c.Description,
        c.Health,
        c.Damage,
        c.Cost,
        ac.AssetPath
    FROM List_Characters lc
    JOIN Characters c ON lc.id_Character = c.id_Character
    OUTER APPLY (
        SELECT TOP 1 AssetPath
        FROM Assets_Characters
        WHERE id_Character = c.id_Character AND AnimationType = 'Preview'
    ) ac
    WHERE lc.id_Player = @PlayerID;

    -- Получаем персонажей, которых у игрока нет
    SELECT 
        c.id_Character,
        c.Name,
        c.Description,
        c.Health,
        c.Damage,
        c.Cost,
        ac.AssetPath
    FROM Characters c
    OUTER APPLY (
        SELECT TOP 1 AssetPath
        FROM Assets_Characters
        WHERE id_Character = c.id_Character AND AnimationType = 'Preview'
    ) ac
    WHERE c.id_Character NOT IN (
        SELECT id_Character
        FROM List_Characters
        WHERE id_Player = @PlayerID
    );
END;

GO

CREATE OR ALTER PROCEDURE GetShopBackgrounds
    @PlayerID INT
AS
BEGIN
    SET NOCOUNT ON;

    -- Фоны, которые есть у игрока
    SELECT 
        b.id_Background,
        b.Name,
        b.Description,
        b.Cost,
        b.AssetPath
    FROM List_Backgrounds lb
    JOIN Backgrounds b ON lb.id_Background = b.id_Background
    WHERE lb.id_Player = @PlayerID;

    -- Фоны, которых нет у игрока
    SELECT 
        b.id_Background,
        b.Name,
        b.Description,
        b.Cost,
        b.AssetPath
    FROM Backgrounds b
    WHERE b.id_Background NOT IN (
        SELECT id_Background
        FROM List_Backgrounds
        WHERE id_Player = @PlayerID
    );
END;

GO

CREATE OR ALTER PROCEDURE BuyBackground
    @PlayerID INT,
    @BackgroundID INT,
    @RemainingMoney INT OUTPUT,
    @ResultCode INT OUTPUT -- 0 = успех, 1 = фон не найден, 2 = уже куплен, 3 = не хватает денег, 4 = ошибка
AS
BEGIN
    SET NOCOUNT ON;

    BEGIN TRY
        BEGIN TRANSACTION;

        -- Проверяем, что фон существует
        IF NOT EXISTS (SELECT 1 FROM Backgrounds WHERE id_Background = @BackgroundID)
        BEGIN
            SET @ResultCode = 1;
            ROLLBACK TRANSACTION;
            RETURN;
        END

        -- Проверяем, что фон еще не куплен
        IF EXISTS (
            SELECT 1 FROM List_Backgrounds 
            WHERE id_Player = @PlayerID AND id_Background = @BackgroundID
        )
        BEGIN
            SET @ResultCode = 2;
            ROLLBACK TRANSACTION;
            RETURN;
        END

        DECLARE @Cost INT;
        SELECT @Cost = Cost FROM Backgrounds WHERE id_Background = @BackgroundID;

        DECLARE @CurrentMoney INT;
        SELECT @CurrentMoney = Money FROM Players WHERE id_Player = @PlayerID;

        -- Проверяем хватает ли денег
        IF @CurrentMoney < @Cost
        BEGIN
            SET @ResultCode = 3;
            ROLLBACK TRANSACTION;
            RETURN;
        END

        -- Снимаем деньги
        UPDATE Players
        SET Money = Money - @Cost
        WHERE id_Player = @PlayerID;

        -- Добавляем фон игроку
        INSERT INTO List_Backgrounds (id_Player, id_Background)
        VALUES (@PlayerID, @BackgroundID);

        -- Возвращаем оставшиеся деньги
        SELECT @RemainingMoney = Money
        FROM Players
        WHERE id_Player = @PlayerID;

        SET @ResultCode = 0;
        COMMIT TRANSACTION;
    END TRY
    BEGIN CATCH
        ROLLBACK TRANSACTION;
        SET @ResultCode = 4;
    END CATCH
END;

GO

CREATE OR ALTER PROCEDURE SelectBackground
    @PlayerID INT,
    @BackgroundID INT,
    @AssetPath NVARCHAR(255) OUTPUT,
    @ResultCode INT OUTPUT -- 0 = успех, 1 = фон не найден, 2 = фон не куплен, 3 = ошибка
AS
BEGIN
    SET NOCOUNT ON;
    SET @AssetPath = NULL;

    BEGIN TRY
        -- Проверяем, что фон существует
        IF NOT EXISTS (SELECT 1 FROM Backgrounds WHERE id_Background = @BackgroundID)
        BEGIN
            SET @ResultCode = 1;
            RETURN;
        END

        -- Проверяем, что фон куплен игроком
        IF NOT EXISTS (
            SELECT 1 FROM List_Backgrounds 
            WHERE id_Player = @PlayerID AND id_Background = @BackgroundID
        )
        BEGIN
            SET @ResultCode = 2;
            RETURN;
        END

        -- Обновляем выбранный активный фон
        UPDATE Players
        SET id_ActiveBackground = @BackgroundID
        WHERE id_Player = @PlayerID;

        -- Возвращаем путь к ассету фона
        SELECT @AssetPath = AssetPath
        FROM Backgrounds
        WHERE id_Background = @BackgroundID;

        SET @ResultCode = 0;
    END TRY
    BEGIN CATCH
        SET @ResultCode = 3;
    END CATCH
END;

GO

CREATE OR ALTER PROCEDURE BuyCharacter
    @PlayerID INT,
    @CharacterID INT,
    @RemainingMoney INT OUTPUT,
    @ResultCode INT OUTPUT -- 0 = успех, 1 = персонаж не найден, 2 = уже куплен, 3 = не хватает денег, 4 = ошибка
AS
BEGIN
    SET NOCOUNT ON;
    SET @RemainingMoney = NULL;

    BEGIN TRY
        BEGIN TRANSACTION;

        -- Проверяем, что персонаж существует
        IF NOT EXISTS (SELECT 1 FROM Characters WHERE id_Character = @CharacterID)
        BEGIN
            SET @ResultCode = 1; 
            ROLLBACK TRANSACTION;
            RETURN;
        END

        -- Проверяем, что персонаж не куплен уже игроком
        IF EXISTS (
            SELECT 1 FROM List_Characters 
            WHERE id_Player = @PlayerID AND id_Character = @CharacterID
        )
        BEGIN
            SET @ResultCode = 2; 
            ROLLBACK TRANSACTION;
            RETURN;
        END

        -- Получаем стоимость персонажа
        DECLARE @Cost INT;
        SELECT @Cost = Cost FROM Characters WHERE id_Character = @CharacterID;

        -- Получаем текущие деньги игрока
        DECLARE @CurrentMoney INT;
        SELECT @CurrentMoney = Money FROM Players WHERE id_Player = @PlayerID;

        -- Проверяем хватает ли денег
        IF @CurrentMoney < @Cost
        BEGIN
            SET @ResultCode = 3; 
            ROLLBACK TRANSACTION;
            RETURN;
        END

        -- Списываем деньги игрока
        UPDATE Players
        SET Money = Money - @Cost
        WHERE id_Player = @PlayerID;

        -- Добавляем персонажа в список купленных
        INSERT INTO List_Characters (id_Player, id_Character)
        VALUES (@PlayerID, @CharacterID);

        -- Возвращаем оставшиеся деньги
        SELECT @RemainingMoney = Money FROM Players WHERE id_Player = @PlayerID;

        SET @ResultCode = 0; 
        COMMIT TRANSACTION;
    END TRY
    BEGIN CATCH
        ROLLBACK TRANSACTION;
        SET @ResultCode = 4; 
    END CATCH
END;

GO

CREATE OR ALTER PROCEDURE SelectCharacter
    @PlayerID INT,
    @CharacterID INT,
    @ResultCode INT OUTPUT -- 0 = успех, 1 = персонаж не найден, 2 = не куплен, 3 = ошибка
AS
BEGIN
    SET NOCOUNT ON;

    BEGIN TRY
        -- Проверка, что персонаж существует
        IF NOT EXISTS (SELECT 1 FROM Characters WHERE id_Character = @CharacterID)
        BEGIN
            SET @ResultCode = 1;
            RETURN;
        END

        -- Проверка, что персонаж куплен игроком
        IF NOT EXISTS (
            SELECT 1 FROM List_Characters 
            WHERE id_Player = @PlayerID AND id_Character = @CharacterID
        )
        BEGIN
            SET @ResultCode = 2; 
            RETURN;
        END

        -- Установка активного персонажа для игрока
        UPDATE Players
        SET id_ActiveCharacter = @CharacterID
        WHERE id_Player = @PlayerID;

        SET @ResultCode = 0;
    END TRY
    BEGIN CATCH
        SET @ResultCode = 3;
    END CATCH
END;

GO
--------------------------------Триггеры
CREATE TRIGGER trg_InsertListBackgrounds
ON Players
AFTER INSERT
AS
BEGIN
	INSERT INTO List_Backgrounds (id_Player, id_Background)
    SELECT 
        i.id_Player, 
        i.id_ActiveBackground
    FROM 
        Inserted i
    WHERE 
        i.id_ActiveBackground IS NOT NULL;
END;

GO

CREATE TRIGGER trg_InsertListCharacters
ON Players
AFTER INSERT
AS
BEGIN
	INSERT INTO List_Characters (id_Player, id_Character)
    SELECT 
        i.id_Player, 
        i.id_ActiveCharacter
    FROM 
        Inserted i
    WHERE 
        i.id_ActiveCharacter IS NOT NULL;
END;

GO

CREATE TRIGGER trg_HandleBattlesOnPlayerDelete
ON Players
AFTER DELETE
AS
BEGIN
    UPDATE Battles
    SET id_Player = NULL
    WHERE id_Player IN (SELECT id_Player FROM deleted)
      AND id_Opponent NOT IN (SELECT id_Player FROM deleted);

    UPDATE Battles
    SET id_Opponent = NULL
    WHERE id_Opponent IN (SELECT id_Player FROM deleted)
      AND id_Player NOT IN (SELECT id_Player FROM deleted);

    DELETE FROM Battles
    WHERE id_Player IS NULL AND id_Opponent IS NULL;
END;
GO

CREATE TRIGGER trg_DeleteFriendsOnDelete
ON Players
AFTER DELETE
AS
BEGIN
    DELETE FROM Friends
    WHERE id_Player IN (SELECT id_Player FROM deleted)
       OR id_Friend IN (SELECT id_Player FROM deleted);
END

