<p align="center">
  <img src="artifacts/cogni-logo.png" width="220" alt="Cogni Logo" />
</p>

<h1 align="center">🧠 Cogni</h1>

<p align="center">
  <b>Cognitive Omniscient Grid for Networked Intelligence</b><br>
  <i>Memoria persistente y buscable para agentes de IA en entornos de desarrollo.</i>
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version"></a>
  <a href="https://sqlite.org/"><img src="https://img.shields.io/badge/SQLite-FTS5-003B57?style=for-the-badge&logo=sqlite&logoColor=white" alt="SQLite FTS5"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License"></a>
  <a href="https://github.com/AdelysAlberto/cogni-memory"><img src="https://img.shields.io/badge/Harnesses-Universal-FF6F61?style=for-the-badge" alt="Harnesses"></a>
</p>

---

## 📌 Visión General

**Cogni** es una CLI para que un agente guarde y recupere decisiones técnicas de forma persistente entre sesiones.

En lugar de releer contexto crudo en cada tarea, el agente consulta una memoria sintética en SQLite (local por proyecto y opcionalmente global). El objetivo práctico es reducir repetición, mantener continuidad y evitar perder acuerdos técnicos.

Es compatible con **Gemini Antigravity**, **Cursor IDE**, **GitHub Copilot**, **OpenCode**, **Hermes CLI**, **OpenAI Codex CLI** y cualquier flujo basado en CLI/IDE que pueda ejecutar comandos.

---

## ✅ Qué Resuelve

* Evita repetir descubrimientos técnicos ya resueltos en sesiones anteriores.
* Reduce lecturas largas de archivos cuando la pregunta ya tiene antecedente.
* Da trazabilidad mínima de decisiones con estructura `What | Why | Where | Learned`.
* Permite operar en modo local-first, sin depender de servicios externos.

---

## 🚀 Contexto Crudo vs. Memoria Sintética

> Nota: los valores son rangos orientativos observados en uso real y dependen del proyecto, del modelo y del arnés.

| Métricas / Capacidad | Sin Cogni (Lectura Tradicional) | Con Cogni (Firmas Sintéticas) |
| :--- | :--- | :--- |
| **Consumo de Tokens** | Miles de tokens por relectura | Menor consumo al reutilizar resumen estructurado |
| **Tiempo de Recuperación** | Segundos de relectura | Milisegundos a decenas de ms según tamaño de BD |
| **Coherencia de Arquitectura** | Se pierde al compactar o reiniciar chat | **Persistente** entre sesiones y proyectos |
| **Duplicación de Decisiones** | Frecuente | Menor, si se guarda y actualiza de forma disciplinada |

---

## ⚡ Características Principales

* 🚀 **Binario Nativo en Go**: sin runtime de Node o Python para ejecutar la CLI.
* 🗄️ **Local-First**: memoria por proyecto en `.cogni/memory.db` y capa global opcional en `~/.cogni/memory.db`.
* 🔍 **Búsqueda FTS5**: búsqueda full-text por título, categoría, tags y resumen.
* 🖥️ **UI Embebida**: inspección visual de memorias desde `cogni ui`.
* 🏷️ **Taxonomía de Tags**: reduce ambigüedad y facilita recuperación consistente.
* 🤖 **Convención Operativa**: define cuándo buscar y cuándo guardar para evitar olvidos del agente.

---

## 🛠️ Instalación Rápida

