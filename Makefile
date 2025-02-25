.PHONY: init restart bench help

# project name
PROJECT_NAME=course_seckill

# initialize project
init:
	@echo "Initializing go mod..."
	@if [ ! -f "go.mod" ]; then \
		go mod init $(PROJECT_NAME); \
	fi
	go mod tidy
	@echo "Starting Docker services..."
	cd deploy && docker-compose up -d
	@echo "Finished initializing project!"

# run project
run:
	@echo "Initializing project..."
	@if [ ! -f "go.mod" ]; then \
		go mod init $(PROJECT_NAME); \
	fi
	go mod tidy
	@echo "Starting Docker services..."
	cd deploy && docker-compose up -d
	@echo "Waiting for services to start..."
	@sleep 10
	@echo "Running project..."
	go run main.go

# restart all services
restart:
	@echo "Stopping services..."
	@if pgrep $(PROJECT_NAME); then \
		pkill $(PROJECT_NAME); \
	fi
	cd deploy
	docker-compose down -v
	@echo "Restarting services..."
	docker-compose up -d
	@echo "Waiting for services to start..."
	@sleep 10
	go run main.go

	# performance test
bench:
	@echo "Running performance test..."
	cd test && ./test.sh

# help information
help:
	@echo "Available make commands:"
	@echo "make init    - Initialize project (create go.mod, install dependencies, start Docker)"
	@echo "make run     - Run project (init + run project)"
	@echo "make restart  - Restart all services (including Docker and application server)"
	@echo "make bench    - Run performance test"
	@echo "make help     - Display help information"
