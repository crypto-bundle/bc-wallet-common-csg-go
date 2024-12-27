# Слой данных. Работа с PostgreSQL. Примеры выборки данных

## Введение

Данный документ содержить в себе примеры использования базового набора инструментов для создания
сервис-компонентов реализующих механизм получения данных из базы данных PostgreSQL. Иными словами
примеры работы с SELECT SQL-запросами.

Ниже приведены примеры для:
- Выборки **одной** записи - [Пример использования](#выборка-одной-записи)
- Выборки **множества записей** - [Пример использования](#выборка-множества-записей)
- Выборка **множества записей с условием вхождения через оператор** `IN` - [Пример использования](#выборка-множества-записей-через-in)
- Выборка **можества записей с callback механизмом** [Пример использования](#выборка-множества-записей-c-callback)

## Примеры использования

#### Выборка одной записи

Выборка одной записи по сути самый простой запрос в плане реализации метод вашего 
сервис-компонента получения данных. Фактически функцию можно разделить на 4 состовляющих:
1. Попытка определить транзакционный контекст запроса.
1. Целевой SQL-запрос на получение данных из БД.
1. Выполнение SQL-запроса через фукцию `QueryRowx` из набора `sqlx` библиотеки.
1. Попытка отсканировать вернувшийся из базы данных результат через функцию `StructScan`.

Важно заметить что, в данном примере используется:
- Концепция программирование - _замыкание_ - для передачи результата из анонимной-функции.
- Не используются именованные возвратные параметры
- Для `StructScan` есть проверка через хэлпер `commonPostgres.EmptyOrError`, так как
  возможна ошибка `sql.ErrNoRows`.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.

```go
package orders_data

import (
  "context"

  commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

  "github.com/jmoiron/sqlx"
)

type ordersDataStore struct {
  dbConn *commonPostgres.Connection
  e errorFormatterService
}

func (s *ordersDataStore) GetOrderItemByID(ctx context.Context,
        identifier uint,
) (*entities.OrderItem, error) {
  var dataItem *entities.OrderItem = nil

  if err := s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
    q := `SELECT "id", "uuid", "entity_uuid", "task_uuid",
              "task_type", "status", 
          "created_at", "updated_at"
        FROM "orders"
        WHERE "id" = $1`

    row := stmt.QueryRowx(q, identifier)
    scannedItem := &entities.OrderItem{}

    defer func() {
      row = nil
      scannedItem = nil
    }()

    err := row.StructScan(scannedItem)
    if err != nil {
      // EmptyOrError используется потому что StructScan может вернуть sql.ErrNoRows,
      // по соглашению мы считаем что если в базе данных не найдено записи - 
      // возвращаем nil как целевое значение, и nil в качестве второго возвратного 
      // параметра ошибки - "return nil, nil"
      return commonPostgres.EmptyOrError(err, "unable to get order item by uuid")
    }

	
	  // Используем замыкание для передачи значения из анонимной функции
    dataItem = scannedItem

    return nil
  }); err != nil {
    // ErrorNoWrap потому что из анонимной функции из 
    // TryWithTransaction уже вернется отформатированная ошибка 
    return nil, s.e.ErrorNoWrap(err)
  }

  // Обрати внимание - здесь возвращаем просто переменную которую заполнили внутри
  // анонимной функции, именнованные параметры не используются.
  return dataItem, nil
}
```

#### Выборка множества записей

Выборка множества записей разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса.
1. Целевой SQL-запрос на получение данных из БД.
1. Выполнение SQL-запроса через фукцию `Queryx` из набора `sqlx` библиотеки.
1. Подготовка слайса к заполнению данными.
1. Цикл по полученным сырым данных из базы данных с использованием функции `rows.Next()`
1. Попытка на каждой итерации цикла отсканировать строку данных через функцию `StructScan`.

Важно заметить что, в данном примере используется:
- Концепция программирование - _замыкание_ - для передачи результата из анонимной-функции, но при этом
  присваивание идет сразу в именнование возвратные параметры.
- Используются именованные возвратные параметры. И после ключевого слова `return` - нет аргументов.
- Для `StructScan` нет проверки через хэлпер `commonPostgres.EmptyOrError`, так как
  ошибка `sql.ErrNoRows` не возможна. Причная по которой мы не ожидаем ошибку `sql.ErrNoRows`, это
  _**"если запустился цикл - значит данные есть"**_.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.

```go
package orders_data

import (
  "context"

  commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

  "github.com/jmoiron/sqlx"
)

type ordersDataStore struct {
  dbConn *commonPostgres.Connection
  e errorFormatterService
}

func (s *ordersDataStore) GetOrderItemsInDateRange(ctx context.Context,
  from time.Time,
  to time.Time,
) (count uint, list []*entities.OrderItem, err error) {
  if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
    query := `SELECT "id", "uuid", "entity_uuid", "task_uuid",
				"task_type", "sttaus",
				"created_at", "updated_at"
			FROM "orders"
			WHERE "created_at" <= $1 AND "created_at" > $2;`

    rows, clbErr := stmt.Queryx(query, from, to)
    if clbErr != nil {
      return s.e.ErrorOnly(clbErr, "unable to execute selection query")
    }

    defer rows.Close()

    blockItems := make([]*entities.OrderItem, 0)
	
    for rows.Next() {
      data := &entities.OrderItem{}

      scanErr := rows.StructScan(data)
      if scanErr != nil {
        // commonPostgres.EmptyOrError НЕ используется потому что мы в данном участке кода
        // мы уже находимся в цикле, а значит какие-то данные уже вернулись и ошибка
        // sql.ErrNoRows невозможна.
        return s.e.ErrorOnly(scanErr, "unable to scan OrderItem db-record")
      }

      blockItems = append(blockItems, data)
    }

    if cnt := uint(len(blockItems)); cnt > 0 {
      list = blockItems
      count = cnt
    }

    blockItems = nil

    return nil
  }); err != nil {
    return 0, nil, s.e.ErrorNoWrap(err)
  }

  // Обрати внимание - здесь возвращаем именованные параметры, которые заполняются
  // в анонимной функции внутри функции TryWithTransaction.
  return
}
```

#### Выборка множества записей через IN

Данный тип получения данных отличается в первую очередь ключевым словом `IN` в SQL-запросе. Для 
обслуживания запроса с динамическим количеством параметров так же используются неокторые специальные
функции.

Выборка множества записей разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса.
1. Целевой SQL-запрос на получение данных из БД.
1. Биндинг динамического списка парметров через функцию `sqlx.In`.
1. Ре-биндинг значений значений для ранее заданных параметров через функцию `Rebind`.
1. Выполнение SQL-запроса через фукцию `sqlx.Queryx`.
1. Подготовка слайса к заполнению данными.
1. Цикл по полученным сырым данных из базы данных с использованием функции `rows.Next()`
1. Попытка на каждой итерации цикла отсканировать строку данных через функцию `StructScan`.

Для данного примера важно заметить следующее:
- Используется концепция программирование - _замыкание_ - для передачи результата из анонимной-функции, но при этом
  присваивание идет сразу в именнование возвратные параметры.
- Используются _именованные возвратные параметры_. И после ключевого слова `return` - нет аргументов.
- Для `StructScan` нет проверки через хэлпер `commonPostgres.EmptyOrError`, так как
  ошибка `sql.ErrNoRows` не возможна. Причная по которой мы не ожидаем ошибку `sql.ErrNoRows`, это
  _**"если запустился цикл - значит данные есть"**_.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.
- Там где заранее неивестно кол-во параметров для оператора `IN` используется плейсхолдер `?`.
- Фунция `sqlx.In` заполняет символ `?` нужным колличеством параметров `$`.
- Фунция `Rebind`, используется для ассоциации значений с их параметрами `$`.

```go
package orders_data

import (
	"context"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type ordersDataStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *ordersDataStore) GerOrdersByStatusesAndTaskTypes(ctx context.Context,
	statuses []types.OrderStatus,
	taskTypes []types.TaskType,
) (count uint, list []*entities.OrderItem, err error) {
	if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
	  const q = `SELECT "id", "uuid", "entity_uuid", "task_uuid",
				"task_type", "status",
				"created_at", "updated_at"
			FROM "orders"
			WHERE "status" IN (?) AND "task_type" IN(?);`
	  
		bondedQuery, args, clbErr := sqlx.In(q, statuses, taskTypes)
		if clbErr != nil {
			return s.e.ErrorOnly(clbErr, "unable to pre-process selection query")
		}

		reBonded := stmt.Rebind(bondedQuery)
		
		rows, clbErr := stmt.Queryx(reBonded, args...)
		if clbErr != nil {
			return s.e.ErrorOnly(clbErr, "unable to execute selection query")
		}

		defer rows.Close()

		blockItems := make([]*entities.OrderItem, 0)
		for rows.Next() {
			data := &entities.OrderItem{}

			scanErr := rows.StructScan(data)
			if scanErr != nil {
				return s.e.ErrorOnly(scanErr, "unable to scan order item db-record")
			}

			blockItems = append(blockItems, data)
		}

		// если счетчик элементов выборки больше 0,
		// то заполняем значение возвратных именнованых параметров
		if cnt := uint(len(blockItems)); cnt > 0 {
			list = blockItems
			count = cnt
		}

		// иначе они остаются со значениями  0 и nil
		blockItems = nil

		return nil
	}); err != nil {
		return 0, nil, s.e.ErrorNoWrap(err)
	}

	// Обрати внимание - здесь возвращаем именованные параметры, которые заполняются
	// в анонимной функции внутри функции TryWithTransaction.
	return
}
```

#### Выборка множества записей c callback

Данный тип получения данных отличается от [Выборки записи чере IN](#выборка-множества-записей-через-in) 
только тем, что используется дополнительный аргумент функции - `loopItemProcessingCallback`.
Функция передаваемая извне будет применяться на каждой итерации цикла `rows.Next()`. 

Такой принцип работы введен как оптимизация количества циклов. Иными словами, если нужно над результатами полученными из базы данных
провести некоторые манипуляции, то необязательно делать такие манипуляции в отдельном цикле - можно сделать
все необходимое сразу в цикле считывания данных из БД.

Выборка множества записей идентична предыдущему примеру, но имеет дополнительный шаг:
1. Цикл по полученным сырым данных из базы данных с использованием функции `rows.Next()`
1. Попытка на каждой итерации цикла отсканировать строку данных через функцию `StructScan`.
1. Попытка на каждой итерации цикла вызвать `loopItemProcessingCallback()` функцию.

В данном примере важно выделить следующее:
- Две функции реализуют получение списка сущностей:
  - Первая функция `GetActiveWalletSessionsByWalletUUID` - просто возвращает нужные значения
  - Вторая функция `GetWalletSessionsByWalletUUIDAndStatusClb` - возвращает нужные значения, но так же
    имеет функционал вызова `callback` фукнции.
  - Первая функция использует вторую, но в качестве `callback`-фукнции передает функцию-пустышку `nopSessionCallback`

```go
package orders_data

import (
	"context"
	
	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

var nopSessionCallback = func(item *entities.MnemonicWalletSession) error {
	return nil
}

type walletsDataStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *walletsDataStore) GetActiveWalletSessionsByWalletUUID(ctx context.Context, 
	walletUUID string, 
) (count uint, list []*entities.MnemonicWalletSession, err error) {
	return s.GetWalletSessionsByWalletUUIDAndStatusClb(ctx, walletUUID,
		[]types.MnemonicWalletSessionStatus{
			types.MnemonicWalletSessionStatusPrepared,
		},
		nopSessionCallback)
}

func (s *walletsDataStore) GetWalletSessionsByWalletUUIDAndStatusClb(ctx context.Context,
	walletUUID string,
	sessionStatuses []types.MnemonicWalletSessionStatus,
	loopItemProcessingCallBack func(item *entities.MnemonicWalletSession) error,
) (count uint, list []*entities.MnemonicWalletSession, err error) {
	if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
	  const q = `SELECT *
			FROM "mnemonic_wallet_sessions"
			WHERE "mnemonic_wallet_uuid" = ? AND
					? BETWEEN "started_at" AND "expired_at" 
				AND
          "status" IN (?)
			ORDER BY "expired_at";`
	  
		query, args, clbErr := sqlx.In(q, walletUUID, time.Now(), sessionStatuses)
		if clbErr != nil {
			return s.e.ErrorOnly(clbErr, "unable to pre-process selection query")
		}

		bonded := stmt.Rebind(query)
		rows, clbErr := stmt.Queryx(bonded, args...)
		if clbErr != nil {
			return s.e.ErrorOnly(clbErr, "unable to execute SQL-query")
		}
		
		defer rows.Close()

		itemsList := make([]*entities.MnemonicWalletSession, 0)
		itemsCount := uint(0)

		for rows.Next() {
			data := &entities.MnemonicWalletSession{}

			scanErr := rows.StructScan(data)
			if scanErr != nil {
				return s.e.ErrorOnly(scanErr, "unable to scan wallet session item db-record")
			}

			scanErr = loopItemProcessingCallBack(data)
			if scanErr != nil {
				return s.e.ErrorOnly(scanErr, "unable to execute item processing callback")
			}

			itemsList = append(itemsList, data)
			itemsCount++
		}

		if itemsCount == 0 {
			itemsList = nil
		}

		count = itemsCount
		list = itemsList

		return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции из 
	  // TryWithTransaction уже вернется отформатированная ошибка 
		return 0, nil, s.e.ErrorNoWrap(err)
	}

	// Обрати внимание - здесь возвращаем именованные параметры, которые заполняются
	// в анонимной функции внутри функции TryWithTransaction.
	return
}

```