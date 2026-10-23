---
title: HTTP. Правила создания HTTP-сервера
description: net/http сервер: конфигурация, таймауты, TLS, маршрутизация, middleware, shutdown
category: App Development
tags: [http, net-http, http-server, middleware, tls, shutdown]
---

---
**Цель**: Регламентировать создание, конфигурирование и жизненный цикл HTTP-серверов на стандартной библиотеке `net/http`
**Категория правил**: App Development, HTTP, server
**Краткое содержание:** Явный http.Server вместо ListenAndServe, обязательные таймауты, TLS и адрес из конфига, ServeMux-маршрутизация в internal/http_api, middleware-стек, горутинный Serve и graceful Shutdown
---

# HTTP. Правила создания HTTP-сервера

## Введение

Канон серверной части — чистый `net/http` (`http.Server` + `http.ServeMux` +
func-middleware), прецедент: `bc-api-gateway/internal/http_api/server.go`
(`server.Init()` собирает mux и `http.Server`, `authMiddleware` обёртывает
обработчик). Альтернативные технологии (например, cloudwego/hertz) допустимы в
отдельных high-load точках входа, но их API здесь не регламентируется.

Связанные документы: [HTTP-хэндлеры](./handlers.md) (структура пакета обработки
запросов), [HTTP-клиент](./client.md).

## Конфигурация сервера

### Правило 1: Только явный http.Server, без http.ListenAndServe и DefaultServeMux

Сервер собирается явным `http.Server` с собственной `http.NewServeMux()`:
пакетный `http.ListenAndServe(addr, nil)` и `http.HandleFunc` используют
`DefaultServeMux` и дефолтный сервер без таймаутов и без контроля жизненного
цикла.

```go
// BAD: «implicit» сервер: DefaultServeMux, никаких таймаутов, ошибка через panic
func run(addr string) {
    http.HandleFunc("/v1/orders", listOrders)

    err := http.ListenAndServe(addr, nil)
    if err != nil {
        panic(err)
    }
}

// GOOD: явный http.Server и свой mux (прецедент gateway: Init() + ListenAndServe(ctx))
func (s *server) Init() error {
    s.httpMux = http.NewServeMux()

    s.httpServer = &http.Server{
        Addr:    s.configSvc.GetHttpBindPort(),
        Handler: s.authMiddleware(s.httpMux),
    }

    return nil
}
```

### Правило 2: Обязательные таймауты ReadHeaderTimeout / ReadTimeout / WriteTimeout / IdleTimeout

Все четыре таймаута задаются при создании `http.Server`: их отсутствие
позволяет медленному/злоумышленному клиенту удерживать соединения (slowloris)
и вешает горутину ответа навсегда. Значения — из SLA эндпоинтов, не «на глаз».

```go
// BAD: сервер без единого таймаута
srv := &http.Server{Addr: cfg.GetHttpBindPort(), Handler: mux}

// GOOD: полный набор таймаутов
srv := &http.Server{
    Addr:              cfg.GetHttpBindPort(),
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       10 * time.Second,
    WriteTimeout:      20 * time.Second,
    IdleTimeout:       120 * time.Second,
}
```

### Правило 3: Адрес, TLS-конфиг и минимальная версия TLS — из конфига сервиса

Пути к сертификатам, адрес и настройки TLS читаются через
`bc-wallet-common-lib-config` (Get-методы, секреты — из secrets engine);
`MinVersion` фиксируется явно. Хардкод путей/портов в коде запрещён.

```go
// BAD: пути сертификатов зашиты, версия протокола не ограничена
srv := &http.Server{Addr: ":443", Handler: mux}

err := srv.ListenAndServeTLS("/etc/ssl/tls.crt", "/etc/ssl/tls.key")

// GOOD: всё из конфига, минимальная версия TLS явная
srv := &http.Server{
    Addr:      cfg.GetHttpBindPort(),
    Handler:   mux,
    TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
}

err = srv.ListenAndServeTLS(cfg.GetTLSCertFile(), cfg.GetTLSKeyFile())
```

## Маршрутизация и middleware

### Правило 1: Маршруты — http.ServeMux пакета internal/http_api

