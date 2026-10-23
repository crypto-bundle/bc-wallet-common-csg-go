---
title: HTTP. Правила создания HTTP-клиентов
description: resty v2 клиенты: инстансы, таймауты, TLS, ретраи, контексты, обработка ответов
category: App Development
tags: [http, http-client, resty, retry, timeout]
---

---
**Цель**: Регламентировать создание и использование HTTP-клиентов на `github.com/go-resty/resty/v2` для внешних вызовов
**Категория правил**: App Development, HTTP, client
**Краткое содержание:** Единый resty.New()-инстанс на сервис, запрет http.DefaultClient и голого http.Get, обязательные SetTimeout и TLS-верификация, ретраи только идемпотентных запросов, SetContext на каждый вызов, маппинг статусов и ошибок через error formatter
---

# HTTP. Правила создания HTTP-клиентов

## Введение

Канон клиентской части — `github.com/go-resty/resty/v2` (сверено: v2.17.1):
цепочечная конфигурация инстанса `*resty.Client`, типизированный разбор ответа
(`SetResult`/`SetError`), встроенные ретраи с условиями. «Голые» вызовы
`net/http` без конфигурации запрещены. Обёртка ошибок следует правилам домена
[errors](../errors/handling.md).

## Создание клиента

### Правило 1: Единый resty-инстанс на внешний сервис, живёт в DI

`*resty.Client` создаётся один раз на внешний сервис на этапе сборки
зависимостей (конструктор/файл конфигурации клиента) и внедряется в
сервис-компоненты: инстанс держит connection pool и транспортную
конфигурацию. `resty.New()` внутри метода — каждый вызов строит новый транспорт.

```go
// BAD: новый клиент на каждый запрос — пул соединений не переиспользуется
func (s *ratesService) GetRate(ctx context.Context) (*Rate, error) {
    resp, err := resty.New().R().
        SetContext(ctx).
        SetResult(&ratesResponse{}).
        Get("/rates")
    if err != nil {
        return nil, s.e.ErrorOnly(err)
    }

    return fromRates(resp.Result().(*ratesResponse)), nil
}

// GOOD: единый инстанс, внедряется в конструктор
func NewRatesProviderService(errFmtSvc errorFormatterService, cfg configService) *ratesProviderService {
    return &ratesProviderService{
        cli: resty.New().
            SetBaseURL(cfg.GetRatesAPIBaseURL()).
            SetTimeout(5 * time.Second),
        e: errFmtSvc,
    }
}
```

### Правило 2: Запрет http.DefaultClient, http.Get и resty без SetTimeout

Клиент обязан иметь явный `SetTimeout`: дефолтный таймаут у `http.Client` и у
resty отсутствует — запрос без дедлайна способен висеть до обрыва транспорта.
`http.DefaultClient` и `http.Get` в сервисном коде запрещены.

```go
// BAD: http.DefaultClient без таймаута — вызов может зависнуть навсегда
resp, err := http.Get(cfg.GetRatesAPIBaseURL() + "/rates")
if err != nil {
    return s.e.ErrorOnly(err)
}

defer resp.Body.Close()

// GOOD: resty с явным таймаутом на инстансе
resp, err := s.cli.R().
    SetContext(ctx).
    SetResult(&ratesResponse{}).
    Get("/rates")
if err != nil {
    return s.e.ErrorOnly(err)
}
```

### Правило 3: TLS-верификация обязательна, InsecureSkipVerify — только локальные тесты

Сертификат проверяется всегда; `InsecureSkipVerify: true` допустим лишь в
тестовом коде (не в рантайме сервиса). Для пининга внутреннего CA — `RootCAs`
из конфига (secrets engine), минимальная версия TLS — явная.

```go
// BAD: проверка сертификата отключена «временно» — MITM на проде
cli := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

// GOOD: верификация со своим CA и явной версией протокола
cli := resty.New().SetTLSClientConfig(&tls.Config{
    MinVersion: tls.VersionTLS12,
    RootCAs:    cfg.GetRatesAPIRootCAs(),
})
```

## Ретраи и идемпотентность

### Правило 1: SetRetryCount только для идемпотентных запросов

Ретраи включаются только клиенту, который выполняет идемпотентные операции
(GET/HEAD, либо PUT с детерминированным телом). Для POST-созданий (заказы,
списания) — отдельный клиент без `SetRetryCount`: повтор на неотвеченном
POST означает повторное списание.

