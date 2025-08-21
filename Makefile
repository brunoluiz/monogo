.PHONY: lint test all

lint:
	docker buildx bake lint

test:
	docker buildx bake test

all: lint test
