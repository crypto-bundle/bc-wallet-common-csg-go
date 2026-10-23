---
title: gRPC. Обработка запросов. Правила для хэндлеров
description: Структура директорий хэндлеров, создание и подключение хэндлер-сервисов, обработка запросов
category: App Development
tags: [grpc, handlers, request-handling, package-layout, bc-wallet-common-lib-grpc]
---

---
**Цель**: Регламентировать создание и подключение gRPC-хэндлеров, организацию их директории и файлов, обработку входящих запросов
**Категория правил**: App Development, gRPC, request handling
**Краткое содержание:** Layout пакета хэндлеров (файл на метод, общий common.go), хэндлер как сервис-компонент с DI, регистрация на grpc.Server, валидация/маппинг ошибок/DTO-границы в методе-обработчике
---

# gRPC. Обработка запросов. Правила для хэндлеров

## Введение

Хэндлер-слой — граница между transport (gRPC) и бизнес-логикой сервиса. Один
proto-сервис (`<package>.<Service>Handlers`) реализуется одной структур-
компонентом из выделенного пакета хэндлеров. Документ регламентирует layout
этого пакета, правила создания и подключения хэндлеров и обработку отдельного
запроса.

Связанные документы: [gRPC-сервер](./server.md) (сборка и запуск сервера),
[health check сервера](./server_healthcheck.md),
[Инъекция зависимостей](../code-writing/dependency_injection_rules.md),
[Обработка ошибок](../errors/handling.md) и
[коды ошибок](../errors/formatting_codes.md) (маппинг в `status.Error`).

## Структура директории и файлов хэндлеров

### Правило 1: Выделенный пакет хэндлеров на grpc-транспорт сервиса

Хэндлер-слой живёт в `internal/grpc/handlers` пакета `handlers`: `internal`
делает слой недоступным импортом из чужих модулей, а сгенерированный proto —
наоборот, публикуется в `pkg/grpc/<api>/proto` для клиентов сервиса. Хэндлеры
в `main`, в пакете proto или вперемешку с бизнес-пакетами запрещены.

```go
// BAD: хэндлер объявлен в main-пакете — слой нельзя ни переиспользовать в тестах,
// ни защитить от импорта извне; grpc-сервер собирается рядом с flag-парсингом.
func startGRPCServer() {
    depositpb.RegisterDepositHandlersServer(srv, &depositHandler{})
}

// GOOD: пакет internal/grpc/handlers независим от сборки приложения;
// main/app только собирает и регистрирует его.
type depositHandlers struct {
    depositSvc depositService
    e          errorFormatterService
}
```

### Правило 2: Один файл — один proto-метод, общее — в common.go

В `common.go` лежат: структура хэндлер-сервиса, factory-функция, интерфейсы
зависимостей (consumer-side). Каждый метод-обработчик — отдельный файл
`handler_<метод>.go`; конвертеры домен↔proto — `marshaller_<метод>.go`.
Сваливать все методы в один файл или дробить один метод по нескольким файлам
нельзя.

```go
// BAD: один файл handlers.go на 30 методов; структура, фабрика, валидация
// и маппинг — вперемешку; diff конфликтует на каждом change-request.

// GOOD: common.go — только структура, зависимости и фабрика:
type depositHandlers struct {
    depositSvc depositService
    tariffCli  tariffService
    e          errorFormatterService
}

func NewDepositHandlersService(
    depositSvc depositService,
    tariffCli tariffService,
    errFmtSvc errorFormatterService,
) *depositHandlers {
    return &depositHandlers{
        depositSvc: depositSvc,
        tariffCli:  tariffCli,
        e:          errFmtSvc,
    }
}
```

Файл `handler_get_deposit.go` содержит ровно один метод-обработчик
`GetDeposit` и его form-валидацию; файл `marshaller_get_deposit.go` —
конвертеры `GetDeposit`-сообщений.

### Правило 3: Именование пакета и файлов по proto-сервису

Имя файла метода — `handler_` + snake_case proto-метода; имя метода-обработчика
— ровно имя proto-метода (регистр PascalCase). Опечатки в префиксе файла
(`hander_`, `hadnler_`) и расхождение имени метода с контрактом — запрещены:
по имени файла должен находиться proto-метод без чтения тела.

