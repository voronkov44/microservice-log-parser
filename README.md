# microservice-log-parser

`microservice-log-parser` - тестовое задание "Парсер логов" на Go. Проект принимает путь до архива, файла или директории из `data/`, парсит данные о логах, сохраняет результат в PostgreSQL, строит топологию и отдаёт REST API для получения информации о логах, узлах, портах и топологии.

Проект сделан как набор отдельных сервисов:

- `app` - внешний REST API и gateway/orchestrator;
- `parser` - чтение и парсинг файлов из `data/`;
- `repository` - работа с PostgreSQL, миграции и транзакционное сохранение;
- `topology` - построение topology view по сохранённым данным;
- `proto` - gRPC-контракты между внутренними сервисами.


Архитектурно проект следует стилю Hexagonal Architecture / Ports and Adapters: в каждом сервисе есть `core`- слой с моделями, интерфейсами и бизнес-логикой, а инфраструктурные детали вынесены в adapters.

---

## Стек

- Go 1.26
- REST API в `app` на `net/http`
- gRPC / protobuf
- PostgreSQL 16
- DB adapter repository service `sqlx` + `pgx`
- Docker Compose
- Логирование через `log/slog`
- Makefile для автоматизации
- unit / smoke / e2e тесты

## Быстрый запуск

### 1. Клонирование

```bash
git clone https://github.com/voronkov44/microservice-log-parser.git
cd microservice-log-parser
```

### 2. Настройка окружения

```bash
cp .env.example .env
```

Архивы и входные логи должны лежать в `data/`. В репозитории `data/*` игнорируется, кроме `data/.gitkeep`, поэтому после свежего clone тестовый архив может отсутствовать. Положите файл, например:

```text
data/log.zip
```

В текущем локальном примере `data/log.zip` содержит:

```text
log/ibdiagnet2.db_csv
log/ibdiagnet2.sharp_an_info
```

### 3. Запуск через Docker Compose

```bash
make up
```

Посмотреть логи всех сервисов:

```bash
make logs
```

Посмотреть логи конкретного сервиса:

```bash
make logs SERVICE=app
```

Проверить контейнеры:

```bash
make ps
```

### 4. Healthcheck

```bash
curl http://localhost:8080/healthz
```

Пример ответа:

```json
{
  "replies": {
    "parser": "ok",
    "repository": "ok",
    "topology": "ok"
  }
}
```

### 5. Запуск парсинга

```bash
curl -X POST http://localhost:8080/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"path":"log.zip"}'
```

Также parser принимает Docker-путь внутри volume:

```bash
curl -X POST http://localhost:8080/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"path":"/data/log.zip"}'
```

Пример ответа:

```json
{
  "log_id": 1,
  "status": "parsed",
  "nodes_count": 5,
  "ports_count": 151,
  "nodes_info_count": 4
}
```

---

## Конфигурация

Пример окружения лежит в `.env.example`. Compose подхватывает `.env` через Makefile:

```make
COMPOSE := ${container_runtime} compose --env-file $(ENV_FILE)
```

Сервисы читают `config.yaml` через `cleanenv`; env-переменные из `config.go` могут переопределять значения из YAML.

### Переменные из `.env.example`

| Variable | Example | Где используется | Описание |
| --- | --- | --- | --- |
| `POSTGRES_USER` | `postgres` | `compose.yaml`, PostgreSQL, repository DSN | Пользователь PostgreSQL |
| `POSTGRES_PASSWORD` | `password` | `compose.yaml`, PostgreSQL, repository DSN | Пароль PostgreSQL |
| `POSTGRES_DB` | `postgres` | `compose.yaml`, PostgreSQL, repository DSN | Имя базы |
| `DATABASE_URL` | `postgres://postgres:password@postgres:5432/postgres?sslmode=disable` | repository config для ручного запуска | Полная строка подключения к PostgreSQL. В compose для repository собирается из `POSTGRES_*` |
| `PGADMIN_DEFAULT_EMAIL` | `admin@test.com` | `pgadmin` | Логин pgAdmin |
| `PGADMIN_DEFAULT_PASSWORD` | `password` | `pgadmin` | Пароль pgAdmin |
| `LOG_LEVEL` | `DEBUG` | все сервисы | Уровень логирования |
| `PORT` | `8080` | `app` | HTTP-порт внешнего REST API. Если задан, app слушает `:<PORT>` |
| `APP_LOG_FILE_PATH` | `/logs/app.log` | `compose.yaml -> LOG_FILE_PATH` | Путь к файлу логов app внутри контейнера |

### Переменные из `config.go`

