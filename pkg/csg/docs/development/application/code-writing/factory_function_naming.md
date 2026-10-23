---
title: Написание кода. Названия factory-функций
description: Правила именования factory-функций при создании сервис-компонентов
category: App Development
tags: [code-writing, factory-functions, naming]
---

---
**Цель**: Регламентировать правила оформления названий factory-функций для создания сервис-компонентов
**Категория правил**: App Development, code writing
**Краткое содержание:** Префиксы (New/Make), суффиксы (Service/Worker), семантическая описательность, стиль CamelCase
---

# Написание кода. Названия factory-функций

## Введение

Factory function — это функция, создающая экземпляр Go-структуры, которая обладает одним или несколькими методами («ресиверами»). Проще говоря, структура с методами = сервис-компонент. Factory-функция возвращает указатель на этот экземпляр и принимает все необходимые зависимости.

Именование factory-функций строго регламентировано — имя должно однозначно сообщать разработчику, *какой* тип создаётся и *в каком виде*.

### Правило 1: Ключевые префиксы (New / Make)

Каждая factory-функция **обязана** начинаться с одного из двух ключевых слов:

- `New` — используется в большинстве случаев, стандартный префикс для Go-пакетов
- `Make` — альтернатива, применяется при необходимости смыслового различия между фабриками одного пакета

Остальные префиксы запрещены (`Create`, `Build`, `Init`, `Construct` и т.п.).

```go
// BAD: Запрещённый префикс
func CreateDepositOrderProcessor() (*depositOrderProcessor, error) { ... }

// BAD: Нет ни New, ни Make
func BuildWorkerService() (*workerService, error) { ... }

// GOOD: Стандартный префикс
func NewDepositOrderProcessorService() (*depositOrderProcessor, error) { ... }

// GOOD: Допустимая альтернатива
func MakeWithdrawOrderFraudCheckerService() (*fraudChecker, error) { ... }
```

### Правило 2: Ключевые суффиксы (Service / Worker)

Имя factory-функции **обязано** заканчиваться на один из ключевых суффиксов:

- `Service` — для бизнес-логики, обработчиков, контроллеров
- `Worker` — для фоновых задач, консюмеров, обработчиков очередей

Суффикс указывает на роль создаваемой структуры в архитектуре приложения.

```go
// BAD: Нет суффикса
func NewDepositHandler() (*depositHandler, error) { ... }

// GOOD: Бизнес-логика → Service
func NewDepositProcessingService() (*depositProcessingService, error) { ... }

// GOOD: Фоновая задача → Worker
func NewEmailDeliveryWorker() (*emailDeliveryWorker, error) { ... }
```

### Правило 3: Семантическая описательность

Название должно чётко описывать задачу сервис-компонента. Избегайте аббревиатур, которые неочевидны при первом чтении. Каждый читатель должен понять назначение функции по её имени без обращения к телу.

```go
// BAD: Неясное название
func NewProcSvc() (*svc, error) { ... }

// GOOD: Описательное название
func NewDepositFraudCheckService() (*fraudCheckService, error) { ... }

// GOOD: Чётко отражает доменную область
func NewPaymentReconciliationWorker() (*reconciliationWorker, error) { ... }
```

### Правило 4: Возвращаемый тип в имени

Имя factory-функции содержит сокращённое название возвращаемой структуры с соответствующим постфиксом для удобства чтения. Постфикс типа должен перекликаться с суффиксом фабрики.

```go
// BAD: Имя функции не связано с возвращаемым типом
func NewAuthService() (*orderProcessor, error) { ... }

// GOOD: "Auth" ↔ "auth", суффиксы совпадают
func NewAuthService() (*authService, error) { ... }

// GOOD: "Wallet" ↔ "wallet", паттерн соблюдён
func NewWalletExportWorker() (*walletExportWorker, error) { ... }
```

### Правило 5: Стиль наименования (CamelCase)

Все factory-функции оформляются строго в стиле **PascalCase** (начинается с заглавной буквы). Многословные имена формируются конкатенацией значимых частей без разделителей-подчёркиваний.

```go
// BAD: snake_case
func New_Delivery_Svc() (*deliverySvc, error) { ... }

// BAD: camelCase (не экспортируется)
func newOrderProcessorWorker() (*orderProcessor, error) { ... }

// GOOD: PascalCase
func NewOrderProcessorService() (*orderProcessorService, error) { ... }
```