```go
// BAD: файл hander_gas_limit.go (опечатка) и метод, не совпадающий с контрактом.
func (h *feeHandlers) GasLimit(ctx context.Context, req *feepb.GasLimitRequest) (*feepb.GasLimitResponse, error) {
    return nil, nil
}

// GOOD: файл handler_get_gas_limit.go, имя метода = имя proto-метода GetGasLimit.
func (h *feeHandlers) GetGasLimit(ctx context.Context, req *feepb.GetGasLimitRequest) (*feepb.GetGasLimitResponse, error) {
    return nil, nil
}
```

## Создание хэндлер-сервис-компонента

### Правило 1: Одна структура реализует весь proto-сервис

Один proto-сервис = один сервис-компонент хэндлеров: все RPC-методы сервиса —
методы одной структуры. Наследование `Unimplemented<X>Server` обязательно —
новые методы контракта не «сломает» молча при обновлении proto. Хэндлер-структура
на каждый метод (исторический паттерн `AddNewWalletHandler`/`Make...Handler`) —
устаревание, новый код так не пишется.

```go
// BAD: отдельная структура и фабрика на каждый метод — зависимости
// дублируются N раз, контракт сервиса «размазан» по пакету.
type getDepositHandler struct {
    depositSvc depositService
}

type listDepositsHandler struct {
    depositSvc depositService
}

// GOOD: одна структура на весь сервис, Unimplemented для совместимости.
type depositHandlers struct {
    depositpb.UnimplementedDepositHandlersServer

    depositSvc depositService
    e          errorFormatterService
}
```

### Правило 2: Factory-функция New(...) и внедрение бизнес-зависимостей

Хэндлер-сервис создаётся factory-функцией `New<ProtoService>HandlersService`
(см. [Названия factory-функций](../code-writing/factory_function_naming.md)) с
передачей зависимостей как consumer-side интерфейсов, объявленных в `common.go`
(см. [Инъекция зависимостей](../code-writing/dependency_injection_rules.md)):
бизнес-сервисы, grpc-клиенты других сервисов (`*LazyConnection`), error
formatter, logger. Конкретные реализации (репозитории, БД) хэндлерам неизвестны.

```go
// BAD: хэндлер сам строит зависимости — привязка к репозиториям и конфигу.
func NewDepositHandlersService(cfg *config.Config) *depositHandlers {
    db := sqlx.MustConnect("pgx", cfg.GetDSN())

    return &depositHandlers{
        depositSvc: postgres.NewDepositRepository(db),
    }
}

// GOOD: зависимости принимаются как интерфейсы, сборка — на этапе main/app.
func NewDepositHandlersService(
    depositSvc depositService,
    errFmtSvc errorFormatterService,
) *depositHandlers {
    return &depositHandlers{
        depositSvc: depositSvc,
        e:          errFmtSvc,
    }
}
```

### Правило 3: Тело метода — оркестрация, а не бизнес-логика

Метод-обработчик выполняет цепочку: валидация входа → вызов бизнес-сервиса →
упаковка ответа. SQL, криптография, вычисления комиссий и состояния в этом слое
запрещены — они принадлежат бизнес-пакетам (`internal/<domain>`). Метод длиннее
~30 строк — сигнал, что в него легла бизнес-логика.

```go
// BAD: расчёт комиссии и запись в БД внутри обработчика.
func (h *feeHandlers) GetGasPrice(ctx context.Context, req *feepb.GetGasPriceRequest) (*feepb.GasPriceResponse, error) {
    row := h.db.QueryRowContext(ctx, "SELECT base, priority FROM gas_prices WHERE chain = ?", req.Chain)

    var base, priority big.Int
    // ... парсинг, арифметика, округления — всё здесь

    return &feepb.GasPriceResponse{Base: base.String()}, nil
}

// GOOD: обработчик только оркестрирует; расчёт — в feeService.
func (h *feeHandlers) GetGasPrice(ctx context.Context, req *feepb.GetGasPriceRequest) (*feepb.GasPriceResponse, error) {
    price, err := h.feeSvc.GasPrice(ctx, req.Chain)
    if err != nil {
        return nil, h.grpcError(err)
    }

    return &feepb.GasPriceResponse{
        Base:     price.Base.String(),
        Priority: price.Priority.String(),
    }, nil
}
```

## Подключение хэндлеров к серверу

### Правило 1: Регистрация Register<...>Server на этапе сборки приложения

Хэндлер-сервис строится factory-функцией в фазе сборки (main/app) и
регистрируется `depositpb.RegisterDepositHandlersServer(srv, handlers)` на
grpc-сервере, собранном по [server.md](./server.md). Порядок: зависимости →
хэндлеры → сервер → регистрация. Регистрация внутри `init()` пакета хэндлеров
или в конструкторе сервера запрещена.

