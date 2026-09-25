# Соглашение по разработке для языка программирование Go

Документация Code Style Guide доступна внутри go-модуля `pkg/csg`:

- [`pkg/csg/docs/app-development/overview.md`](pkg/csg/docs/app-development/overview.md) — Общие правила Go

## Настройка окружения (`setup/`)
- [Настройка Git](pkg/csg/docs/setup/git_setup.md)

## Git Flow (`git-flow/`)
- [Правила работы с Git](pkg/csg/docs/git-flow/git_flow.md)

## ToolKit (`toolkit/`)
- [Обзор ToolKit](pkg/csg/docs/toolkit/overview.md)
- [Copywrite](pkg/csg/docs/toolkit/copywrite.md)
- [GolangCI Lint](pkg/csg/docs/toolkit/golangci-lint.md)
- [Migrator](pkg/csg/docs/toolkit/migrator.md)

## Code Review (`review/`)
- [GitHub Code Review](pkg/csg/docs/review/github_code_review.md)

## Repository Layouts (`repository-layouts/`)
- [Обзор структурных правил](pkg/csg/docs/repository-layouts/overview.md)
- [Код приложения](pkg/csg/docs/repository-layouts/app/code.md)
- [Деплоймент](pkg/csg/docs/repository-layouts/app/deployment.md)
- [Миграции БД](pkg/csg/docs/repository-layouts/migrations/index.md)
- [Конфигурационные файлы](pkg/csg/docs/repository-layouts/configuration-files.md)
- [Линтеры](pkg/csg/docs/repository-layouts/linters.md)
- [Билд-скрипты](pkg/csg/docs/repository-layouts/builders.md)
- [CI/CD](pkg/csg/docs/repository-layouts/cicd.md)
- [Лицензирование](pkg/csg/docs/repository-layouts/license.md)
- [Документация](pkg/csg/docs/repository-layouts/documentation.md)

## Разработка приложений (`app-development/`)
- [Архитектура](pkg/csg/docs/app-development/architecture/)
  - [Структура пакетов](pkg/csg/docs/app-development/architecture/package_structure.md)
  - [main.go](pkg/csg/docs/app-development/architecture/main_go.md)
  - [Структура файлов](pkg/csg/docs/app-development/architecture/file_structure.md)
- [Конфигурация](pkg/csg/docs/app-development/config/index.md)
- [Ошибки](pkg/csg/docs/app-development/errors/)
  - [Коды ошибок](pkg/csg/docs/app-development/errors/codes.md)
  - [Обработка ошибок](pkg/csg/docs/app-development/errors/handling.md)
- [HTTP](pkg/csg/docs/app-development/http/index.md)
- [gRPC](pkg/csg/docs/app-development/grpc/index.md)
- [PostgreSQL](pkg/csg/docs/app-development/postgresql/)
  - [Подключение к БД](pkg/csg/docs/app-development/postgresql/index.md)
  - [INSERT](pkg/csg/docs/app-development/postgresql/insert.md)
  - [SELECT](pkg/csg/docs/app-development/postgresql/select.md)
  - [UPDATE](pkg/csg/docs/app-development/postgresql/update.md)
  - [DELETE](pkg/csg/docs/app-development/postgresql/delete.md)
  - [Транзакции](pkg/csg/docs/app-development/postgresql/transactions.md)
  - [Data Layer](pkg/csg/docs/app-development/postgresql/data_layer.md)
- [NATS](pkg/csg/docs/app-development/nats/index.md)
- [Processing](pkg/csg/docs/app-development/processing/index.md)

## Процессы (`processes/`)
- [Периметры ответственности](pkg/csg/docs/processes/perimeter_rules.md)

## Диаграммы (`diagrams/`)
- [Integration Branch](pkg/csg/docs/diagrams/integration-branch.puml)

## Примеры (`examples/`)
- [License Example](pkg/csg/docs/examples/license/)
  - [README](pkg/csg/docs/examples/license/README.md)
  - [cmd.go](pkg/csg/docs/examples/license/cmd.go)
  - [LICENSE.txt](pkg/csg/docs/examples/license/LICENSE.txt)
- [Env Example](pkg/csg/docs/examples/env/)
  - [env-hdwallet-controller-manager-api-example.env](pkg/csg/docs/examples/env/env-hdwallet-controller-manager-api-example.env)
