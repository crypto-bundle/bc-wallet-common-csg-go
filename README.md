# Документация проекта `crypto-bundle`

---
**Цель**: Главный навигационный индекс документации
**Категория правил**: Индекс
**Краткое содержание:** Переход ко всем разделам документации с относительными ссылками
---
> Все ссылки относительны корню репозитория (`./`). Для прямого просмотра через GitHub используйте полные пути вида: `https://github.com/crypto-bundle/bc-wallet-common-csg-go/blob/master/docs/<путь>`

## Настройка окружения (`setup/`)
- [Настройка Git](./setup/git_setup.md)

## Git Flow (`git-flow/`)
- [Правила работы с Git](./git-flow/git_flow.md)

## ToolKit (`toolkit/`)
- [Обзор ToolKit](./toolkit/overview.md)
- [Copywrite](./toolkit/copywrite.md)
- [GolangCI Lint](./toolkit/golangci-lint.md)
- [Migrator](./toolkit/migrator.md)

## Code Review (`review/`)
- [GitHub Code Review](./review/github_code_review.md)

## Repository Layouts (`repository-layouts/`)
- [Обзор структурных правил](./repository-layouts/overview.md)
- [Код приложения](./repository-layouts/app/code.md)
- [Деплоймент](./repository-layouts/app/deployment.md)
- [Миграции БД](./repository-layouts/migrations/index.md)
- [Конфигурационные файлы](./repository-layouts/configuration-files.md)
- [Линтеры](./repository-layouts/linters.md)
- [Билд-скрипты](./repository-layouts/builders.md)
- [CI/CD](./repository-layouts/cicd.md)
- [Лицензирование](./repository-layouts/license.md)
- [Документация](./repository-layouts/documentation.md)

## Разработка приложений (`app-development/`)
- [Общие правила Go](./app-development/overview.md)
- [Архитектура](./app-development/architecture/)
  - [Структура пакетов](./app-development/architecture/package_structure.md)
  - [main.go](./app-development/architecture/main_go.md)
  - [Структура файлов](./app-development/architecture/file_structure.md)
- [Конфигурация](./app-development/config/index.md)
- [Ошибки](./app-development/errors/)
  - [Коды ошибок](./app-development/errors/codes.md)
  - [Обработка ошибок](./app-development/errors/handling.md)
- [HTTP](./app-development/http/index.md)
- [gRPC](./app-development/grpc/index.md)
- [PostgreSQL](./app-development/postgresql/)
  - [Подключение к БД](./app-development/postgresql/index.md)
  - [INSERT](./app-development/postgresql/insert.md)
  - [SELECT](./app-development/postgresql/select.md)
  - [UPDATE](./app-development/postgresql/update.md)
  - [DELETE](./app-development/postgresql/delete.md)
  - [Транзакции](./app-development/postgresql/transactions.md)
  - [Data Layer](./app-development/postgresql/data_layer.md)
- [NATS](./app-development/nats/index.md)
- [Processing](./app-development/processing/index.md)

## Процессы (`processes/`)
- [Периметры ответственности](./processes/perimeter_rules.md)

## Диаграммы (`diagrams/`)
- [Integration Branch](./diagrams/integration-branch.puml)

## Примеры (`examples/`)
- [License Example](./examples/license/)
  - [README](./examples/license/README.md)
  - [cmd.go](./examples/license/cmd.go)
  - [LICENSE.txt](./examples/license/LICENSE.txt)
- [Env Example](./examples/env/)
  - [env-hdwallet-controller-manager-api-example.env](./examples/env/env-hdwallet-controller-manager-api-example.env)
