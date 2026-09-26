# Рекомендации для AI-агентов при работе с CSG

## Навигация по документации (`pkg/csg/docs/`)

- [app-development](./docs/app-development/) — Архитектура Go-приложения, обработка ошибок, конфигурация, работа с БД, gRPC, HTTP, NATS
- [setup](./docs/setup/) — Настройка окружения разработчика и базовые требования к среде разработки
- [git-flow](./docs/git-flow/) — Правила ветвления, оформления PR'ов и работы с Git в проекте
- [toolkit](./docs/toolkit/) — Обзор утилит и библиотек проекта: копиррайт, линтеры, миграции
- [review](./docs/review/) — Процедуры code review, включая GitHub Code Review
- [repository-layouts](./docs/repository-layouts/) — Структура репозитория: код, деплоймент, миграции, конфиги, CI/CD, линтеры, лицензии
- [processes](./docs/processes/) — Процессы разработки, периметры ответственности команд и зон
- [diagrams](./docs/diagrams/) — Диаграммы интеграций, включая интеграционные ветки (Integration Branch)
- [examples](./docs/examples/) — Примеры файлов проекта: LICENSE, .env, другие boilerplate-шаблоны

---
**Цель**: Инструкция по соблюдению Code Style Guide (CSG) при генерации кода
---

## Кратко

Этот файл описывает структуру и содержит выжимку основных правил Code Style Guide. Полный текст CSG доступен внутри пакета `pkg/csg` через встроенную файловую систему (Go Embed), в директории `./docs/`. Все агенты, участвующие в разработке на Go в рамках проекта `crypto-bundle`, обязаны соблюдать требования [CSG](https://github.com/crypto-bundle/bc-wallet-common-csg-go).

CSG распространяется по нескольким каналам:
- Как отдельный репозиторий с документацией внутри `./docs/` (внутри пакета `pkg/csg/`)
- Как go-модуль, содержащий в себе документацию для конечного потребителя — разработчика, который импортирует её как зависимость в проект

## Использование в проекте

Для подключения CSG как зависимости к своему проекту выполните:

```bash
go get github.com/crypto-bundle/bc-wallet-common-csg-go/pkg/csg@latest
```

Затем добавьте blank-import в `main.go`, чтобы убедиться, что embedded docs были инициализированы:

```go
package main

import (
	_ "github.com/crypto-bundle/bc-wallet-common-csg-go/pkg/csg"
)

func main() {
	// ...
}
```

После этого документация будет доступна внутри скомпилированного бинарного файла(для developement сборки) через встроенную файловую систему (Go Embed). Для программной работы используйте публичный API пакета: `ListDocFiles`, `FileContent`, `WalkDoc`.

### Embed для dev-сборки (`//go:embed` + build tag `development`)

Файлы документации из каталога `docs/` подключаются через `//go:embed` только при разработке приложения. В продакшен-сборке они не включаются в бинарник — для их чтения используется прямая работа с ФС проекта или пакетный API. Для разделения сборок применяется build-тег `development`:

```go
//go:build development

package main

import (
    "embed"
)

//go:embed docs
var embeddedDocs embed.FS
```

При сборке без тега (production):

```bash
go build -tags production .
```

Embed-инструкции не будут выполнены, а переменная `embeddedDocs` останется пустой. В этом случае приложение опирается на публичный API пакета `csg`: `ListDocFiles`, `FileContent`, `WalkDoc`.

### Запуск csgRunnerService только для dev-сборки

Сервис-компонент `csgRunnerService` (и его реализация `csgService`) предназначен для разработки и отладки. Он **не должен запускаться** в production-сборках. При инициализации приложения проверяйте окружение через `baseConfigService.IsDev()` / `baseConfigService.IsProd()`:

```go
package main

import (
    "context"

    "github.com/crypto-bundle/bc-wallet-common-csg-go/pkg/csg"
    csgTypes "github.com/crypto-bundle/bc-wallet-common-csg-go/pkg/csg/types"
)

func main() {
    // ... получение зависимостей loggerSvc, configSvc, errorFmtSvc ...

    csgSvc := csg.New(configSvc, errorFmtSvc, loggerSvc)

    if configSvc.IsDev() {
        // Запускаем только в dev-режиме
        go func() {
            if err := csgSvc.Run(context.Background()); err != nil {
                // логирование ошибки
            }
        }()
    }
}
```

При вызове `Run(ctx)` в dev-режиме происходит вывод краткой справки по доступным документам CSG. Это полезно для навигации разработчика во время локальной разработки.

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