| Variable | Service | Default | Описание |
| --- | --- | --- | --- |
| `LOG_LEVEL` | app/parser/repository/topology | `DEBUG` | Уровень логирования. Внутренние сервисы ожидают `DEBUG`, `INFO` или `ERROR` |
| `LOG_FILE_PATH` | app | `logs/app.log` | Файл логов app; compose передаёт его из `APP_LOG_FILE_PATH` |
| `APP_ADDRESS` | app | `localhost:8080` | HTTP address app, если не задан `PORT` |
| `PORT` | app | пусто | Приоритетный способ задать порт app; превращается в `:<PORT>` |
| `APP_TIMEOUT` | app | `5s` | Timeout для REST handlers при вызове core/gRPC |
| `APP_READ_HEADER_TIMEOUT` | app | `5s` | `ReadHeaderTimeout` HTTP-сервера |
| `APP_SHUTDOWN_TIMEOUT` | app | `10s` | Timeout graceful shutdown HTTP-сервера |
| `PARSER_GRPC_ADDRESS` | app | `localhost:8081` | gRPC address parser service |
| `REPOSITORY_GRPC_ADDRESS` | app/topology | `localhost:8082` | gRPC address repository service |
| `TOPOLOGY_GRPC_ADDRESS` | app | `localhost:8083` | gRPC address topology service |
| `PARSER_ADDRESS` | parser | `localhost:8081` | Address gRPC-сервера parser |
| `DATA_DIR` | parser | `../data` | Базовая директория входных файлов |
| `REPOSITORY_ADDRESS` | repository | `localhost:8082` | Address gRPC-сервера repository |
| `DB_ADDRESS` | repository | `postgres://postgres:password@localhost:5432/postgres?sslmode=disable` | DB DSN из `config.yaml`/env |
| `DATABASE_URL` | repository | пусто | Если задан, переопределяет `DB_ADDRESS` |
| `TOPOLOGY_ADDRESS` | topology | `localhost:8083` | Address gRPC-сервера topology |

### Адреса в Docker Compose

| Service | External port | Internal address |
| --- | --- | --- |
| `app` | `8080:8080` | `PORT=8080` |
| `parser` | `28081:8080` | `PARSER_ADDRESS=:8080` |
| `repository` | `28082:8080` | `REPOSITORY_ADDRESS=:8080` |
| `topology` | `28083:8080` | `TOPOLOGY_ADDRESS=:8080` |
| `postgres` | `5432:5432` | `postgres:5432` |
| `pgadmin` | `18888:80` | `pgadmin:80` |

`pgAdmin` и проброс PostgreSQL наружу нужны для локальной разработки и диагностики.

---

## REST API

Основное задание закрывают эти endpoint-ы:

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Проверить доступность внутренних сервисов |
| `POST` | `/api/v1/parse` | Создать log, распарсить файл из `data/`, сохранить результат |
| `GET` | `/api/v1/log/{log_id}` | Получить metadata сохранённого лога |
| `GET` | `/api/v1/topology/{log_id}` | Получить topology view по parsed log |
| `GET` | `/api/v1/node/{node_id}` | Получить данные одного node |
| `GET` | `/api/v1/port/{node_id}` | Получить ports конкретного node |

