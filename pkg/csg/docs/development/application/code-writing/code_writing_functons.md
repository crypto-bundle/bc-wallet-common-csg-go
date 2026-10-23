# Написание кода. Правила оформления кода функций

---
**Цель**: Регламентировать правила оформления сигнатур и тел функций.
**Категория правил**: App Development, code writing
**Краткое содержание:** Правила оформления сигнатуры и тела функции. Отступы, проверка ошибок, передача зависимостей.
---

## Сигнатура функции

### Оформление параметров

#### Правило 1: Параметры на отдельных строках

Когда у функции более одного параметра, каждый параметр размещается на собственной строке:

```go
func MyFunction(
    ctx context.Context,
    param1 string,
    param2 int,
    param3 bool,
) {
    // ...
}
```

При отсутствии параметров скобки остаются пустыми:

```go
func MyFunction() {
    // ...
}
```

#### Правило 2: Закрывающая круглая скобка на новой строке

Закрывающая скобка `)` идёт на отдельной строке после последнего параметра:

```go
func MyFunction(
    param1 string,
    param2 int,
) {
    // ...
}
```

### Возврат значения

#### Правило 3: Возврат указателя через `return &struct{...}`

Фабричные функции возвращают указатель на экземпляр структуры. Поля заполняются в именованном виде, каждое поле — на отдельной строке с выравниванием по имени поля:

```go
func NewHandler(name string, port int) *handler {
    return &handler{
        name: name,
        port: port,
    }
}
```

## Тело функции

### Правило 4: Избегать использования `else` — применять guard-паттерн (early return)

Логика работы функции должна следовать правилу «сверху вниз»: сначала отсекаем ошибки и некорректные входные данные, потом выполняем основную логику. Не следует писать:

```go
// BAD: Неправильно — else блоки скрывают основную логику и требуют вложенности
func Process(data string) error {
    if data == "" {
        return fmt.Errorf("data is empty")
    } else {
        if len(data) > 100 {
            return fmt.Errorf("data too long")
        } else {
            // основная логика с ещё одной вложенностью
            if err := validate(data); err != nil {
                return err
            } else {
                return execute(data)
            }
        }
    }
}
```

Следует писать так:

```go
// GOOD: Правильно — каждый guard возвращает сразу, основная логика на нулевой вложенности
func Process(data string) error {
    if data == "" {
        return fmt.Errorf("data is empty")
    }

    if len(data) > 100 {
        return fmt.Errorf("data too long")
    }

    if err := validate(data); err != nil {
        return err
    }

    return execute(data)
}
```

### Правило 5: Максимальная вложенность if — 2 уровня

Вложенность блоков `if` более двух уровней ухудшает читаемость. При достижении второго уровня следует выносить логику во вспомогательные функции или применять guard-паттерн:

```go
// BAD: Неправильно — вложенность 3 уровня
func AssignRole(user User, role string) error {
    if user.IsActive {
        if role != "" {
            if role == Admin {
                // ...
            } else {
                // ...
            }
        }
    }

    return nil
}

// GOOD: Правильно — вложенность не превышает 2 уровня
func AssignRole(user User, role string) error {
    if !user.IsActive {
        return fmt.Errorf("user is not active")
    }

    if role == "" {
        return fmt.Errorf("role is empty")
    }

    return grantRole(role) // дополнительная логика вынесена
}
```

### Правило 6: Запретить использование `goto`

В Go недопустимо использовать оператор `goto`. Для выхода из многоуровневых циклов применяется именованный блок `label` вместе с `break`:

```go
// BAD: Неправильно — goto
func FindUser(users []User, id string) *User {
found:
    for i := range users {
        if users[i].ID == id {
            goto found
        }
    }

    return nil
}

// GOOD: Правильно — break с меткой
func FindUser(users []User, id string) *User {
search:
    for i := range users {
        if users[i].ID == id {
            break search
        }
    }

    if i >= len(users) {
        return nil
    }

    return &users[i]
}
```

### Правило 7: Отделять пустой строкой блоки кода и операторы `if`, `for`, `switch`

**7.1 Пустая строка после закрывающей скобки блока.**
После каждого блока, ограничивающего зону видимости (`{` … `}`), **всегда** ставится пустая строка перед следующим фрагментом кода. Это относится ко всем операторам, создающим блок: `if`, `else if`, `else`, `for`, `switch`, `select`.

Это единственное правило, которое можно нарушать — потому что закрывающая `}` визуально сливается с последующей строкой. Запомните: **после `}` всегда пустая строка, если только за ней не следует `else`, `)`, `,` или конец функции**.

```go
// BAD: Неправильно — нет пустой строки после }
func Process(data []byte) error {
    if data == nil {
        return ErrNil
    }
    if len(data) == 0 {
        return ErrEmpty
    }
    for _, b := range data {
        if b < 0x20 {
            return ErrInvalid
        }
        processed = append(processed, b)
    }
    return nil
}

// GOOD: Правильно — пустая строка после каждой }
func Process(data []byte) error {
    if data == nil {
        return ErrNil
    }

    if len(data) == 0 {
        return ErrEmpty
    }

    for _, b := range data {
        if b < 0x20 {
            return ErrInvalid
        }

        processed = append(processed, b)
    }

    return nil
}
```

**7.2 Пустая строка перед `return` внутри блока `if` / `else`.**
Если `return` — **единственный** оператор в теле блока `if`, `else if` или `else`, отбивка пустой строкой перед ним **не требуется**. Это правило особенно актуально для guard-паттернов, где каждая ветка содержит один быстрый возврат. Если же в блоке есть дополнительные операторы перед `return` (присваивание, вызов функции, логирование), отбивка становится обязательной:

```go
// GOOD: return единственный оператор в каждом блоке if/else — без пустых строк перед return
func Validate(input Input) error {
    if input.Name == "" {
        return ErrEmptyName
    }

    if !input.Valid {
        return ErrInvalid
    }

    if input.Expired {
        return ErrExpired
    }

    return process(input) // return на уровне функции — всегда с отбивкой
}

// BAD: Неправильно — пустые строки там, где они не нужны
func BadValidate(input Input) error {
    if input.Name == "" {

        return ErrEmptyName
    }

    if !input.Valid {

        return ErrInvalid
    }

    if input.Expired {

        return ErrExpired
    }

    return process(input)
}
```

**7.3 Пустая строка между смысловыми блоками.**
Отдельные операции внутри одного блока (присваивания, вызовы функций) могут идти подряд без пустых строк. Однако каждый новый *смысловой этап* (проверка, трансформация, возврат) выделяется пустой строкой. Типичные границы:

- Проверка ошибки → присваивание
- Присваивание → вызов другой функции
- Операции с данными → `return`

Пример — блочные операторы + смысловые переходы:

```go
func HandleRequest(req Request) (*Response, error) {
    // guard: проверка ошибки
    if req.Err != nil {
        return nil, fmt.Errorf("precondition failed: %w", req.Err)
    }

    // guard: проверка флага
    if !req.Enabled {
        return nil, ErrDisabled
    }

    // guard: проверка статуса
    if req.Status == StatusCancelled {
        return nil, ErrCancelled
    }

    // основная обработка
    result := compute(req.Payload)

    return &Response{
        Result:  result,
        Version: version,
    }, nil
}
```

**7.4 Пустая строка перед `case` / `default` и `return` внутри `switch` / `select`.**

Каждый блок `case` и `default` обязательно предваряется пустой строкой относительно предыдущего блока кода или другого блока `case`/`default`. Это правило **не имеет исключений**.

Внутри блока `case` / `default` отбивка пустой строкой перед `return` требуется **только если в блоке более одной строки кода**. Если `return` — единственный оператор в `case`/`default`, отбивка не нужна:

```go
// GOOD: switch — каждый case отделён пустой строкой; return с отбивкой только при >1 строки
func StatusLabel(code int32) string {
    switch code {
    case StatusCodeOK:
        return "ok" // return единственный оператор — без пустой строки

    case StatusCodePending:
        return "pending"

    case StatusCodeFailed:
        log.Warn("unexpected failed code", code)

        return "failed" // return после оператора — отбивка обязательна
    }

    return "unknown" // default-case «вне switch» — на уровне функции, отбивка всегда
}

// BAD: Неправильно — нет разделителя между case, нет отбивки перед финальным return
func BadStatusLabel(code int32) string {
    switch code {
    case StatusCodeOK:
        return "ok"
    case StatusCodePending: // нет пустой строки перед case
        return "pending"
    case StatusCodeFailed:
        log.Warn("unexpected failed code", code)
        return "failed" // есть лог, но нет отбивки перед return
    }
    return "unknown" // нет отбивки перед return на уровне функции
}

// GOOD: select — каждый case/select отделён, те же правила для return
func Listen(ctx context.Context, ch <-chan Event) error {
    for {
        select {
        case event := <-ch:
            process(event)

            continue // операция перед continue — отбивка обязательна
        case <-ctx.Done():
            return ctx.Err() // return единственный в case — без пустой строки

        case <-time.After(timeout):
            return ErrTimeout
        }
    }
}
```

### Правило 8: Перед каждой строкой `return` ставить пустую строку

Строка `return` всегда предваряется пустой строкой. Это зрительно отдляет результат возврата от предыдущей логики и делает финальную точку функции заметной при чтении.

**Исключение:** если `return` — **единственный** оператор в логическом блоке `if` / `switch`, отбивка пустой строкой **не нужна**.

На уровне тела функции (1 уровень вложенности) отбивка перед `return` нужна **всегда**, исключений нет.
```go
// GOOD: внутри if/switch return без отбивки; на уровне функции — всегда с отбивкой
func FindUser(users []User, id string) *User {
    for i := range users {
        if users[i].ID == id {
            return &users[i] // return единственный в блоке if — без пустой строки
        }
    }

    return nil // return на уровне функции (1 ур.) — отбивка обязательна
}

// BAD: guards внутри if не требуют отбивки; итоговым return на уровне функции — всегда нужен разрыв
func ProcessOrder(order *Order) error {
    if order.ID == "" {
        return fmt.Errorf("missing order ID") // guard — return один внутри if, без пустой строки
    }

    items, err := enrichItems(order.Items)
    if err != nil {
        return fmt.Errorf("enrich items: %w", err) // guard — return один внутри if, без пустой строки
    }

    order.Items = items       // операция перед итоговым return
    return nil                // BAD: нет пустой строки, хотя это level-1 return
}

// GOOD: то же, но правильно: все level-1 returns отделены пустыми строками
func ProcessOrder(order *Order) error {
    if order.ID == "" {
        return fmt.Errorf("missing order ID")
    }

    items, err := enrichItems(order.Items)
    if err != nil {
        return fmt.Errorf("enrich items: %w", err)
    }

    order.Items = items

    return nil
}
```

### Правило 9: Обработка ошибок — ссылка на docs/errors/handling.md

Все детали обработки ошибок смотри в [Обработка ошибок](../errors/handling.md).

### Правило 10: Оборачивание кодов ошибок — ссылка на docs/errors/codes.md

Каждая ошибка, передаваемая за пределы функции или пакета, должна быть обернута в код ошибки согласно [Форматирование ошибок. Коды ошибок](../errors/codes.md).