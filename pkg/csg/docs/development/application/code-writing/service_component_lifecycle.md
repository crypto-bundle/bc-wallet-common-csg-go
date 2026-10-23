---
title: Написание кода. Жизненный цикл сервис-компонента
description: Стадии инициализации, запуска и завершения работы сервис-компонентов
category: App Development
tags: [code-writing, lifecycle, init-run-shutdown]
---

---
**Цель**: Регламентировать стадии жизненного цикла каждого сервис-компонента (Init, Run, Shutdown)
**Категория правил**: App Development, code writing
**Краткое содержание:** Init(ctx), Run(ctx), завершение по отмене контекста, мониторинг состояния через lib-healthcheck
---

# Написание кода. Жизненный цикл сервис-компонента

## Введение

Разные сервис-компоненты могут иметь различный срок службы и роли, но **базовые правила жизненного цикла** едины для всех. Каждый компонент проходит три стадии: инициализация, запуск, завершение. Все они тесно связаны с `context.Context`.

### Правило 1: Инициализация — `Init(ctx context.Context) error`

Метод `Init` вызывается один раз после создания экземпляра фабрики, но до начала выполнения основной логики. В `Init` выполняется:

- Валидация доступности внутренних ресурсов (проверка соединения с БД, чтение конфига)
- Пре-подготовка данных, необходимых для корректного запуска
- Возврат ошибки, если что-то не удалось подготовить

`Init` **не выполняет** бизнес-логику — только подготовку.

```go
// BAD: Бизнес-логика внутри Init
func (s *worker) Init(ctx context.Context) error {
	s.processOrder(ctx) // Не нужно делать здесь

	return nil
}

// GOOD: Только валидация ресурсов
func (s *worker) Init(ctx context.Context) error {
	if s.db == nil {
		return errors.New("db connection not configured")
	}

	return nil
}
```

### Правило 2: Запуск — `Run(ctx context.Context) error`

Метод `Run` — основная рабочая лошадка компонента. Вызывается после `Init`:

- Выполняет основную бизнес-логику (обработка очередей, фоновые задачи, HTTP-сервисы)
- Должна корректно реагировать на отмену контекста (`ctx.Done()`)
- Возвращает ошибку только при критическом сбое, делающем дальнейшую работу невозможной

```go
// BAD: Нет реакции на отмену контекста
func (s *worker) Run(ctx context.Context) error {
	for {
		s.processOne(ctx) // Цикл не останавливается при ctx.Cancel()
	}
}

// GOOD: Корректная работа с контекстом
func (s *worker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case item := <-s.queue.Chan():
			if err := s.processOne(item); err != nil {
				return err
			}

		case msg := <-s.events.Chan():
			if err := s.handleEvent(msg); err != nil {
				return err
			}
		}
	}
}
```

### Правило 3: Завершение по отмене контекста

Завершение работы сервиса **обязательно** происходит по завершении или отмене контекста, ранее переданного в `Run(ctx context.Context) error`. Это означает:

- При вызове `cancel()` из родительского контекста все горутины сервиса должны завершиться
- Очистка ресурсов (закрытие соединений, сброс буферов) выполняется в defer'ах или в конце `Run`
- После `Run` экземпляр компонента считается недоступным для повторного использования

```go
// BAD: Горутина не знает о контексте, висит бесконечно
func (s *worker) Run(ctx context.Context) error {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			s.doPeriodicTask() // Никогда не проверяет ctx
		}
	}()

	return nil
}

// GOOD: Все горутины слушают ctx.Done(), очистка в defer
func (s *worker) Run(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.cleanup() // Очистка ресурсов
				return
			case <-ticker.C:
				s.doPeriodicTask(ctx)
			}
		}
	}()

	<-done // Ждём завершения горутины перед возвратом

	return ctx.Err()
}
```

### Правило 4: Мониторинг состояния через bc-wallet-common-lib-healthcheck

Если сервис-компонент содержит параметры состояния, применимые к метрикам или отчёту о состоянии приложения, опрос этих параметров должен использовать библиотеку `bc-wallet-common-lib-healthcheck`:

- Регистрация check'ов через API healthcheck (HTTP-endpoint readiness/liveness)
- Сохранение промежуточного состояния через вспомогательный сервис-компонент
- Чтение метрик через Prometheus-экспортер из того же пакета

```go
// BAD: Прямой опрос без регистрации в healthcheck
func (s *worker) getHealth() string {
	return fmt.Sprintf("items_processed=%d", s.counter) // Не подключено к healthcheck
}

// GOOD: Регистрация check'а в healthcheck
func (s *worker) RegisterChecks(hc *healthcheck.HealthChecker) {
	hc.AddCheck("worker_items_processed", healthcheck.CheckFunc(func(ctx context.Context) error {
		if s.counter > s.maxItems {
			return fmt.Errorf("counter overflow: %d > %d", s.counter, s.maxItems)
		}
		return nil
	}))
}
```