Дополнительные debug/data endpoint-ы:

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/api/v1/log/{log_id}/nodes` | Получить все nodes по `log_id` |
| `GET` | `/api/v1/log/{log_id}/ports` | Получить все ports по `log_id` |

Параметры для data endpoint-ов:

| Query parameter | Endpoint-ы | Default | Описание |
| --- | --- | --- | --- |
| `include_raw=true` | `/api/v1/node/{node_id}`, `/api/v1/port/{node_id}`, `/api/v1/log/{log_id}/nodes`, `/api/v1/log/{log_id}/ports` | `false` | Добавить `raw_json` в ответ |
| `limit` | `/api/v1/port/{node_id}`, `/api/v1/log/{log_id}/ports` | `100` | Размер страницы, максимум `500` |
| `offset` | `/api/v1/port/{node_id}`, `/api/v1/log/{log_id}/ports` | `0` | Смещение |

`raw_json` не возвращается по умолчанию. Он отдаётся только при `include_raw=true`, чтобы обычные ответы не раздувались.

---

## Примеры REST API

### Healthcheck

```bash
curl http://localhost:8080/healthz
```

Ответ:

```json
{
  "replies": {
    "parser": "ok",
    "repository": "ok",
    "topology": "ok"
  }
}
```

Если один из внутренних сервисов недоступен, healthcheck вернёт `503 Service Unavailable`, а соответствующий сервис будет отмечен как `unavailable`.

### Parse log

```bash
curl -X POST http://localhost:8080/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"path":"log.zip"}'
```

Request body:

```json
{
  "path": "log.zip"
}
```

Ответ:

```json
{
  "log_id": 1,
  "status": "parsed",
  "nodes_count": 5,
  "ports_count": 151,
  "nodes_info_count": 4
}
```

`path` должен указывать на файл или директорию внутри `DATA_DIR`. В Docker Compose это volume `./data:/data`, поэтому для файла `data/log.zip` можно передавать `log.zip` или `/data/log.zip`.

### Get log metadata

```bash
curl http://localhost:8080/api/v1/log/1
```

Ответ:

```json
{
  "id": 1,
  "file_path": "log.zip",
  "status": "parsed",
  "nodes_count": 5,
  "ports_count": 151,
  "uploaded_at": "2026-05-15T04:20:30Z",
  "parsed_at": "2026-05-15T04:20:31Z"
}
```

Для failed log в ответе также появится поле `error`.

### Get topology

```bash
curl http://localhost:8080/api/v1/topology/1
```

Ответ:

```json
{
  "log_id": 1,
  "summary": {
    "nodes_count": 5,
    "ports_count": 151,
    "edges_count": 4,
    "hosts_count": 1,
    "switches_count": 4
  },
  "nodes": [
    {
      "id": 1,
      "log_id": 1,
      "node_guid": "node-guid-1",
      "node_desc": "Host A",
      "node_type": 1,
      "node_kind": "host",
      "declared_ports_count": 2,
      "parsed_ports_count": 2,
      "serial_number": "SN-001",
      "product_name": "Host Adapter"
    }
  ],
  "groups": [
    {
      "name": "Hosts",
      "kind": "host",
      "node_ids": [1],
      "node_guids": ["node-guid-1"]
    },
    {
      "name": "Switches",
      "kind": "switch",
      "node_ids": [2, 3, 4, 5],
      "node_guids": ["switch-guid-1", "switch-guid-2", "switch-guid-3", "switch-guid-4"]
    }
  ],
  "edges": [
    {
      "source_node_id": 1,
      "source_node_guid": "node-guid-1",
      "source_port_num": 1,
      "source_port_guid": "port-guid-1",
      "target_node_id": 2,
      "target_node_guid": "switch-guid-1",
      "target_port_num": 1,
      "target_port_guid": "switch-port-guid-1",
      "relation": "inferred_host_switch",
      "link_width_active": 4,
      "link_speed_active": 100,
      "port_state": 4
    }
  ]
}
```

Важно: `edges` являются эвристическими связями, построенными по доступным данным `nodes` и `ports`. Это полезное приближение топологии и bonus к ТЗ, но не гарантированная физическая схема сети.

### Get node details

```bash
curl http://localhost:8080/api/v1/node/1
```

Ответ:

```json
{
  "id": 1,
  "log_id": 1,
  "node_guid": "node-guid-1",
  "node_desc": "Host A",
  "node_type": 1,
  "node_kind": "host",
  "num_ports": 2,
  "class_version": 1,
  "base_version": 2,
  "system_image_guid": "system-image-guid-1",
  "port_guid": "port-guid-1",
  "info": {
    "id": 1,
    "node_id": 1,
    "node_guid": "node-guid-1",
    "serial_number": "SN-001",
    "part_number": "PN-001",
    "revision": "A1",
    "product_name": "Host Adapter"
  }
}
```

С `raw_json`:

```bash
curl "http://localhost:8080/api/v1/node/1?include_raw=true"
```

Фрагмент ответа:

```json
{
  "id": 1,
  "node_guid": "node-guid-1",
  "raw_json": "{\"node_guid\":\"node-guid-1\",\"node_desc\":\"Host A\"}",
  "info": {
    "id": 1,
    "raw_json": "{\"serial_number\":\"SN-001\"}"
  }
}
```

### Get ports by node

```bash
curl "http://localhost:8080/api/v1/port/1?limit=2&offset=0"
```

Ответ:

```json
{
  "count": 2,
  "total": 2,
  "limit": 2,
  "offset": 0,
  "ports": [
    {
      "id": 1,
      "log_id": 1,
      "node_id": 1,
      "node_guid": "node-guid-1",
      "port_guid": "port-guid-1",
      "port_num": 1,
      "lid": 101,
      "local_port_num": 1,
      "port_state": 4,
      "port_phy_state": 5,
      "link_width_active": 4,
      "link_speed_active": 100
    }
  ]
}
```

С `raw_json`:

```bash
curl "http://localhost:8080/api/v1/port/1?limit=1&include_raw=true"
```

### Дополнительные endpoint-ы

Эти ручки не входят в основной список ТЗ, но реально есть в `app/main.go` и полезны для просмотра сохранённых данных.

#### Get nodes by log

```bash
curl http://localhost:8080/api/v1/log/1/nodes
```

Ответ:

```json
{
  "count": 2,
  "nodes": [
    {
      "id": 1,
      "log_id": 1,
      "node_guid": "node-guid-1",
      "node_desc": "Host A",
      "node_type": 1,
      "node_kind": "host",
      "num_ports": 2,
      "class_version": 1,
      "base_version": 2,
      "system_image_guid": "system-image-guid-1",
      "port_guid": "port-guid-1"
    }
  ]
}
```

С `raw_json`:

```bash
curl "http://localhost:8080/api/v1/log/1/nodes?include_raw=true"
```

#### Get ports by log

```bash
curl "http://localhost:8080/api/v1/log/1/ports?limit=2&offset=0"
```

Ответ:

```json
{
  "count": 2,
  "total": 151,
  "limit": 2,
  "offset": 0,
  "ports": [
    {
      "id": 1,
      "log_id": 1,
      "node_id": 1,
      "node_guid": "node-guid-1",
      "port_guid": "port-guid-1",
      "port_num": 1,
      "lid": 101,
      "local_port_num": 1,
      "port_state": 4,
      "port_phy_state": 5,
      "link_width_active": 4,
      "link_speed_active": 100
    }
  ]
}
```

С `raw_json`:

```bash
curl "http://localhost:8080/api/v1/log/1/ports?limit=1&include_raw=true"
```

---

## HTTP-статусы и ошибки

Ошибки обычно возвращаются в JSON-виде:

```json
{
  "error": "resource is not found"
}
```

| Status | Когда используется |
| --- | --- |
| `200 OK` | Успешный healthcheck, parse, get log, get topology, get node, get ports |
| `400 Bad Request` | Невалидный JSON, пустой `path`, неизвестные поля, некорректный id, выход path за пределы `data/`, ошибка валидации parser |
| `404 Not Found` | Файл в `data/` не найден, log/node не найден |
| `409 Conflict` | Некорректный статусный переход log или запрос topology для log не в статусе `parsed` |
| `413 Request Entity Too Large` | Body `POST /api/v1/parse` больше 1 MiB |
| `500 Internal Server Error` | Неожиданная внутренняя ошибка |
| `503 Service Unavailable` | Недоступен parser/repository/topology или healthcheck не прошёл |

## API collection
Для удобной ручной проверки API в корне проекта лежит коллекция для Insomnia:
```text
microservice-log-parser-insomnia.yaml
```
Postman-коллекцию я отдельно не добавлял, так как для локальной проверки использую Insomnia. Коллекцию можно импортировать в Insomnia и сразу проверять основные endpoint-ы проекта:

- healthcheck;
- parse log archive;
- get log metadata;
- get topology;
- get node details;
- get node ports;
- дополнительные ручки для просмотра nodes/ports по log_id.

В коллекции используются переменные окружения:
```text
base_url = http://localhost:8080
log_id = 1
node_id = 1
```

---

## Makefile

### Команды из корневого `Makefile`

| Command | Description |
| --- | --- |
| `make up` | Выполнить `down`, затем поднять compose с `--build -d` |
| `make down` | Остановить compose |
| `make clean` | Остановить compose и удалить volumes |
| `make logs` | Смотреть логи всех сервисов |
| `make logs SERVICE=app` | Смотреть логи конкретного сервиса |
| `make ps` | Показать статус compose-сервисов |
| `make run-tests` | Запустить заранее собранный container image `tests:latest` в `--network=host` |
| `make test` | Алиас на `make test-unit` |
| `make test-unit` | Запустить unit-тесты в `log-services` |
| `make test-e2e-sh` | Поднять compose и запустить `tests/e2e.sh` |
| `make test-smoke` | Поднять compose и запустить Go smoke tests из `tests` |
| `make test-e2e` | Алиас на `make test-smoke` |
| `make test-all` | Запустить `lint`, `test-unit`, `test-smoke` |
| `make lint` | Запустить lint в `log-services` |
| `make proto` | Запустить генерацию protobuf через `log-services/Makefile` target `protobuf` |
| `make tools` | Установить dev tools: `protolint`, `goimports`, `grpcurl`, protoc plugins, `golangci-lint v2.4.0` |

Для `make logs SERVICE=...` доступны:

```text
app, parser, repository, topology, postgres, pgadmin
```

### Команды из `log-services/Makefile`

```bash
make -C log-services <target>
```

| Command | Description |
| --- | --- |
| `make -C log-services lint` | Запустить `protolint` и `golint` |
| `make -C log-services protobuf` | Сгенерировать Go-код из `parser.proto`, `repository.proto`, `topology.proto` |
| `make -C log-services protolint` | Запустить `protolint .` |
| `make -C log-services golint` | Запустить `golangci-lint run -E gocritic -v ./...` |
| `make -C log-services fmt` | Запустить `go fmt ./...` |
| `make -C log-services tidy` | Запустить `go mod tidy` |
| `make -C log-services test` | Запустить `go test ./...` |
| `make -C log-services test-unit` | Запустить `go test ./...` |
| `make -C log-services test-integration` | Запустить integration DB tests с build tag `integration` |

Для `test-integration` нужен `TEST_DATABASE_URL`. Если переменная пустая, integration-тесты repository пропускаются.

---

## Структура проекта

```text
microservice-log-parser/
├── compose.yaml
├── data/
│   └── .gitkeep
├── logs/
│   └── .gitkeep
├── log-services/
│   ├── app/
│   │   ├── adapters/
│   │   ├── config/
│   │   ├── core/
│   │   ├── pkg/
│   │   └── main.go
│   ├── parser/
│   │   ├── adapters/
│   │   ├── config/
│   │   ├── core/
│   │   ├── parser/
│   │   └── main.go
│   ├── repository/
│   │   ├── adapters/
│   │   │   ├── db/
│   │   │   └── grpc/
│   │   ├── config/
│   │   ├── core/
│   │   └── main.go
│   ├── topology/
│   │   ├── adapters/
│   │   ├── config/
│   │   ├── core/
│   │   └── main.go
│   ├── proto/
│   │   ├── parser/
│   │   ├── repository/
│   │   └── topology/
│   ├── Dockerfile.app
│   ├── Dockerfile.parser
│   ├── Dockerfile.repository
│   ├── Dockerfile.topology
│   ├── Makefile
│   ├── go.mod
│   └── go.sum
├── tests/
│   ├── e2e.sh
│   ├── smoke_test.go
│   └── go.mod
├── .env.example
├── .gitignore
├── Makefile
└── README.md
```

---

## Архитектура

### Сервисы

| Service | Responsibility |
| --- | --- |
| `app` | Внешний REST API, orchestration layer. Создаёт log, вызывает parser, сохраняет результат через repository, отдаёт REST-ответы |
| `parser` | Читает файл/архив/директорию из `DATA_DIR`, парсит nodes, ports и nodes_info |
| `repository` | Применяет миграции, хранит данные в PostgreSQL, обеспечивает транзакционное сохранение и статусную модель |
| `topology` | Получает snapshot данных из repository и строит summary, groups и inferred edges |
| `proto` | gRPC contracts и generated Go-код для внутренних сервисов |
| `tests` | Smoke/e2e проверки полного REST-сценария |

### Hexagonal Architecture / Ports and Adapters

В проекте core-слой отделён от adapter-слоя:

| Сервис | Core | Adapters |
| --- | --- | --- |
| `app` | `app/core`: orchestration logic, ports `Parser`, `Repository`, `TopologyProvider` | REST handlers, gRPC clients parser/repository/topology |
| `parser` | `parser/core`: service и `Engine` port | gRPC server, parser engine в `parser/parser` |
| `repository` | `repository/core`: service, DB port, модели | gRPC server, PostgreSQL adapter |
| `topology` | `topology/core`: topology building logic, repository port | gRPC server, repository gRPC client |

Сервисы зависят от интерфейсов, а не от конкретной инфраструктуры. Например, `app/core.Service` ничего не знает про protobuf или PostgreSQL: он вызывает `Repository`, `Parser` и `TopologyProvider`. Конкретные gRPC clients находятся в `app/adapters`.

### Внутренние gRPC API

| Proto | Service methods |
| --- | --- |
| `proto/parser/parser.proto` | `Ping`, `Parse` |
| `proto/repository/repository.proto` | `Ping`, `CreateLog`, `SaveParsedLog`, `FailLog`, `GetLog`, `GetNode`, `GetPortsByNode`, `GetNodesByLog`, `GetPortsByLog`, `GetTopologyData` |
| `proto/topology/topology.proto` | `Ping`, `GetTopology` |

Во внутренних gRPC-серверах включена reflection-регистрация.

---

## Flow работы приложения

### Parse flow

```text
POST /api/v1/parse
  -> app validates JSON body and path
  -> app calls repository.CreateLog(path)
  -> repository creates logs row with status = processing
  -> app calls parser.Parse(path)
  -> parser reads archive/file/dir from DATA_DIR
  -> parser returns nodes, ports, nodes_info
  -> app calls repository.SaveParsedLog(log_id, parsed)
  -> repository saves nodes/ports/nodes_info in one transaction
  -> repository updates logs.status to parsed
  -> client receives log_id, status, counts
