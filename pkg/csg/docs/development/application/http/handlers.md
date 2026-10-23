---
title: HTTP. Обработка запросов. Правила для хэндлеров
description: Структура директории хэндлеров, создание и подключение обработчиков HTTP-сервера, обработка входящих запросов
category: App Development
tags: [http, http-handlers, request-handling, package-layout, net-http]
---

---
**Цель**: Регламентировать создание и подключение HTTP-хэндлеров сервисного уровня экосистемы на `net/http`, организацию их директории и файлов
**Категория правил**: App Development, HTTP, request handling
**Краткое содержание:** Layout пакета хэндлеров (файл на эндпоинт, форма валидации), хэндлер как сервис-компонент с DI, регистрация маршрутов, валидация/маппинг статусов в методе-обработчике
---

# HTTP. Обработка запросов. Правила для хэндлеров

## Введение

Хэндлер-слой — граница между transport (`net/http`) и бизнес-логикой сервиса.
Один домен обслуживается пакетом `internal/http_api`: структура сервера и
маршрутизация живут в `server.go` + `common.go`, каждый эндпоинт — отдельный
файл хэндлера и файл формы валидации. Прецедент: `bc-api-gateway/internal/http_api`.

Для API, работающих поверх gRPC-бэкенда, используется grpc-gateway (примеры:
callback-api, merchant_api) — этот паттерн не описывается в CSG. Данный
документ касается исключительно самописных REST-хэндлеров.

Связанные документы: [HTTP-сервер](./server.md), [gRPC-хэндлеры](../grpc/request_handling.md).

## Структура пакета HTTP-хэндлеров

### Правило 1: Хэндлеры — пакет `internal/http_api`, сервер в common.go/server.go

Пакет `internal/http_api` хранит структуру `server` (common.go), сборку mux и
сервера (`Init()`) и жизненный цикл (`ListenAndServe()`) из server.go, а также
отдельные файлы хэндлеров и middleware. Пакет `main` не содержит логики обработки
запросов; хэндлеры не регистрируются через пакетные `http.HandleFunc` из `init()`
других пакетов.

```go
// BAD: хэндлер зарегистрирован через глобальный HandleFunc в init — невозможно переиспользовать
func init() {
    http.HandleFunc("/v1/orders", getOrderHandler)
}

// GOOD: пакет internal/http_api изолирован от сборки приложения;
// main/app только создаёт и запускает экземпляр server.
type server struct {
    httpMux    *http.ServeMux
    httpServer *http.Server

    e      errorFormatterService
    cfg    configService
}

func (s *server) Init() error {
    s.httpMux = http.NewServeMux()

    return nil
}
```

### Правило 2: Один эндпоинт — один файл handler_<endpoint>.go, форма — в forms/<endpoint>.go

Каждый обрабатываемый маршрут имеет выделенный файл `handler_create_merchant.go`
(или аналогичное именование). Форма парсинга и валидации входных данных лежит
в пакете `internal/forms/<endpoint>.go` — это повторяет layout gateway, где уже
существуют `forms/add_account_currency.go` и т.п. Складывать все методы в один
`server.go` или дробить один метод по разным файлам нельзя.

```go
// BAD: всё в одном файле — серверная логика смешана с формированием ответа;
// diff становится нефункциональным на любом изменении маршрута.
// BAD: нет отдельных форм-файлов — валидация размазана по телам хэндлеров.
```

```go
// GOOD: handler_create_merchant.go — ровно один метод хэндлера + вызов формы:
func (s *server) CreateMerchant(w http.ResponseWriter, r *http.Request) {
	var input forms.CreateMerchantInput
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "malformed json")

		return
	}

	valid, verr := input.Form.Validate()
	if !valid {
		writeJSONError(w, http.StatusBadRequest, verr.Error())

		return
	}

	m, err := s.merchantSvc.Create(r.Context(), input)
	if err != nil {
		writeJSONError(w, mapDomainStatus(err), "merchant create failed")

		return
	}

	writeJSON(w, http.StatusCreated, toMerchantResponse(m))
}
```

