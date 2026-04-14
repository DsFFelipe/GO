
# CONFIGURAÇÕES DE REDE (IPs)
 
SERVER_IP ?= 172.16.103.11
CLIENT_IP ?= 172.16.103.11
ATUADOR_IP ?=  172.16.103.11

# Porta intera (listen)
SENSOR_PORT ?= 6767
CLIENTE_PORT ?= 6767
ATUADOR_PORT ?= 4243
MONITOR_UDP_PORT ?= 3343

MONITOR_TCP_PORT ?= 4342



# Portas externas
SERVIDOR_UDP_PORT ?= 6767
SERVIDOR_TCP_PORT ?= 6767

SERVIDOR_ATUADOR_PORT ?= 4243
HOST_MONITOR_UDP ?= 3343

HOST_MONITOR_TCP ?= 4342


# Configurações de Simulação
SENSOR_LOCAL ?= "Padrao"


# SERVIDOR

build-servidor:
	cd servidor && docker build -t img-servidor -f sv.dockerfile .

run-servidor:
	docker run -d --rm \
		-p $(SERVIDOR_UDP_PORT):$(SENSOR_PORT)/udp \
		-p $(SERVIDOR_TCP_PORT):$(CLIENTE_PORT)/tcp \
		-p $(SERVIDOR_ATUADOR_PORT):$(ATUADOR_PORT)/tcp \
		-e SENSOR_PORT=$(SENSOR_PORT) \
		-e CLIENTE_PORT=$(CLIENTE_PORT) \
		-e ATUADOR_PORT=$(ATUADOR_PORT) \
		-e CLIENTE_ADDR=$(CLIENT_IP):$(HOST_MONITOR_UDP) \
		-e CLIENTE_TCP_ADDR=$(CLIENT_IP):$(HOST_MONITOR_TCP) \
		--name container-servidor \
		img-servidor


# CLIENTE

build-cliente:
	cd cliente && docker build -t img-cliente -f c.dockerfile .

# Terminal 1: Interface de Telemetria
run-cliente-monitor:
	docker run -it --rm \
		-p $(HOST_MONITOR_UDP):$(MONITOR_UDP_PORT)/udp \
		-p $(HOST_MONITOR_TCP):$(MONITOR_TCP_PORT)/tcp \
		-e MONITOR_UDP_PORT=$(MONITOR_UDP_PORT) \
		-e MONITOR_TCP_PORT=$(MONITOR_TCP_PORT) \
		-e CLIENTE_MODO=MONITOR \
		--name container-cliente-monitor \
		img-cliente

# Terminal 2: Interface de Comandos Manuais
run-cliente-comandos:
	docker run -it --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):$(SERVIDOR_TCP_PORT) \
		-e CLIENTE_MODO=COMANDO \
		--name container-cliente-comandos \
		img-cliente


# SENSORES (UDP)

build-sensor1:
	cd sensor/sensor1 && docker build -t img-sensor1 -f ss1.dockerfile .

run-sensor1:
	docker run -it --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):$(SERVIDOR_UDP_PORT) \
		-e SENSOR_LOCAL=$(SENSOR_LOCAL) \
		--name container-sensor1-$(SENSOR_LOCAL) \
		img-sensor1

build-sensor2:
	cd sensor/sensor2 && docker build -t img-sensor2 -f ss2.dockerfile .

run-sensor2:
	docker run -d --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):$(SERVIDOR_UDP_PORT) \
		--name container-sensor2 \
		img-sensor2


# ATUADORES (TCP)

build-atuador1:
	cd atuador/atuador1 && docker build -t img-atuador1 -f a1.dockerfile .

run-atuador1:
	docker run -d --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):$(SERVIDOR_ATUADOR_PORT) \
		--name container-atuador1 \
		img-atuador1

build-atuador2:
	cd atuador/atuador2 && docker build -t img-atuador2 -f a2.dockerfile .

run-atuador2:
	docker run -it --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):$(SERVIDOR_ATUADOR_PORT) \
		--name container-atuador2 \
		img-atuador2

build-testes:
	cd Testes && docker build -t img-loadtest -f lt.dockerfile .

run-loadtest:
	docker run -it --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):$(SERVIDOR_UDP_PORT) \
		-e NUM_SENSORES=1000 \
		--name container-loadtest \
		img-loadtest