```

Если parser или save падают после `CreateLog`, app вызывает `repository.FailLog` с отдельным background context и timeout 5 секунд. `FailLog` переводит log в `failed` только из статуса `processing`.

### Topology flow

```text
GET /api/v1/topology/{log_id}
  -> app validates log_id
  -> app calls topology service
  -> topology calls repository.GetTopologyData(log_id)
  -> repository returns log + nodes + ports in repeatable-read snapshot
  -> topology checks log.status == parsed
  -> topology groups hosts/switches/unknown/other kinds
  -> topology builds inferred edges
  -> app returns topology response
```

---

## Parser implementation

Parser работает только с файлами внутри configured `DATA_DIR`. В Docker Compose это `/data`, смонтированная из `./data`.

### Поддерживаемые источники

| Source | Реализация |
| --- | --- |
| Директория | Рекурсивный обход через `filepath.WalkDir` |
| `.zip` | `archive/zip` |
| `.tar` | `archive/tar` |
| `.tar.gz`, `.tgz` | `archive/tar` + `compress/gzip` |
| `.gz` | `compress/gzip`, один распакованный файл |
| Обычный файл | Чтение как один source file |

### Поддерживаемые форматы внутри файлов

| Format | Как определяется | Что делает parser |
| --- | --- | --- |
| `.db_csv` / имя содержит `db_csv` | suffix/name check | Делит файл на секции `START_*` / `END_*`, затем парсит записи |
| `.csv` | suffix `.csv` | Читает CSV, auto-detect `,` или `;` по первой непустой строке |
| key-value sections | fallback | Читает строки `key=value` или `key: value`, секции разделяются пустыми строками или `---` |

Parser классифицирует записи как `node`, `port` или `info` по имени файла и набору полей.

### Что собирается

| Entity | Основные поля |
| --- | --- |
| `Node` | `node_guid`, `node_desc`, `node_type`, `node_kind`, `num_ports`, `class_version`, `base_version`, `system_image_guid`, `port_guid`, `raw_json` |
| `Port` | `node_guid`, `port_guid`, `port_num`, `lid`, `local_port_num`, `port_state`, `port_phy_state`, `link_width_active`, `link_speed_active`, `raw_json` |
| `NodeInfo` | `node_guid`, `serial_number`, `part_number`, `revision`, `product_name`, `raw_json` |

### Валидация и ограничения

| Ограничение | Значение |
| --- | --- |
| Максимальный размер одного файла | `64 MiB` |
| Максимальное количество файлов в архиве/директории | `1000` |
| Максимальный суммарный размер архива/директории | `128 MiB` |
| Body limit для `POST /api/v1/parse` | `1 MiB` |

Дополнительная защита:

- path нормализуется и обязан оставаться внутри `DATA_DIR`;
- абсолютные `/data/...` пути в Docker корректно мапятся на configured data dir;
- повреждённые архивы возвращают ошибку parser;
- некорректные numeric fields возвращают parse error;
- незакрытые или несовпавшие `START_` / `END_` секции возвращают parse error;
- malformed key-value lines возвращают parse error;
- если после парсинга не найдено ни одного node, parser возвращает parse error;
- если `ports` или `nodes_info` ссылаются на неизвестный `node_guid`, parser создаёт node с `node_kind = "unknown"`;
- дубли nodes, node info и ports объединяются на этапе finalize.

`node_kind` берётся из поля, если оно есть. Если поля нет, parser пытается вывести тип из `node_desc` или `node_type`: `host`, `switch` или `unknown`.

---

## Repository implementation

Repository service хранит данные в PostgreSQL и применяет миграции автоматически при старте.

### Схема PostgreSQL

| Table | Назначение |
| --- | --- |
| `logs` | Метаданные загруженного/обработанного файла |
| `nodes` | Узлы, найденные parser service |
| `ports` | Порты узлов |
| `nodes_info` | Дополнительная информация по node |

### Статусная модель `logs`

| Status | Meaning |
| --- | --- |
| `processing` | Log создан, parsing/save ещё не завершены |
| `parsed` | Данные успешно распарсены и сохранены |
| `failed` | Parsing или save завершились ошибкой |

В БД есть `CHECK` constraint:

```sql
CHECK (status IN ('processing', 'parsed', 'failed'))
```

### Constraints и индексы

| Constraint / index | Назначение |
| --- | --- |
| `nodes.log_id -> logs.id ON DELETE CASCADE` | Удаление log удаляет nodes |
| `nodes_info.node_id -> nodes.id ON DELETE CASCADE` | Удаление node удаляет node info |
| `ports.log_id -> logs.id ON DELETE CASCADE` | Удаление log удаляет ports |
| `ports.node_id -> nodes.id ON DELETE CASCADE` | Удаление node удаляет ports |
| `nodes_log_guid_unique` | Уникальный `node_guid` внутри одного log |
| `nodes_info_node_unique` | Один `nodes_info` на node |
| `idx_ports_log_node_port_unique` | Уникальный port по `(log_id, node_id, port_num)` |
| `idx_nodes_log_id`, `idx_ports_log_id`, другие indexes | Быстрые выборки по log/node/guid |

### Транзакционное сохранение

`SaveParsedLog` выполняется в одной транзакции:

```text
BEGIN
  SELECT status FROM logs WHERE id = $1 FOR UPDATE
  validate status == processing
  DELETE old nodes for log_id
  INSERT nodes
  INSERT nodes_info
  INSERT ports
  UPDATE logs SET status='parsed', counts, parsed_at=now()
