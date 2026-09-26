# Форматирование ошибок. Правила написания кода

---
**Цель**: Определить patterns форматирования ошибок и выбор форматтера
**Категория правил**: App Development, ошибки
**Краткое содержание:** Как выбирать и применять error-formatter для форматирования ошибок
---
## Введение

В данном документе описаны рекомендации по организации кода приложения в контексте
_форматирования ошибок_.

Документация описывает, как пользоваться библиотеками форматирования
ошибок, входящими в базовый тулкит проекта:
- https://github.com/crypto-bundle/bc-wallet-common-lib-tinyerrors
- https://github.com/crypto-bundle/bc-wallet-common-lib-errors

Дополнительную информацию по кодам ошибок см. [errors/codes.md](./codes.md).

## Описание форматтеров

Указанные библиотеки предоставляют три типа сервис-компонентов для форматирования ошибок.
Каждый тип доступен через пакетные функции и через фабричные конструкторы инстансов.

### Структура описания каждого форматтера

| Поле | Описание |
|------|----------|
| Названия | Пакетная функция + фабричная функция конструктора |
| Тип обёртки | Результат вызова конструктора или пакетной функции |
| Формат вывода | Как выглядит итоговая `.Error()` строка |
| Параметры | Какие значения передаются и каким Kind |
| Извлечение данных | Что можно вернуть обратно (код, scope, details, public code) |
| Рекомендации | Когда использовать, когда не использовать |

---

#### Правило 1: Default (basic) форматтер

Самый простой форматтер — без scope префикса и без хранения значений.
Подходит для случаев когда нужна только текстовая цепочка оборачивания.

| Поле | Значение |
|------|----------|
| Пакетная функция | `errfmt.Error(err, details...)`, `errfmt.Errorf(err, fmt, args...)` |
| Фабричный конструктор | `errfmt.NewErrorBasicFormatter()` — возвращает `*service` |
| Тип обёртки | plain `error` (строка через `fmt.Errorf("%w -> details")`) |
| Формат вывода | `original_error -> detail1, detail2` |
| Параметры | Только текстовые детали |
| Извлечение данных | Ничего — нет ни кодов, ни scope, ни структурированных значений |
| Когда использовать | Простые обёртывания без необходимости программной обработки; middleware / HTTP layer который оборачивает уже enriched ошибку |
| Когда НЕ использовать | Нужны коды ошибок, идентификация источника, многомерная информация |

Фабричный инстанс (`*service`) предоставляет те же методы, но применительно к одной конфигурации:

```go
import (
    "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

func main() {
    svc := errfmt.NewErrorBasicFormatter()

    // Методы инстанса: .Error(), .Errorf(), .ErrorOnly(),
    // .NewError(), .NewErrorf(), .ErrorNoWrap(), .ErrorWithCode(),
    // .ErrorCodeIsOneOf()
    err := svc.Error(someErr, "operation failed")
}
```

Пакетные функции удобны когда ошибка создаётся один раз в одном месте:

```go
// Плохо: создавать инстанс ради одного вызова
svc := errfmt.NewErrorBasicFormatter()
return svc.Error(err, "context")

// Хорошо: пакетная функция
return errfmt.Error(err, "context")
```

---

#### Правило 2: ScopedError форматтер

Добавляет scope префикс к каждой ошибке. Позволяет идентифицировать источник ошибки
в логах и при дебаге.

| Поле | Значение |
|------|----------|
| Пакетная функция | `errfmt.ScopedError(err, scope, details...)` — возвращает `*scopedError` |
| Фабричный конструктор | `errfmt.NewScopedErrorFormatter(scope string)` — возвращает `*serviceScoped` |
| Тип обёртки | `*scopedError` |
| Формат вывода | `[<scope>]: <inner.Error()> -> <details>` |
| Параметры | Scope — строка; Details — набор строк |
| Извлечение данных | `ErrorCodeIsOneOf()` — да (через ErrorWithCode); scope извлекается парсингом строки |
| Когда использовать | Один namespace ошибок на пакет/сервис, повторное использование formatter как зависимости |
| Когда НЕ использовать | Обособленные утилитарные функции, ошибки собираемые один раз |

