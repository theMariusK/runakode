# ---
all: build build_images

build:
	@echo "Building the application..."
	go build -o ./runakode
	@echo "Application successfully built."

build_images:
	@echo "Building Docker images..."
	docker build -t python-runner ./sandboxes/python/
	docker build -t go-runner ./sandboxes/go/
	@echo "Docker images successfully built."

clean:
	@echo "Cleaning the project..."
	@echo "Removing the application..."
	[ -f ./runakode ] && rm -f ./runakode || true
	@echo "Application removed."
	@echo "Removing built Docker images..."
	docker image rm python-runner
	docker image rm go-runner
	@echo "Docker images removed."