COMMIT
```

Если любая операция падает, транзакция откатывается. Integration-тесты проверяют, что при duplicate ports log остаётся в `processing`, а данные не сохраняются частично.

### Защита статусных переходов

`SaveParsedLog` разрешён только для `processing` log. Для защиты используется:

```sql
SELECT status
FROM logs
WHERE id = $1
FOR UPDATE
```

`FailLog` тоже разрешён только из `processing`:

```sql
UPDATE logs
SET status = 'failed', error_text = $2, parsed_at = now()
WHERE id = $1 AND status = 'processing'
```

Если log уже `parsed` или `failed`, repository возвращает `ErrInvalidStatus`, который на REST-уровне превращается в `409 Conflict`.

### Consistent snapshot для topology

`GetTopologyData` читает log, nodes и ports в read-only transaction с isolation level `RepeatableRead`. Это даёт topology service согласованный snapshot данных для одного `log_id`.

---

## Topology implementation

Topology service получает `log_id`, берёт snapshot из repository и строит topology response.

Что делает topology:

- проверяет, что `log_id > 0`;
- получает `log`, `nodes`, `ports` через `repository.GetTopologyData`;
- проверяет `log.status == parsed`;
- считает summary: nodes, ports, edges, hosts, switches;
- нормализует `node_kind`;
- добавляет к topology nodes `declared_ports_count` и `parsed_ports_count`;
- прокидывает `serial_number` и `product_name` из `nodes_info`, если они есть;
- строит groups по kind: `host`, `switch`, `unknown`, остальные kind в алфавитном порядке;
- строит inferred edges.

### Groups

Пример group:

```json
{
  "name": "Switches",
  "kind": "switch",
  "node_ids": [2, 3, 4, 5],
  "node_guids": ["switch-guid-1", "switch-guid-2", "switch-guid-3", "switch-guid-4"]
}
```

### Inferred edges

Topology строит edges только по active ports:

```text
port.ID > 0
port.NodeID > 0
port.PortNum > 0
port.PortState == 4
```

Типы inferred-связей:

| Relation | Как строится |
| --- | --- |
| `inferred_host_switch` | Host port соединяется с неиспользованным switch port, сначала по совпадению `link_width_active` и `link_speed_active`, затем fallback на первый свободный |
| `inferred_switch_backbone` | Switches упорядочиваются по id, соседние switches соединяются через первые свободные active switch ports |

Важно: эти `edges` являются эвристическими связями. Они строятся по доступной информации из `ports` и `nodes`, а не по гарантированным физическим link records. Их стоит читать как приближённую топологию.

---

## Graceful shutdown

Все четыре сервиса слушают `SIGINT` и `SIGTERM`.

### app

`app/main.go` использует `signal.NotifyContext` и `http.Server.Shutdown`.

При shutdown:

- HTTP-сервер перестаёт принимать новые запросы;
- активные HTTP-запросы получают время завершиться;
- применяется timeout `APP_SHUTDOWN_TIMEOUT`, по умолчанию `10s`;
- gRPC clients parser/repository/topology закрываются через `Close`;
- файл логов app закрывается cleanup-функцией logger.

### parser / repository / topology

Внутренние gRPC-сервисы:

- слушают `SIGINT` / `SIGTERM`;
- вызывают `grpc.Server.GracefulStop`;
- repository закрывает DB connection;
- topology закрывает gRPC client к repository.

---

## Logging

Логирование реализовано через `log/slog`.

### app logger

`app/pkg/logger` пишет:

- в stdout;
- дополнительно в файл `LOG_FILE_PATH`, если путь задан.

Compose монтирует `./logs:/logs` и передаёт:

```env
LOG_FILE_PATH=/logs/app.log
```

### parser / repository / topology

Внутренние сервисы используют `slog.NewTextHandler(os.Stderr, ...)`. Docker Compose собирает stdout/stderr в обычные container logs.

### Request logging middleware

`app` логирует каждый HTTP-запрос:

| Field | Meaning |
| --- | --- |
| `status` | HTTP status code |
| `method` | HTTP method |
| `path` | URL path |
| `query` | Raw query, если есть |
| `bytes` | Количество записанных bytes |
| `duration` | Длительность запроса |

`Recover` middleware ловит panic, пишет stack trace и возвращает `500 Internal Server Error`.

### Безопасность логов

Repository при старте логирует DB DSN через `maskDSN`: пароль в `postgres://user:password@...` заменяется на `xxxxx`.