```go
import (
    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

// --- Через фабричный инстанс ---

// Создаём scoped formatter один раз и передаём как зависимость
scopeFormatter := errfmt.NewScopedErrorFormatter("payment")

func (s *PaymentService) Process(ctx context.Context) error {
    // .Error(err, details...) — добавляет scope + details
    err := s.processor.Do(ctx)
    if err != nil {

        return scopeFormatter.Error(err, "transaction amount exceeds limit")
    }

    return nil
}

// --- Через пакетные функции ---

// Один вызов — без создания инстанса
if err != nil {

    return errfmt.ScopedError(err, "payment", "invalid gateway response")
}

// Создание новой ошибки (без обёртывания существующей)
return errfmt.NewScopedError("payment", "initialization failed")

// Printf-стиль
return errfmt.ScopedErrorf(err, "payment", "gateway timeout after %d ms", timeoutMs)
```

Методы `*serviceScoped` (`svc.Error()`, `svc.Errorf()`, `svc.NewError()`, `svc.NewErrorf()`,
`svc.ErrorOnly()`, `svc.ErrorNoWrap()`, `svc.ErrorWithCode()`, `svc.ErrorCodeIsOneOf()`)
работают идентично пакетным функциям, но автоматически используют scope указанный при создании.

---

#### Правило 3: ValuedError форматтер

Хранит несколько типов метаданных внутри ошибки: scope, код, детали, публичный код.
Поддерживает ре-wrap merge — если scope совпадает при двойном оборачивании,
детали объединяются вместо создания нового уровня обёртки.

| Поле | Значение |
|------|----------|
| Пакетная функция | `errfmt.ValuedError(err, values[]Value, details...)` — возвращает `*valuedError` |
| Фабричный конструктор | `errfmt.NewValuedErrorFormatter(values...Value)` — возвращает `ErrorFormatterService` |
| Тип обёртки | `*valuedError` |
| Формат вывода | `[scope]: inner -> details` (если scope+details),
`inner -> details` (только details),
`[scope]: inner` (только scope),
`inner` (пусто) |
| Параметры | Values массив — каждый Value имеет Kind (Scope, Code, Details, PublicCode) и значение |
| Извлечение данных | `ValuedErrorGetCode()` — извлекает KindCode;
`errors.As(err, &*valuedError)` — полная структура со всеми Values |
| Когда использовать | Ошибки нужно структурировать для программной обработки (REST response, metrics, tracing); нужны несколько измерений одновременно |
| Когда НЕ использовать | Простая логика где достаточно scope-only; performance критичные участки (дополнительный overhead обёртки) |

**Value system** — каждая Value содержит Kind и любое значение:

```go
value := errfmt.NewValue(errfmt.KindCode, 2005)          // int code
value := errfmt.NewValue(errfmt.KindScope, "wallet")     // string scope
value := errfmt.NewValue(errfmt.KindDetails, "reason")   // string detail
value := errfmt.NewValue(errfmt.KindPublicCode, "E001")  // visible to clients
```

**Через пакетные функции:**

```go
import (
    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

values := []errfmt.Value{
    errfmt.NewValue(errfmt.KindCode, 1001),
    errfmt.NewValue(errfmt.KindDetails, "account frozen"),
}

// Обёртывание существующей ошибки
wrapped := errfmt.ValuedError(baseErr, values, "transfer rejected")

// Новая ошибка (без обёртывания)
newErr := errfmt.ValuedNewError(values, "validation failed")

// Printf-стиль
wrapped := errfmt.ValuedErrorf(baseErr, values, "amount %.2f invalid", amount)
```

**Через фабричный инстанс:**

```go
formatter := errfmt.NewValuedErrorFormatter(
    errfmt.NewValue(errfmt.KindScope, "payment"),
    errfmt.NewValue(errfmt.KindCode, 3001),
)

// Инстанс использует свои значения по умолчанию
wrapped := formatter.Error(baseErr, "timeout")

// Можно передать дополнительные детали
wrapped := formatter.ErrorOnly(baseErr, "extra detail")
```

**Re-wrap merge:**

При последовательном оборачивании тем же инстанцем (scope совпадает) детали мержатся:

```go
// Уровни извлечения:
// Level 1 (самый внешний): "re_wrap_scope: valued_err_scope: test error -> detail1, detail2, detail3"
// Level 2 (после одного Unwrap): "valued_err_scope: test error -> detail1, detail2"
// Level 3 (глубокий target): "test error"

// При разных scope — создаётся вложенная цепочка с обоими scope
```

**Извлечение данных:**

```go
code := errfmt.ValuedErrorGetCode(err) // Returns KindCode value or -1 (ValueCodeMissing) if absent

var valErr *errfmt.valuedError // Note: внутренняя структура, используйте публичные методы
if errors.As(err, &valErr) { ... } // Для продвинутого использования
```

---

#### Правило 4: Таблица сравнения форматтеров

