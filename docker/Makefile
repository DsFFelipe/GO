# LAN padrão
SERVER_IP ?= 192.168.15.71
CLIENT_IP ?= 192.168.15.71
ACTUATOR_IP ?= 192.168.15.71
SENSOR_LOCAL ?= "Padrao"

# ==========================================
# SERVIDOR
# ==========================================

build-servidor:
	cd servidor && docker build -t img-servidor -f sv.dockerfile .

run-servidor:
	docker run -d --rm \
		-p 8080:8080/udp \
		-p 8081:8080/tcp \
		-p 8082:8082/tcp \
		-e CLIENTE_ADDR=$(CLIENT_IP):8083 \
		-e CLIENTE_TCP_ADDR=$(CLIENT_IP):8084 \
		--name container-servidor \
		img-servidor

# ==========================================
# CLIENTE
# ==========================================
build-cliente:
	cd cliente && docker build -t img-cliente -f c.dockerfile .

# Terminal 1: Abre a interface de telemetria
run-cliente-monitor:
	docker run -it --rm \
		-p 8083:8083/udp \
		-p 8084:8084/tcp \
		-e CLIENTE_MODO=MONITOR \
		--name container-cliente-monitor \
		img-cliente

# Terminal 2: Abre a interface de inserção de comandos
run-cliente-comandos:
	docker run -it --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):8081 \
		-e CLIENTE_MODO=COMANDO \
		--name container-cliente-comandos \
		img-cliente

# ==========================================
# SENSOR 1 (Pluviômetro)
# ==========================================
build-sensor1:
	cd sensor/sensor1 && docker build -t img-sensor1 -f ss1.dockerfile .

# - it
run-sensor1:
	docker run -it --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):8080 \
		-e SENSOR_LOCAL=$(SENSOR_LOCAL) \
		--name container-sensor1-$(SENSOR_LOCAL) \
		img-sensor1

# ==========================================
# SENSOR 2 (Temperatura do Reator Nuclear)
# ==========================================
build-sensor2:
	cd sensor/sensor2 && docker build -t img-sensor2 -f ss2.dockerfile .

# Executa o sensor do reator em modo "detached" (-d)
run-sensor2:
	docker run -d --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):8080 \
		--name container-sensor2\
		img-sensor2

# ==========================================
# ATUADOR 1
# ==========================================
build-atuador1:
	cd atuador/atuador1 && docker build -t img-atuador1 -f a1.dockerfile .

run-atuador1:
	docker run -d --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):8082 \
		--name container-atuador1 \
		img-atuador1

# ==========================================
# ATUADOR 2 (Sirene de Emergência)
# ==========================================
build-atuador2:
	cd atuador/atuador2 && docker build -t img-atuador2 -f a2.dockerfile .

run-atuador2:
	docker run -d --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):8082 \
		--name container-atuador2 \
		img-atuador2