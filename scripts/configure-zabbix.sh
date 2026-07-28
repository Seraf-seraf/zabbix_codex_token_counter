#!/bin/sh

set -eu

project_directory=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file="$project_directory/docker/docker-compose.yaml"
sql_file="$project_directory/exporter/scripts/configure-zabbix.sql"

start_services() {
	docker-compose -f "$compose_file" up -d zabbix-server codex-exporter
}

docker-compose -f "$compose_file" stop codex-exporter zabbix-server
trap start_services 0

docker-compose -f "$compose_file" exec -T postgres \
	psql --username zabbix --dbname zabbix --set ON_ERROR_STOP=1 < "$sql_file"

trap - 0
start_services