| Возможность | Default (basic) | Scoped | Valued |
|---|---|---|---|
| Scope префикс | ❌ | ✅ `[scope]:` | ✅ `[scope]:` |
| Детали | ✅ `, detail` | ✅ `, detail` | ✅ `, detail` |
| Код ошибки | ❌ | ✅ `ErrorWithCode(int)` | ✅ `NewValue(KindCode, int)` |
| Public code | ❌ | ❌ | ✅ `NewValue(KindPublicCode, string)` |
| Ре-wrap merge | N/A | ❌ (новая обёртка) | ✅ (merge при том же scope) |
| Multiple dims | ❌ | ❌ (только scope) | ✅ (scope+code+details+public) |
| Извлечение кода | ❌ | ✅ `ErrorCodeIsOneOf()` | ✅ `ValuedErrorGetCode()` |
| `errors.As(*T)` | N/A | ✅ `*scopedError` | ✅ `*valuedError` |

---

#### Правило 5: Руководство по выбору форматтера

Используйте следующую блок-схему принятия решения для выбора форматтера:

```
Нужна ли структура для программной обработки? (извлечение кодов, метаданных)
├── Нет → нужен только текстовый wrap
│   ├── Один раз, обособленное место? → пакетные функции: errfmt.Error(err, details)
│   └── Повторно, как зависимость? → errfmt.NewErrorBasicFormatter()
└── Да, нужна дополнительная информация
    ├── Только источник (namespace)? → ScopedError
    │   ├── Утилита, один вызов? → errfmt.ScopedError(err, scope, ...)
    │   └── Зависимость, много методов? → errfmt.NewScopedErrorFormatter(scope)
    └── Нужно несколько измерений? → ValuedError
        ├── Только коды и данные? → errfmt.NewValuedErrorFormatter(values...)
        └── Нужен public code (для клиентов)? → включите KindPublicCode в values
```

---

#### Правило 6: Паттерн внедрения в коде

Определите абстрактный интерфейс в пакете сервиса, чтобы не зависеть от конкретной реализации:

```go
package wallet

type Formatter interface {
    ErrorNoWrap(err error) error
    Error(err error, details ...string) error
    Errorf(err error, format string, args ...any) error
    ErrorOnly(err error, details ...string) error
    ErrorWithCode(err error, code int) error
    ErrorCodeIsOneOf(err error, codes ...int) (int, bool)
}

// Конструктор сервиса принимает интерфейс
type Service struct {
    repo       Repository
    formatter  Formatter
}

func NewService(repo Repository, formatter Formatter) *Service {

    return &Service{repo: repo, formatter: formatter}
}
```

Bootstrap — создание формальтера и передача его как зависимости:

```go
package main

import (
    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
    "myapp/app/wallet"
)

func main() {
    formatter := errfmt.NewScopedErrorFormatter("wallet")

    repository := repo.NewWalletRepository(db)
    service := wallet.NewService(repository, formatter)

    // Использование
    http.HandleFunc("/transfer", func(w http.ResponseWriter, r *http.Request) {
        err := service.Transfer(r.Context(), req)
        handleError(w, err)
    })
}
```

---

#### Правило 7: Паттерн передачи бизнес-кодов

Покажите полный цикл: создание business-error с кодом → обработка → ответ API.

**Сервис: присоединяем код через ErrorWithCode:**

```go
package wallet

import (
    "fmt"

    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
    "myapp/app/wallet"
)

func (s Service) Transfer(ctx context.Context, req *TransferRequest) error {
    source, err := s.repo.GetBalance(ctx, req.From)
    if err != nil {

        return errfmt.ErrorWithCode(
            errfmt.Error(err, "source wallet not found: "+req.From),
            wallet.ErrWalletNotFound, // 1001
        )
    }
    if source < req.Amount {

        return errfmt.ErrorWithCode(nil, wallet.ErrInsufficientBalance) // 1003
    }

    return nil
}
```

**Handler: извлекаем код через ErrGetCode для определения ответа:**

```go
func HandleTransfer(w http.ResponseWriter, r *http.Request) {
    code := errfmt.ErrGetCode(doTransfer(r.Context(), req))
    switch code {
    case 1003: // InsufficientBalance
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "code":    code,
            "message": translate(code),
        })
    case 1001: // WalletNotFound
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusNotFound)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "code":    code,
            "message": translate(code),
        })
    default:
        w.WriteHeader(http.StatusInternalServerError)
    }
    return
}
```

Использование `ErrGetCode` просто — достаёт прикреплённый int-код (или `-1` если кода нет).
Для проверки принадлежности к множеству используйте `ErrorCodeIsOneOf`:

