---
title: gRPC. Правила оформления gRPC-сервера
description: Сборка опций, keepalive, лимиты сообщений, интерцепторы и жизненный цикл gRPC-сервера
category: App Development
tags: [grpc, grpc-server, keepalive, lifecycle, bc-wallet-common-lib-grpc]
---

---
**Цель**: Регламентировать создание, конфигурирование и запуск gRPC-серверов сервисов экосистемы на базе `bc-wallet-common-lib-grpc`
**Категория правил**: App Development, gRPC, server
**Краткое содержание:** Базис из DefaultServeOptions, лимиты сообщений из констант библиотеки, обоснованные отклонения keepalive-политик, цепочки интерцепторов, жизненный цикл ListenAndServe/GracefulStop, reflection только в не-продакшен
---

# gRPC. Правила оформления gRPC-сервера

## Введение

gRPC-серверы сервисов crypto-bundle разворачиваются поверх
`github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/server` (далее —
`commonGRPCServer`). Пакет поставляет базовые keepalive/enforcement-политики и
константы лимитов сообщений; все серверы экосистемы обязаны собираться от этих
дефолтов, чтобы поведение соединения было предсказуемым для balance'ров и
клиентов.

Связанные документы: [health check сервера](./server_healthcheck.md)
(обязателен для каждого сервера), [gRPC-клиент](./client.md).

Пример корректного импорта:

```go
import (
    commonGRPCServer "github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/server"
    "google.golang.org/grpc"
)
```

## Сборка опций сервера

### Правило 1: Только DefaultServeOptions как базис

Любой `grpc.NewServer` обязан начинаться с `commonGRPCServer.DefaultServeOptions()`.
Функция возвращает стандартные для экосистемы keepalive-параметры
(`MaxConnectionIdle` 15s, `MaxConnectionAge` 30s, `Time` 2m1s, `Timeout` 1m10s)
и enforcement-политику (`MinTime` 5s, `PermitWithoutStream` true). Самодельные
keepalive-настройки запрещены — рассинхрон с ожиданиями клиентов приводит к
разрыву соединений GOAWAY-ошибками `ENHANCE_YOUR_CALM`.

```go
// BAD: keepalive задан руками «с головы», безотносительно к дефолтам экосистемы
srv := grpc.NewServer(
    grpc.KeepaliveParams(grpcKeepalive.ServerParameters{
        MaxConnectionIdle: 30 * time.Minute,
        Time:              time.Hour,
        Timeout:           10 * time.Second,
    }),
)

// GOOD: базис из библиотеки
srv := grpc.NewServer(commonGRPCServer.DefaultServeOptions()...)
```

### Правило 2: Размеры сообщений — из констант библиотеки

`DefaultServeOptions()` не выставляет лимиты размеров сообщений — их сервис
добавляет сам, используя константы `DefaultServerMaxReceiveMessageSize`
(`math.MaxInt32`) и `DefaultServerMaxSendMessageSize` (24 MiB). Магические числа
в `MaxRecvMsgSize`/`MaxSendMsgSize` запрещены: лимит должен быть единым для всех
сервисов и читаться из имени.

```go
// BAD: магические числа, отличающиеся от стандарта экосистемы
options := append(commonGRPCServer.DefaultServeOptions(),
    grpc.MaxRecvMsgSize(4*1024*1024),
    grpc.MaxSendMsgSize(2*1024*1024),
)

// GOOD: константы библиотеки
options := append(commonGRPCServer.DefaultServeOptions(),
    grpc.MaxRecvMsgSize(commonGRPCServer.DefaultServerMaxReceiveMessageSize),
    grpc.MaxSendMsgSize(commonGRPCServer.DefaultServerMaxSendMessageSize),
)

srv := grpc.NewServer(options...)
```

### Правило 3: Отклонение keepalive — только от дефолтных функций библиотеки

Если SLA сервиса требует иных keepalive-параметров, отклонение допустимо только
от значений, возвращаемых `DefaultKeepaliveServerOptions()` /
`DefaultEnforcementServerOptions()`: меняем ровно нужное поле, причиной служит
замер, зафиксированный комментарием. Переопределяющие опции добавляются после
базиса — в grpc-go последующая опция того же типа вытесняет предыдущую.