```go
// --- пример содержимого forms/create_merchant.go ---
type CreateMerchantInput struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

func (f *CreateMerchantInput) Validate() (bool, error) {
	if f.Name == "" || f.Country == "" {
		return false, fmt.Errorf("name and country are required")
	}

	return true, nil
}
```

### Правило 3: Middleware — отдельные файлы пакета, обёртки http.Handler

Аутентификация, recovery, логирование и другие перекрёстные аспекты реализуются
как функции типа `func(http.Handler) http.Handler`, расположенные в своих файлах
middleware_auth.go, middleware_recovery.go и т.д. внутри того же пакета `internal/http_api`.
Логика авторизации внутри тела конкретного хэндлера запрещена. Порядок стека
фиксирован: recover → logging/tracing → auth → mux.

```go
// BAD: авторизация внутри каждого метода — копипаст логики и отсутствие единой точки контроля
func (s *server) createMerchant(w http.ResponseWriter, r *http.Request) {
	claims, ok := extractClaims(r.Header)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}
}
```

```go
// GOOD: middleware — одна функция, оборачивающая следующий шаг.
// Локальный пример стека: один вызов next.ServeHTTP внутри обёртки.
func authHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// проверка прав доступа
		next.ServeHTTP(w, r)
	})
}
```

## Создание HTTP-хэндлера

### Правило 1: Хэндлер — метод сервис-компонента, зависимости — интерфейсы

Хэндлер реализует метод структуры `server`, которая инкапсулирует все необходимые
зависимости: бизнес-сервисы, error formatter, конфигуратор. Конкретные реализации
(репозитории БД, конфигурация соединения) передаются через consumer-side
интерфейсы, объявленные в `common.go` (см. [Инъекция зависимостей](../code-writing/dependency_injection_rules.md)).
Замыкание над глобальным состоянием запрещено.

```go
// BAD: замыкание над глобальной переменной — зависимость захардкожена, тестировать невозможно
type globalState struct {
	dep *someDependency
}

var current = &globalState{}

func handleOrderGlobal(w http.ResponseWriter, r *http.Request) {
	result := current.dep.Process(orderFromRequest(r))
}
```

```go
// GOOD: один экземпляр server, внедряется в конструктор приложения; каждый хэндлер работает с полями receiver.
func NewServer(deps []interface{}, e errorFormatterService, cfg configService) *server {
	return &server{
		e:      e,
		cfg:    cfg,
		deps:   deps,
	}
}

func (s *server) createMerchantHandler(w http.ResponseWriter, r *http.Request) {
	var input forms.CreateMerchantInput
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")

		return
	}

	m, err := s.merchantSvc.Create(r.Context(), input)
	if err != nil {
		writeJSONError(w, mapDomainStatus(err), err.Error())

		return
	}

	writeJSON(w, http.StatusCreated, toMerchantProto(m))
}
```

### Правило 2: Регистрация маршрутов — один таблицу маршрутов при сборке

Таблица маршрутов собирается в одном месте: `server.Init()` или явный список
пар `{Method, Path}`. Пакетный `HandleFunc` из `init()` любого пакета
нарушает принцип единственной точки регистрации и приводит к гонкам порядка
инициализации.

```go
// BAD: маршруты «размазаны» по multiple init() разных пакетов
func init() {
    http.HandleFunc("/v1/orders", listOrders)
}
func init() {
    http.HandleFunc("/v1/merchant", createMerchant)
}

// GOOD: единая таблица маршрутов в Init() сервера
func (s *server) Init() error {
    s.httpMux.Handle("POST /v1/merchant", s.createMerchantHandler())
    s.httpMux.Handle("GET /v1/orders/{id}", s.getOrderHandler())
    s.httpMux.Handle("DELETE /v1/orders/{id}", s.cancelOrderHandler())

    return nil
}
```

## Обработка запроса

### Правило 1: Проверка метода и Content-Type на входе