```go
matched, ok := errfmt.ErrorCodeIsOneOf(err, 1001, 1003)
if !ok {
    // Неизвестный код — техническая ошибка, не business error
} else {
    // matched содержит конкретный код из списка
}
```

Альтернативно, используя scoped инстанс:

```go
package wallet

import (
    "context"
    "encoding/json"
    "net/http"

    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

const (
    ErrWalletNotFound      = 1001
    ErrInsufficientBalance = 1003
)

type TransferRequest struct {
    From   string
    To     string
    Amount int64
}

var formatter = errfmt.NewScopedErrorFormatter("wallet")

func doTransfer(ctx context.Context, req *TransferRequest) error {
    bal, err := repo.Balance(ctx, req.From)
    if err != nil {
        // Scope + контекст через ScopedError, затем код через ErrorWithCode
        return errfmt.ErrorWithCode(
            formatter.ErrorOnly(err, fmt.Sprintf("failed to query source wallet: %s", req.From)),
            ErrWalletNotFound,
        )
    }
    if bal < req.Amount {
        // Код присваивается отдельно, без обёртывания
        return errfmt.ErrorWithCode(nil, ErrInsufficientBalance)
    }

    return nil
}

// HTTP handler извлекает код через ErrorCodeIsOneOf
func HandleTransfer(w http.ResponseWriter, r *http.Request) {
    err := doTransfer(r.Context(), &TransferRequest{})
    if err != nil {
        matched, ok := errfmt.ErrorCodeIsOneOf(err, ErrWalletNotFound, ErrInsufficientBalance)
        if !ok {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
            return
        }
        status := http.StatusBadRequest
        switch matched {
        case ErrWalletNotFound:
            status = http.StatusNotFound
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "code":    matched,
            "message": translate(matched),
        })
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

---

## Правила

### Правило 8: Используем bc-wallet-common-lib-errors

Для форматирования ошибок используем библиотеку `bc-wallet-common-lib-errors`. Не создаём свои
форматтеры и не используем сторонние пакеты для тех же целей. Все обёртывания проходят через
`errfmt`: `Error()`, `ErrorOnly()`, `Errorf()`, `NewError()`, `NewErrorf()`, `ErrorNoWrap()`.

```go
// Плохо: свой форматтер
func wrapErr(err error) error {
    return fmt.Errorf("custom prefix: %w", err) // нет структурированности
}

// Хорошо: штатная библиотека
import (
    "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

// errfmt.ErrorOnly формирует: original_error -> detail1, detail2
return errfmt.ErrorOnly(err, "operation failed")
```

### Правило 9: Оборачиваем внешние ошибки

Ошибка, возвращаемая из импортированного пакета или интерфейса, должна быть обернута `error-formatter`,
чтобы сохранить контекст происхождения и обеспечить консистентную семантику.

Например, если репозиторий возвращает plain `error`, сервис оборачивает его в ScopedError или
ValuedError перед возвратом на уровень handler'а.

```go
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
}

// Плохо: внешний error уходит без обработки
func (svc UserService) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := svc.repo.FindByID(ctx, id)
    if err != nil {

        return nil, err // непрозрачный stacktrace из репозитория
    }

    return user, nil
}

// Хорошо: ошибка обработана формальтером
import (
    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

func (svc UserService) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := svc.repo.FindByID(ctx, id)
    if err != nil {
        // Обёртываем scope + контекст
        return nil, errfmt.ScopedError(err, "user_service", "failed to find user by id")
    }

    return user, nil
}
```

### Правило 10: Без двойного форматирования

Не форматируйте ошибку дважды — избегаем ситуации re-wrap одной и той же ошибки.
Такая ситуация может возникать, например, при использовании замыканий или механизма
вызовов функций как колбэков.

Библиотека `errformatter` поддерживает intelligent re-wrap:
при обёртывании уже формализованной ошибки тем же инстантом `*serviceValued`
детали мержатся вместо создания лишнего уровня.

```go
// Плохо: двойное оборачивание внутри callback
func doBatch(ctx context.Context, ids []string) error {
    var results []string
    err := batchProcess(ids, func(id string) error {
        item, err := processOne(ctx, id)
        if err != nil {
            return errfmt.Error(err, "process item "+id) // первое форматирование
        }
        results = append(results, item.ID)

        return nil
    })
    if err != nil {
        return errfmt.Error(err, "batch operation") // второе — ре-wrap потеряет контекст
    }

    return nil
}