```go
// BAD: полностью самодельный ServerParameters с произвольными значениями
srv := grpc.NewServer(grpc.KeepaliveParams(grpcKeepalive.ServerParameters{
    MaxConnectionIdle: 2 * time.Minute,
    MaxConnectionAge:  2*time.Hour + time.Minute,
}))

// GOOD: копируем дефолты, отклоняемся в одном поле с обоснованием
keepalive := commonGRPCServer.DefaultKeepaliveServerOptions()
// стриминг транзакций живёт дольше 30s — не рвём соединения по возрасту
keepalive.MaxConnectionAge = 24 * time.Hour

srv := grpc.NewServer(
    append(commonGRPCServer.DefaultServeOptions(),
        grpc.KeepaliveParams(keepalive),
    )...,
)
```

## Интерцепторы и регистрация сервисов

### Правило 1: Цепочки интерцепторов вместо одиночных

Одиночные `grpc.UnaryInterceptor` / `grpc.StreamInterceptor` deprecated и
допускают ровно одну передачу — повторная опция приводит к ошибке сборки
сервера. Используется однократный `grpc.ChainUnaryInterceptor` /
`grpc.ChainStreamInterceptor` со списком интерцепторов. Методы health-сервиса
авторизации не подлежат — см. [server_healthcheck.md](./server_healthcheck.md),
раздел «Готовность и остановка» (правило про интерцептор).

```go
// BAD: два одиночных UnaryInterceptor — ошибка при создании сервера
srv := grpc.NewServer(
    grpc.UnaryInterceptor(authInterceptor),
    grpc.UnaryInterceptor(metricsInterceptor),
)

// GOOD: одна цепочка + телеметрия через stats handler
srv := grpc.NewServer(
    append(commonGRPCServer.DefaultServeOptions(),
        grpc.ChainUnaryInterceptor(authInterceptor, metricsInterceptor),
        grpc.ChainStreamInterceptor(authStreamInterceptor),
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
    )...,
)
```

### Правило 2: Reflection — только в не-продакшен сборках

`reflection.Register` даёт произвольному клиенту схему методов сервиса. В
production это недопустимо; reflection регистрируется только в dev/staging по
конфигу окружения.

```go
// BAD: reflection доступен всегда, включая production
reflection.Register(srv)

// GOOD: только для не-продакшен-окружения
if cfg.IsDev() || cfg.IsStaging() {
    reflection.Register(srv)
}
```

## Жизненный цикл сервера

### Правило 1: Serve под управлением контекста с корректной остановкой

`Serve` блокирует до остановки сервера, поэтому запускается в отдельной
горутине, а ошибка из него не отбрасывается: она форматируется через
error formatter и возвращается вызывающему коду. Остановка — только
`GracefulStop` по `ctx.Done()` (перед `GracefulStop` вызывается
`Shutdown()` health-сервиса — см.
[server_healthcheck.md](./server_healthcheck.md)).

```go
// BAD: ошибка Serve молча теряется, остановки по контексту нет.
// Метод сервис-компонента grpcAPIService с полями server *grpc.Server, listener net.Listener.
func (s *grpcAPIService) ListenAndServe(ctx context.Context) error {
    go func() {
        _ = s.server.Serve(s.listener)
    }()

    <-ctx.Done()
    s.server.Stop() // жёсткая остановка: активные RPC обрываются

    return nil
}

// GOOD: error formatter и health-сервис — поля компонента (e, health).
// Serve в горутине, ошибка форматируется и возвращается, остановка — GracefulStop.
func (s *grpcAPIService) ListenAndServe(ctx context.Context) error {
    errCh := make(chan error, 1)

    go func() {
        if err := s.server.Serve(s.listener); err != nil {
            errCh <- s.e.Error(err, "grpc serve stopped")
        }
    }()

    select {
    case <-ctx.Done():
        // снимает все сервисы в NOT_SERVING, затем плавно докручиваем соединения
        s.health.Shutdown()
        s.server.GracefulStop()

        return nil
    case err := <-errCh:
        return err
    }
}
```
