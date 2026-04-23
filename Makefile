.PHONY: dev dev-up down logs build shell-frontend shell-backend clean help

COMPOSE_DEV  = docker-compose -f develop/docker-compose.yml
COMPOSE_PROD = docker-compose -f production/docker-compose.yml

# ─── Dev ─────────────────────────────────────────────────────────────────────

dev:            ## Запустить dev-окружение (foreground)
	$(COMPOSE_DEV) up

dev-up:          ## Запустить dev-окружение (background)
	$(COMPOSE_DEV) up -d

down:           ## Остановить dev-окружение
	$(COMPOSE_DEV) down

logs:           ## Логи dev-окружения
	$(COMPOSE_DEV) logs -f

# ─── Shells ───────────────────────────────────────────────────────────────────

shell-frontend: ## Shell внутри frontend-контейнера
	$(COMPOSE_DEV) exec frontend sh

shell-backend:  ## Shell внутри backend-контейнера
	$(COMPOSE_DEV) exec backend sh

# ─── Production ──────────────────────────────────────────────────────────────

prod-up:        ## Запустить prod-окружение
	$(COMPOSE_PROD) up -d

prod-down:      ## Остановить prod-окружение
	$(COMPOSE_PROD) down

prod-logs:      ## Логи prod-окружения
	$(COMPOSE_PROD) logs -f

prod-build:     ## Собрать prod-образы
	$(COMPOSE_PROD) build

install-service: ## Установить systemd-сервис (автозапуск при перезагрузке)
	@sed 's|{{PROJECT_DIR}}|$(shell pwd)|g' production/moto-rental.service \
	    | sudo tee /etc/systemd/system/moto-rental.service > /dev/null
	sudo systemctl daemon-reload
	sudo systemctl enable moto-rental
	@echo "Сервис установлен. Запустить сейчас: sudo systemctl start moto-rental"

uninstall-service: ## Удалить systemd-сервис
	sudo systemctl disable moto-rental
	sudo rm -f /etc/systemd/system/moto-rental.service
	sudo systemctl daemon-reload

# ─── Clean ───────────────────────────────────────────────────────────────────

clean:          ## Удалить контейнеры, образы, volumes
	$(COMPOSE_DEV) down --rmi local --volumes
	rm -rf frontend/dist frontend/node_modules backend/tmp

# ─── Help ─────────────────────────────────────────────────────────────────────

help:           ## Показать список команд
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*##"}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