---

## Тестирование

### Команды

```bash
make test
make test-unit
make test-smoke
make test-e2e
make test-e2e-sh
make test-all
make lint
```

`make test` и `make test-unit` запускают unit-тесты в `log-services`.

`make test-smoke` поднимает compose и запускает Go smoke tests:

```bash
cd tests && BASE_URL=${BASE_URL:-http://localhost:8080} go test -v ./...
```

`make test-e2e-sh` поднимает compose и запускает shell e2e:

```bash
BASE_URL=${BASE_URL:-http://localhost:8080} ./tests/e2e.sh
```

### Что покрыто тестами

| Area | Файлы | Что проверяется |
| --- | --- | --- |
| app core | `log-services/app/core/service_test.go` | parse flow, порядок `CreateLog -> Parse -> SaveParsedLog`, `FailLog` при ошибках, topology delegation |
| REST handlers | `log-services/app/adapters/rest/handlers_test.go` | parse, get log/node/ports/topology, mapping errors to statuses, pagination, `include_raw`, invalid ids, large body |
| request decode | `log-services/app/pkg/req/decode_test.go` | strict JSON decode, invalid bodies |
| middleware | `log-services/app/adapters/rest/middleware/*_test.go` | logging middleware и recover middleware |
| parser core | `log-services/parser/core/service_test.go` | validation path и delegation to engine |
| parser engine | `log-services/parser/parser/archive_test.go` | zip/tar/tar.gz/gz/plain/dir, archive limits, path validation, strict parsing, unknown nodes |
| repository core | `log-services/repository/core/service_test.go` | bad arguments и propagation DB errors |
| repository startup helpers | `log-services/repository/main_test.go` | маскирование пароля в PostgreSQL DSN |
| repository integration | `log-services/repository/adapters/db/storage_integration_test.go` | DB lifecycle, migrations, FK-related reads, status transitions, rollback on duplicate ports, not found cases |
| topology core | `log-services/topology/core/service_test.go` | parsed-only topology, summary, groups, inferred edges, no active ports |
| smoke/e2e | `tests/smoke_test.go`, `tests/e2e.sh` | полный сценарий через REST |

