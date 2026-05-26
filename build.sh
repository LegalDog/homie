#!/bin/bash
set -e

IMAGE_NAME="homie"
IMAGE_TAG="${1:-latest}"
REGISTRY="${2:-ghcr.io/openclaw}"

echo "🏗️  Building ${IMAGE_NAME}..."

# Build the image
docker build -t "${IMAGE_NAME}:${IMAGE_TAG}" .

# Tag for registry
if [ "$IMAGE_TAG" != "latest" ]; then
    docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
fi
docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${REGISTRY}/${IMAGE_NAME}:latest"

echo "✅ Build complete!"
echo ""
echo "📤 Push to registry:"
echo "   docker push ${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
echo "   docker push ${REGISTRY}/${IMAGE_NAME}:latest"
echo ""
echo "🚀 Run locally:"
echo "   docker run -d --name homie -p 8080:8080 -v \$(pwd)/data:/app/data ${IMAGE_NAME}:${IMAGE_TAG}"
