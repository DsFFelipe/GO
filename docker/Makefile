# Default LAN IPs (Override these in the terminal when running the commands)
SERVER_IP ?= 192.168.0.71
CLIENT_IP ?= 192.168.0.118
ACTUATOR_IP ?= 192.168.0.118
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
		-e CLIENTE_ADDR=$(CLIENT_IP):8080 \
		-e ATUADOR_ADDR=$(ACTUATOR_IP):8080 \
		--name container-servidor \
		img-servidor

# ==========================================
# CLIENTE
# ==========================================
build-cliente:
	cd cliente && docker build -t img-cliente -f c.dockerfile .

# Requires -it (interactive TTY) to capture terminal input for the client menu
run-cliente:
	docker run -it --rm \
		-p 8080:8080/udp \
		-e SERVIDOR_ADDR=$(SERVER_IP):8081 \
		--name container-cliente \
		img-cliente

# ==========================================
# SENSOR 1 (Pluviômetro)
# ==========================================
build-sensor1:
	cd sensor/sensor1 && docker build -t img-sensor1 -f ss1.dockerfile .

# Requires -it to capture terminal input for changing the locality
run-sensor1:
	docker run -it --rm \
		-e SERVIDOR_ADDR=$(SERVER_IP):8080 \
		-e SENSOR_LOCAL=$(SENSOR_LOCAL) \
		--name container-sensor1-$(SENSOR_LOCAL) \
		img-sensor1

# ==========================================
# ATUADOR 1
# ==========================================
build-atuador1:
	cd atuador/atuador1 && docker build -t img-atuador1 -f a1.dockerfile .

run-atuador1:
	docker run -d --rm \
		-p 8080:8080/tcp \
		-e PORTA=8080 \
		--name container-atuador1 \
		img-atuador1