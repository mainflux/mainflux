# Copyright (c) Mainflux
# SPDX-License-Identifier: Apache-2.0

MF_DOCKER_IMAGE_NAME_PREFIX ?= mainflux
BUILD_DIR = build
SERVICES = auth users things http coap ws lora influxdb-writer influxdb-reader mongodb-writer \
	mongodb-reader cassandra-writer cassandra-reader postgres-writer postgres-reader timescale-writer timescale-reader cli \
	bootstrap opcua twins mqtt provision certs smtp-notifier smpp-notifier
DOCKERS = $(addprefix docker_,$(SERVICES))
DOCKERS_DEV = $(addprefix docker_dev_,$(SERVICES))
CGO_ENABLED ?= 0
GOARCH ?= amd64
VERSION ?= $(shell git describe --abbrev=0 --tags)
COMMIT ?= $(shell git rev-parse HEAD)
TIME ?= $(shell date +%F_%T)
USER_REPO ?= $(shell git remote get-url origin | sed -e 's/.*\/\([^/]*\)\/\([^/]*\).*/\1_\2/' )
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD  2>/dev/null || git describe --tags --abbrev=0  2>/dev/null )
empty:=
space:= $(empty) $(empty)
# Docker compose project name should follow this guidelines: https://docs.docker.com/compose/reference/#use--p-to-specify-a-project-name
DOCKER_PROJECT ?= $(shell echo $(subst $(space),,$(USER_REPO)_$(BRANCH)) | tr -c -s '[:alnum:][=-=]' '_' | tr '[:upper:]' '[:lower:]')
DOCKER_COMPOSE_COMMANDS_SUPPORTED := up down config
DEFAULT_DOCKER_COMPOSE_COMMAND  := up
GRPC_MTLS_CERT_FILES_EXISTS = 0
DOCKER_PROFILE ?= $(MF_MQTT_BROKER_TYPE)_$(MF_MESSAGE_BROKER_TYPE)
ifneq ($(MF_MESSAGE_BROKER_TYPE),)
    MF_MESSAGE_BROKER_TYPE := $(MF_MESSAGE_BROKER_TYPE)
else
    MF_MESSAGE_BROKER_TYPE=nats
endif

ifneq ($(MF_MQTT_BROKER_TYPE),)
    MF_MQTT_BROKER_TYPE := $(MF_MQTT_BROKER_TYPE)
else
    MF_MQTT_BROKER_TYPE=nats
endif

ifneq ($(MF_ES_STORE_TYPE),)
    MF_ES_STORE_TYPE := $(MF_ES_STORE_TYPE)
else
    MF_ES_STORE_TYPE=nats
endif

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) \
	go build -mod=vendor -tags $(MF_MESSAGE_BROKER_TYPE) --tags $(MF_ES_STORE_TYPE) -ldflags "-s -w \
	-X 'github.com/mainflux/mainflux.BuildTime=$(TIME)' \
	-X 'github.com/mainflux/mainflux.Version=$(VERSION)' \
	-X 'github.com/mainflux/mainflux.Commit=$(COMMIT)'" \
	-o ${BUILD_DIR}/mainflux-$(1) cmd/$(1)/main.go
endef

