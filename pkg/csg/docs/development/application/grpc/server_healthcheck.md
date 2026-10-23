---
title: gRPC. Health check на стороне сервера (grpc_health_v1)
description: Регистрация и жизненный цикл gRPC Health Checking Protocol в сервисах экосистемы
category: App Development
tags: [grpc, health-check, grpc-health-v1, readiness, shutdown, bc-wallet-common-lib-grpc]
---

---
**Цель**: Регламентировать внедрение gRPC Health Checking Protocol (`grpc.health.v1.Health`) в gRPC-серверы экосистемы
**Категория правил**: App Development, gRPC, health check
**Краткое содержание:** Обязательная регистрация health-сервиса, именование сервисов статуса, корректный Shutdown перед GracefulStop, исключение health из авторизации
---

# gRPC. Health check на стороне сервера (`grpc_health_v1`)

## Введение

Health-протокол реализован в grpc-go: серверная часть —
`google.golang.org/grpc/health` (`grpchealth`), контракты и генерат —
`google.golang.org/grpc/health/grpc_health_v1` (`healthpb`). Для экосистемы он
обязателен по двум причинам: балансировщик
`round_robin_crypto_bundle` фильтрует субсоединения по `SERVING`
(см. [client_dialing_balancing.md](./client_dialing_balancing.md)), а
readiness/liveness-пробы платформ определяют готовность реплики именно по
этому протоколу.

Пример корректных импортов:

```go
import (
    grpchealth "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)
```

## Регистрация health-сервиса

### Правило 1: Каждый gRPC-сервер обязан зарегистрировать grpc_health_v1

`grpchealth.NewServer()` при создании сразу помечает безымянный сервис `""`
как `SERVING` (это «общий» статус сервера) — руками выставлять `""` не нужно.
Регистрация выполняется до `Serve`, вместе с остальными сервисами сервера.

```go
// BAD: сервер без health-сервиса — round_robin_crypto_bundle не увидит READY SubConn
srv := grpc.NewServer(options...)
depositpb.RegisterDepositHandlersServer(srv, handlers)

// GOOD: health-сервис зарегистрирован штатно
srv := grpc.NewServer(options...)

healthSvc := grpchealth.NewServer()
healthpb.RegisterHealthServer(srv, healthSvc)

depositpb.RegisterDepositHandlersServer(srv, handlers)
```

### Правило 2: Статусы именованных сервисов — по полному proto-имени

Если сервис готов/не готов независимо от общего статуса сервера, его статус
выставляют отдельно. Ключ сервиса — полное имя proto-сервиса
(`"<package>.<Service>"`), идентичное имени в регистрации grpc. Короткие и
произвольные имена запрещены: `Check` для неизвестного имени возвращает
`codes.NotFound`, и клиент никогда не найдёт такой статус.

```go
// BAD: сокращённое имя — клиенты с полным именем получат codes.NotFound
healthSvc.SetServingStatus("deposit", healthpb.HealthCheckResponse_SERVING)

// GOOD: полное proto-имя сервиса, совпадающее с генератом
healthSvc.SetServingStatus("deposit.v1.DepositHandlers", healthpb.HealthCheckResponse_SERVING)

// деградацию зависимостей отражаем снятием статуса
healthSvc.SetServingStatus("deposit.v1.DepositHandlers", healthpb.HealthCheckResponse_NOT_SERVING)
```

## Готовность и остановка

### Правило 1: Перед GracefulStop — Shutdown() health-сервиса

`healthSvc.Shutdown()` переводит все сервисы в `NOT_SERVING` и запрещает
последующие смены статуса; после него нужно дать балансировщикам и пробам
получить обновление (секунды достаточно), только затем `GracefulStop`. Без
этого окно «сервер ещё отвечает на Check, но уже дренируется» приносит в
подбор мёртвые или умирающие реплики.

```go
// BAD: сервер останавливают без снятия health-статуса — пробы всё это время получают SERVING
<-ctx.Done()
srv.GracefulStop()

// GOOD: сняли статус, подождали propagation, плавно остановили
<-ctx.Done()

healthSvc.Shutdown()

// время на распространение NOT_SERVING до балансировщиков/проб
time.Sleep(3 * time.Second)

srv.GracefulStop()
```

### Правило 2: Методы health-сервиса не закрывать авторизацией

Auth-интерцептор обязан пропускать методы
`/grpc.health.v1.Health/` без проверки credentials: протобу и пробам
платформы нечем аутентифицироваться, и закрытый авторизацией health
выглядит как полностью недоступный сервис.

```go
// BAD: авторизация применяется ко всем методам, health отбивается UNAUTHENTICATED
func authUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    if err := authorize(ctx); err != nil {
        return nil, status.Error(codes.Unauthenticated, err.Error())
    }

    return handler(ctx, req)
}

// GOOD: health-методы в белом списке интерцептора
func authUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
        return handler(ctx, req)
    }

    if err := authorize(ctx); err != nil {
        return nil, status.Error(codes.Unauthenticated, err.Error())
    }

    return handler(ctx, req)
}
```
