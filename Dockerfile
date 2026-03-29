# --- ETAPA 1: Compilación ---
# Usamos 'latest' o '1.24-alpine' para evitar problemas con la versión 1.25 que aún es muy nueva
FROM golang:alpine AS builder

# Instalar dependencias necesarias
RUN apk add --no-cache git

WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Compilar el binario
# El nombre del módulo 'dental-app' ya está configurado en tu go.mod
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# --- ETAPA 2: Ejecución ---
FROM alpine:latest

# Instalamos tzdata para el Timezone de Colombia y ca-certificates para HTTPS
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copiar el binario y los assets estáticos desde la etapa de construcción
COPY --from=builder /app/main .
COPY --from=builder /app/assets ./assets

# Railway usará la variable PORT automáticamente
EXPOSE ${PORT}

CMD ["./main"]