---
title: gRPC. Правила подключения gRPC-клиента
description: Dial-опции, LazyConnection, TLS-переопределения и ретраи для gRPC-клиентов
category: App Development
tags: [grpc, grpc-client, lazy-connection, retry, bc-wallet-common-lib-grpc]
---

---
**Цель**: Регламентировать способ подключения сервисов-клиентов к gRPC-серверам экосистемы через `bc-wallet-common-lib-grpc`
**Категория правил**: App Development, gRPC, client
**Краткое содержание:** Базис DefaultDialOptions, LazyConnection как единственный штатный способ межсервисного подключения, переопределения хвостовыми опциями, ретраи только дефолтного набора кодов
---

# gRPC. Правила подключения gRPC-клиента

## Введение

Клиентская часть `github.com/crypto-bundle/bc-wallet-common-lib-grpc`
(пакет `pkg/client`, далее — `commonGRPCClient`) поставляет базовый набор
dial-опций (keepalive, блокирующий коннект, retry-интерцептор) и ленивое
соединение `LazyConnection`. Все межсервисные вызовы в экосистеме строятся
поверх этих компонентов.

Связанные документы: [dialer'ы и балансировка](./client_dialing_balancing.md),
[health check клиента](./client_healthcheck.md).

Пример корректного импорта:

```go
import (
    commonGRPCClient "github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/client"
    "google.golang.org/grpc"
)
```

## Сборка опций подключения

### Правило 1: Только DefaultDialOptions как базис

`commonGRPCClient.DefaultDialOptions()` возвращает стандартный набор:
`insecure` transport credentials, `WithBlock`, keepalive (`Time` 2m,
`Timeout` 1m9s, `PermitWithoutStream` true) и цепочку unary интерцепторов с
retry. Голый `grpc.Dial` без этих опций запрещён — такое соединение не
переживает паузы балансировщика и выполняет звонки без ретраев.

```go
// BAD: «голый» вызов без keepalive и ретраев, deprecated-вариант grpc.Dial
conn, err := grpc.Dial("deposit-service.wallet:9091",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)

// GOOD: базис из библиотеки; живой коннект — через LazyConnection (см. ниже)
options := commonGRPCClient.DefaultDialOptions()
_ = options
```

### Правило 2: Переопределения — только хвостовыми опциями

Дописывать и переопределять поведение можно только опциями, добавленными
**после** базиса: grpc-go применяет последнюю опцию того же типа
(например, TLS-credentials перекрывают встроенный `insecure`). Копировать
состав `DefaultDialOptions()` руками и «править» копию запрещено — при
обновлении библиотеки сервис теряет её дефолты молча.

```go
// BAD: рукописная копия набора опций — разъедется с библиотекой при первом релизе
options := []grpc.DialOption{
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithBlock(),
    grpc.WithKeepaliveParams(grpcKeepalive.ClientParameters{Time: time.Minute}),
}

// GOOD: базис + переопределение в конце списка
lazyConn := commonGRPCClient.NewLazyConnection(errFmt, "api.partner.example:443",
    grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
)
```

### Правило 3: Транспорт — insecure по умолчанию, TLS только явно

Внутрисетевые соединения кластера используют `insecure.NewCredentials()` — это
уже вшито в `DefaultDialOptions()`, добавлять повторно не нужно. Опция
`grpc.WithInsecure()` запрещена (deprecated). Выход за периметр кластера —
только явный TLS хвостовой опцией (см. правило 2).

```go
// BAD: deprecated-опция поверх базиса
options := append(commonGRPCClient.DefaultDialOptions(), grpc.WithInsecure())

// GOOD: для кластера — ничего не добавляем; для внешнего канала — явный TLS
lazyConn := commonGRPCClient.NewLazyConnection(errFmt, addr,
    grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
)
```

## LazyConnection — штатное межсервисное подключение

### Правило 1: Единый LazyConnection на target, живёт в DI

`commonGRPCClient.NewLazyConnection(errFmtSvc, address, opts...)` —
обязательный способ подключения к другим сервисам: коннект выполняется
лениво при первом `Invoke`/`NewStream` (блокируясь до 10s по контексту
вызова), все ошибки оборачиваются через переданный error formatter,
otelgrpc-stats handler добавляется автоматически. Экземпляр создаётся один
раз на target на этапе сборки зависимостей и внедряется в конструкторы.
`*LazyConnection` реализует `grpc.ClientConnInterface`, поэтому подходит
сгенерированным proto-клиентам напрямую.

```go
// BAD: grpc.Dial в каждом методе — новое соединение на вызов, утечка коннектов
func (s *tariffService) GetTariff(ctx context.Context, id string) (*Tariff, error) {
    conn, err := grpc.DialContext(ctx, "tariff-service.wallet:9091",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, s.e.ErrorOnly(err)
    }

    resp, err := tariffpb.NewTariffHandlersClient(conn).GetTariff(ctx, &tariffpb.GetTariffRequest{Id: id})
    if err != nil {
        return nil, s.e.ErrorOnly(err)
    }

    return fromProto(resp), nil
}

// GOOD: один LazyConnection на target, внедряется в конструктор
func NewWithdrawOrderService(
    lazyConn *commonGRPCClient.LazyConnection,
    errFmtSvc errorFormatterService,
) *withdrawOrderService {
    return &withdrawOrderService{
        tariffCli: tariffpb.NewTariffHandlersClient(lazyConn),
        e:         errFmtSvc,
    }
}
```

### Правило 2: Каждый вызов — с дедлайном контекста

Первый `Invoke`/`NewStream` блокируется на установку соединения до
`connectTimeout` (10 секунд), поэтому контекст вызова обязан несть дедлайн,
ограниченный SLA метода, — иначе вызов переживёт таймаут сервера и
upstream-контекст.

```go
// BAD: контекст без дедлайна — первый вызов может висеть до 10s сверх SLA
err := s.conn.Invoke(ctx, method, req, resp)

// GOOD: явный дедлайн на вызов
callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()

err = s.conn.Invoke(callCtx, method, req, resp)
```

### Правило 3: Close — ровно один раз на остановке приложения

`LazyConnection.Close()` вызывается в фазе завершения процесса — после
остановки собственных серверов. Вызывать `Close` между запросами, а также
игнорировать его при остановке — запрещено.

```go
// BAD: соединение закрывается после каждого вызова — «лень» превращается в dial на каждый RPC
func (s *tariffService) GetTariff(ctx context.Context, id string) error {
    defer s.conn.Close()

    return s.conn.Invoke(ctx, method, req, resp)
}

// GOOD: закрытие на shutdown-фазе приложения
func (app *application) shutdown() {
    app.grpcAPIServer.GracefulStop()
    app.tariffConn.Close()
    app.depositConn.Close()
}
```

## Ретраи и интерцепторы

### Правило 1: Ретраи — только дефолтный набор кодов

Retry уже встроен в `DefaultDialOptions()` (см. `DefaultRetryOptions()`):
3 попытки, линейный backoff 450ms, коды `Aborted` и `Unavailable`.
Расширять набор кодов на `Internal`, `DeadlineExceeded`,
`FailedPrecondition` и прочие ошибки без гарантии идемпотентности запрещено —
это путь к двойному списанию. Дублировать retry-интерцептор поверх базового
нельзя: число попыток перемножится.

```go
// BAD: расширяем коды ретрая на неидемпотентные ошибки и дублим интерцептор
options := append(commonGRPCClient.DefaultDialOptions(),
    grpc.WithChainUnaryInterceptor(
        grpcRetry.UnaryClientInterceptor(
            grpcRetry.WithMax(5),
            grpcRetry.WithCodes(grpcCodes.Internal, grpcCodes.Unavailable, grpcCodes.DeadlineExceeded),
        ),
    ),
)

// GOOD: дефолтные опции как есть — retry уже внутри DefaultDialOptions()
options := commonGRPCClient.DefaultDialOptions()
```

### Правило 2: Свои интерцепторы — отдельной цепочкой, не заменой

Прикладные интерцепторы (аудит, проброс заголовков) добавляются
`grpc.WithChainUnaryInterceptor` хвостом: grpc-go объединяет несколько таких
опций в одну цепочку, библиотечные retry-обёртки сохраняются. Заменять
интерцепторную часть базиса своей цепочкой запрещено.

```go
// BAD: свой интерцептор вместо базиса — сервис теряет retry и keepalive
options := []grpc.DialOption{
    grpc.WithChainUnaryInterceptor(auditInterceptor),
}

// GOOD: свой интерцептор дописан к базису
lazyConn := commonGRPCClient.NewLazyConnection(errFmt, addr,
    grpc.WithChainUnaryInterceptor(auditInterceptor),
)
```
