# Обработка ошибок. Правила написания кода

---
**Цель**: Определить patterns обработки и распространения ошибок
**Категория правил**: App Development, ошибки
**Краткое содержание:** Patterns обработки, оборачивания и возврата ошибок
---
## Введение

В данном документе описаны правила и рекомендации по организации кода приложения в контексте _обработки ошибок_.

Документация описывает принципы проверки, оберачивания и распространения ошибок при работе с
внешними пакетами и интерфейсами.

Дополнительную информацию по форматированию ошибок см. [errors/formatting.md](./formatting.md).
Дополнительную информацию по кодам ошибок см. [errors/codes.md](./codes.md).

## Правила

### Правило 1: Не допускай shadowed error

При присваивании результата вызова функции с двумя возвращаемыми значениями (результат + error)
не используй `:=` внутри блока `if`, если это создаст новую переменную, которая скроет
предыдущую ошибку. Shadow-переменная ломает цепочку unwrap и лишает вызывающий код
информации об оригинальной ошибке.

```go
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
}

// Плохо: новая err внутри if скрывает внешнюю переменную
func (svc UserService) GetUser(ctx context.Context, id string) (*User, error) {
    var user *User
    if user, err := svc.repo.FindByID(ctx, id); err != nil { // shadowing!

        return nil, err // теряется ссылка на оригинальную ошибку из репозитория
    }

    return user, nil
}

// Хорошо: присваивание существующей переменной, без shadowing
func (svc UserService) GetUser(ctx context.Context, id string) (*User, error) {
    user, err := svc.repo.FindByID(ctx, id)
    if err != nil {

        return nil, errfmt.ScopedError(err, "user_service", "failed to find user by id")
    }

    return user, nil
}

// Альтернативно: объявление err отдельно от условия
func (svc UserService) GetUserV2(ctx context.Context, id string) (*User, error) {
    var err error
    var user *User
    if user, err = svc.repo.FindByID(ctx, id); err != nil {

        return nil, errfmt.ScopedError(err, "user_service", "failed to find user by id")
    }

    return user, nil
}
```

### Правило 2: Не запирай переменную ошибки внутри зоны видимости if

Не ограничивай область видимости переменной ошибки только блоком `if`. Переменная должна быть
доступна за пределами условия — для логирования, передачи вызывающему уровню или оборачивания.

```go
// Плохо: ошибка доступна только внутри блока if
func doSomething(ctx context.Context) error {
    result, err := expensiveOperation(ctx)
    if err != nil {
        log.Printf("operation failed, retrying...") // нет доступа к err здесь
        return fmt.Errorf("retry failed") // потеряна исходная причина
    }
    // ... дальнейшая обработка
    return nil
}

// Хорошо: ошибка видна вне блока if — для логирования и оборачивания
func doSomething(ctx context.Context) error {
    result, err := expensiveOperation(ctx)
    if err != nil {
        log.Printf("expensive operation failed: %v", err)

        return errfmt.ErrorOnly(err, "context detail available in logs")
    }
    // ... дальнейшая обработка
    return nil
}

// Ещё пример: оборачивание с сохранением контекста
func processRequest(ctx context.Context, req *Request) error {
    payload, err := validateAndParse(req.Body)
    if err != nil {
        // Ошибка доступна снаружи — можно добавить контекст
        return errfmt.ErrorWithCode(
            errfmt.Error(err, "payload validation"),
            ErrValidationFailed,
        )
    }
    return handlePayload(ctx, payload)
}
```

### Правило 3: Оборачиваем внешние ошибки

Ошибка, возвращаемая из импортированного пакета или интерфейса, должна быть обернута `error-formatter`,
чтобы сохранить контекст происхождения и обеспечить консистентную семантику.
См. также [formatting.md → Правило 8](../formatting.md#правило-8-оборачиваем-внешние-ошибки).

Например, если репозиторий возвращает plain `error`, сервис оборачивает его в ScopedError
или ValuedError перед возвратом на уровень handler'а.

```go
// Ресурсный слой — plain error
func (r Repository) GetByID(ctx context.Context, id string) (*Wallet, error) {
    w, err := r.db.Wallet().FindByID(ctx, id)
    if err != nil {

        return nil, err // plain error остаётся внутри слоя репозитория
    }

    return w, nil
}

// Сервис — оборачиваем error-formatter'ом
func (svc Service) GetByID(ctx context.Context, id string) (string, error) {
    wallet, err := svc.repo.GetByID(ctx, id)
    if err != nil {
        // Оборачиваем scope + контекст
        return "", errfmt.ScopedError(
            err,
            "wallet_service",
            fmt.Sprintf("failed to load wallet by id: %s", id))
    }

    return wallet.ID, nil
}

// Или через фабричный инстанс
var walletFormatter = errfmt.NewScopedErrorFormatter("wallet_service")

func (svc Service) GetByIDV2(ctx context.Context, id string) (string, error) {
    wallet, err := svc.repo.GetByID(ctx, id)
    if err != nil {

        return "", walletFormatter.Error(err, fmt.Sprintf("wallet lookup failed: %s", id))
    }

    return wallet.ID, nil
}
```

### Правило 4: Избегай double-formatting (re-wrap)

Не оборачивай ошибку повторно тем же формальтером после того как она уже была обработана.
Библиотека `errformatter` поддерживает intelligent re-wrap: при обёртывании уже формализованной
ошибки тем же инстантом `*serviceValued` детали мержатся вместо создания лишнего уровня.
При использовании пакетных функций повторное оборачивание теряет информацию о стеке.
См. также [formatting.md → Правило 9](../formatting.md#правило-9-без-двойного-форматирования).

```go
// Плохо: двойное оборачивание внутри callback / замыкания
func doBatch(ctx context.Context, ids []string) error {
    var results []string
    err := batchProcess(ids, func(id string) error {
        item, err := processOne(ctx, id)
        if err != nil {
            return errfmt.Error(err, "process item "+id) // первое оборачивание
        }
        results = append(results, item.ID)

        return nil
    })
    if err != nil {
        // Повторное оборачивание — re-wrap теряет оригинальный stack trace
        return errfmt.Error(err, "batch operation failed")
    }

    return nil
}

// Хорошо: обрабатываем на одном уровне, один раз оборачиваем
func doBatch(ctx context.Context, ids []string) error {
    var results []string
    for _, id := range ids {
        item, err := processOne(ctx, id)
        if err != nil {
            // Один раз оборачиваем прямо здесь
            return errfmt.ErrorOnly(err, "processing item "+id)
        }
        results = append(results, item.ID)
    }

    return nil
}

// С factory инстансом: intelligent re-wrap merge
var batchFormatter = errfmt.NewValuedErrorFormatter(
    errfmt.NewValue(errfmt.KindScope, "batch_processor"),
)

func doBatchMerged(ctx context.Context, ids []string) error {
    var results []string
    for _, id := range ids {
        item, err := processOne(ctx, id)
        if err != nil {
            // Инстанс мержит детали при том же scope — это безопасно
            return batchFormatter.ErrorOnly(err, "processing item "+id)
        }
        results = append(results, item.ID)
    }

    return nil
}
```
