# **Sistema de Telemetria e Controle Distribuído**

Este projeto implementa uma arquitetura cliente-servidor orientada a LAN para telemetria e controle de processos críticos simulados. Desenvolvido em Go e conteinerizado via Docker, o sistema emprega uma abordagem multicanal: utiliza datagramas UDP para fluxo contínuo de telemetria e conexões TCP para garantir a integridade na entrega de comandos e transições de estado.

A topologia da rede, incluindo o endereçamento IP (LAN) e o mapeamento de portas dos contêineres, é parametrizável diretamente através do Makefile.

## **Arquitetura do Sistema**

O ecossistema é modular e distribuído em quatro componentes principais:

### **1\. Servidor (Broker Central)**

Atua como o nó central de roteamento e processamento lógico da rede.

* **Ingestão de Dados (UDP):** Recebe o fluxo contínuo de telemetria dos sensores. O protocolo UDP maximiza o *throughput* e evita *overhead* em dados cuja perda ocasional é tolerável.  
* **Broker de Comandos (TCP):** Mantém conexões persistentes com atuadores para roteamento de comandos de estado e realiza *handshakes* de registro de hardware. Recebe também comandos síncronos da interface do cliente.  
* **Lógica Autônoma:** Avalia o estado da planta industrial. Executa rotinas de atuação automática baseadas em limiares predefinidos (ex: acionamento de contenções de segurança ou disparo de alarmes de reator).

### **2\. Sensores (Emissores UDP)**

Unidades de monitoramento que transmitem a telemetria periodicamente para a porta UDP do servidor.

* **Sensor 1 (Contador Geiger):** Responsável por emitir os níveis de radiação detectados. Suporta alteração dinâmica do parâmetro de localidade em tempo de execução via interface de linha de comando (*stdin*).  
* **Sensor 2 (Temperatura do Reator):** Simula um gerador contínuo de dados térmicos críticos operando no range de 350°C a 1050°C.

### **3\. Atuadores (Receptores TCP)**

Dispositivos físicos simulados que requerem confirmação de entrega para alteração de estado estrutural.

* **Atuador 1 (Barreira de Contenção):** Realiza registro via identificador REGISTRO:BARREIRA. Responde aos comandos de abertura e fechamento.  
* **Atuador 2 (Sirene de Emergência):** Realiza registro via identificador REGISTRO:ALARME. Responde aos comandos de ativação e desativação do alerta acústico.

### **4\. Cliente (Interfaces de Operação)**

O binário do cliente suporta dois modos operacionais configurados por variáveis de ambiente:

* **Modo Monitor (CLIENTE\_MODO=MONITOR):** *Dashboard* interativo via terminal. Implementa instâncias passivas de escuta em portas UDP e TCP locais para refletir em tempo real a telemetria da planta e o status do maquinário (atuadores).  
* **Modo Comando (CLIENTE\_MODO=COMANDO):** CLI síncrona que permite a intervenção de um operador através do envio de comandos TCP diretos (ABRIR, FECHAR, LIGAR\_ALARME, DESLIGAR\_ALARME) para o broker.

## **Configuração de Rede (LAN)**

Sendo um sistema voltado para infraestruturas LAN, todos os IPs e portas de roteamento Docker estão centralizados no arquivo Makefile. Antes do *deploy* em ambiente físico, ajuste o cabeçalho do arquivo com os endereços da sub-rede:

\# Exemplo de configuração no Makefile  
SERVER\_IP ?= 192.168.15.187  
CLIENT\_IP ?= 192.168.15.187  
SENSOR\_PORT ?= 8080  
\# ...

## **Compilação e Execução**

### **1\. Build das Imagens**

A compilação de todas as imagens Docker deve ser o primeiro passo lógico:

make build-servidor build-cliente build-sensor1 build-sensor2 build-atuador1 build-atuador2

### **2\. Implantação da Infraestrutura Core**

Inicie o nó central seguido do maquinário de resposta física (atuadores):

make run-servidor  
make run-atuador1  
make run-atuador2

### **3\. Execução das Interfaces Cliente**

Abra terminais isolados para separar a escuta passiva da intervenção ativa:

make run-cliente-monitor \# Terminal 1  
make run-cliente-comandos \# Terminal 2

### **4\. Ativação da Malha Sensorial**

Inicie a transmissão de dados injetando os sensores na rede:

make run-sensor1  
make run-sensor2

*Nota para testes de estresse:* Utilize o script docker/rodar\_sensores.bat para provisionar em *batch* uma malha densa (10 instâncias concorrentes) do Sensor 2 e avaliar o comportamento do Broker sob carga contínua.