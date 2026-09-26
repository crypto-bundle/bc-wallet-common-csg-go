# Форматирование ошибок. Коды ошибок. Правила написания кода

---
**Цель**: Регламентировать формат систем кодов ошибок приложения
**Категория правил**: App Development, ошибки
**Краткое содержание:** Структура error codes и их категоризация
---
## Введение

В данном документе описаны правила и рекомендации по организации кода приложения в контексте
_форматирования ошибок_. Также в данном документе описаны правила описания и формирования
так называемых _**кодов-ошибок**_.

Документация описывает, как пользоваться библиотеками форматирования
ошибок, входящими в базовый тулкит проекта:
- https://github.com/crypto-bundle/bc-wallet-common-lib-tinyerrors
- https://github.com/crypto-bundle/bc-wallet-common-lib-errors

## Правила

### Правило 1: Специальная форма для внешних ошибок

Ошибки, которые должны передаваться за пределы приложения, должны быть сформированы
специальным образом — с присоединённым кодом ошибки, используя `ErrorWithCode` или
`ValuedError` из пакета `bc-wallet-common-lib-errors/pkg/errformatter`.

```go
import (
    errfmt "github.com/crypto-bundle/bc-wallet-common-lib-errors/pkg/errformatter"
)

type WalletRepo interface {
    Get(ctx context.Context, id string) (*Wallet, error)
}

func (r Repository) GetByID(ctx context.Context, id string) (*Wallet, error) {
    // Ресурсные слои могут возвращать plain error.
    w, err := r.db.Wallet().FindByID(ctx, id)
    if err != nil {
        return nil, err // plain error остаётся внутри приложения
    }
    return w, nil
}

// На границе приложения оборачиваем в ErrorWithCode.
func (svc Service) GetByID(ctx context.Context, id string) (string, error) {
    wallet, err := svc.repo.GetByID(ctx, id)
    if err != nil {
        return "", errfmt.ErrorWithCode(
            errfmt.ErrorOnly(err, fmt.Sprintf("failed to load wallet: %s", id)),
            1001, // WalletNotFound
        )
    }
    return wallet.ID, nil
}
```

Важно: plain error допустим внутри внутренних слоёв (repository, domain).
На границах (gRPC handler, HTTP endpoint, RPC caller) — всегда используем `ErrorWithCode`
или `ValuedError`. Вызывающий слой извлекает код через `ErrGetCode` или проверяет
через `ErrorCodeIsOneOf`.

```go
// Вызывающий уровень извлекает код для принятия решения
func HandleRequest(ctx context.Context, id string) (http.ResponseWriter, error) {
    _, err := svc.GetByID(ctx, id)
    if err != nil {
        code := errfmt.ErrGetCode(err) // 1001 или -1 если кода нет
        switch code {
        case 1001: // WalletNotFound
            return http.StatusNotFound, nil
        default:
            return http.StatusInternalServerError, nil
        }
    }
    return http.StatusOK, nil
}
```

### Правило 2: Ошибка ≠ код ошибки

Ошибка типа `error` **не равна** коду ошибки. Код — это метаданные внутри обёрнутой ошибки,
которые можно извлечь через `ErrGetCode` или `ErrorCodeIsOneOf`.

```go
// Плохо: пытаемся сравнить error с кодом напрямую
if someErr == 1001 { ... }

// Хорошо: извлекаем код из обёрнутой ошибки
err := svc.DoSomething(ctx)
code := errfmt.ErrGetCode(err)
if code == 1001 {
    // обработка WalletNotFound
}

// Альтернативно — проверка принадлежности к множеству кодов
_, ok := errfmt.ErrorCodeIsOneOf(err, 1001, 1002)
if ok {
    // одна из известных кодов ошибок найдена
}
```

Если вызывающий код сравнивает `error` напрямую с каким-то значением — такого не будет работать,
потому что `ErrorCode` — это int-значение, встроенное внутрь цепочки обёрнутых ошибок,
а не сама переменная `error`.

### Правило 3: gRPC Status Code ≠ код ошибки

