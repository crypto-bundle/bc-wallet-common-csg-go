# bc-wallet-common-migrator

---
**Цель**: Управлять миграциями базы данных через wrapper над goose
**Категория правил**: ToolKit, миграции
**Краткое содержание:** Библиотека bc-wallet-common-migrator для применения SQL-миграций с доступом к секретам Vault
---
## Описание

`Bc-wallet-common-migrator` это утилита для применения миграций на SQL-Like базы данных.
Приложение основано на библиотеке для миграции - [Goose library](https://github.com/pressly/goose)

## Цель cоздания

Единственная причина по которой был создано данное приложение-обертка - 
добавление возможности получение разнообразных "чувствительных" данных из HashiCorp Vault.
Под чувствительными данными подразумевается:
- логин
- пароль
- сертификат подключения к базе данных

### Установка
```bash
go install github.com/crypto-bundle/bc-wallet-common-migrator/cmd@latest
```
Эта команда установит последнюю версию приложение `bc-wallet-common-migrator` в вашу $GOPATH/bin директорию.

### Использование
```
Usege:
    bc-wallet-common-migrator [OPTIONS] GOOSE_COMMAND
    
Options:
    -dir string
        directory with migration files (default "./migrations")
    -envPath string
        environment file with migration settings
```

Правила организации файлов и директорий для мигратора описаны в документе [Миграции](3_go_repository_layouts_migrations.md).