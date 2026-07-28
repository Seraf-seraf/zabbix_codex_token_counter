# Экспортёр OTLP в Zabbix

Сервис получает метрики OTLP по gRPC через официальную модель OpenTelemetry
Collector `pdata`, запускает первый подходящий специализированный обработчик
(либо универсальный обработчик по умолчанию) и отправляет данные пакетами
по протоколу Zabbix sender.

## Настройка

| Переменная                    | Значение по умолчанию                                  |
|-------------------------------|--------------------------------------------------------|
| `OTLP_LISTEN_ADDR`            | `0.0.0.0:4317`                                         |
| `ZABBIX_SERVER_ADDR`          | `zabbix-server:10051`                                  |
| `ZABBIX_HOST`                 | `codex-wrapper`                                        |
| `ZABBIX_KEY_PREFIX`           | `otel`                                                 |
| `ZABBIX_SEND_TIMEOUT`         | `5s`                                                   |
| `ZABBIX_BATCH_SIZE`           | `100`                                                  |
| `ZABBIX_FLUSH_INTERVAL`       | `1s` (зарезервировано для будущей очереди обработки)   |
| `PROCESSOR_GENERIC_ENABLED`   | `true`                                                 |
| `LOG_LEVEL`                   | `info`                                                 |

Обработчик Codex распознаёт метрику `codex.turn.token_usage` (а также устаревший
вариант без префикса), считывает строковый атрибут `token_type` и отправляет
числовое значение или сумму гистограммы под ключом
`codex.tokens.<token_type>`. Поддерживаются типы `input`, `cached_input`,
`cache_write_input`, `output`, `reasoning_output` и `total`. Точки данных
с неизвестным или отсутствующим типом токена либо без числового значения
записываются в журнал и пропускаются.

Универсальный обработчик сохраняет ключ неизменным и намеренно не добавляет
к нему атрибуты OTLP. Для значений Gauge и Sum используется ключ
`otel.<metric_name>`. Для гистограмм создаются ключи `.count`, `.sum`, `.min`
и `.max`, если соответствующие значения существуют.

## Запуск

```sh
go test ./...
go run ./cmd/exporter
```

Из корневого каталога репозитория:

```sh
docker compose up --build
```

Перед отправкой значений создайте в Zabbix узел сети и элементы данных типа
Zabbix trapper: экспортёр не создаёт их через API Zabbix.

Текущая конфигурация Codex хранится в пользовательском файле
`~/.codex/config.toml`:

```toml
[otel]
environment = "local"
metrics_exporter = { otlp-grpc = {
  endpoint = "http://127.0.0.1:4317"
}}
```