gRPC Status Code **не равен** коду ошибки. gRPC статус описывает transport-level результат вызова,
а код ошибки — бизнес-смысл проблемы. Одна и та же бизнес-ошибка может быть закодирована в разный
gRPC status в зависимости от контекста.

```go
// Плохо: маппим все NotFound'ы в один gRPC code без анализа бизнес-кода
func FindUser(ctx context.Context, id string) (*User, error) {
    user, err := repo.Find(ctx, id)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "user not found")
    }
    return user, nil
}

// Хорошо: анализируем бизнес-код и выбираем gRPC status соответственно
func FindUser(ctx context.Context, id string) (*User, error) {
    user, err := repo.Find(ctx, id)
    if err != nil {
        matched, ok := errfmt.ErrorCodeIsOneOf(err, 2001, 2005, 2010)
        if !ok {
            // не известный код — внутренняя ошибка
            return nil, status.Errorf(codes.Internal, "%v", err)
        }
        switch matched {
        case 2001: // UserNotFound
            return nil, status.Error(codes.NotFound, "user not found")
        case 2005: // UserDisabled
            return nil, status.Error(codes.PermissionDenied, "account disabled")
        default: // другие известные коды
            return nil, status.Error(codes.Unknown, "operation failed")
        }
    }
    return user, nil
}
```

### Правило 4: .String() / .I18N() ≠ текст ошибки

Методы-ресиверы `.String()` и `.I18n()` у типа `TinyErrCodeInt` (из tinyerrors) используются
для логирования и отладки, а не для показа пользователю. Они возвращают технические идентификаторы,
а не локализованный текст.

```go
// Плохо: отдаём форматированный код напрямую в ответ API
return c.JSON(http.StatusBadRequest, map[string]string{
    "error": fmt.Sprintf("%v", errfmt.ErrGetCode(err)),
})

// Хорошо: код в логи, пользовательский текст — через i18n/mapping
log.Printf("business error: code=%d", errfmt.ErrGetCode(err))
return c.JSON(http.StatusBadRequest, map[string]interface{}{
    "code":    errfmt.ErrGetCode(err),
    "message": translate(errfmt.ErrGetCode(err)),
})
```

Для извлечения информации из структурированных ошибок (`*valuedError`) используйте
`ValuedErrorGetCode`:

```go
code := errfmt.ValuedErrorGetCode(err) // извлекает KindCode или -1 если отсутствует
```

### Правило 5: Диапазоны кодов по приложениям

Коды ошибок на уровне приложения должны иметь чёткий диапазон значений, привязанный к конкретному приложению.
Это позволяет избежать конфликтов при использовании общих пакетов.

Библиотека не предоставляет механизмов автоматической нумерации диапазонов —
определение констант остается за разработчиком приложения. Рекомендуется создавать
константы в пакете приложения (например, `apperrors`):

| Приложение          | Диапазон     | Пример                              |
|---------------------|-------------|--------------------------------------|
| Wallet Service      | 1000–1999   | `WalletNotFound = 1001`              |
| Auth Service         | 2000–2999   | `AuthFailed = 2001`, `AccountLocked = 2003` |
| Payment Service     | 3000–3999   | `PaymentRejected = 3001`             |

```go
// app/wallet/errors.go
package wallet

const (
    ErrWalletNotFound       = 1001
    ErrWalletFrozen         = 1002
    ErrInsufficientBalance  = 1003
)

// Сервис использует ErrorWithCode для присоединения кода
func (s Service) Transfer(ctx context.Context, from, to string, amount int64) error {
    bal, err := s.repo.Balance(ctx, from)
    if err != nil {
        return errfmt.ErrorWithCode(err, wallet.ErrWalletNotFound)
    }
    if bal < amount {
        return errfmt.ErrorWithCode(nil, wallet.ErrInsufficientBalance)
    }
    return nil
}
```

Если приложение имеет несколько независимых бизнес-доменов, каждый домен
определяет свой пакет ошибок с собственными константами-диапазонами.
Эти константы локальны для внутреннего использования — на экспорт идёт
обёрнутая ошибка, а не сырые числа.
