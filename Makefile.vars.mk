## These are some common variables for Make

PROJECT_ROOT_DIR = .
PROJECT_NAME ?= metrics-collector
PROJECT_OWNER ?= vshn

## BUILD:go
BIN_FILENAME ?= $(PROJECT_NAME)

## BUILD:docker
DOCKER_CMD ?= docker

IMG_TAG ?= latest
# Image URL to use all building/pushing image targets
CONTAINER_IMG ?= ghcr.io/$(PROJECT_OWNER)/$(PROJECT_NAME):$(IMG_TAG)

ANTORA_PREVIEW_CMD ?= $(DOCKER_CMD) run --rm --publish 35729:35729 --publish 2020:2020 --volume "${PWD}/.git":/preview/antora/.git --volume "${PWD}/docs":/preview/antora/docs docker.io/vshn/antora-preview:3.0.1.1 --style=syn --antora=docs
