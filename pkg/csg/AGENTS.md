# Рекомендации для AI-агентов при работе с CSG

## Навигация по документации (`docs/`)

- [app-development](./docs/app-development/) — Общие правила Go
  - [overview](./docs/app-development/overview.md) — Обзор архитектуры приложения
- [setup](./docs/setup/)
  - [git_setup](./docs/setup/git_setup.md) — Настройка Git
- [git-flow](./docs/git-flow/)
  - [git_flow](./docs/git-flow/git_flow.md) — Правила работы с Git
- [toolkit](./docs/toolkit/)
  - [overview](./docs/toolkit/overview.md) — Обзор ToolKit
  - [copywrite](./docs/toolkit/copywrite.md) — Copywrite
  - [golangci-lint](./docs/toolkit/golangci-lint.md) — GolangCI Lint
  - [migrator](./docs/toolkit/migrator.md) — Migrator
- [review](./docs/review/)
  - [github_code_review](./docs/review/github_code_review.md) — GitHub Code Review
- [repository-layouts](./docs/repository-layouts/)
  - [overview](./docs/repository-layouts/overview.md) — Обзор структурных правил
  - [app/code](./docs/repository-layouts/app/code.md) — Код приложения
  - [app/deployment](./docs/repository-layouts/app/deployment.md) — Деплоймент
  - [migrations/index](./docs/repository-layouts/migrations/index.md) — Миграции БД
  - [configuration-files](./docs/repository-layouts/configuration-files.md) — Конфигурационные файлы
  - [linters](./docs/repository-layouts/linters.md) — Линтеры
  - [builders](./docs/repository-layouts/builders.md) — Билд-скрипты
  - [cicd](./docs/repository-layouts/cicd.md) — CI/CD
  - [license](./docs/repository-layouts/license.md) — Лицензирование
  - [documentation](./docs/repository-layouts/documentation.md) — Документация
- [processes](./docs/processes/)
  - [perimeter_rules](./docs/processes/perimeter_rules.md) — Периметры ответственности
- [diagrams](./docs/diagrams/)
  - [integration-branch.puml](./docs/diagrams/integration-branch.puml) — Integration Branch
- [examples](./docs/examples/)
  - [license](./docs/examples/license/) — License Example
    - [README](./docs/examples/license/README.md)
    - [cmd.go](./docs/examples/license/cmd.go)
    - [LICENSE.txt](./docs/examples/license/LICENSE.txt)
  - [env](./docs/examples/env/) — Env Example
    - [env-hdwallet-controller-manager-api-example.env](./docs/examples/env/env-hdwallet-controller-manager-api-example.env)

---
**Цель**: Инструкция по соблюдению Code Style Guide (CSG) при генерации кода
---

## Кратко

Этот файл описывает структуру и содержит выжимку основных правил Code Style Guide. Полный текст CSG доступен внутри пакета `pkg/csg` через встроенную файловую систему (Go Embed), в директории `./docs/`. Все агенты, участвующие в разработке на Go в рамках проекта `crypto-bundle`, обязаны соблюдать требования [CSG](https://github.com/crypto-bundle/bc-wallet-common-csg-go).

CSG распространяется по нескольким каналам:
- Как отдельный репозиторий с документацией внутри `./docs/` (внутри пакета `pkg/csg/`)
- Как go-модуль содержащий в себе документацию для конечного потребителя - разработчика который импортирует ее ка зависимость в проект

## Основные принципы

- Не создавать новые библиотеки без обоснования и separate git-репозитория
- Соблюдать имена модулей согласно рекомендациям именования в CSG
- Использовать только библиотечные компоненты из ToolKit
- Следовать SemVer для версионирования библиотек

## Архитектура пакета

- Структура пакетов: один сервис-компонент = одна или несколько связанных таблиц/доменов
- main.go — минимальная точка входа, собирает конфиг через embedding
- File structure: следовать правилам из [`docs/app-development/architecture/file_structure.md`](./docs/app-development/architecture/file_structure.md)

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

## ToolKit Libraries

Использовать только библиотеки из проекта ToolKit для всех стандартных задач. Для подробностей искать инструкции в `doc.go` пакета библиотеки или в `AGENTS.md` внутри репозитория библиотеки (если есть).

| Library | Цель |
| --- | --- |
| `bc-wallet-common-lib-config` | Менеджер конфигурации (ENV, JSON, secrets engine) |
| `bc-wallet-common-lib-tinyerrors` | Минимальная реализация форматтера ошибок |
| `bc-wallet-common-lib-errors` | Расширенное форматирование ошибок (`ScopedErrorFormatter`) |
| `bc-wallet-common-lib-logger` | Билдеры для логгеров (`zap.Logger`, `slog.Logger`, `log.Logger`) |
| `bc-wallet-common-lib-profiler` | Обёртка над профилировщиком, кастомный HTTP-сервер |
| `bc-wallet-common-lib-healthcheck` | Универсальная проверка здоровья приложения (HTTP-сервер) |
| `bc-wallet-common-lib-metrics` | Телеметрия, HTTP-сервер, пулы данных для Prometheus |
| `bc-wallet-common-lib-jwt` | Создания и валидации JWT-токенов |
| `bc-wallet-common-lib-tracer` | Обёртка над OpenTracing, подключение к Jaeger |
| `bc-wallet-common-lib-grpc` | Создание экземпляров gRPC-сервера / gRPC-клиента |
| `bc-wallet-common-lib-vault` | Работа с HashiCorp Vault, драйверы, обновление токена в runtime |
| `bc-wallet-common-lib-postgres` | Конфиг подключения к PostgreSQL, транзакционный стейтмент |
| `bc-wallet-common-lib-nats-queue` | Подключение к Nats, консюмеры и продюсеры |
| `bc-wallet-common-lib-redis` | Подключение к Redis / DragonFlyDb |
| `bc-wallet-common-migrator` | Утилита-обёртка над goose для миграций БД |

## PostgreSQL

- Один сервис-компонент = одна(и) таблица(ы)
- SQL-запросы выполнять внутри `TryWithTransaction` / `MustWithTransaction`
- Одна функция ресивер = один SQL-запрос внутри транзакции
- Не использовать SQL-функции времени (`now()`, `current_date` и т.д.)
- Не использовать `DEFAULT`, передавать чёткие значения
- Не использовать `fmt.Sprintf` для подстановки параметров — только `sqlx`

## Process & Perimeters

- Соблюдать периметры ответственности (см. [`docs/processes/perimeter_rules.md`](./docs/processes/perimeter_rules.md))
- Follow Git-Flow для всех веток и PR

## Что проверять перед коммитом

1. [ ] Код следует правилам архитектуры пакетов
2. [ ] Ошибки обернуты через lib-errors
3. [ ] Конфиг читается через Get-методы
4. [ ] SQL-запросы оформлены в TryWithTransaction
5. [ ] Нет direct field access к конфигу
6. [ ] Нет sql.ErrNoRows на уровне выше data layer