define make_docker
	$(eval svc=$(subst docker_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg GOARM=$(GOARM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg TIME=$(TIME) \
		--tag=$(MF_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile .
endef

define make_docker_dev
	$(eval svc=$(subst docker_dev_,,$(1)))

	docker build \
		--no-cache \
		--build-arg SVC=$(svc) \
		--tag=$(MF_DOCKER_IMAGE_NAME_PREFIX)/$(svc) \
		-f docker/Dockerfile.dev ./build
endef

ADDON_SERVICES = bootstrap cassandra-reader cassandra-writer certs \
					influxdb-reader influxdb-writer lora-adapter mongodb-reader mongodb-writer \
					opcua-adapter postgres-reader postgres-writer provision smpp-notifier smtp-notifier \
					timescale-reader timescale-writer twins

EXTERNAL_SERVICES = vault prometheus

ifneq ($(filter run%,$(firstword $(MAKECMDGOALS))),)
  temp_args := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  DOCKER_COMPOSE_COMMAND := $(if $(filter $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)), $(filter $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)), $(DEFAULT_DOCKER_COMPOSE_COMMAND))
  $(eval $(DOCKER_COMPOSE_COMMAND):;@)
endif

ifneq ($(filter run_addons%,$(firstword $(MAKECMDGOALS))),)
  temp_args := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  RUN_ADDON_ARGS :=  $(if $(filter-out $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)), $(filter-out $(DOCKER_COMPOSE_COMMANDS_SUPPORTED),$(temp_args)),$(ADDON_SERVICES) $(EXTERNAL_SERVICES))
  $(eval $(RUN_ADDON_ARGS):;@)
endif

ifneq ("$(wildcard docker/ssl/certs/*-grpc-*)","")
GRPC_MTLS_CERT_FILES_EXISTS = 1
else
GRPC_MTLS_CERT_FILES_EXISTS = 0
endif

FILTERED_SERVICES = $(filter-out $(RUN_ADDON_ARGS), $(SERVICES))

all: $(SERVICES)

.PHONY: all $(SERVICES) dockers dockers_dev latest release run run_addons grpc_mtls_certs check_mtls check_certs

clean:
	rm -rf ${BUILD_DIR}

cleandocker:
	# Stops containers and removes containers, networks, volumes, and images created by up
	docker-compose -f docker/docker-compose.yml --profile $(DOCKER_PROFILE) -p $(DOCKER_PROJECT) down --rmi all -v --remove-orphans

ifdef pv
	# Remove unused volumes
	docker volume ls -f name=$(MF_DOCKER_IMAGE_NAME_PREFIX) -f dangling=true -q | xargs -r docker volume rm
endif

install:
	cp ${BUILD_DIR}/* $(GOBIN)

test:
	go test -mod=vendor -v -race -count 1 -tags test $(shell go list ./... | grep -v 'vendor\|cmd')

proto:
	protoc -I. --go_out=. --go_opt=paths=source_relative pkg/messaging/*.proto
	protoc -I. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./*.proto

$(FILTERED_SERVICES):
	$(call compile_service,$(@))

$(DOCKERS):
	$(call make_docker,$(@),$(GOARCH))

$(DOCKERS_DEV):
	$(call make_docker_dev,$(@))

dockers: $(DOCKERS)
dockers_dev: $(DOCKERS_DEV)

define docker_push
	for svc in $(SERVICES); do \
		docker push $(MF_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(1); \
	done
endef

changelog:
	git log $(shell git describe --tags --abbrev=0)..HEAD --pretty=format:"- %s"

latest: dockers
	$(call docker_push,latest)

release:
	$(eval version = $(shell git describe --abbrev=0 --tags))
	git checkout $(version)
	$(MAKE) dockers
	for svc in $(SERVICES); do \
		docker tag $(MF_DOCKER_IMAGE_NAME_PREFIX)/$$svc $(MF_DOCKER_IMAGE_NAME_PREFIX)/$$svc:$(version); \
	done
	$(call docker_push,$(version))

rundev:
	cd scripts && ./run.sh

grpc_mtls_certs:
	$(MAKE) -C docker/ssl users_grpc_certs things_grpc_certs

check_tls:
ifeq ($(GRPC_TLS),true)
	@unset GRPC_MTLS
	@echo "gRPC TLS is enabled"
	GRPC_MTLS=
else
	@unset GRPC_TLS
	GRPC_TLS=
endif

check_mtls:
ifeq ($(GRPC_MTLS),true)
	@unset GRPC_TLS
	@echo "gRPC MTLS is enabled"
	GRPC_TLS=
else
	@unset GRPC_MTLS
	GRPC_MTLS=
endif

check_certs: check_mtls check_tls
ifeq ($(GRPC_MTLS_CERT_FILES_EXISTS),0)
ifeq ($(filter true,$(GRPC_MTLS) $(GRPC_TLS)),true)
ifeq ($(filter $(DEFAULT_DOCKER_COMPOSE_COMMAND),$(DOCKER_COMPOSE_COMMAND)),$(DEFAULT_DOCKER_COMPOSE_COMMAND))
	$(MAKE) -C docker/ssl users_grpc_certs things_grpc_certs
endif
endif
endif

define edit_docker_config
	sed -i "s/MF_MQTT_BROKER_TYPE=.*/MF_MQTT_BROKER_TYPE=$(1)/" docker/.env
	sed -i "s/MF_MQTT_BROKER_HEALTH_CHECK=.*/MF_MQTT_BROKER_HEALTH_CHECK=$$\{MF_$(shell echo ${MF_MQTT_BROKER_TYPE} | tr 'a-z' 'A-Z')_HEALTH_CHECK}/" docker/.env
	sed -i "s/MF_MQTT_ADAPTER_WS_TARGET_PATH=.*/MF_MQTT_ADAPTER_WS_TARGET_PATH=$$\{MF_$(shell echo ${MF_MQTT_BROKER_TYPE} | tr 'a-z' 'A-Z')_WS_TARGET_PATH}/" docker/.env
	sed -i "s/MF_MESSAGE_BROKER_TYPE=.*/MF_MESSAGE_BROKER_TYPE=$(2)/" docker/.env
	sed -i "s,file: .*.yml,file: $(2).yml," docker/brokers/docker-compose.yml
	sed -i "s,MF_MESSAGE_BROKER_URL=.*,MF_MESSAGE_BROKER_URL=$$\{MF_$(shell echo ${MF_MESSAGE_BROKER_TYPE} | tr 'a-z' 'A-Z')_URL\}," docker/.env
	sed -i "s,MF_MQTT_ADAPTER_MQTT_QOS=.*,MF_MQTT_ADAPTER_MQTT_QOS=$$\{MF_$(shell echo ${MF_MQTT_BROKER_TYPE} | tr 'a-z' 'A-Z')_MQTT_QOS\}," docker/.env
endef

change_config:
ifeq ($(DOCKER_PROFILE),nats_nats)
	sed -i "s/- broker/- nats/g" docker/docker-compose.yml
	sed -i "s/- rabbitmq/- nats/g" docker/docker-compose.yml
	sed -i "s,MF_NATS_URL=.*,MF_NATS_URL=nats://nats:$$\{MF_NATS_PORT}," docker/.env
	$(call edit_docker_config,nats,nats)
else ifeq ($(DOCKER_PROFILE),nats_rabbitmq)
	sed -i "s/nats/broker/g" docker/docker-compose.yml
	sed -i "s,MF_NATS_URL=.*,MF_NATS_URL=nats://nats:$$\{MF_NATS_PORT}," docker/.env
	sed -i "s/rabbitmq/broker/g" docker/docker-compose.yml
	$(call edit_docker_config,nats,rabbitmq)
else ifeq ($(DOCKER_PROFILE),vernemq_nats)
	sed -i "s/nats/broker/g" docker/docker-compose.yml
	sed -i "s/rabbitmq/broker/g" docker/docker-compose.yml
	sed -i "s,MF_NATS_URL=.*,MF_NATS_URL=nats://broker:$$\{MF_NATS_PORT}," docker/.env
	$(call edit_docker_config,vernemq,nats)
else ifeq ($(DOCKER_PROFILE),vernemq_rabbitmq)
	sed -i "s/nats/broker/g" docker/docker-compose.yml
	sed -i "s/rabbitmq/broker/g" docker/docker-compose.yml
	$(call edit_docker_config,vernemq,rabbitmq)
else
	$(error Invalid DOCKER_PROFILE $(DOCKER_PROFILE))
endif

run: check_certs change_config
ifeq ($(MF_ES_STORE_TYPE), redis)
	sed -i "s/MF_ES_STORE_TYPE=.*/MF_ES_STORE_TYPE=redis/" docker/.env
	sed -i "s/MF_ES_STORE_URL=.*/MF_ES_STORE_URL=$$\{MF_REDIS_URL}/" docker/.env
	docker-compose -f docker/docker-compose.yml --profile $(DOCKER_PROFILE) --profile redis -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)
else
	sed -i "s,MF_ES_STORE_TYPE=.*,MF_ES_STORE_TYPE=$$\{MF_MESSAGE_BROKER_TYPE}," docker/.env
	sed -i "s,MF_ES_STORE_URL=.*,MF_ES_STORE_URL=$$\{MF_$(shell echo ${MF_MESSAGE_BROKER_TYPE} | tr 'a-z' 'A-Z')_URL\}," docker/.env
	docker-compose -f docker/docker-compose.yml --profile $(DOCKER_PROFILE) -p $(DOCKER_PROJECT) $(DOCKER_COMPOSE_COMMAND) $(args)
endif

run_addons: check_certs
	$(call change_config)
	$(foreach SVC,$(RUN_ADDON_ARGS),$(if $(filter $(SVC),$(ADDON_SERVICES) $(EXTERNAL_SERVICES)),,$(error Invalid Service $(SVC))))
	@for SVC in $(RUN_ADDON_ARGS); do \
		MF_ADDONS_CERTS_PATH_PREFIX="../."  docker-compose -f docker/addons/$$SVC/docker-compose.yml -p $(DOCKER_PROJECT) --env-file ./docker/.env $(DOCKER_COMPOSE_COMMAND) $(args) & \
	done
