up:
	docker-compose -f docker/docker-compose.yaml up -d
.PHONY: up

down:
	docker-compose -f docker/docker-compose.yaml down
.PHONY: down