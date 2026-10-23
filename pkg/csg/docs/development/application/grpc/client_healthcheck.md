---
title: gRPC. Health check на стороне клиента (grpc_health_v1)
description: Проверка готовности gRPC-сервисов через grpc.health.v1.Health из клиентского кода
category: App Development
tags: [grpc, health-check, grpc-health-v1, readiness, client, bc-wallet-common-lib-grpc]
---

---
**Цель**: Регламентировать использование gRPC Health Checking Protocol на стороне клиента экосистемы
**Категория правил**: App Development, gRPC, health check
**Краткое содержание:** Check в gate-местах с дедлайном, корректный ключ сервиса и обработка codes.NotFound, Watch вместо поллинга, запрет пер-методовых проверок
---

# gRPC. Health check на стороне клиента (`grpc_health_v1`)

## Введение

Клиентская работа с `grpc.health.v1.Health` нужна ровно в трёх местах:
wait-for-ready при старте сервиса, ответ на внешнюю пробу (HTTP/readiness
эндпоинт через `bc-wallet-common-lib-healthcheck`) и диагностика в
административных методах. Для бизнес-вызовов health-проверки не выполняются —
готовность подключений обеспечивает связка keepalive + retry +
`round_robin_crypto_bundle` (см.
[client_dialing_balancing.md](./client_dialing_balancing.md)).

`healthpb.NewHealthClient` принимает `grpc.ClientConnInterface`, поэтому
работать можно и с `*grpc.ClientConn`, и напрямую с `*LazyConnection` из
`bc-wallet-common-lib-grpc` — он реализует этот интерфейс.

Пример корректного импорта:

```go
import (
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)
```

## Проверка готовности

### Правило 1: Check — только в gate-местах и с дедлайном контекста

Одиночная проверка `Check` выполняется с явным коротким дедлайном (секунды),
а ответ в `NOT_SERVING` и транспортная ошибка различаются явно. Ошибки
health-вызова оборачиваются через error formatter, как любые ошибки
grpc-вызовов.

```go
// BAD: Check без дедлайна — gate старта может зависнуть на весь upstream-контекст
resp, err := healthpb.NewHealthClient(lazyConn).Check(ctx, &healthpb.HealthCheckRequest{Service: ""})

// GOOD: клиент поверх LazyConnection, дедлайн, разбор статуса
healthCli := healthpb.NewHealthClient(lazyConn)

checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()

resp, err := healthCli.Check(checkCtx, &healthpb.HealthCheckRequest{Service: "deposit.v1.DepositHandlers"})
if err != nil {
    return errFmt.ErrorOnly(err)
}

if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
    return errFmt.NewErrorf("deposit service not serving: %v", resp.GetStatus())
}
```

### Правило 2: Ключ сервиса — пустая строка или полное proto-имя

`Service: ""` запрашивает общий статус сервера (сервер помечает его `SERVING`
с момента создания health-сервиса). Для отдельного сервиса передаётся полное
proto-имя (`"deposit.v1.DepositHandlers"`); неизвестный серверу ключ
возвращает `codes.NotFound` — это ошибка реестра статусов, а не недоступность,
и молча трактовать её как `NOT_SERVING` нельзя.

```go
// BAD: произвольное сокращённое имя — стабильный codes.NotFound вместо статуса
resp, err := healthCli.Check(checkCtx, &healthpb.HealthCheckRequest{Service: "deposit"})

// BAD: NotFound трактуется как «сервис жив, но не готов»
if err != nil && status.Code(err) == codes.NotFound {
    return nil // молча считаем здоровым
}

// GOOD: общий статус сервера либо точное полное имя
resp, err := healthCli.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
```

## Подписка на изменения статуса

### Правило 1: Непрерывное отслеживание — Watch, а не поллинг Check

`Watch` стримит смены статуса: первый кадр для неизвестного сервиса —
`SERVICE_UNKNOWN`, и это не ошибка, стрим продолжается и доставит статус при
регистрации. Поллинг `Check` с коротким интервалом запрещён — он создаёт
паразитный трафик и race-окно между проверкой и вызовом.

```go
// BAD: busy-poll Check раз в 100ms вместо подписки
for {
    resp, err := healthCli.Check(ctx, &healthpb.HealthCheckRequest{Service: "deposit.v1.DepositHandlers"})
    if err == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING {
        return nil
    }

    time.Sleep(100 * time.Millisecond)
}

// GOOD: Watch с обработкой SERVICE_UNKNOWN как «ждём»
stream, err := healthCli.Watch(watchCtx, &healthpb.HealthCheckRequest{Service: "deposit.v1.DepositHandlers"})
if err != nil {
    return errFmt.ErrorOnly(err)
}

for {
    resp, err := stream.Recv()
    if err != nil {
        return errFmt.ErrorOnly(err)
    }

    switch resp.GetStatus() {
    case healthpb.HealthCheckResponse_SERVING:
        return nil
    case healthpb.HealthCheckResponse_SERVICE_UNKNOWN, healthpb.HealthCheckResponse_NOT_SERVING:
        // ждём новых статусов; watchCtx ограничивает общее ожидание готовности
        continue
    }
}
```

### Правило 2: Не выполнять Check перед каждым бизнес-RPC

Per-method health-проверки в клиентском коде запрещены: health-фильтрацию
уже выполняет `round_robin_crypto_bundle` (`HealthCheck: true` в balancer'е),
а недоступность закрывает retry из `DefaultDialOptions()`. Двойная проверка
добавляет RTT на каждый вызов и не даёт гарантий (статус может измениться
сразу после Check).

```go
// BAD: health-гейт перед каждым вызовом — лишний RTT на каждый RPC
func (s *withdrawOrderService) GetTariff(ctx context.Context, id string) (*Tariff, error) {
    if err := s.checkHealth(ctx); err != nil {
        return nil, err
    }

    return s.tariffCli.GetTariff(ctx, &tariffpb.GetTariffRequest{Id: id})
}

// GOOD: прямые вызовы, готова ли связность — решает balancer/retry
func (s *withdrawOrderService) GetTariff(ctx context.Context, id string) (*Tariff, error) {
    return s.tariffCli.GetTariff(ctx, &tariffpb.GetTariffRequest{Id: id})
}
```
