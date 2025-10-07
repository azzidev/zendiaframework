.PHONY: test build run clean docs

# Variáveis
BINARY_NAME=zendia-example
MAIN_PATH=./examples/main.go

# Testes
test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Build
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Executar exemplo
run:
	go run $(MAIN_PATH)

# Limpar arquivos gerados
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -f coverage.html

# Gerar documentação Swagger
docs:
	swag init -g $(MAIN_PATH) -o ./docs

# Instalar dependências
deps:
	go mod tidy
	go mod download

# Verificar código
lint:
	golangci-lint run

# Formatar código
fmt:
	go fmt ./...

# Verificar vulnerabilidades
security:
	gosec ./...

# Pipeline completa
ci: fmt lint test build

# Ajuda
help:
	@echo "Comandos disponíveis:"
	@echo "  test         - Executar testes"
	@echo "  test-coverage- Executar testes com cobertura"
	@echo "  build        - Compilar exemplo"
	@echo "  run          - Executar exemplo"
	@echo "  clean        - Limpar arquivos gerados"
	@echo "  docs         - Gerar documentação Swagger"
	@echo "  deps         - Instalar dependências"
	@echo "  lint         - Verificar código"
	@echo "  fmt          - Formatar código"
	@echo "  security     - Verificar vulnerabilidades"
	@echo "  ci           - Pipeline completa"