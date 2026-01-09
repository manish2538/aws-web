#!/bin/bash

# ============================================
# AWS Local Dashboard - Docker Hub Publisher
# ============================================

set -e

# Configuration - UPDATE THESE VALUES
DOCKER_USERNAME="manish2538"
IMAGE_NAME="aws-local-dashboard"
VERSION="${VERSION:-1.0.0}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  AWS Local Dashboard - Docker Publisher${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Docker Hub username is set
if [ -z "$DOCKER_USERNAME" ]; then
    echo -e "${YELLOW}Docker Hub username not set.${NC}"
    read -p "Enter your Docker Hub username: " DOCKER_USERNAME
    if [ -z "$DOCKER_USERNAME" ]; then
        echo -e "${RED}Error: Docker Hub username is required.${NC}"
        exit 1
    fi
fi

FULL_IMAGE_NAME="${DOCKER_USERNAME}/${IMAGE_NAME}"

echo -e "Image: ${BLUE}${FULL_IMAGE_NAME}${NC}"
echo -e "Version: ${BLUE}${VERSION}${NC}"
echo ""

# Check if logged in to Docker Hub
if ! docker info 2>/dev/null | grep -q "Username"; then
    echo -e "${YELLOW}Not logged in to Docker Hub. Logging in...${NC}"
    docker login
fi

# Build the image
echo ""
echo -e "${BLUE}Building Docker image...${NC}"
docker build -t "${IMAGE_NAME}" .

# Tag the image
echo ""
echo -e "${BLUE}Tagging image...${NC}"
docker tag "${IMAGE_NAME}" "${FULL_IMAGE_NAME}:latest"
docker tag "${IMAGE_NAME}" "${FULL_IMAGE_NAME}:${VERSION}"

echo -e "${GREEN}✓ Tagged as ${FULL_IMAGE_NAME}:latest${NC}"
echo -e "${GREEN}✓ Tagged as ${FULL_IMAGE_NAME}:${VERSION}${NC}"

# Push to Docker Hub
echo ""
echo -e "${BLUE}Pushing to Docker Hub...${NC}"
docker push "${FULL_IMAGE_NAME}:latest"
docker push "${FULL_IMAGE_NAME}:${VERSION}"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  ✓ Successfully published to Docker Hub${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Image URL: ${BLUE}https://hub.docker.com/r/${FULL_IMAGE_NAME}${NC}"
echo ""
echo -e "Users can now run:"
echo -e "${YELLOW}docker run -d -p 8080:8080 -v ~/.aws:/root/.aws:ro ${FULL_IMAGE_NAME}${NC}"
echo ""

