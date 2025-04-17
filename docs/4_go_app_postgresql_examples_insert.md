# Слой данных. Работа с PostgreSQL. Примеры добавления данных

## Введение

Данный документ содержить в себе примеры использования базового набора инструментов для создания
сервис-компонентов реализующих механизм добавления новых записей в базу данных PostgreSQL. Иными словами
примеры работы с INSERT SQL-запросами.

Ниже приведены примеры для:
- Вставки **одной** записи без возвратного значения- [Пример использования](#добавление-одной-записи-без-выборки-результата)
- Вставки **одной** записи с возвратным значением- [Пример использования](#добавление-одной-записи-с-результатом)
- Вставка **одной записи с требованием транзакционности** - [Пример использования](#вставка-одной-записи-с-требованием-транзакционности)
- Вставки **множества записей c выборкой** - [Пример использования](#добавление-множества-записей-одним-запросом-с-выборкой-данных)
- Вставки **множества записей без выборки данных** - [Пример использования](#добавление-множества-записей-одним-запросом-без-выборки-данных)
- Вставки **множества записей c выборкой отсортированных данных** - [Пример использования](#добавление-множества-записей-одним-запросом-с-выборкой-отсортированных-данных)

## Примеры использования

#### Добавление одной записи без выборки результата

Данный пример добавления данных описывае способ сохранения одной записи в
определенную таблицу используя один SQL-запрос. В рамках данного примере расмотрена ситуация
в которой после вставки не возвращаются данные о вставленной записи.

Функция добавления записи разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на добавление множества данных в БД.
1. Биндинг динамического списка именованных параметров через функцию `BindNamed`.
1. Выполнение SQL-запроса через фукцию `Exec` с передачей значений для вставки.
1. Получение кол-ва измененных строк
1. Проверка что измененных строк было больше 0, иначе ошибка.

Для данного примера важно отметить следующее:
- Используется концепция программирования - _замыкание_ - для передачи результата из анонимной-функции.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.
- Запрос выполняется без выборки данных, а значит используется функция `Exec` из библиотеки `sqlx`.
- После вставки данных проверяется кол-во измененных(добавленых) строк таблицы.

```go
package orders_data

import (
	"context"
	"time"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type addressPoolDataStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *addressPoolDataStore) AddAddressPoolItem(ctx context.Context,
	item *entities.AddressPoolItem,
) error {
	if err := s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
	  now := time.Now()

	  const query = `
			INSERT INTO address_pool
				(address_uuid, 
				 wallet_uuid, 
				 blockchain_uuid, 
				 address, 
				 status, 
				 created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id;`

	  row := stmt.QueryRowx(query,
		  address.AddressUUID,
		  address.WalletUUID,
		  address.BlockchainUUID,
		  address.Address,
		  address.Status,
		  now,
	  )

	  var addressID uint
	  errTx := row.Scan(&addressID);
	  if errTx != nil {
		  return s.e.Error(errTx, "unable to insert address record")
	  }

	  address.ID = addressID
	  address.CreatedAt = now

		return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции из 
	  // TryWithTransaction уже вернется отформатированная ошибка 
		return s.e.ErrorNoWrap(err)
	}

	return nil
}
```

#### Добавление одной записи с результатом

Данный тип добавления данных является наиболее распространенным и используется для добавления
одной записи с фиксированным числом параметров для вставки.

Функция добавления записи разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на добавление данных в БД.
1. Выполнение SQL-запроса через фукцию `sqlx.QueryRowx` с передачей значений для вставки.
1. Попытка на отсканировать данные из результата SQL-запроса через функцию `Scan`.
1. Если успешен предыдущий шаг - присваиваем результат через механизм замыкания.

Для данного примера важно заметить следующее:
- Используется концепция программирование - _замыкание_ - для передачи результата из анонимной-функции.
- Использование функции `Scan` обусловлено тем, что наш SQL-запрос содержит в себе директиву
  `RETURNING "id"`, а значит имеет возвращаемое значение.
- Для `Scan` нет проверки через хэлпер `commonPostgres.EmptyOrError`, так как
  ошибка `sql.ErrNoRows` не возможна при INSERT запросе.
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

func (s *ordersDataStore) AddOrderItem(ctx context.Context,
  item *entities.OrderItem,
) (*entities.OrderItem, error) {
  if err := s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
    var itemID uint

    q := `INSERT INTO "orders" ("uuid", "entity_uuid", "task_uuid",
				"task_type", "status", 
				"created_at", "updated_at")
			VALUES($1, $2, $3, $4, $5, $6, $7)
			RETURNING "id";`

    row := stmt.QueryRowx(q,
      item.UUID, item.EntityUUID, item.TaskUUID,
      item.TaskType, item.Status,
      item.CreatedAt, item.UpdatedAt)

    err := row.Scan(&itemID)
    if err != nil {
      // EmptyOrError НЕ используется потому что при вызове Scan для INSERT RETURNING
      // запроса sql.ErrNoRows не вернется. А значит EmptyOrError испльзовать смысла нет.
      return s.e.ErrorOnly(err, ErrUnableExecuteQuery.Error())
    }

    item.ID = itemID

    return nil
  }); err != nil {
    // ErrorNoWrap потому что из анонимной функции из 
    // TryWithTransaction уже вернется отформатированная ошибка 
    return nil, s.e.ErrorNoWrap(err)
  }

  return item, nil
}
```

#### Вставка одной записи с требованием транзакционности

Данный тип добавления данных данных используется не часто, но иллюстрирует работу с sql-запросами
которым необходимо выполнение внутри транзакционого контекста - он же _Transactional Statement_.

Функция добавления записи разделена на несколько состовляющих:
1. Получение транзакционного стейтмента через `MustWithTransaction`.
1. Целевой SQL-запрос на добавление данных в БД.
1. Выполнение SQL-запроса через фукцию `sqlx.QueryRowx` с передачей значений для вставки.
1. Попытка на отсканировать данные из результата SQL-запроса через функцию `Scan`.
1. Если успешен предыдущий шаг - присваиваем результат через механизм замыкания.

Для данного примера важно заметить следующее:
- Используется концепция программирование - _замыкание_ - для передачи результата из анонимной-функции.
- Использование функции `Scan` обусловлено тем, что наш SQL-запрос содержит в себе директиву
  `RETURNING "counter_value"`, а значит имеет возвращаемое значение.
- Для `Scan` нет проверки через хэлпер `commonPostgres.EmptyOrError`, так как
  ошибка `sql.ErrNoRows` не возможна при INSERT запросе.
- SQL-запрос написанный в примере может только внутри транзакционного стейтмента.
  Для этого используется функция `MustWithTransaction`, данная функция вернет ошибка если в 
  контексте `ctx` необнаружена переменная хранящая транзакционный стейтмент.

```go
package wallets_data

import (
	"context"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type pgRepository struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}


func (s *pgRepository) GetNextWalletSessionCounterValue(ctx context.Context,
	tokenIdentifier uuid.UUID,
) (int32, error) {
	var nextNonce int32

	err := s.dbConn.MustWithTransaction(ctx, func(stmt *sqlx.Tx) error {
		var tmp int32

		q := `
			INSERT INTO wallet_sessions_access_tokens_counters (token_uuid, counter_value)
				VALUES ($1, 0)
			ON CONFLICT (token_uuid) 
				DO UPDATE
					SET counter_value = wallet_sessions_access_tokens_counters.counter_value + 1
			RETURNING counter_value`

		row := stmt.QueryRowx(q, tokenIdentifier)
		clbErr := row.Scan(&tmp)
		if clbErr != nil {
			return clbErr
		}

		nextNonce = tmp

		return nil
	})

	if err != nil {
		return -1, err
	}

	return nextNonce, nil
}
```

#### Добавление множества записей одним запросом с выборкой данных
Данный пример добавления данных описывае способ сохранения множества однотипны данных в 
определенную таблицу используя один SQL-запрос. В рамках данного примере расмотрена ситуация
в которой после вставки данных, в качестве результата у нас будет список из успешно вставленных данных.
При этом данные вернуться в том порядке в котором они были вставлены.

Функция добавления записи разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на добавление множества данных в БД.
1. Биндинг динамического списка именованных параметров через функцию `BindNamed`.
1. Выполнение SQL-запроса через фукцию `Queryx` с передачей значений для вставки.
1. Подготовка слайса к заполнению данными.
1. Цикл по полученным сырым данных из базы данных с использованием функции `rows.Next()`
1. Попытка на каждой итерации цикла отсканировать строку данных через функцию `StructScan`.
1. Если цикл на предыдущих шагах успешен - присваиваем значения в возвратные переменные
   через механизм замыкания.

Для данного примера важно заметить следующее:
- Используется концепция программирование - _замыкание_ - для передачи результата из анонимной-функции.
- Использование функции `Scan` обусловлено тем, что наш SQL-запрос содержит в себе директиву
  `RETURNING "id"`, а значит имеет возвращаемое значение.
- Для `Scan` нет проверки через хэлпер `commonPostgres.EmptyOrError`, так как
  ошибка `sql.ErrNoRows` не возможна при INSERT запросе.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.

```go
package orders_data

import (
	"context"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type addressPoolDataStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *addressPoolDataStore) AddMultipleAddressPoolItems(ctx context.Context,
	items []*entities.AddressPoolItem,
) (count uint64, list []*entities.AddressPoolItem, err error) {
	var itemsList []*entities.AddressPoolItem = nil

	if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
		q := `
        INSERT INTO "address_pool" ("uuid", "wallet_uuid", 
					"sub_wallet_uuid",
					"hd_wallet_account_index", "hd_wallet_internal_index", "hd_wallet_address_index", 
					"address", "created_at")
				VALUES(:uuid, :wallet_uuid, :sub_wallet_uuid,
					:hd_wallet_account_index, :hd_wallet_internal_index, :hd_wallet_address_index,
					:address, :created_at)
        RETURNING *`
		
		bondedSql, args, queryErr := stmt.BindNamed(q, items)
		if queryErr != nil {
			return s.e.ErrorOnly(queryErr, ErrUnableExecuteQuery.Error())
		}

		rows, queryErr := stmt.Queryx(bondedSql, args...)
		if queryErr != nil {
      // commonPostgres.EmptyOrError НЕ используется потому что при вызове Scan для INSERT RETURNING
      // запроса sql.ErrNoRows не вернется. А значит EmptyOrError испльзовать смысла нет.
      return s.e.ErrorOnly(queryErr, ErrUnableExecuteQuery.Error())
		}

		defer rows.Close()

		itemsList = make([]*entities.AddressPoolItem, 0)
		itemsCount := uint64(0)

		for rows.Next() {
			itemData := &entities.AddressPoolItem{}

			iterErr := rows.StructScan(data)
			if iterErr != nil {
				return iterErr
			}

			itemsList = append(itemsList, itemData)
			itemsCount++
		}

		count = itemsCount
		list = itemsList

		return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции из 
	  // TryWithTransaction уже вернется отформатированная ошибка 
		return 0, nil, s.e.ErrorNoWrap(err)
	}

	return
}
```

#### Добавление множества записей одним запросом без выборки данных
Данный пример добавления данных описывае способ сохранения множества однотипных данных в
определенную таблицу используя один SQL-запрос. В рамках данного примере расмотрена ситуация
в которой после вставки данных не возвращаются данные о вставленных записях.

Функция добавления записи разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на добавление множества данных в БД.
1. Биндинг динамического списка именованных параметров через функцию `BindNamed`.
1. Выполнение SQL-запроса через фукцию `Exec` с передачей значений для вставки.
1. Получение кол-ва измененных строк
1. Проверка что измененных строк было больше 0, иначе ошибка.

Для данного примера важно отметить следующее:
- Используется концепция программирования - _замыкание_ - для передачи результата из анонимной-функции.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.
- Запрос выполняется без выборки данных, а значит используется функция `Exec` из библиотеки `sqlx`.
- После вставки данных проверяется кол-во измененных(добавленых) строк таблицы.

```go
package orders_data

import (
	"context"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type addressPoolDataStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *addressPoolDataStore) AddMultipleAddressPoolItems(ctx context.Context,
	items []*entities.AddressPoolItem,
) error {
	if err := s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
		q := `
        INSERT INTO "address_pool" ("uuid", "wallet_uuid", 
					"sub_wallet_uuid",
					"hd_wallet_account_index", "hd_wallet_internal_index", "hd_wallet_address_index", 
					"address", "created_at")
				VALUES(:uuid, :wallet_uuid, :sub_wallet_uuid,
					:hd_wallet_account_index, :hd_wallet_internal_index, :hd_wallet_address_index,
					:address, :created_at)`
		
		bondedSql, args, clbErr := stmt.BindNamed(q, items)
		if clbErr != nil {
			return s.e.ErrorOnly(clbErr, "unable to create query with bonded names")
		}

	  qResult, clbErr := stmt.Exec(bondedSql, args...)
	  if clbErr != nil {
		  return s.e.ErrorOnly(clbErr, ErrUnableExecuteQuery.Error())
	  }

	  affectedRows, clbErr := qResult.RowsAffected()
	  if clbErr != nil {
		    return s.e.ErrorOnly(clbErr, "unable to check query result - affected rows")
	  }

	  if affectedRows == 0 {
		  return s.e.ErrorOnly(ErrHaveNoAffectedRows)
	  }

		return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции из 
	  // TryWithTransaction уже вернется отформатированная ошибка 
		return s.e.ErrorNoWrap(err)
	}

	return nil
}
```

#### Добавление множества записей одним запросом с выборкой отсортированных данных
Данный пример добавления данных описывае способ сохранения множества однотипных данных в
определенную таблицу используя один SQL-запрос. В рамках данного примере расмотрена ситуация
в которой после вставки возвращаются вставленные записях отсортированные по определенному полю.

Функция добавления записи разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на добавление множества данных в БД с оберткой вида `WITH "inserted" AS ... SELECT`.
1. Биндинг динамического списка именованных параметров через функцию `BindNamed`.
1. Выполнение SQL-запроса через фукцию `Queryx` с передачей значений для вставки.
1. Подготовка слайса к заполнению данными.
1. Цикл по полученным сырым данных из базы данных с использованием функции `rows.Next()`
1. Попытка на каждой итерации цикла отсканировать строку данных через функцию `StructScan`.
1. Если цикл на предыдущих шагах успешен - присваиваем значения в возвратные переменные
   через механизм замыкания.

Для данного примера важно отметить следующее:
- Используется концепция программирования - _замыкание_ - для передачи результата из анонимной-функции.
  При этом результат выполнения сразу в случае успеха сразу присваивается в именованные возвратные параметры.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.
- Запрос выполняется c выборкой вставленных данных, а значит используется функция `Queryx` из библиотеки `sqlx`.
- Добавленные данные возвращаеются как **отсортированный список**, для сортировки используется SQL-код вида
  `WITH "inserted" AS ... SELECT ... ORDER BY`.

```go
package orders_data

import (
	"context"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type addressPoolDataStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *addressPoolDataStore) AddMultipleTaskItemsClb(ctx context.Context,
	items []*entities.TaskItem,
) (count uint, list []*entities.TaskItem, err error) {
	if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
		const q = `
			WITH "inserted" AS (
				INSERT INTO "tasks" ("uuid", "entity_uuid",
					"status", 
					"type", "creator_type",
					"created_at", "updated_at")
				VALUES(:uuid, :entity_uuid, 
					:status,
					:type, :creator_type,
					:created_at, :updated_at) 
				ON CONFLICT DO NOTHING
				RETURNING *
			)
			SELECT * FROM "inserted"
			ORDER BY "created_at";`

		bondedSql, args, queryErr := stmt.BindNamed(q, items)
		if queryErr != nil {
		    return s.e.ErrorOnly(queryErr, "unable to create query with bonded names")
		}

		rows, queryErr := stmt.Queryx(bondedSql, args...)
		if queryErr != nil {
		return s.e.ErrorOnly(queryErr, ErrUnableExecuteQuery.Error())
		}

		defer rows.Close()

		itemsList := make([]*entities.TaskItem, len(items))
		itemsCount := uint(0)

		for rows.Next() {
			itemData := &entities.TaskItem{}

			iterErr := rows.StructScan(itemData)
			if iterErr != nil {
		     return s.e.ErrorOnly(iterErr, "unable to scan sql-result on struct")
			}

			itemsList[itemsCount] = itemData
			itemsCount++
		}

		if itemsCount == 0 { 
			// в таком случае возвращаем - count = 0, list = nil, err = nil
			return nil
		}

		count = itemsCount
		list = itemsList

		return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции 
	  // TryWithTransaction уже вернется отформатированная ошибка 
	  return 0, nil, s.e.ErrorNoWrap(err)
	}

	return
}
```