// Хорошо: обрабатываем на одном уровне
func doBatch(ctx context.Context, ids []string) error {
    var results []string
    for _, id := range ids {
        item, err := processOne(ctx, id)
        if err != nil {
            return errfmt.ErrorOnly(err, "process item "+id)
        }
        results = append(results, item.ID)
    }

    return nil
}
```

При работе с пакетными функциями — вызывайте `Error()` / `ErrorOnly()` один раз,
на верхнем уровне обработки. Для повторных вызовов используйте фабричный инстанс
(`NewScopedErrorFormatter()`, `NewValuedErrorFormatter()`), который кэширует scope и Values.

### Правило 11: Выбираем форматтер по потребностям

Выбирайте форматтер ошибок исходя из контекста возникновения ошибки. См. [Руководство по выбору](#правило-5-руководство-по-выбору-форматтера) для блок-схемы принятия решения.

#### Единый namespace пакета → ScopedErrorFormatter

Если у пакета или сервис-компонента единый namespace ошибок, используйте `ScopedErrorFormatter`:

```go
package payment

import errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"

var formatter = errfmt.NewScopedErrorFormatter("payment")

// Ресивер сервиса использует shared formatter
func (ps PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    if req.Amount <= 0 {

        return fmt.Errorf("invalid amount: %.2f", req.Amount)
    }

    return nil
}
```

Инстанс `*serviceScoped` автоматически добавляет `[payment]:` префикс ко всем ошибкам.
Также доступны методы `.Errorf()`, `.ErrorWithCode(code int)`, `.ErrorCodeIsOneOf(codes...)`.

#### Ошибка в ресивере → formatter как зависимость

Если ошибка возникает внутри функции-ресивера сервис-компонента, используйте форматтер
как зависимость, удовлетворяющую интерфейсу (`Formatter`). Это обеспечивает тестируемость и замену
реализации без переписывания всех методов.

```go
package wallet

type Formatter interface {
    ErrorNoWrap(err error) error
    Error(err error, details ...string) error
    Errorf(err error, format string, args ...any) error
    ErrorOnly(err error, details ...string) error
    ErrorWithCode(err error, code int) error
    ErrorCodeIsOneOf(err error, codes ...int) (int, bool)
}

type WalletService struct {
    repo      Repository
    formatter Formatter
}

func NewWalletService(repo Repository, formatter Formatter) *WalletService {

    return &WalletService{repo: repo, formatter: formatter}
}

func (ws *WalletService) Transfer(ctx context.Context, from, to string, amount int64) error {
    source, err := ws.repo.GetBalance(ctx, from)
    if err != nil {

        return ws.formatter.Error(err, "source wallet not found")
    }
    if source < amount {

        return ws.formatter.Error(nil, fmt.Sprintf(
            "wallet %s: balance %d < %d", from, source, amount))
    }

    return nil
}
```

#### Обособленная функция → статические вызовы

Если ошибка возникает в обособленной функции, используйте форматирование через вызов
библиотечной (`bc-wallet-common-lib-errors`) экспортируемой функции из пакета `errformatter`.

```go
package utils

import errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"

// Утилита без состояния — не нужно хранить formatter
func ParseConfig(data []byte) (Config, error) {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {

        return Config{}, errfmt.ErrorOnly(err, "invalid config payload")
    }
    if cfg.Timeout == 0 {

        return Config{}, errfmt.Error(errors.New("timeout zero"), "timeout must be non-zero")
    }

    return cfg, nil
}
```

### Правило 12: Оборачиваем бизнес-коды через ValuedError

Для оборачивания бизнес-кодов ошибок используйте форматтеры с поддержкой значений (например,
`ValuedError`). Это позволяет хранить код ошибки внутри структуры ошибки и впоследствии
извлекать его через `ValuedErrorGetCode` или `ErrorCodeIsOneOf`. Подробности см. в
[errors/codes.md](./codes.md) — раздел про коды ошибок.

```go
// Плохо: код ошибки потерян в строке
if user.Status == UserDisabled {

    return errors.Errorf("user disabled: id=%s", userID) // строка — не структура
}

// Хорошо: business code внутри структурированной ошибки
import (
    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

values := []errfmt.Value{
    errfmt.NewValue(errfmt.KindCode, 2005), // AuthUserDisabled
    errfmt.NewValue(errfmt.KindScope, "auth"),
}

return errfmt.ValuedNewError(values, "user account is disabled")

// Извлечение кода на границе (handler / middleware)
code := errfmt.ErrGetCode(err)         // извлекает int или -1 (ValueCodeMissing)
matched, ok := errfmt.ErrorCodeIsOneOf(err, 2001, 2005) // проверка принадлежности
if ok {
    // matched содержит конкретный код из списка
}
```