```go
// BAD: единый клиент с ретраями для всего — POST /withdraws повторится на таймауте
cli := resty.New().
    SetBaseURL(cfg.GetProviderBaseURL()).
    SetTimeout(5 * time.Second).
    SetRetryCount(3)

// GOOD: read-клиент с ретраями, write-клиент — без них
readCli := resty.New().
    SetBaseURL(cfg.GetRatesAPIBaseURL()).
    SetTimeout(5 * time.Second).
    SetRetryCount(2).
    SetRetryWaitTime(200 * time.Millisecond).
    SetRetryMaxWaitTime(2 * time.Second)

writeCli := resty.New().
    SetBaseURL(cfg.GetOrdersAPIBaseURL()).
    SetTimeout(10 * time.Second)
```

### Правило 2: AddRetryCondition — сетевые ошибки и 5xx/429, без «ретраим всё»

Условие повтора задаётся `AddRetryCondition`: сетевая ошибка (`err != nil`)
или статусы `429`/`5xx`. Повтор при `4xx` клиента-ошибках запрещён — провайдер
уже отверг запрос, повтор лишь умножает нагрузку.

```go
// BAD: повтор на любой ответ, включая 400 Bad Request
readCli.AddRetryCondition(func(r *resty.Response, err error) bool {
    return true
})

// GOOD: только сетевые сбои и «повторно-пригодные» статусы
readCli.AddRetryCondition(func(r *resty.Response, err error) bool {
    if err != nil {
        return true
    }

    return r.StatusCode() == http.StatusTooManyRequests ||
        r.StatusCode() >= http.StatusInternalServerError
})
```

## Выполнение запросов

### Правило 1: SetContext(ctx) на каждый вызов

Таймаут инстанса не заменяет контекст: `SetContext(ctx)` обязателен на каждом
`R()`, чтобы отмена и дедлайн upstream-операции распространялись на HTTP-вызов.
Контекст, потерянный на уровне клиента, не восстанавливается.

```go
// BAD: контекст не передан — отмена вызовущего не влияет на запрос
resp, err := s.cli.R().SetResult(&ratesResponse{}).Get("/rates")

// GOOD: ctx каждого вызова явен
resp, err := s.cli.R().
    SetContext(ctx).
    SetResult(&ratesResponse{}).
    Get("/rates")
```

### Правило 2: Типизированный разбор ответа — SetResult/SetError, лимиты тела

Успешное тело разбирается в типизированный `SetResult`, ошибка провайдера — в
`SetError`; «сырое» чтение `io.ReadAll(resp.Body)` без ограничения запрещено
(Ответ провайдера может быть любого размера). Для стриминга больших тел —
`SetDoNotParseResponse(true)` + `io.LimitReader`.

```go
// BAD: тело читается без лимита — hostile 1 ГБ ответ выедает память
body, err := io.ReadAll(resp.Body)
if err != nil {
    return s.e.ErrorOnly(err)
}

// GOOD: resty парсит по типам; size-политика — на стороне транспорта клиента
resp, err := s.cli.R().
    SetContext(ctx).
    SetResult(&ratesResponse{}).
    SetError(&apiErrorResponse{}).
    Get("/rates")
if err != nil {
    return s.e.ErrorOnly(err)
}
```

### Правило 3: Ошибки транспорта и 4xx/5xx оборачиваются через error formatter

`err != nil` (транспорт/таймаут/разбор) и `resp.IsError()` (не-2xx) — обе
ветки форматируются error formatter'ом сервиса; статусы маппятся в доменную
ошибку с кодом, «голый» `err` наружу не уходит.

```go
// BAD: 5xx от провайдера — тихий успех, тело интерпретирует вызывающий
if err != nil {
    return s.e.ErrorOnly(err)
}

return fromRates(resp.Result().(*ratesResponse)), nil

// GOOD: оба класса ошибок смаплены через formatter
if err != nil {
    return s.e.Errorf(err, "rates api transport")
}

if resp.IsError() {
    if apiErr, ok := resp.Error().(*apiErrorResponse); ok {
        return s.e.NewErrorf("rates api: status %d: %s", resp.StatusCode(), apiErr.Message)
    }

    return s.e.NewErrorf("rates api: unexpected status %d", resp.StatusCode())
}

return fromRates(resp.Result().(*ratesResponse)), nil
```
