---
title: gRPC. Dialer'ы и балансировка клиента
description: DNS SRV-резолвер, net/unix dialer'ы и round_robin balancer из bc-wallet-common-lib-grpc
category: App Development
tags: [grpc, load-balancing, dns-srv, unix-socket, dialer, bc-wallet-common-lib-grpc]
---

---
**Цель**: Регламентировать подключение DNS SRV-резолвинга, кастомных dialer'ов и балансировщика нагрузки на стороне gRPC-клиента
**Категория правил**: App Development, gRPC, client
**Краткое содержание:** Связка dns.NewResolver + dialer/net для балансировки через DNS, unix-socket dialer для сайдкаров, balancer round_robin_crypto_bundle с health-фильтрацией
---

# gRPC. Dialer'ы и балансировка клиента

## Введение

Библиотека `github.com/crypto-bundle/bc-wallet-common-lib-grpc` поставляет три
механизма поверх базовых опций клиента: резолвер DNS SRV-записей
(`pkg/dns`), сетевой и unix-сокетный dialer'ы
(`pkg/client/dialer/net`, `pkg/client/dialer/unix`) и собственный
round-robin balancer (`pkg/client/roundrobin`). Все они надстраиваются над
`DefaultDialOptions()` / `LazyConnection` хвостовыми опциями — базис не
заменяется (см. [client.md](./client.md)).

Пример корректных импортов:

```go
import (
    commonGRPCClient "github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/client"
    netDialer "github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/client/dialer/net"
    unixDialer "github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/client/dialer/unix"
    roundRobin "github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/client/roundrobin"
    commonDNS "github.com/crypto-bundle/bc-wallet-common-lib-grpc/pkg/dns"
    "google.golang.org/grpc"
)
```

## Разрешение адресов через DNS SRV

### Правило 1: SRV-резолвинг — только связкой pkg/dns + pkg/client/dialer/net

Обнаружение реплик сервиса по DNS SRV-записи выполняется стандартной связкой:
`commonDNS.NewResolver(errFmt)` реализует интерфейс резолвера
(`LookupSRV(service, proto, name)`), `netDialer.NewNetDialer(errFmt, resolver)`
предоставляет метод `Dialer` для `grpc.WithContextDialer`. Прямые вызовы
`net.LookupSRV` в сервисном коде и захардкоженные списки `ip:port` в
конфигурации запрещены.

```go
// BAD: собственноручный LookupSRV в сервисе — мимо error formatter, мимо единой семантики
addr, err := net.LookupSRV("grpc", "tcp", "tariff-service.wallet")

// GOOD: связка из библиотеки, подключенная к LazyConnection хвостовой опцией
resolverSvc := commonDNS.NewResolver(errFmt)
dialSvc := netDialer.NewNetDialer(errFmt, resolverSvc)

lazyConn := commonGRPCClient.NewLazyConnection(errFmt, "tariff-service.wallet",
    grpc.WithContextDialer(dialSvc.Dialer),
)
```

### Правило 2: Target — домен SRV-записи, порт не указывать

`resolver.Resolve(service, proto, name)` вызывает
`net.LookupSRV(service, proto, name)` и возвращает `cname:port` из первой
SRV-записи. Поэтому в конфиге/вызове находится только доменная часть
(`tariff-service.wallet`); порт в target недопустим — он возьмётся из записи,
а дублирование порта ломает резолвинг. Остальные записи обрабатывает
balancer (см. ниже), сервис не выбирает реплику руками.

```go
// BAD: порт в таргете — LookupSRV получит невалидное имя
lazyConn := commonGRPCClient.NewLazyConnection(errFmt, "tariff-service.wallet:9091",
    grpc.WithContextDialer(dialSvc.Dialer),
)

// GOOD: только доменная часть
lazyConn := commonGRPCClient.NewLazyConnection(errFmt, "tariff-service.wallet",
    grpc.WithContextDialer(dialSvc.Dialer),
)
```

## Балансировка нагрузки

### Правило 1: Balancer — round_robin_crypto_bundle, не стандартный round_robin

Пакет `pkg/client/roundrobin` в `init` регистрирует balancer с именем
`round_robin_crypto_bundle`: round-robin по READY SubConn c `HealthCheck: true`
(фильтрация по grpc_health_v1) и случайной стартовой позицией пикера
(защита от «стада» на первый бэкенд при перестройке пикера). Имя берётся только
из константы `roundRobin.BalancerName`.