### 1. Vía Script de Instalación Universal (Recomendado)

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/AdelysAlberto/cogni-memory/main/install.sh)
```

*El script detectará automáticamente los arneses de IA instalados (`.gemini`, `.cursor`, `.claude`, `.agents`, `.copilot`, `.opencode`, `.hermes`, `.codex`) y registrará la skill de Cogni.*

*Para GitHub Copilot en VS Code, además de la skill, el instalador crea una instrucción global en `~/.config/Code/User/prompts/cogni-copilot.instructions.md` (Linux) o `~/Library/Application Support/Code/User/prompts/cogni-copilot.instructions.md` (macOS) para reforzar búsqueda/guardado obligatorio cuando el CLI `cogni` está disponible.*

### 2. Compilando desde el Código Fuente (Go 1.22+)

```bash
git clone https://github.com/AdelysAlberto/cogni-memory.git cogni
cd cogni
make install
```

*El binario quedará listo en `$HOME/.local/bin/cogni`.*

---

## 🔐 Por Qué Instalar el Binario (Para Escépticos y Seguridad)

Instalar el binario no es solo comodidad; también es control operativo:

1. **Superficie de ejecución acotada**: ejecutas una CLI única en Go, en vez de depender de varios runtimes y paquetes transitorios.
2. **Comportamiento estable**: el mismo comando `cogni` funciona igual desde distintos agentes (Copilot, Cursor, CLI), reduciendo variaciones.
3. **Local-first real**: por defecto, la memoria vive en tu máquina (SQLite), sin enviar datos a servicios remotos por diseño de base.
4. **Auditable**: el código fuente está disponible; puedes compilar tú mismo y evitar binarios precompilados si lo prefieres.

Si prefieres máxima cautela, evita ejecutar scripts remotos directos y revisa primero:

```bash
curl -fsSL https://raw.githubusercontent.com/AdelysAlberto/cogni-memory/main/install.sh -o /tmp/cogni-install.sh
less /tmp/cogni-install.sh
bash /tmp/cogni-install.sh
```

O compila desde fuente:

```bash
git clone https://github.com/AdelysAlberto/cogni-memory.git
cd cogni-memory
make install
```

### Qué modifica el instalador

* Crea `~/.local/bin/cogni`.
* Crea/usa `~/.cogni/` para datos locales.
* Puede usar `~/.cogni-src/` como caché de fuente.
* Copia la skill y reglas en carpetas de los arneses seleccionados.
* **Configura automáticamente el servidor MCP** en los arneses compatibles (OpenCode, Cursor, Claude, Gemini, Hermes, Codex).
* En Copilot VS Code, puede crear `~/.config/Code/User/prompts/cogni-copilot.instructions.md` (Linux).

No reemplaza archivos del proyecto actual ni requiere privilegios root para el flujo normal (salvo intentos opcionales de instalar Go si no existe).

---

## 🔌 Integración MCP (Model Context Protocol)

Cogni incluye un servidor MCP nativo que permite a los agentes de IA interactuar con la memoria mediante herramientas estructuradas, sin depender exclusivamente de la CLI.

### Configuración Automática por Arnés

Al ejecutar `cogni init` e seleccionar tu entorno, Cogni inyecta automáticamente la configuración MCP en el archivo correspondiente:

| Arnés | Archivo de Configuración MCP |
| :--- | :--- |
| **OpenCode** | `~/.config/opencode/opencode.json` |
| **Claude Desktop** | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) |
| **Claude Code CLI** | `~/.claude.json` |
| **Cursor IDE** | `~/.cursor/mcp.json` |
| **Gemini Antigravity** | `~/.gemini/config/mcp_config.json` |
| **Hermes CLI** | `~/.hermes/mcp.json` |
| **Codex CLI** | `~/.codex/config.toml` |

### Formato Generado para OpenCode

Para **OpenCode**, Cogni genera automáticamente la estructura correcta bajo `mcp.servers`:

```jsonc
// ~/.config/opencode/opencode.json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "servers": {
      "cogni": {
        "type": "local",
        "command": ["/Users/tu-usuario/.local/bin/cogni", "mcp"],
        "enabled": true
      }
    }
  }
}
```

> **Nota**: OpenCode usa el formato `mcp.servers` (V2), **no** el formato `mcpServers` usado por Claude/Cursor. El instalador de Cogni detecta automáticamente el arnés y genera el formato correcto.

### Formato Generado para Claude

**Claude Desktop** (aplicación gráfica):
```jsonc
// ~/Library/Application Support/Claude/claude_desktop_config.json (macOS)
{
  "mcpServers": {
    "cogni": {
      "type": "stdio",
      "command": "/Users/tu-usuario/.local/bin/cogni",
      "args": ["mcp"]
    }
  }
}
```

**Claude Code** (CLI):
```jsonc
// ~/.claude.json
{
  "mcpServers": {
    "cogni": {
      "type": "stdio",
      "command": "/Users/tu-usuario/.local/bin/cogni",
      "args": ["mcp"]
    }
  }
}
```

> **Nota importante**: Claude Code también soporta configuración a nivel de proyecto con `.mcp.json` en la raíz del proyecto. Para configuración compartida con el equipo, ejecuta `claude mcp add cogni --scope project` después de instalar Cogni.

### Formato Generado para Codex CLI

Codex CLI lee la configuración de `~/.codex/config.toml` (o `<proyecto>/.codex/config.toml` para overrides del proyecto) y registra los servidores MCP bajo la tabla `[mcp_servers]`. Cogni preserva el resto de tus claves existentes (modelo, profiles, flags, etc.) y solo hace upsert de la entrada `[mcp_servers.cogni]`:

```toml
# ~/.codex/config.toml
mcp_oauth_credentials_store = "auto"

[mcp_servers]

