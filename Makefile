.PHONY: backend frontend dev build-frontend run-backend-with-build docker docker-build docker-run docker-stop docker-logs docker-publish clean

# ============================================
# Local Development
# ============================================

# Run Go backend against live AWS using your local AWS CLI config.
backend:
	cd backend && COMMAND_CONFIG_PATH=./command-config.json go run ./cmd/server

# Run React frontend via Vite dev server (proxies /api to localhost:8080).
frontend:
	cd frontend && npm install && npm run dev

# Run backend and frontend together for local development.
# Note: backend runs in the background; stop it manually when you're done.
dev:
	cd backend && COMMAND_CONFIG_PATH=./command-config.json go run ./cmd/server &
	cd frontend && npm install && npm run dev

# Build the React frontend (outputs to frontend/dist).
build-frontend:
	cd frontend && npm install && npm run build

# Serve the built frontend using the Go backend.
run-backend-with-build: build-frontend
	cd backend && STATIC_DIR=../frontend/dist COMMAND_CONFIG_PATH=./command-config.json go run ./cmd/server

# ============================================
# Docker
# ============================================

# Build and run Docker container (uses run.sh)
docker:
	./run.sh

# Build Docker image only
docker-build:
	docker build -t aws-local-dashboard .

# Run Docker container (assumes image is already built)
docker-run:
	@mkdir -p data
	docker run -d \
		--name aws-dashboard \
		-p 8080:8080 \
		-v ~/.aws:/root/.aws:ro \
		-v $(PWD)/data:/app/data \
		aws-local-dashboard
	@echo "Dashboard running at http://localhost:8080"

# Stop and remove Docker container
docker-stop:
	docker stop aws-dashboard 2>/dev/null || true
	docker rm aws-dashboard 2>/dev/null || true
	@echo "Container stopped and removed"

# View Docker container logs
docker-logs:
	docker logs -f aws-dashboard

# Publish to Docker Hub (set DOCKER_USERNAME env var)
docker-publish:
	@if [ -z "$(DOCKER_USERNAME)" ]; then \
		echo "Error: DOCKER_USERNAME is required. Usage: make docker-publish DOCKER_USERNAME=yourusername"; \
		exit 1; \
	fi
	./publish.sh

# ============================================
# Cleanup
# ============================================

# Clean build artifacts
clean:
	rm -rf frontend/dist
	rm -rf frontend/node_modules
	rm -rf data/.aws-local-dashboard-profiles.json
	docker stop aws-dashboard 2>/dev/null || true
	docker rm aws-dashboard 2>/dev/null || true
	docker rmi aws-local-dashboard 2>/dev/null || true
	@echo "Cleaned up build artifacts and Docker resources"
