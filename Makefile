# Makefile
# --------
.PHONY: build run docker-build docker-run docker-stop setup redeploy phptest logs

APP_NAME=erudite_push_stream
CONTAINER_NAME=erudite_push_stream_i
PORT=10026

setup:
	go mod tidy
	go mod download

# Run development server locally
run:
	go run .

# Build Docker image
build:
	docker build -t $(APP_NAME):latest .

# Stop and remove Docker container if it exists
docker-stop:
	-docker stop $(CONTAINER_NAME)
	-docker rm $(CONTAINER_NAME)

# Run Docker container with optional env file
docker-run:
	@if [ -f config.env ]; then \
		echo "Using config.env"; \
		docker run -d --name $(CONTAINER_NAME) \
			--env-file=config.env \
			--restart=always \
			-p 127.0.0.1:$(PORT):$(PORT) \
			$(APP_NAME):latest; \
	else \
		docker run -d --name $(CONTAINER_NAME) \
			--restart=always \
			-p 127.0.0.1:$(PORT):$(PORT) \
			-e ENABLE_LOG=true \
			$(APP_NAME):latest; \
	fi

# Tail logs
logs:
	docker logs --tail 100 -f $(CONTAINER_NAME)

# Pull + rebuild and redeploy container
redeploy:
	git pull
	$(MAKE) docker-stop
	$(MAKE) build
	$(MAKE) docker-run

# Run legacy PHP test
phptest:
	php test.php