[mcp_servers.cogni]
command = "/Users/tu-usuario/.local/bin/cogni"
args = ["mcp"]
enabled = true
```

> **Nota importante**: Codex CLI **no usa** el formato `mcpServers` (camelCase) ni el wrapping `mcp.servers` de OpenCode. La sección debe llamarse `[mcp_servers]` (snake_case). El instalador de Cogni detecta el archivo `config.toml` dentro de `~/.codex/` y aplica automáticamente el formato correcto.

Para configuración compartida con tu equipo, también puedes versionar `<proyecto>/.codex/config.toml` (sólo se carga en proyectos confiables) con el mismo bloque `[mcp_servers.cogni]`.

### Herramientas MCP Disponibles

Una vez configurado, el agente tendrá acceso a estas herramientas:

| Herramienta | Descripción |
| :--- | :--- |
| `cogni_search` | Búsqueda compacta de memorias (FTS5) |
| `cogni_get` | Recuperación completa de una memoria por ID o TopicKey |
| `cogni_save` | Guarda o actualiza (upsert) una firma de memoria |
| `cogni_update` | Actualiza una memoria existente por ID |
| `cogni_context` | Contexto activo reciente del proyecto |
| `cogni_session_summary` | Guarda resumen de sesión |
| `cogni_stats` | Métricas de uso y tokens ahorrados |

### Verificación Manual

Para verificar que el servidor MCP está funcionando:

```bash
# Iniciar el servidor MCP manualmente (para debugging)
cogni mcp

# Verificar que el binario está en PATH
which cogni
```

---

## 🔄 Flujo Operativo del Agente

```text
               ┌──────────────────────────────────────────────┐
               │    [Operador / Agente inicia solicitud]     │
               └──────────────────────┬───────────────────────┘
                                      │
                                      ▼
                        ¿Existe decisión/patrón previo?
                         cogni search --query "auth"
                                      │
                   ┌──────────────────┴──────────────────┐
                   ▼                                     ▼
                [ SÍ ]                                 [ NO ]
    Recupera firma semántica              Diseña solución técnica,
     y notifica en chat:                  ejecuta cambio y guarda:
   🧠 Memoria Recuperada                  cogni save 💾
