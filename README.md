# Мониторинг метрик Codex в Zabbix

Проект принимает метрики Codex по протоколу OTLP, преобразует их в элементы
Zabbix sender и отправляет в локальный стенд Zabbix 7.4.

## Состав проекта

- `exporter/` — экспортёр OTLP в Zabbix на Go;
- `exporter/scripts/` — SQL для настройки узла и элементов данных экспортёра;
- `docker/` — PostgreSQL, Zabbix server, веб-интерфейс и экспортёр;
- `scripts/` — общие сценарии управления проектом.

## Требования

- Docker;
- Docker Compose;
- `make`.

## Запуск

Запустите контейнеры:

```sh
make up
```

Создайте в базе Zabbix узел и элементы данных:

```sh
make configure
```

Сценарий можно запускать повторно: существующий узел и элементы данных не
дублируются. На время изменения базы сценарий останавливает Zabbix server и
экспортёр, а после применения SQL запускает их снова. Это предотвращает
конкурентное изменение внутренних идентификаторов и заставляет сервер
перечитать конфигурацию.

Веб-интерфейс доступен по адресу <http://localhost:8080>, OTLP gRPC — на
`localhost:4317`, Zabbix sender — на `localhost:10051`.

Остановка стенда:

```sh
make down
```

## Метрики

SQL создаёт узел `codex-wrapper` с отображаемым именем `Codex exporter` в
группе `Applications`. На узле настраиваются активные элементы типа
`Zabbix trapper`:

| Название                       | Ключ                             |
|--------------------------------|----------------------------------|
| Codex input tokens             | `codex.tokens.input`             |
| Codex cached input tokens      | `codex.tokens.cached_input`      |
| Codex cache write input tokens | `codex.tokens.cache_write_input` |
| Codex output tokens            | `codex.tokens.output`            |
| Codex reasoning output tokens  | `codex.tokens.reasoning_output`  |
| Codex total tokens             | `codex.tokens.total`             |

Все элементы хранят беззнаковые целые значения с единицей измерения
`!tokens`, историей за 31 день и трендами за 365 дней.

## Разработка экспортёра

Команды Go выполняются из каталога `exporter`:

```sh
cd exporter
go test ./...
go vet ./...
go run ./cmd/exporter
```

Настройка и устройство экспортёра описаны в
[`exporter/README.md`](exporter/README.md).

## Примечание о SQL

Сценарий изменяет внутренние таблицы Zabbix и рассчитан на схему Zabbix 7.4
из `docker/docker-compose.yaml`. При переходе на другую версию Zabbix SQL
необходимо сверить с её схемой базы данных.