Smoke/e2e сценарий:

```text
GET /healthz
POST /api/v1/parse
GET /api/v1/log/{log_id}
GET /api/v1/topology/{log_id}
GET /api/v1/node/{node_id}
GET /api/v1/port/{node_id}
negative cases: invalid ids, invalid body, missing file
```

Для repository integration tests нужен `TEST_DATABASE_URL`:

```bash
TEST_DATABASE_URL='postgres://postgres:password@localhost:5432/postgres?sslmode=disable' \
  make -C log-services test-integration
```

Если `TEST_DATABASE_URL` пустой, integration tests пропускаются.

---

## Линт и качество

### Lint

```bash
make lint
```

Запускает:

```text
protolint .
golangci-lint run -E gocritic -v ./...
```

### Protobuf generation

```bash
make proto
```

Под капотом:

```bash
make -C log-services protobuf
```

Генерируются Go-файлы для:

- `proto/parser/parser.proto`;
- `proto/repository/repository.proto`;
- `proto/topology/topology.proto`.

### Formatting

В `log-services/Makefile` есть:

```bash
make -C log-services fmt
make -C log-services tidy
```

---


## Важные замечания

- `data/log.zip` может быть локальным файлом и может не храниться в git: `.gitignore` игнорирует `data/*`, кроме `data/.gitkeep`.
- Чтобы запустить parse, положите архив/лог в `data/` и передайте путь в `POST /api/v1/parse`.
- Parser не разрешает читать файлы за пределами `DATA_DIR`.
- `pgAdmin` на `http://localhost:18888` и проброс PostgreSQL `localhost:5432` предназначены для локальной разработки.
- `POST /api/v1/parse` синхронный: клиент получает ответ после parsing и сохранения в PostgreSQL.
- `edges` в topology являются inferred/эвристическими, а не гарантированной физической схемой сети.
- `raw_json` хранится в БД, но в REST-ответах скрыт по умолчанию и отдаётся только через `include_raw=true`.

---

## Итог

`microservice-log-parser` реализует рабочий микросервисный pipeline для парсинга логов:

- отдельные сервисы `app`, `parser`, `repository`, `topology`;
- REST API снаружи и gRPC внутри;
- core-слой отделён от adapter-слоя;
- PostgreSQL schema с FK, constraints, indexes и automatic migrations;
- транзакционное сохранение parsed log;
- защищённые статусные переходы `processing -> parsed/failed`;
- graceful shutdown для всех сервисов;
- структурные логи через `slog`;
- Docker Compose для локального запуска;
- unit, smoke/e2e tests и lint;
- topology response с groups, summary и inferred edges.