```

---

## 🧠 Directivas y Disparadores Obligatorios (Skill Standard)

Todo agente integrado con Cogni sigue el estándar **WHEN TO SAVE / WHEN TO SEARCH**:

### 1. Disparadores Obligatorios de Guardado (`cogni save`)

El agente debe guardar memoria INMEDIATAMENTE tras:

* 🐛 **bugfix**: Solución a un error o bug no trivial.
* 📐 **architecture / decision**: Elección de librerías, modelo de datos o diseño de sistema.
* 💡 **discovery**: Descubrimiento no obvio sobre el comportamiento del sistema.
* ⚙️ **config**: Setup de entorno, herramientas o scripts.
* 🎨 **pattern**: Convención de naming, estructura de archivos o estándar técnico.
* 👤 **preference**: Restricción o preferencia explicada por el usuario.
* 📋 **session**: Resumen de sesión o hito alcanzado al cerrar sesión o tras compactar contexto.

### 2. Estructura de Firma Sintética de Alta Densidad (<5% Tokens)

Cogni reemplaza relecturas masivas de código por **Firmas Sintéticas de Alta Densidad**:

```yaml
Topic: <domain>/<subdomain>/<topic> (ej: standards/i18n/ui)
What: <Qué se hizo o decidió en 1 oración corta>
Why: <Motivación o causa raíz técnica>
Where: <Archivos o rutas clave afectadas>
Learned: <Gotchas o hallazgos no obvios>
```

Format de firma sintética unificada (`--summary` o flags discretos `--what`, `--why`, `--where`, `--learned`):
```text
What: ... | Why: ... | Where: ... | Learned: ...
```

### 3. Herramientas de Diagnóstico y Ciclo de Vida de Sesión

* **`cogni stats`**: Muestra métricas de salud de memoria, número de registros y tokens ahorrados.
* **`cogni session-summary`**: Guarda los avances y descubrimientos del proyecto al cerrar sesión o compactar contexto para reanudar el trabajo en < 100 tokens.
* **Tras compactación de contexto (`FIRST ACTION REQUIRED`)**:
  1. Llama inmediatamente a `cogni session-summary` con el resumen compactado.
  2. Llama a `cogni context` para recuperar el estado activo.
  3. Continúa con la tarea.

### 4. Notificaciones Visuales en Chat

* **Al Recuperar**: `🧠 **Memoria Recuperada**: [<proyecto>] "<titulo_o_tema>" (Tags: #tag1, #tag2)`
* **Al Guardar**: `💾 **Memoria Guardada**: [<proyecto>] "<titulo_breve>" (Category: #category, Tags: #tag1, #tag2)`

---

## 💻 Referencia de Comandos CLI

```bash
# 1. Bootstrapping rápido de contexto activo (< 100 tokens)
cogni context

# 2. Guardar memoria sintética con flags estructurados
cogni save \
  --topic-key "arch/db/indexes" \
  --title "Fixed N+1 Query in Product List" \
  --what "Added index on category_id and joined queries" \
  --why "Resolves slow load on 10k rows" \
  --where "src/db/products.go" \
  --learned "SQLite EXPLAIN QUERY PLAN required" \
  --category "bugfix" \
  --tags "database,sqlite,products-list"

# 3. Guardar resumen de fin de sesión o post-compactación
cogni session-summary \
  --goal "Optimizar auth y contexto" \
  --accomplished "Endpoints creados, tablas migradas" \
  --where "src/auth/jwt.go"

# 4. Buscar firmas semánticas con FTS5 (local y global)
cogni search --query "products"

# 5. Obtener contenido completo hidratado por TopicKey o ID (Fase 2)
cogni get arch/db/indexes
cogni get --id 6

# 6. Actualizar memoria existente por ID para evitar duplicados
cogni update --id 6 --summary "What: Updated auth to JWT + Rotation | Why: Security audit | Where: src/auth/jwt.go"

# 7. Promover una memoria local a la BD global centralizada
cogni promote --id 6

# 8. Eliminar una firma por ID
cogni remove --id 6

# 9. Exportar memorias en Markdown o JSON
cogni share --format markdown > memorias.md
cogni share --format json

# 10. Ver métricas de tokens ahorrados y estadísticas
cogni stats

# 11. Abrir el Dashboard Gráfico en el navegador
cogni ui

# 12. Instalar o actualizar la Skill en arneses de IA
cogni skill
```

---

## 🏷️ Regla de las 3 Capas de Tags

Para evitar etiquetas ambiguas o duplicadas, cada firma semántica organiza de 3 a 5 tags en 3 capas deterministas:

1. **Capa 1 - Concepto Principal / Dominio**: Término genérico (`pagination`, `auth`, `state-management`, `api-rest`, `database`).
2. **Capa 2 - Tecnología / Herramienta**: Stack exacto (`go`, `sqlite`, `zustand`, `react`, `express`, `css-modules`).
3. **Capa 3 - Módulo / Entidad Específica**: Dominio del proyecto (`users-table`, `products-list`, `jwt-middleware`).

---

## 🏗️ Arquitectura del Repositorio

```text
cogni-memory/
├── cmd/cogni/main.go          # Punto de entrada de la CLI
├── internal/
│   ├── cli/                   # Handlers de comandos (save, search, update, promote, remove, share, ui, skill)
│   ├── core/                  # Entidades de dominio, tags y resolución de workspace Git
│   ├── server/                # Servidor HTTP embebido y endpoints REST de la Web UI
│   └── storage/               # Repositorio SQLite Pure-Go con soporte FTS5
├── web/                       # Assets estáticos embebidos (Dashboard Web UI)
│   ├── embed.go
│   └── public/
├── SKILL.md                   # Especificación canónica de la Skill para Agentes de IA
├── Makefile                   # Tareas de compilación, testeo, instalación y releases
├── release.sh                 # Script automatizado de tags y releases (patch / minor / major)
└── install.sh                 # Instalador universal multi-arnés de IA
```

---

## 🛠️ Desarrollo y Release de Versiones

El sistema de versiones es **centralizado por Tag de Git y `-ldflags`**. Para generar una nueva versión desde desarrollo:

```bash
# Incrementar versión PATCH (ej: v2.0.3 -> v2.0.4)
make release-patch

# Incrementar versión MINOR (ej: v2.0.3 -> v2.1.0)
make release-minor

# Incrementar versión MAJOR (ej: v2.0.3 -> v3.0.0)
make release-major
```

El comando automatiza la compilación con la versión exacta inyectada en Go, crea la `git tag`, hace el `git push` a GitHub y (si tienes el `gh` CLI instalado) sube el Release a GitHub con su binario correspondiente.

---

## 👨‍💻 Autor y Mantenimiento

Desarrollado y mantenido por **Adelys Alberto** ([@AdelysAlberto](https://github.com/AdelysAlberto)).

---

## 📄 Licencia

Este proyecto está distribuido bajo la licencia **MIT**. Consulta el archivo [LICENSE](LICENSE) para más detalles.
