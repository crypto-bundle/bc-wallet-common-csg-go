# Рекомендации для AI-агентов при работе с CSG

---
**Цель**: Инструкция по соблюдению Code Style Guide (CSG) при генерации кода
---

## Кратко

Все агенты, участвующие в разработке на Go в рамках проекта `crypto-bundle`, обязаны соблюдать требования [CSG](https://github.com/crypto-bundle/bc-wallet-common-csg-go). CSG встроен в бинарник через `go:embed` и доступен через пакет `pkg/csg`.

## Основные принципы

- Не создавать новые библиотеки без обоснования и separate git-репозитория
- Соблюдать имена модулей согласно рекомендациям именования в CSG
- Использовать только библиотечные компоненты из ToolKit
- Следовать SemVer для версионирования библиотек

## Архитектура пакета

- Структура пакетов: один сервис-компонент = одна или несколько связанных таблиц/доменов
- main.go — минимальная точка входа, собирает конфиг через embedding
- File structure: следовать правилам из [docs/app-development/architecture/file_structure.md](./docs/app-development/architecture/file_structure.md)

## Ошибки

- Обёртывать все ошибки из импортируемых пакетов через `bc-wallet-common-lib-errors`
- Не допускать shadowed error
- Не форматировать ошибку дважды
- Использовать `ScopedErrorFormatter` для единого namespace пакета
- Не возвращать `sql.ErrNoRows` за пределы data-layer пакета — использовать `EmptyOrError`

## Конфигурация

- Использовать `bc-wallet-common-lib-config`
- Все Go-структуры конфига собирать в одну через embedding
- Получение данных только через `Get` методы, прямое обращение к полям недопустимо
- Методы подготовки: `Prepare()`, `PrepareWith(...interface{})`

## PostgreSQL

- Один сервис-компонент = одна(и) таблица(ы)
- SQL-запросы выполнять внутри `TryWithTransaction` / `MustWithTransaction`
- Одна функция ресивер = один SQL-запрос внутри транзакции
- Не использовать SQL-функции времени (`now()`, `current_date` и т.д.)
- Не использовать `DEFAULT`, передавать чёткие значения
- Не использовать `fmt.Sprintf` для подстановки параметров — только `sqlx`

## Process & Perimeters

- Соблюдать периметры ответственности (см. [docs/processes/perimeter_rules.md](./docs/processes/perimeter_rules.md))
- Follow Git-Flow для всех веток и PR

## Что проверять перед коммитом

1. [ ] Код следует правилам архитектуры пакетов
2. [ ] Ошибки обернуты через lib-errors
3. [ ] Конфиг читается через Get-методы
4. [ ] SQL-запросы оформлены в TryWithTransaction
5. [ ] Нет direct field access к конфигу
6. [ ] Нет sql.ErrNoRows на уровне выше data layer
