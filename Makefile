IMAGE_NAME := my-registry/my-secure-service
TAG := $(shell git rev-parse --short HEAD 2>/dev/null || echo "latest")

.PHONY: all build sbom scan sign keygen

all: build sbom scan sign

build:
	docker build -t $(IMAGE_NAME):$(TAG) .

sbom:
	syft $(IMAGE_NAME):$(TAG) -o cyclonedx-json > sbom.json

scan:
	# Using Trivy as example (ensure trivy is installed)
	trivy image --severity CRITICAL --exit-code 1 --ignore-unfixed $(IMAGE_NAME):$(TAG)

sign:
	cosign sign --key cosign.key $(IMAGE_NAME):$(TAG)
	cosign attest --key cosign.key --type cyclonedx --predicate sbom.json $(IMAGE_NAME):$(TAG)

keygen:
	cosign generate-key-pair
