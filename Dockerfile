# Stage de build
FROM maiconfr/go-oracle-basic:1.0 AS builder
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

WORKDIR /app
#COPY . .
COPY "main.go" .
COPY "configIntegracao.go" .
COPY "functions/" functions/
COPY "utils/" utils/
COPY logs/ app/logs/
RUN go mod init integracao-hes-fleury && go mod tidy && go build -o integracao-hes-fleury

#RUN go mod init integracao-hes-fleury && go mod tidy && go build -o integracao-hes-fleury main.go

# Stage final
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates

WORKDIR /app

# Dependências Oracle
RUN apt-get update && apt-get install -y libaio1 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/lib/oracle /usr/lib/oracle
COPY --from=builder /etc/ld.so.conf.d/oracle-instantclient.conf /etc/ld.so.conf.d/oracle-instantclient.conf

ENV LD_LIBRARY_PATH=/usr/lib/oracle/19.3/client64/lib:$LD_LIBRARY_PATH


COPY --from=builder /app/integracao-hes-fleury .
COPY wait-for-it.sh /wait-for-it.sh
RUN chmod +x /wait-for-it.sh

ENTRYPOINT ["/wait-for-it.sh", "rabbitmq:5672", "--timeout=30", "--strict", "--", "./integracao-hes-fleury"]