Первое действие хэндлера — валидация HTTP-метода и Content-Type. Запрос с неверным
методом завершается 405 Method Not Allowed; запрос с ожидаемым телом, но без
`application/json` — 415 Unsupported Media Type. Игнорирование этих проверок
приводит к десериализации случайных входных данных.

```go
// BAD: хэндлер читает тело даже при GET — мусор десериализуется
func (h *ordersHandler) List(w http.ResponseWriter, r *http.Request) {
    var params OrderListParams
    _ = json.NewDecoder(r.Body).Decode(&params) // игнорируем ошибку, если body пустой
}

// GOOD: проверка метода и Content-Type перед чтением тела
func (h *ordersHandler) Create(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

        return
    }

    if ct := r.Header.Get("Content-Type"); ct != "application/json" {
        http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)

        return
    }
}
```

### Правило 2: Тело — io.LimitReader, декодирование — строгий JSON

Тело запроса всегда ограничивается `io.LimitReader(body, maxBodyBytes)` перед
json.Decoder; чтение без лимита уязвимо к DoS. После декодирования ошибки
обрабатываются явно — invalid JSON и невалидные поля формируют 400 Bad Request
через форму валидации.

```go
// BAD: чтение тела без лимита, raw decode без проверки ошибок
err := json.NewDecoder(r.Body).Decode(&input)

// GOOD: ограничение + строгий декодер с проверкой
var input forms.CreateOrderInput
if err := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes)).Decode(&input); err != nil {
    http.Error(w, "malformed json", http.StatusBadRequest)

    return
}

valid, verr := input.Form.Validate()
if !valid {
    writeJSONError(w, http.StatusBadRequest, verr.Error())

    return
}
```

### Правило 3: Бизнес-ошибки и успешный ответ — маппинг статуса

Результат бизнес-вызова маппится в HTTP-статус: найденный объект → 2xx, найденный
но заблокированный → 400/409, неизвестный ресурс → 404, внутренняя ошибка → 500.
Маппинг выполняется через код ошибки из error formatter'а (см.
[formatting_codes.md](../errors/formatting_codes.md)); «голый» текст `err.Error()`
в теле ответа наружу не уходит (безопасность).

```go
// BAD: весь err попадает в тело ответа — клиент получает системную информацию
func badErrorHandling(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
```

```go
// GOOD: маппинг кодов через formatter, безопасный текст для клиента
func handleOrderCreate(h *handler, r *http.Request, input any) {
	m, err := h.orderSvc.Create(r.Context(), input)
	if err != nil {
		code, ok := h.e.ErrorCodeIsOneOf(err, ErrOrderNotFound, ErrOrderBlocked, ErrInternal)
		if !ok {
			code = ErrInternal
		}

		switch code {
		case ErrOrderNotFound:
			http.Error(r.Context().Value("writer").(http.ResponseWriter), "not found", http.StatusNotFound)
		case ErrOrderBlocked:
			http.Error(r.Context().Value("writer").(http.ResponseWriter), "blocked", http.StatusConflict)
		default:
			http.Error(r.Context().Value("writer").(http.ResponseWriter), "internal error", http.StatusInternalServerError)
		}

		return
	}

	writeJSON(r.Context().Value("writer").(http.ResponseWriter), http.StatusCreated, toOrderResponse(m))
}
```

### Правило 4: Ответ — WriteHeader + Encode, двойной write запрещён

Заголовок статуса должен писаться строго до записи тела. Первый `Write` или
`WriteHeader` фиксирует статус (по умолчанию 200), последующие `Write` молча
игнорируются. Двойной `WriteHeader` паникует. helper `writeJSON` в `common.go`
единообразно устанавливает `Content-Type: application/json`, пишет код статуса и
marshal payload.

```go
// BAD: двойной WriteHeader — паника runtime
func doubleWriteHeader(w http.ResponseWriter) {
	w.WriteHeader(http.StatusCreated)
	w.WriteHeader(http.StatusCreated) // panic: header already written
}
```

```go
// GOOD: единый helper — header + encode один раз
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(v)
}
```
