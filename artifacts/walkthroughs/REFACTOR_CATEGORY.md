# Categoria refactor en MCP

## Cambio

- `cogni_save` y `cogni_search` anuncian `refactor` en un catalogo compartido.
- Se conservan las ocho categorias existentes y no se modifica el esquema SQLite.
- El README distingue reorganizacion de codigo, configuracion y arquitectura.

## Verificacion

- Prueba de contrato de ambos enums, incluidas las categorias previas.
- Flujo real de guardado, busqueda por categoria, recuperacion y upsert con `refactor`.
- `go vet ./...`, `go test ./...`, `go build ./...` y `git diff --check` exitosos.

## Revision

Tio Bob: `APPROVED_WITH_OBSERVATIONS`, sin bloqueos ni regresiones de seguridad detectadas.
Se atendio la observacion de comprobar error y categoria tras recuperar el upsert.

## Distribucion

Version patch prevista: `v2.3.6`. Los clientes deben usar el binario actualizado y
reiniciar o reconectar su servidor MCP para cargar el nuevo esquema.