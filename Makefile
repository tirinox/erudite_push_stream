# Makefile
# --------
.PHONY: build run docker-build docker-run docker-stop setup redeploy phptest

setup:
	go mod tidy

# Run development server
run:
	go run .

# Build Docker image
build:
	docker build -t erudite_push_stream:latest .

# Stop and delete Docker container if it exists
docker-stop:
	docker stop erudite_push_stream_i || true
	docker rm erudite_push_stream_i || true

# Run Docker container
docker-run:
	docker run -d --name erudite_push_stream_i -p 127.0.0.1:10026:10026 erudite_push_stream:latest

logs:
	docker logs --tail 100 -f erudite_push_stream_i

redeploy:
	git pull
	# stop
	$(MAKE) docker-stop
	$(MAKE) build
	$(MAKE) docker-run


phptest:
	php test.php
