---
title: Написание кода. Инъекция зависимостей
description: Категории и порядок передачи зависимостей в factory-функциях сервис-компонентов
category: App Development
tags: [code-writing, dependency-injection, factory-functions]
---

---
**Цель**: Регламентировать категории, приоритеты и порядок передачи зависимостей при создании сервис-компонентов
**Категория правил**: App Development, code writing
**Краткое содержание:** Нулевой/первый/второй/третий приоритеты, каждый параметр на отдельной строке, nil-проверки, интерфейсы вместо concrete типов
---

# Написание кода. Инъекция зависимостей

## Введение

Сервис-компонент не может работать автономно — ему требуются зависимости разных уровней. Порядок передачи аргументов в factory-функцию **строго детерминирован** и отражает архитектурный приоритет каждой категории. Это упрощает чтение сигнатуры и предотвращает путаницу между разработчиками.

```
Порядок → | ctx | service-deps | infra-deps | logical-deps
            [0]       [1]             [2]           [3]
```

### Правило 1: Контекст — нулевой приоритет

Если factory-функция требует `context.Context`, этот параметр всегда передаётся первым — перед любой другой зависимостью. Стандартно контекст передавать не нужно; он нужен только если функция или её тело выполняет долгие операции, требующие отмены.

```go
// BAD: Контекст после других параметров
func NewAuthService(cfg configService, logger *slog.Logger, ctx context.Context) (*authService, error) { ... }

// GOOD: Контекст на первом месте
func NewAuthService(ctx context.Context, cfg configService, logger *slog.Logger) (*authService, error) { ... }
```

### Правило 2: Служебные (базовые) зависимости — первый приоритет

Эти зависимости обязательны с точки зрения архитектуры приложения. Передаются первыми после контекста:

| Позиция | Зависимость | Тип |
|---------|-------------|-----|
| 1-я | Форматтер ошибок | `errorFormatterService` |
| 2-я | Логгер | `loggerBuilderService` / `*slog.Logger` / `*log.Logger` / `*zap.Logger` |
| 3-я | Сервис конфигурации | `configService` |
| 4-я | Трассировщик | `tracerSvc` (если передается как отдельный сервис-компонент) |

```go
// BAD: Нарушен порядок служебных зависимостей (logger перед errFmtSvc)
func NewWorker(log *slog.Logger, errFmt errorFormatterService, cfg configService) (*worker, error) { ... }

// GOOD: Строго по приоритету
func NewWorker(errFmt errorFormatterService, log loggerBuilderService, cfg configService, tracer tracerSvc) (*worker, error) { ... }
```

### Правило 3: Инфраструктурные зависимости — второй приоритет

Зависимости внутренней инфраструктуры, передаются после служебных. Между категориями — пустая строка для визуального разделения:

| Порядок | Зависимость |
|---------|-------------|
| 1 | Vault (менеджер секретов) |
| 2 | Persistent storage (PostgreSQL, ClickHouse) |
| 3 | Cache storage (Redis, KeyDB) |
| 4 | Брокер очередей (NATS, Kafka, RabbitMQ) |
| 5 | Клиенты внутренних API |

```go
// BAD: Нет пустой строки между категориями; инфраструктурные вперемешку с базовыми
func NewWorker(errFmtSvc errorFormatterService, pgxPool *pgxpool.Pool, loggerSvc loggerBuilderService, redisConn *redis.Connection) (*worker, error) { ... }

// GOOD: Пустая строка отделяет категории; инфраструктурные идут подряд
func NewWorker(
	errFmtSvc errorFormatterService,
	loggerSvc loggerBuilderService,
	// Пустая строка — далее инфраструктурные зависимости
	vaultClient vault.Client,
	pgxPool *pgxpool.Pool,
	redisConn *redis.Connection,
	natsConn *nats.Conn,
) (*worker, error) { ... }
```

### Правило 4: Логические зависимости — третий приоритет

Другие сервис-компоненты доменной области, необходимые данной функции как вспомогательные сущности. Внешние API, консюмеры/продюсеры бизнес-очередей. Последние в сигнатуре.

```go
// BAD: Логические зависимости в середине сигнатуры (до инфраструктурных)
func NewWorker(pgConn postgres.Conn, domainA ServiceA, cacheSvc cache.Service, domainB ServiceB) (*worker, error) { ... }

// GOOD: Логические зависимости — самые последние
func NewWorker(
	errFmtSvc errorFormatterService,
	loggerSvc loggerBuilderService,
	// Пустая строка — инфраструктурные
	pgConn postgres.Conn,
	cacheSvc cache.Service,
	// Пустая строка — логические (доменные) зависимости
	domainA ServiceA,
	domainB ServiceB,
) (*worker, error) { ... }
```

### Правило 5: Каждый параметр на отдельной строке

Если у функции более одного параметра — каждый параметр пишется на новой строке. Многострочная форма применяется всегда, даже если формально возможен single-line (один параметр). Это упрощает diff'ы при добавлении/удалении аргументов.

```go
// BAD: Параметры в одну строку
func NewWorker(errFmtSvc errorFormatterService, loggerSvc loggerBuilderService, cfg configService) (*worker, error) { ... }

// GOOD: Каждый параметр на отдельной строке
func NewWorker(
	errFmtSvc errorFormatterService,
	loggerSvc loggerBuilderService,
	cfg configService,
) (*worker, error) { ... }
```

### Правило 6: Интерфейсы вместо concrete типов

Передаваемые зависимости должны имплементировать интерфейс, определённый на уровне пакета. Запрещено принимать конкретные реализации напрямую в сигнатуре — это нарушает принцип инверсии зависимостей (DIP).

```go
// BAD: Concrete тип в сигнатуре
func NewWorker(db *sqlx.DB, redisClient *redis.Client) (*worker, error) { ... }

// GOOD: Интерфейс
func NewWorker(repo RepoStore, cache Client) (*worker, error) { ... }
```

### Правило 7: Обязательная проверка на nil

Каждая зависимость проверяется на `nil` внутри factory-функции до использования. Задача фабрики — не пропустить пустую зависимость и исключить панику (go panic) при runtime. Используется error-formatter сервиса, а не `fmt.Errorf`.

```go
// BAD: Отсутствие проверки на nil
func NewWorker(cfg configService, db postgres.Conn) (*worker, error) {
	return &worker{cfg: cfg, db: db}, nil
}

// GOOD: Каждая зависимость отдельно, пустая строка перед итоговым return
func NewWorker(errFmtSvc errorFormatterService, cfg configService, db postgres.Conn) (*worker, error) {
	if errFmtSvc == nil {
		return nil, errFmtSvc.Error(ErrMissingSvcDependencies, "missing error-formatter service component")
	}

	if cfg == nil {
		return nil, errFmtSvc.Error(ErrMissingSvcDependencies, "missing config service component")
	}

	if db == nil {
		return nil, errFmtSvc.Error(ErrMissingSvcDependencies, "missing postgres connection")
	}

	return &worker{
		e:  errFmtSvc,
		cfg: cfg,
		db:  db,
	}, nil
}
```