```go
// BAD: стандартный round_robin — без health-фильтрации и со стартом на первом бэкенде
serviceConfig := `{"loadBalancingConfig": [{"round_robin": {}}]}`

// GOOD: библиотечный balancer по константе roundRobin.BalancerName
serviceConfig := fmt.Sprintf(`{"loadBalancingConfig": [{"%s": {}}]}`, roundRobin.BalancerName)
```

### Правило 2: Health-фильтрация balancer'а требует health-сервис на сервере

Базовый balancer собран с `base.Config{HealthCheck: true}`: SubConn
переходит в READY только когда сервер отвечает `SERVING` по
`grpc.health.v1.Health`. Сервер без зарегистрированного health-сервиса для
такого клиента означает «нет доступных субсоединений» (`ErrNoSubConnAvailable`
у каждого вызова). Комбинация обязательна в обе стороны: клиент с
`round_robin_crypto_bundle` — только к серверам из
[server_healthcheck.md](./server_healthcheck.md), и наоборот.

```go
// GOOD: полная штатная связка — SRV-резолвер + balancer + health-фильтрация
resolverSvc := commonDNS.NewResolver(errFmt)
dialSvc := netDialer.NewNetDialer(errFmt, resolverSvc)

serviceConfig := fmt.Sprintf(`{"loadBalancingConfig": [{"%s": {}}]}`, roundRobin.BalancerName)

lazyConn := commonGRPCClient.NewLazyConnection(errFmt, "tariff-service.wallet",
    grpc.WithContextDialer(dialSvc.Dialer),
    grpc.WithDefaultServiceConfig(serviceConfig),
)
```

## Unix socket (сайткары и локальные агенты)

### Правило 1: Локальная связность — socketDialer с ротацией сокетов

Для общения с сайдкаром/агентом на хосте используется
`unixDialer.NewUnitFileSocketDialer(logger, errFmt, dirName, filePattern)`:
он обходит набор socket-файлов по glob-паттерну с ротацией между попытками
подключения. Один зафиксированный путь сокета и «ручной» `net.Dial("unix", …)`
в обход grpc запрещены: при перезапуске сайдкара меняется имя файла, и такой
клиент остаётся с мёртвым путём.

```go
// BAD: один захардкоженный путь — перезапуск сайдкара с новым сокетом ломает клиента
conn, err := net.Dial("unix", "/var/run/bc-sidecar/api.sock")

// GOOD: dialer с ротацией по паттерну; Prepare возвращает готовый DialCallback
dialler := unixDialer.NewUnitFileSocketDialer(logger, errFmt, "/var/run/bc-sidecar", "api-*.sock")

dialFn, err := dialler.Prepare()
if err != nil {
    return errFmt.ErrorOnly(err)
}

lazyConn := commonGRPCClient.NewLazyConnection(errFmt,
    "passthrough:///unix:///var/run/bc-sidecar/api.sock",
    grpc.WithContextDialer(dialFn),
)
```

### Правило 2: Prepare() вызывать в рантайме, не в init пакета

Внутри `prepare()` dialer крутит бесконечный цикл с паузой 1 секунду, пока в
каталоге не появятся файлы, подходящие по паттерну. Вызов на уровне `init()`
или package-level переменной блокирует старт процесса без возможности
отмены. `Prepare()` вызывается в фазе инициализации сервис-компонента, где
есть контекст запуска; `logger` (`*slog.Logger` из
`bc-wallet-common-lib-logger`) обязателен — dialer пишет в него ошибки
перебора сокетов.

```go
// BAD: Prepare в init — бесконечный цикл блокирует старт процесса
func init() {
    dialler = unixDialer.NewUnitFileSocketDialer(logger, errFmt, "/var/run/bc-sidecar", "api-*.sock")

    cb, _ = dialler.Prepare()
}

// GOOD: инициализация в конструкторе сервис-компонента
func NewSidecarClientService(logger *slog.Logger, errFmtSvc errorFormatterService) (*sidecarClientService, error) {
    dialler := unixDialer.NewUnitFileSocketDialer(logger, errFmtSvc, "/var/run/bc-sidecar", "api-*.sock")

    dialFn, err := dialler.Prepare()
    if err != nil {
        return nil, errFmtSvc.ErrorOnly(err)
    }

    return &sidecarClientService{
        conn: commonGRPCClient.NewLazyConnection(errFmtSvc,
            "passthrough:///unix:///var/run/bc-sidecar/api.sock",
            grpc.WithContextDialer(dialFn),
        ),
    }, nil
}
```