```go
// BAD: init-регистрация — порядок инициализации неуправляем, тесты не собирают
// сервер без побочных эффектов.
func init() {
    depositpb.RegisterDepositHandlersServer(globalServer, NewDepositHandlersService(globalDeps))
}

// GOOD: сборка в app-слое.
func newApplication(cfg *appConfig) (*application, error) {
    handlers := NewDepositHandlersService(depositSvc, errFmtSvc)

    srv := grpc.NewServer(serverOptions()...)
    depositpb.RegisterDepositHandlersServer(srv, handlers)

    return &application{grpcServer: srv}, nil
}
```

### Правило 2: Пакет хэндлеров не знает про net и жизненный цикл grpc-сервера

Пакет `handlers` импортирует только сгенерированный proto, `codes`/`status` и
бизнес-интерфейсы. `net.Listen`, `grpc.Server`, `Serve`/`GracefulStop` —
компетенция слоя сборки (см. [server.md](./server.md), жизненный цикл).
Хэндлер не останавливает сервер, не заводит горутин и не хранит указатель на
`*grpc.Server`.

```go
// BAD: хэндлер управляет сервером — слой.transport знает про transport.
func (h *depositHandlers) GetDeposit(ctx context.Context, req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    d, err := h.depositSvc.Get(ctx, req.Id)
    if err != nil {
        h.server.GracefulStop() // «сервис нездоров» — решение не хэндлера

        return nil, h.grpcError(err)
    }
    // ...
}

// GOOD: хэндлер возвращает ошибку; реакция на неё — решение app-слоя.
func (h *depositHandlers) GetDeposit(ctx context.Context, req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    d, err := h.depositSvc.Get(ctx, req.Id)
    if err != nil {
        return nil, h.grpcError(err)
    }

    return depositToProto(d), nil
}
```

### Правило 3: Бизнес-сервисы, health и reflection регистрируются раздельно

Регистрация бизнес-сервисов (хэндлеры), health-сервиса и reflection — три
отдельных шага сборки, каждый на своём месте: хэндлеры — по списку API
сервиса, health — по [server_healthcheck.md](./server_healthcheck.md),
reflection — только не-PROD по [server.md](./server.md). Helper «зарегистрировать
всё сразу» внутри пакета хэндлеров запрещён.

```go
// BAD: универсальная функция в handlers, регистрирующая health/reflection —
// транспортные детали утекли в хэндлер-слой.
func (h *depositHandlers) RegisterAll(srv *grpc.Server) {
    depositpb.RegisterDepositHandlersServer(srv, h)

    healthpb.RegisterHealthServer(srv, grpchealth.NewServer())
    reflection.Register(srv)
}

// GOOD: раздельные шаги в app-сборке.
func newApplication(cfg *appConfig) (*application, error) {
    srv := grpc.NewServer(serverOptions()...)

    depositpb.RegisterDepositHandlersServer(srv, depositHandlers)

    healthSvc := grpchealth.NewServer()
    healthpb.RegisterHealthServer(srv, healthSvc)

    return &application{grpcServer: srv, healthSvc: healthSvc}, nil
}
```

## Обработка отдельного запроса

### Правило 1: Валидация входных сообщений на входе в метод

Первое действие метода — загрузка и валидация запроса через form
(`forms.<Method>Form.LoadAndValidate(ctx, req)`): обязательность полей,
диапазоны, формат идентификаторов. Невалидный вход отсекается кодом
`InvalidArgument` до обращения к бизнес-слою. Молчаливые «улучшения» значений
(truncate, подстановка дефолтов) вместо отказа — запрещены.

```go
// BAD: запрос уходит в бизнес-слой без проверки — пустой id доходит до БД.
func (h *depositHandlers) GetDeposit(ctx context.Context, req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    d, err := h.depositSvc.Get(ctx, req.Id)
    if err != nil {
        return nil, h.grpcError(err)
    }

    return depositToProto(d), nil
}

// GOOD: form-валидация первым шагом, InvalidArgument при провале.
func (h *depositHandlers) GetDeposit(ctx context.Context, req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    form := &forms.GetDepositForm{}

    valid, err := form.LoadAndValidate(ctx, req)
    if err != nil {
        if !valid {
            return nil, status.Error(grpcCodes.InvalidArgument, err.Error())
        }

        return nil, h.grpcError(err)
    }

    d, err := h.depositSvc.Get(ctx, form.Id)
    if err != nil {
        return nil, h.grpcError(err)
    }

    return depositToProto(d), nil
}
```

