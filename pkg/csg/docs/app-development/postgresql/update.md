# Слой данных. Работа с PostgreSQL. Примеры обновления данных

---
**Цель**: Привести примеры UPDATE операций в PostgreSQL
**Категория правил**: App Development, SQL
**Краткое содержание:** Примеры обновления записей — single, batch, conditional update
---
## Введение

Данный документ содержить в себе примеры использования базового набора инструментов для создания
сервис-компонентов реализующих механизм **обновления записей** в базе данных PostgreSQL. Иными словами
примеры работы с **UPDATE** SQL-запросами.

Ниже приведены примеры для:
- Обновление **одной** записи без возвратного значения- [Пример использования](#обновление-одной-записи-без-выборки-результата)
- Обновление **одной** записи с возвратным значением- [Пример использования](#обновление-одной-записи-с-выборкой-результата)
- Обновление **множества записей с возвратными значениями**- [Пример использования](#обновление-множества-записей-с-выборкой-результата)
- Обновление **множества записей c выборкой отсортированных данных** - [Пример использования](#обновление-множества-записей-с-выборкой-отсортированных-данных)

## Примеры использования

#### Обновление одной записи без выборки результата

Данный способ обновления данных самый простой в своей реализации, 
здесь просто формируется запрос на обновление данных, и не ожидается
каких либо возвратных значений.

Функция представленная ниже разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на обновление данных.
1. Выполнение SQL-запроса через фукцию `Exec` с передачей необходимы хзначений.
1. Получение кол-ва измененных строк.
1. Проверка что измененных строк было больше 0, иначе ошибка.

Для данного примера важно заметить следующее:
- Используется концепция программирование - _замыкание_ - для передачи результата из анонимной-функции.
- Используются именованные возвратные параметры, и результат выполнения SQL-запроса
  присваивается напрямую в именованный возвратный параметр.
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

type walletSessionStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *walletSessionStore) UpdateWalletSessionStatusBySessionUUID(ctx context.Context,
	sessionUUID string,
	newStatus types.MnemonicWalletSessionStatus,
) error {
	if err := s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
		const q = `
      UPDATE "mnemonic_wallet_sessions" 
			SET "status" = $1,
				"updated_at" = $2
			WHERE "uuid" = $3`

	  qResult, clbErr := stmt.Exec(q,
			newStatus, time.Now(), sessionUUID)
	  if clbErr != nil {
		  return s.e.ErrorOnly(clbErr, "unable to execute query")
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

#### Обновление одной записи с выборкой результата

Данный тип обновления данных является наиболее распространенным и используется для обновления свойств
одной конкретной модели данных.

Функция представленная ниже разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на обновление данных.
1. Выполнение SQL-запроса через фукцию `sqlx.QueryRowx` с передачей необходимы хзначений.
1. Попытка отсканировать данные из результата SQL-запроса через функцию `StructScan`.
1. Если успешен предыдущий шаг - присваиваем результат выполнения функции через механизм замыкания.

Для данного примера важно заметить следующее:
- Используется концепция программирование - _замыкание_ - для передачи результата из анонимной-функции.
- Использование функции `StructScan` обусловлено тем, что наш SQL-запрос содержит в себе директиву
  `RETURNING *`, а значит имеет возвращаемое значение.
- Для `StructScan` нет проверки через хэлпер `commonPostgres.EmptyOrError`, так как
  ошибка `sql.ErrNoRows` не возможна при UPDATE запросе.
- Используются именованные возвратные параметры, и результат выполнения SQL-запроса 
  присваивается напрямую в именованный возвратный параметр. 
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.

```go
package orders_data

import (
	"context"
	"time"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type walletSessionStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *walletSessionStore) UpdateWalletSessionStatusBySessionUUID(ctx context.Context,
	sessionUUID string,
	newStatus types.MnemonicWalletSessionStatus,
) (result *entities.MnemonicWalletSession, err error) {
	if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
		const q = `
      UPDATE "mnemonic_wallet_sessions" 
			SET "status" = $1,
				"updated_at" = $2
			WHERE "uuid" = $3
			RETURNING *` 
		
		row := stmt.QueryRowx(q,
			newStatus, time.Now(), sessionUUID)

		sessionItem := &entities.MnemonicWalletSession{}
		clbErr := row.StructScan(sessionItem)
		if clbErr != nil {
      // EmptyOrError НЕ используется потому что при вызове Scan для UPDATE ... RETURNING
      // запроса sql.ErrNoRows не вернется. А значит EmptyOrError испльзовать смысла нет.
      return s.e.ErrorOnly(clbErr, "unable to scan result")
		}

		result = sessionItem

		return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции из 
	  // TryWithTransaction уже вернется отформатированная ошибка 
	  return nil, s.e.ErrorNoWrap(err)
	}

	return
}
```

#### Обновление множества записей с выборкой результата

Данный тип обновления данных предназначен для обновления множества записей в базе данных,
и получения списка данных которые были затронуты запросом.

По сути данный способ во многом схож с механизмом вставки новых значений, 
только целевой запрос SQL-запрос UPDATE. Так же используется ключевое слово RETURNING 
для обозначения того какие данные хотим получить после запроса.

Ниже приведенный код разделен на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса.
1. Целевой SQL-запрос на обновление данных.
1. Биндинг динамического списка парметров через функцию `sqlx.In`.
1. Ре-биндинг значений значений для ранее заданных параметров через функцию `Rebind`.
1. Выполнение SQL-запроса через фукцию `sqlx.Queryx`.
1. Подготовка слайса к заполнению данными.
1. Цикл по полученным сырым данных из базы данных с использованием функции `rows.Next()`
1. Попытка на каждой итерации цикла отсканировать строку данных через функцию `StructScan`.
1. Если цикл завершился успешно, то используя механизм _замыкания_, присваиваем
   значения возвращаемым именнованным параметрам.

```go
package orders_data

import (
	"context"

	commonPostgres "github.com/crypto-bundle/bc-wallet-common-lib-postgres/pkg/postgres"

	"github.com/jmoiron/sqlx"
)

type walletSessionStore struct {
	dbConn *commonPostgres.Connection
	e errorFormatterService
}

func (s *walletSessionStore) UpdateMultipleWalletsStatus(ctx context.Context,
	walletUUIDs []string,
	newStatus types.MnemonicWalletStatus,
) (count uint, list []*entities.MnemonicWallet, err error) {
	if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
	  const q = `
      UPDATE "mnemonic_wallets"
      SET "status" = ?
      WHERE 
        "uuid" IN (?)
      RETURNING *`
		
		query, args, clbErr := sqlx.In(q, 
			newStatus, walletUUIDs)
	  if clbErr != nil {
		  return s.e.ErrorOnly(clbErr, "unable to prepare sql-query with IN condition")
	  }

		bonded := stmt.Rebind(query)

		returnedRows, clbErr := stmt.Queryx(bonded, args...)
		if clbErr != nil {
      return s.e.ErrorOnly(clbErr, "unable to make sql-query")
		}
		
		defer returnedRows.Close()

		walletsList := make([]*entities.MnemonicWallet, 0)

		var counter uint
		
		for returnedRows.Next() {
			walletItem := &entities.MnemonicWallet{}

			loopErr := returnedRows.StructScan(walletItem)
			if loopErr != nil {
        // EmptyOrError НЕ используется потому что при вызове Scan для UPDATE ... RETURNING
        // запроса sql.ErrNoRows не вернется. А значит EmptyOrError испльзовать смысла нет.
				return s.e.ErrorOnly(loopErr, "unable to scan row from sql query result")
			}

			walletsList = append(walletsList, walletItem)

      counter++
		}

		list = walletsList
		count = counter

		return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции из 
	  // TryWithTransaction уже вернется отформатированная ошибка 
	  return 0, nil, s.e.ErrorNoWrap(err)
	}

	return
}

```

#### Обновление множества записей с выборкой отсортированных данных
Данный пример добавления данных описывае способ сохранения множества однотипных данных в
определенную таблицу используя один SQL-запрос. В рамках данного примере расмотрена ситуация
в которой после вставки возвращаются вставленные записях отсортированные по определенному полю.

Функция добавления записи разделена на несколько состовляющих:
1. Попытка определить транзакционный контекст запроса через `TryWithTransaction`.
1. Целевой SQL-запрос на добавление множества данных в БД с оберткой вида `WITH "updated" AS ... SELECT`.
1. Биндинг динамического списка парметров через функцию `sqlx.In`.
1. Ре-биндинг значений значений для ранее заданных параметров через функцию `Rebind`.
1. Выполнение SQL-запроса через фукцию `Queryx` с передачей значений для вставки.
1. Подготовка слайса к заполнению данными.
1. Цикл по полученным сырым данных из базы данных с использованием функции `rows.Next()`
1. Попытка на каждой итерации цикла отсканировать строку данных через функцию `StructScan`.
1. Если цикл завершился успешно, то используя механизм _замыкания_, присваиваем
   значения возвращаемым именнованным параметрам.

Для данного примера важно отметить следующее:
- Используется концепция программирования - _замыкание_ - для передачи результата из анонимной-функции.
  При этом результат выполнения в случае успеха сразу присваивается в именованные возвратные параметры.
- SQL-запрос написанный в примере может использоваться как внутри транзакционного стейтмента,
  так и без. Для этого используется функция `TryWithTransaction`.
- Запрос выполняется c выборкой вставленных данных, а значит используется функция `Queryx` из библиотеки `sqlx`.
- Обновленные данные возвращаеются как **отсортированный список**, для сортировки используется SQL-код вида
  `WITH "updated" AS ... SELECT ... ORDER BY`.

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

func (s *walletSessionStore) UpdateMultipleWalletsStatus(ctx context.Context,
	walletUUIDs []string,
	newStatus types.MnemonicWalletStatus,
) (count uint, list []*entities.MnemonicWallet, err error) {
	if err = s.dbConn.TryWithTransaction(ctx, func(stmt sqlx.Ext) error {
		const q = `
			WITH "updated" AS (
        UPDATE "mnemonic_wallets"
            SET "status" = ?
        WHERE 
          "uuid" IN (?)
        RETURNING *
			)
			SELECT * FROM "updated"
			ORDER BY "created_at";`

	  query, args, clbErr := sqlx.In(q,
		  newStatus, walletUUIDs)
	  if clbErr != nil {
		  return s.e.ErrorOnly(clbErr, "unable to prepare sql-query with IN condition")
	  }

	  bonded := stmt.Rebind(query)

	  returnedRows, clbErr := stmt.Queryx(bonded, args...)
	  if clbErr != nil {
		  return s.e.ErrorOnly(clbErr, "unable to make sql-query")
	  }

	  defer returnedRows.Close()

	  walletsList := make([]*entities.MnemonicWallet, 0)

	  var counter uint

	  for returnedRows.Next() {
		  walletItem := &entities.MnemonicWallet{}

		  loopErr := returnedRows.StructScan(walletItem)
		  if loopErr != nil {
        // EmptyOrError НЕ используется потому что при вызове Scan для UPDATE ... RETURNING
        // запроса sql.ErrNoRows не вернется. А значит использовать EmptyOrError смысла нет.
			  return s.e.ErrorOnly(loopErr, "unable to scan row from sql query result")
		  }

		  walletsList = append(walletsList, walletItem)

		  counter++
	  }

	  list = walletsList
	  count = counter

	  return nil
	}); err != nil {
	  // ErrorNoWrap потому что из анонимной функции 
	  // TryWithTransaction уже вернется отформатированная ошибка 
	  return 0, nil, s.e.ErrorNoWrap(err)
	}

	return
}
```