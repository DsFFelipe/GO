@echo off
set SERVER_IP=192.168.15.71

echo Iniciando 10 sensores de temperatura...
for /L %%i in (1,1,10) do (
    docker run -d --rm -e SERVIDOR_ADDR=%SERVER_IP%:8080 --name container-sensor2-%%i img-sensor2
)
echo Sensores iniciados!
pause