### Правило 2: Контекст и дедлайны запроса

`ctx` входящего вызова передаётся всем зависимостям без подмены; создавать
`context.Background()` внутри обработчика или отрывать от него trace/metadata
нельзя. Таймауты, если нужны, — только `context.WithTimeout(ctx, …)` от
входящего контекста, с `defer cancel()`.

```go
// BAD: контекст запроса выброшен — потерян дедлайн, trace и отмена клиента.
func (h *depositHandlers) GetDeposit(req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    d, err := h.depositSvc.Get(context.Background(), req.Id)
    if err != nil {
        return nil, h.grpcError(err)
    }

    return depositToProto(d), nil
}

// GOOD: ctx течёт сквозь весь вызов; локальный дедлайн — от него.
func (h *depositHandlers) GetDeposit(ctx context.Context, req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    callCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()

    d, err := h.depositSvc.Get(callCtx, req.Id)
    if err != nil {
        return nil, h.grpcError(err)
    }

    return depositToProto(d), nil
}
```

### Правило 3: Маппинг доменных ошибок в grpc status

Каждая ошибка, возвращаемая из обработчика, — `status.Error(code, msg)` с кодом,
вычисленным из кода error formatter (см.
[formatting_codes.md](../errors/formatting_codes.md)): известная доменная ошибка →
осмысленный grpc-код, неизвестная → `Internal`. Наружу не уходит «сырая» `err` и не
утекают внутренние детали (тексты БД, стеки). Хелпер `grpcError(err)` живёт в
`common.go` пакета хэндлеров.

```go
// BAD: внутренний текст ошибки уехал клиенту без кода вовсе.
func (h *depositHandlers) GetDeposit(ctx context.Context, req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    d, err := h.depositSvc.Get(ctx, req.Id)
    if err != nil {
        return nil, err
    }

    return depositToProto(d), nil
}

// BAD: всё подряд свёрнуто в Internal с дампом err — клиент слепнет.
func (h *depositHandlers) ListDeposits(ctx context.Context, req *depositpb.ListDepositsRequest) (*depositpb.ListDepositsResponse, error) {
    ds, err := h.depositSvc.List(ctx, req.Page)
    if err != nil {
        return nil, status.Error(grpcCodes.Internal, fmt.Sprintf("%+v", err))
    }

    return depositsToProto(ds), nil
}

// GOOD: коды доменных ошибок смаплены, неизвестное свёрнуто в Internal.
func (h *depositHandlers) grpcError(err error) error {
    code, ok := h.e.ErrorCodeIsOneOf(err, ErrDepositNotFound, ErrDepositBlocked)
    if !ok {
        return status.Error(grpcCodes.Internal, "internal error")
    }

    switch code {
    case ErrDepositNotFound:
        return status.Error(grpcCodes.NotFound, "deposit not found")
    case ErrDepositBlocked:
        return status.Error(grpcCodes.FailedPrecondition, "deposit is blocked")
    default:
        return status.Error(grpcCodes.Internal, "internal error")
    }
}
```

### Правило 4: Граница DTO — только proto-сообщения

Граница transport/домен — сообщения proto. Доменные сущности не сериализуются в
строки/JSON-поля ответов и не возвращаются через `interface{}`; конвертеры —
только в `marshaller_<метод>.go` с явным перечислением полей. Молчаливый пропуск
поля при маппинге запрещён — каждое поле контракта заполнено или обосновано
комментарием.

```go
// BAD: доменная сущность протекла наружу как JSON в строке — контракт врёт,
// consumers распарсивают неизвестно что.
func (h *depositHandlers) GetDeposit(ctx context.Context, req *depositpb.GetDepositRequest) (*depositpb.GetDepositResponse, error) {
    d, err := h.depositSvc.Get(ctx, req.Id)
    if err != nil {
        return nil, h.grpcError(err)
    }

    return &depositpb.GetDepositResponse{
        DepositJson: fmt.Sprintf("%+v", d),
    }, nil
}

// GOOD: явный маппинг полей в marshaller-файле.
func depositToProto(d *entities.Deposit) *depositpb.GetDepositResponse {
    return &depositpb.GetDepositResponse{
        Id:         d.ID.String(),
        Amount:     d.Amount.String(),
        Status:     depositStatusToProto(d.Status),
        CreatedAt:  timestamppb.New(d.CreatedAt),
    }
}
```
