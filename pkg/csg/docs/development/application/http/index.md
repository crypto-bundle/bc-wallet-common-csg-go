---
title: HTTP. Правила сервер-клиент интеграции
description: Каталог правил HTTP-домена: сервер на net/http, клиенты на resty v2
category: App Development
tags: [http, net-http, resty, overview]
---

---
**Цель**: Регламентировать правила HTTP сервер-клиент интеграции
**Категория правил**: App Development, HTTP
**Краткое содержание:** Создание http handlers, middleware, client configuration, status codes
---

# HTTP. Правила сервер-клиент интеграции

## Введение

HTTP-домен CSG стандартен по канону (2026-10-01): сервер — только стандартная
библиотека `net/http` (прецедент — `bc-api-gateway/internal/http_api`); клиент —
`github.com/go-resty/resty/v2`. Альтернативные серверные технологии (например,
cloudwego/hertz) допускаются в отдельных high-load точках входа, но их API
настоящими правилами не регламентируется. `gin` и `valyala/fasthttp` в канон не
входят — новый код на них не пишется.

Служебные HTTP-эндпоинты (health, metrics, profiler) закрыты библиотеками
`bc-wallet-common-lib-healthcheck` / `-metrics` / `-profiler` и здесь не
дублируются. HTTP→gRPC- шины (`bc-wallet-common-callback-api` на grpc-gateway) —
отдельная технология вне scope домена.

## Состав документов

- [HTTP-сервер: правила создания и жизненного цикла](./server.md) — `net/http`:
  `http.Server`, таймауты, TLS, маршрутизация, middleware, shutdown.
- [HTTP-клиент: правила resty v2](./client.md) — создание клиентов, таймауты,
  TLS, ретраи, контексты, обработка ответов.
