# go-egts [![GoDoc](https://godoc.org/github.com/LdDl/go-egts?status.svg)](https://godoc.org/github.com/LdDl/go-egts) [![Sourcegraph](https://sourcegraph.com/github.com/LdDl/go-egts/-/badge.svg)](https://sourcegraph.com/github.com/LdDl/go-egts?badge) [![Go Report Card](https://goreportcard.com/badge/github.com/LdDl/go-egts)](https://goreportcard.com/report/github.com/LdDl/go-egts) [![GitHub tag](https://img.shields.io/github/tag/LdDl/go-egts.svg)](https://github.com/LdDl/go-egts/releases) [![Build Status](https://travis-ci.com/LdDl/go-egts.svg?branch=master)](https://travis-ci.com/LdDl/go-egts)

[Английская версия (english version)](README_RU.md)

Библиотека на Go для декодирования и кодирования пакетов формата EGTS (Era Glonass Telematics Standard). Репозиторий также содержит приложение `egts_gateway` - простой сервер TCP на основе этой библиотеки для приёма, сохранения и ретрансляции телеметрии.

## Содержание

- [О проекте](#о-проекте)
- [Установка](#установка)
- [Примеры использования библиотеки](#примеры-использования-библиотеки)
- [Gateway](#gateway)
- [Docker](#docker)
- [Тесты](#тесты)
- [Поддержка](#поддержка)
- [Лицензия](#лицензия)

## О проекте

EGTS расшифровывается как Era Glonass Telematics Standard. Библиотека декодирует и кодирует пакеты, записи и подзаписи EGTS. Подробнее о формате можно узнать из [описания протокола](https://docs.cntd.ru/document/1200095098) и [примеров разбора пакетов](docs_rus).

`egts_gateway` принимает соединения EGTS с опциональной аутентификацией и отправляет пакеты в stdout, файлы с ротацией, другие серверы EGTS (т.е. ретрансляция), RabbitMQ и Valkey/Redis Streams. Можно включить несколько получателей одновременно.

## Установка

### Библиотека Go

```bash
go get github.com/LdDl/go-egts
```

### Установка gateway через Go

```bash
go install github.com/LdDl/go-egts/cmd/egts_gateway@latest
```

### Сборка из исходников

Для сборки достаточно выполить команду из корня репозитория, чтобы собрать бинарный файл для текущей платформы:

```bash
go build -o egts_gateway ./cmd/egts_gateway
```

### Готовые бинарные файлы

Можно скачать заранее подготовленный архив для своей платформы со страницы [последнего релиза](https://github.com/LdDl/go-egts/releases/latest):

| Платформа | Архив |
| --- | --- |
| Linux amd64 | [linux-amd64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/linux-amd64-egts_gateway.tar.gz) |
| Linux arm64 | [linux-arm64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/linux-arm64-egts_gateway.tar.gz) |
| macOS amd64 | [darwin-amd64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/darwin-amd64-egts_gateway.tar.gz) |
| macOS arm64 | [darwin-arm64-egts_gateway.tar.gz](https://github.com/LdDl/go-egts/releases/latest/download/darwin-arm64-egts_gateway.tar.gz) |
| Windows amd64 | [windows-amd64-egts_gateway.zip](https://github.com/LdDl/go-egts/releases/latest/download/windows-amd64-egts_gateway.zip) |

Каждый архив содержит `egts_gateway` или `egts_gateway.exe`. Распакуйте файл в каталог, указанный в `PATH`.

Быстрая установка на Linux amd64:

```bash
curl -fsSL https://github.com/LdDl/go-egts/releases/latest/download/linux-amd64-egts_gateway.tar.gz \
  | sudo tar -xz -C /usr/local/bin egts_gateway
```

Для Linux arm64 необходимо заменить `linux-amd64` на `linux-arm64`. На macOS указать `darwin-amd64` для Intel или `darwin-arm64` для Apple Silicon.

## Примеры использования библиотеки

На примере [egts_server](cmd/egts_server) и [egts_client](cmd/egts_client) можно познакомиться с тем, как использовать библиотеку. Запускать примеры надо в отдельных терминалах:

- Запуск сервера

    ```shell
    go run cmd/egts_server/main.go
    ```

- Запуск клиента

    ```shell
    go run cmd/egts_client/main.go
    ```

После запуска сервера и клиента будет примерно такой вывод:

- На стороне клиента

    ```shell
    go run cmd/egts_client/main.go
    2021/12/16 21:18:15 Response code: {0 0      0 0 0 0 0 0 0 0 0 <nil> 0 0}
    2021/12/16 21:18:15 Packet: {1 0 00 11 0 00 0 11 0 16 1 0 0 0 0 245 0xc0000040c0 4587 0}
    ```

- На стороне сервера

    ```shell
    go run cmd/egts_server/main.go
    2021/12/16 21:18:08 Accept connection on port 8081
    2021/12/16 21:18:15 Calling handleConnection for remote address: [::1]:50840
    2021/12/16 21:18:15 PosData is:
            OID: 825791382 | Longitude: 48.362186 | Latitude: 54.287315 | Time: 2021-12-16 21:18:15.1412741 +0300 MSK m=+7.036938801
    2021/12/16 21:18:15 Result code has been sent to '[::1]:50840'
    ```

`Packet.Encode()` возвращает `([]byte, error)`. Необходимо ошибку проверять перед отправкой закодированных байтов. Кодировщик проверяет флаги и длины пакетов, а также возвращает ошибки из вложенных записей и подзаписей.

## Gateway

Запуск с настройками по умолчанию:

```bash
egts_gateway
```

Сервер слушает `0.0.0.0:8081`, принимает соединения без аутентификации, записывает пакеты в stdout, а логи приложения в stderr. Для остановки нажмите `Ctrl+C`.

### Настройка через TOML

Вот тут есть [пример конфигурации](egts_gateway.toml). Его можно отредактировать и можно явно указать путь к файлу:

```bash
curl -fsSL https://raw.githubusercontent.com/LdDl/go-egts/master/egts_gateway.toml -o egts_gateway.toml
egts_gateway -conf egts_gateway.toml
```

При запуске с `-conf` используются настройки из TOML и значения по умолчанию. Переменные окружения и `.env` игнорируются. Относительные пути к данным отсчитываются от рабочего каталога процесса.

Чтобы требовать аутентификацию входящих соединений, нужно указать `auth_cfg.enabled = true` и задайть `auth_cfg.password`. Для исходящей ретрансляции EGTS аутентификация также опциональна.

### Настройка через переменные окружения

Без `-conf` gateway читает переменные окружения. Если в рабочем каталоге есть `.env`, его значения переопределяют окружение процесса. Доступные переменные перечислены в [.env.example](.env.example).

Например, для сохранения пакетов в файлы с ротацией:

```bash
EGTS_SERVER_PORT=8081 \
EGTS_PACKETS_STDOUT=false \
EGTS_PACKETS_FILE_ENABLED=true \
egts_gateway
```

Для нескольких получателей задаётся список ID через запятую, а ID добавляется в конец имени переменной. Например, такие записи в `.env` включают два направления ретрансляции EGTS:

```dotenv
EGTS_RELAY_IDS=monitoring,backup
EGTS_RELAY_ENABLED_MONITORING=true
EGTS_RELAY_HOST_MONITORING=127.0.0.1
EGTS_RELAY_PORT_MONITORING=8082
EGTS_RELAY_ENABLED_BACKUP=true
EGTS_RELAY_HOST_BACKUP=127.0.0.1
EGTS_RELAY_PORT_BACKUP=8083
```

Для RabbitMQ и Valkey используются `EGTS_RABBITMQ_IDS` и `EGTS_VALKEY_IDS` с тем же правилом добавления ID, например `EGTS_RABBITMQ_PASSWORD_BACKUP` и `EGTS_VALKEY_PASSWORD_BACKUP`. В TOML надо добавлять по одной записи `[[destinations_cfg.egts]]`, `[[destinations_cfg.rabbitmq]]` или `[[destinations_cfg.valkey]]` на каждого получателя.

### Получатели пакетов

| Получатель | Поведение |
| --- | --- |
| stdout | Один объект JSON с данными пакета на строку. |
| Файл | NDJSON в `./data/packets/packets.ndjson` с ротацией по размеру файла, количеству архивов, их возрасту и общему размеру. |
| EGTS | Пересылка пакетов на один или несколько серверов с ожиданием ответов EGTS. |
| RabbitMQ | Публикация сообщений JSON в очереди с сохранением на диск и ожиданием подтверждений публикации. |
| Valkey/Redis | Добавление записей в настроенные потоки Streams с полями `message_id` и `payload`. |

Логирование приложения настраивается отдельно через `logs_cfg`. При записи в файл по умолчанию используется `./data/logs/application.log`. Gateway отклоняет конфигурации, в которых логи приложения, файлы пакетов или дампы очереди используют одно место для записи. У файлов логов и пакетов отдельные настройки ротации.

Объект JSON содержит `record.received_at`, `record.source`, декодированный `record.packet` и `record.raw`. Поле `raw` содержит исходный пакет EGTS, закодированный в Base64. Идентификаторы сообщений RabbitMQ и значения `message_id` в Valkey/Redis сохраняются при повторных попытках доставки и восстановлении из дампа.

Для Valkey/Redis надо оставить `username` пустым, чтобы использовать `AUTH password` (это legacy-режим работы), или указать и `username` и `password` для `AUTH username password` (на основе ACL доступов). Для сервера без аутентификации нужно оставить оба поля пустыми. Удалением старых записей из потока управляет потребитель или администратор сервера.

### Доставка и восстановление

`delivery_cfg.ack_mode` определяет, когда терминал получает подтверждение успешного приёма:

| Режим | Подтверждение |
| --- | --- |
| `queued` | После помещения пакета в очередь в памяти. Используется по умолчанию. |
| `delivered` | После приёма пакета каждым включённым получателем. |

Очередь вмещает до `queue_capacity` пакетов, по умолчанию 1024. При заполнении gateway отклоняет новые пакеты до освобождения места. Повторные попытки доставки выполняются независимо для каждого получателя, поэтому недоступность одного не блокирует доставку остальным.

При остановке ожидающие доставки пакеты сохраняются в `dump_directory`, по умолчанию `./data/queue`. Если ошибки доставки получателю продолжаются в течение `dump_after_seconds`, по умолчанию 60 секунд, gateway сохраняет очередь и завершает работу с ошибкой. При следующем запуске очередь восстанавливается, а доставка оставшимся получателям повторяется. Для восстановления дампа необходимо сохранить ID нужных получателей и оставить их включёнными в конфигурации.

В режиме `queued` подтверждаются данные, находящиеся в памяти. Принудительное завершение процесса или сбой хоста до сохранения дампа может привести к потере пакетов из очереди. Повторная доставка может создавать дубликаты, если получатель принял пакет, но его подтверждение потерялось. Потребители могут использовать постоянный идентификатор сообщения для устранения дубликатов.

Примеры gateway содержат конфигурации и проверки доставки пакетов для [нескольких получателей EGTS](gateway/examples/relay), [RabbitMQ](gateway/examples/rabbitmq) и [Valkey/Redis](gateway/examples/valkey).

## Docker

Образ публикуется как [dimahkiin/egts_gateway](https://hub.docker.com/r/dimahkiin/egts_gateway). Запуск последнего образа без фонового режима:

```bash
docker run --rm --name egts_gateway --stop-timeout 30 \
  -p 127.0.0.1:8081:8081 \
  -v egts_gateway_data:/app/data \
  docker.io/dimahkiin/egts_gateway:latest
```

Именованный том сохраняет файлы пакетов, логи в файлах и дампы очереди. По умолчанию пакеты записываются в stdout. Для приёма соединений с других хостов необходимо заменить проброс порта на `8081:8081`. Для остановки контейнера достаточно классического `Ctrl+C`.

Чтобы использовать скачанный выше файл TOML:

```bash
docker run --rm --name egts_gateway --stop-timeout 30 \
  -p 127.0.0.1:8081:8081 \
  --mount type=bind,source="$(pwd)/egts_gateway.toml",target=/app/egts_gateway.toml,readonly \
  -v egts_gateway_data:/app/data \
  docker.io/dimahkiin/egts_gateway:latest -conf /app/egts_gateway.toml
```

Для настройки через ENV необходимо добавить флаг `--mount type=bind,source="$(pwd)/.env",target=/app/.env,readonly` перед именем образа в первой команде. Gateway загрузит этот файл самостоятельно. Внутри контейнера сервер должен слушать `0.0.0.0:8081`, а данные следует сохранять в `/app/data`.

### Docker Compose

Из корня репозитория достаточно запустить опубликованный образ с конфигурацией TOML из проекта:

```bash
EGTS_GATEWAY_IMAGE=dimahkiin/egts_gateway:latest \
docker compose -f gateway/compose.yaml up --no-build --pull always
```

Для настройки через ENV нужно подготовить `.env` на основе `.env.example`, затем отредактировать его и выполнить команду:

```bash
EGTS_GATEWAY_IMAGE=dimahkiin/egts_gateway:latest \
docker compose -f gateway/compose.yaml -f gateway/compose.env.yaml up --no-build --pull always
```

`EGTS_GATEWAY_PORT` задаёт опубликованный порт хоста, `EGTS_GATEWAY_BIND_HOST` задаёт адрес привязки на хосте, а `EGTS_GATEWAY_CONFIG` выбирает файл TOML. Для собственного файла конфигурации необходимо использовать абсолютный путь. `EGTS_GATEWAY_ENV_FILE` выбирает файл ENV при запуске соответствующего варианта. Том `gateway_data` сохраняется при пересоздании контейнера.

[Compose для инфраструктуры](gateway/infrastructure/compose.yaml) содержит два экземпляра RabbitMQ и два экземпляра Valkey для тестирования. Чтобы запустить их в одной сети Compose с gateway, необходимо включить нужных получателей в `egts_gateway.toml` и указать хосты `rabbitmq_monitoring`, `rabbitmq_backup`, `valkey_monitoring` или `valkey_backup`. Внутри этой сети необходимо использовать порт `5672` для каждого экземпляра RabbitMQ и `6379` для каждого экземпляра Valkey.

```bash
EGTS_GATEWAY_IMAGE=dimahkiin/egts_gateway:latest \
docker compose -f gateway/compose.yaml -f gateway/infrastructure/compose.yaml up --no-build --pull always
```

## Тесты

Запуск всех модульных тестов:

```bash
go test ./...
```

Интеграционные тесты используют инфраструктуру из примера и включаются явно. Для этого необходимо запустить нужные сервисы RabbitMQ или Valkey в одном терминале:

```bash
docker compose -f gateway/infrastructure/compose.yaml up
```

При использовании учётных данных и портов из примера можно запустить интеграционные тесты в другом терминале:

```bash
GO_EGTS_RABBITMQ_INTEGRATION=1 GO_EGTS_VALKEY_INTEGRATION=1 go test ./gateway/...
```

## Поддержка

Если возникли проблемы или вопросы, [создайте issue](https://github.com/LdDl/go-egts/issues/new/choose).

Pull request приветствуются.

## Лицензия

MIT. Подробнее в [LICENSE.md](https://github.com/LdDl/go-egts/blob/master/LICENSE.md).
