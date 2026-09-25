# Лицензирование кода. Правила оформления информации

---
**Цель**: Регламентировать права на код и лицензирование проектов
**Категория правил**: Layouts, license
**Краткое содержание:** Правила лицензирования, copyright info, оформление заголовков в файлах
---

## Введение

Стандратной практикой при разработке програмного обеспечения является
закрепление прав на владение кодовой базой. Для всех репозиториев в рамках
проекта crypto-bundle должна быть закреплена лицензия и информация о правообладателе.

## Правила

Фактически все сводится к следующим ключевым пунктам:
- Правами владения на код является заказчик в лице {CompanyName}.
- Используемая линцензия должна быть отмечена в следующих файлах:
  - `README.md` в секции `License` - шаблонный текст представлен в [README.md](./examples/license/README.md).
  - `LICENSE.txt` -  шаблонный текст представлен в [LICENSE.txt](./examples/license/LICENSE.txt).
  - Все `*.go` файлы внутри проекта - шаблонный текст представлен в [cmd.go](./examples/license/cmd.go).
  - В файлах `Chart.yaml` при описании Helm-chart'a в репозитории - указать `crypto-bundle Limited Proprietary License`.
- Должен быть добавлен файл `.golangci.yml` в корне репозитория с правилами для линтера `goheader`.
- Локально у разработчика должна быть установлена утилита [copywrite](https://github.com/hashicorp/copywrite).
- Должен быть добавлен файл `.copywrite.hcl` в корне репозитория с конфигурацией для утилиты [copywrite](https://github.com/hashicorp/copywrite).
- Функционал автоматического добавления баннера с текстом лицензии в исходные коды должен
  быть добавлен в `Makefile` файл как отдельная цель.

#### Текст

Стандартный текст о правах на интелектуальную собственность выглядит следующим образом

```
Copyright (C) 2025 crypto-bundle Limited - All Rights Reserved

Source codes of stored in this repository are protected under international copyright law. All rights
reserved and protected by the copyright holders.

Files contained in this repository are confidential and only available to authorized individuals with the
permission of the copyright holders. If you encounter this file and do not have
permission, please contact the copyright holders and delete this file.
```