Таблица маршрутов живёт в одном месте (пакет `internal/http_api`, см.
[handlers.md](./handlers.md)) и использует Go 1.22+ шаблон
`"<METHOD> /path"`. Регистрация через пакетный `http.HandleFunc` из `init()`
любых пакетов запрещена.

```go
// BAD: маршруты разбросаны по init-функциям разных пакетов
func init() {
	http.HandleFunc("/v1/merchant/create", createMerchant)
}
```

```go
// GOOD: явная регистрация на собственном mux c указанием метода
func setupRoutes(s *server) {
	s.httpMux.Handle("POST /v1/merchant", s.createMerchantHandler())
	s.httpMux.Handle("GET /v1/orders/{orderUUID}", s.getOrderHandler())
}
```

### Правило 2: Middleware — обёртки http.Handler, фиксированный порядок стека

Каждый middleware — функция `func(http.Handler) http.Handler`; стек собирается
один раз на поле `Handler` сервера в порядке: recover → logging/tracing → auth
→ mux. Авторизация внутри тела хэндлера запрещена.

```go
// BAD: auth-логика внутри хэндлера; recover отсутствует
func (s *server) createMerchantHandler(w http.ResponseWriter, req *http.Request) {
	if !s.isAuthorized(req) {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}
	// ... бизнес-логика
}

// ---

// GOOD: стек обёрток на Handler сервера (прецедент: authMiddleware в gateway)
func buildServer(srv *server) *http.Server {
	return &http.Server{
		Addr: cfg.GetHttpBindPort(),
		Handler: recoverMiddleware(
			loggingMiddleware(
				s.authMiddleware(s.httpMux),
			),
		),
	}
}
```

### Правило 3: Публичный API и служебные эндпоинты — раздельные серверы/порты

health, metrics и profiler обслуживает их собственная пара портов из
`bc-wallet-common-lib-healthcheck` / `-metrics` / `-profiler` (зона тех
документов). Публичный `http.Server` не должен нести `/healthz`-подобные
маршруты: пробы платформы не проходят бизнес-auth, а общий порт открывает
служебные данные наружу.

```go
// BAD: служебный маршрут на публичном сервере под authMiddleware
s.httpMux.Handle("GET /healthz", s.healthHandler)

// GOOD: публичный mux — только бизнес-маршруты; health/metrics — серверы библиотек
s.httpMux.Handle("POST /v1/merchant", s.createMerchantHandler())
s.httpMux.Handle("GET /v1/orders/{orderUUID}", s.getOrderHandler())
```

## Жизненный цикл

### Правило 1: Serve в горутине с возвратом ошибки через error formatter

Блокирующий `ListenAndServe` запускается в горутине, ошибка (кроме ожидаемого
`http.ErrServerClosed`) форматируется error formatter и возвращается
вызывающему коду; `panic` из-за падения HTTP-сервера запрещён.

```go
// BAD: блокирующий вызов в main-горутине — контекст приложения не работает
func (s *server) ListenAndServe(ctx context.Context) error {
    return s.httpServer.ListenAndServe()
}

// GOOD: goroutine + errCh + ErrServerClosed отфильтрован
func (s *server) ListenAndServe(ctx context.Context) error {
    errCh := make(chan error, 1)

    go func() {
        err := s.httpServer.ListenAndServe()
        if err != nil && !errors.Is(err, http.ErrServerClosed) {
            errCh <- s.e.Error(err, "http serve stopped")
        }
    }()

    select {
    case <-ctx.Done():
        return s.shutdown()
    case err := <-errCh:
        return err
    }
}
```

### Правило 2: Graceful Shutdown(ctx) по завершению контекста приложения

Остановка — `Shutdown(shutdownCtx)` с ограничением по времени (дренируем
активные запросы), а не `Close()` (обрывает соединения). Контекст shutdown
создаётся от `context.Background()` — контекст приложения уже отменён.

```go
// BAD: жёсткое закрытие listener'а, активные ответы оборваны
func (s *server) shutdown() error {
    s.httpServer.Close()

    return nil
}

// GOOD: graceful drain с явным лимитом
func (s *server) shutdown() error {
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err := s.httpServer.Shutdown(shutdownCtx)
    if err != nil {
        return s.e.ErrorOnly(err)
    }

    return nil
}
```
