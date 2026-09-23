package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AdelysAlberto/cogni/internal/core"
	"github.com/AdelysAlberto/cogni/internal/mcp"
	"github.com/AdelysAlberto/cogni/internal/network"
	"github.com/AdelysAlberto/cogni/internal/server"
	"github.com/AdelysAlberto/cogni/internal/storage"
)

var Version = "dev"

func Execute(args []string) int {
	if len(args) < 1 {
		printUsage()
		return 0
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "init":
		return handleInit(cmdArgs)
	case "save":
		return handleSave(cmdArgs)
	case "context":
		return handleContext(cmdArgs)
	case "session-summary", "compact":
		return handleSessionSummary(cmdArgs)
	case "search":
		return handleSearch(cmdArgs)
	case "get":
		return handleGet(cmdArgs)
	case "mcp":
		server := mcp.NewServer(Version)
		if err := server.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error ejecutando servidor MCP: %v\n", err)
			return 1
		}
		return 0
	case "update":
		// If called without memory flags or with --check, route to upgrade
		if len(cmdArgs) == 0 || (len(cmdArgs) == 1 && (cmdArgs[0] == "--check" || cmdArgs[0] == "-c")) {
			return handleUpgrade(cmdArgs)
		}
		return handleUpdate(cmdArgs)
	case "upgrade":
		return handleUpgrade(cmdArgs)
	case "remove", "delete":
		return handleRemove(cmdArgs)
	case "share", "export":
		return handleShare(cmdArgs)
	case "sync", "import":
		return handleSync(cmdArgs)
	case "list":
		return handleList(cmdArgs)
	case "stats":
		return handleStats(cmdArgs)
	case "clean", "optimize", "vacuum":
		return handleOptimize(cmdArgs)
	case "promote":
		return handlePromote(cmdArgs)
	case "ui":
		return handleUI(cmdArgs)
	case "tray", "bar":
		return handleTray(cmdArgs)
	case "skill", "skills":
		promptAndInstallSkills("", len(cmdArgs) > 0 && cmdArgs[0] == "--all")
		return 0
	case "uninstall":
		return handleUninstall(cmdArgs)
	case "version", "--version", "-v":
		fmt.Printf("🧠 Cogni %s\n", "v"+strings.TrimPrefix(Version, "v"))
		return 0
	case "help", "--help", "-h":
		printUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "Comando desconocido: %s\n\n", cmd)
		printUsage()
		return 1
	}
}

func printUsage() {
	usage := `🧠 Cogni — Cognitive Omniscient Grid for Networked Intelligence
"Así como el Byte es la unidad de datos, Cogni es la unidad de conocimiento sintético de tu agente."

Uso:
  cogni <comando> [argumentos...]

Comandos Principales:
  init             Inicializa Cogni globalmente (~/.cogni/) e instala skills de IA
  save             Guarda o actualiza (upsert) una firma de memoria sintética
  context          Muestra el contexto activo reciente del proyecto (alta señal, bajo token)
  session-summary  Guarda un resumen estructurado al cerrar sesión o tras compactar
  search           Busca firmas de memoria con FTS5 (previews compactas para ahorrar tokens)
  get              Recupera una memoria completa por ID o por TopicKey determinístico
  mcp              Inicia el servidor nativo MCP (Model Context Protocol) por stdio
  update           Actualiza una memoria existente por su ID
  promote          Promueve una memoria de local a global (o viceversa)
  remove           Elimina una memoria por su ID
  share            Inicia sesión efímera P2P cifrada o exporta paquete de memorias
  sync             Sincroniza memorias P2P de un compañero (código 3 slots) o archivo
  clean, optimize  Compacta SQLite, vacía WAL y reconstruye índice FTS5 (VACUUM)
  list             Lista las memorias registradas
  stats            Muestra métricas y tokens ahorrados
  ui               Abre el dashboard gráfico interactivo en el navegador
  bar, tray        Abre la app residente en el Top Bar (macOS) o Bandeja del Sistema
  skill            Instala o actualiza el Skill en tus arneses de IA
  uninstall        Desinstala Cogni, elimina el binario y limpia las skills
  version          Muestra la versión de Cogni


Flags de init:
  --project   Inicializa solo el almacén local (.cogni/) en el proyecto actual, sin instalar skills
  --all       Instala las skills en todos los arneses de IA sin preguntar
  --no-skills Omitir instalación de skills de IA (solo init global)

Flags Globales:
  --db        Ruta personalizada al archivo SQLite
  --json      Imprime la salida en formato JSON puro

Ejemplos:
  cogni init                   # Instala cogni global + configura skills de IA
  cogni save --topic-key "arch/auth/jwt" --title "Auth JWT" --summary "What: ... | Why: ... | Where: ... | Learned: ..." --category architecture --tags "auth,jwt"
  cogni search --query "jwt"   # Búsqueda compacta (IDs y previews)
  cogni get --id 6             # Recuperación completa por ID
  cogni get arch/auth/jwt      # Recuperación completa por TopicKey
  cogni mcp                    # Inicia servidor MCP para agentes
  cogni ui                     # Abre dashboard web
`
	fmt.Print(usage)
}

func getStorage(customPath string, forceGlobal bool) (*storage.Storage, error) {
	dbPath := core.ResolveDatabasePath(customPath, forceGlobal)
	return storage.New(dbPath)
}

func prettyPath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

func handleInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	project := fs.Bool("project", false, "Inicializa el almacén local (.cogni/) en el proyecto actual")
	noSkills := fs.Bool("no-skills", false, "Omitir instalación de skills de IA")
	allSkills := fs.Bool("all", false, "Instalar automáticamente en todos los arneses de IA")
	harnessFlag := fs.String("harness", "", "Especifica el arnés de IA a instalar (antigravity, cursor, claude, pi, opencode, local, copilot, hermes, codex, all, none)")
	// --global mantenido como alias de retrocompatibilidad
	_ = fs.Bool("global", false, "")
	_ = fs.Parse(args)

	if *project {
		localDir := filepath.Join(".", ".cogni")
		if err := os.MkdirAll(localDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creando directorio .cogni: %v\n", err)
			return 1
		}
		dbPath := filepath.Join(localDir, "memory.db")
		s, err := storage.New(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error inicializando base de datos local: %v\n", err)
			return 1
		}
		s.Close()
		fmt.Printf("✅ Cogni local inicializado en: %s\n", dbPath)
		fmt.Printf("💡 Proyecto detectado: %s\n", core.DetectProjectName())
	} else {
		dir := core.GetGlobalCogniDir()
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creando directorio global: %v\n", err)
			return 1
		}
		dbPath := filepath.Join(dir, "memory.db")
		s, err := storage.New(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error inicializando base de datos global: %v\n", err)
			return 1
		}
		s.Close()

		if !*noSkills {
			promptAndInstallSkills(*harnessFlag, *allSkills)
		} else {
			fmt.Printf("✅ Cogni global inicializado en: %s\n", prettyPath(dbPath))
		}
	}

	return 0
}

func promptAndInstallSkills(harnessFlag string, autoAll bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	harnesses := core.GetHarnessSkillPaths(home)

	// Intentar cargar configuración guardada previa al actualizar
	cfg, _ := core.LoadConfig(home)
	if harnessFlag == "" && autoAll && cfg != nil && len(cfg.SelectedHarnesses) > 0 {
		for _, h := range cfg.SelectedHarnesses {
			if paths, ok := harnesses[h]; ok {
				for _, p := range paths {
					_ = core.InstallSkill(p)
				}
			}
		}

		_ = core.InstallRules(home, cfg.SelectedHarnesses)
		mcpResults := core.ConfigureHarnessMCP(home, cfg.SelectedHarnesses)
		injectedDirectives := core.InjectAgentDirectives(home, cfg.SelectedHarnesses)

		fmt.Println("🔄 Cogni Upgrade: Arneses de IA actualizados:")
		fmt.Printf("  ✔ Arneses activos:    %s\n", strings.Join(cfg.SelectedHarnesses, ", "))
		if len(mcpResults) > 0 {
			fmt.Printf("  ✔ Servidores MCP:     Configurados en %d destinos\n", len(mcpResults))
		}
		if len(injectedDirectives) > 0 {
			fmt.Printf("  ✔ Directivas AGENTS:  Preservadas e inyectadas en %d destinos\n", len(injectedDirectives))
		}
		return
	}

	availableHarnesses := []SelectItem{
		{Key: "all", Title: "TODOS los entornos (Recomendado)", Description: "Configura automáticamente todos los arneses detectados"},
		{Key: "pi", Title: "Pi Coding Agent", Description: "~/.pi/agent/ (MCP, skills, rules, AGENTS.md)"},
		{Key: "antigravity", Title: "Gemini Antigravity", Description: "~/.gemini/config/ (MCP + Always-On Rules)"},
		{Key: "cursor", Title: "Cursor IDE", Description: "~/.cursor/ (MCP + Always-On Rules)"},
		{Key: "claude", Title: "Claude Code / Desktop", Description: "~/.claude/ & claude_desktop_config.json"},
		{Key: "opencode", Title: "OpenCode", Description: "~/.config/opencode/ (MCP v2 + Skills + Rules)"},
		{Key: "local", Title: "Agentes Estándar", Description: "~/.agents/ (Skills + Rules + Workspace)"},
		{Key: "copilot", Title: "GitHub Copilot", Description: "VS Code Copilot User Prompts & Instructions"},
		{Key: "hermes", Title: "Hermes CLI", Description: "~/.hermes/ (MCP + Skills + Rules)"},
		{Key: "codex", Title: "OpenAI Codex CLI", Description: "~/.codex/config.toml (TOML MCP + Skills)"},
		{Key: "none", Title: "Omitir instalación de skills", Description: "Solo inicializar base de datos local/global"},
	}

	selectedKey := ""
	if harnessFlag != "" {
		hLower := strings.ToLower(harnessFlag)
		switch hLower {
		case "antigravity", "1":
			selectedKey = "antigravity"
		case "cursor", "2":
			selectedKey = "cursor"
		case "claude", "3":
			selectedKey = "claude"
		case "pi", "4":
			selectedKey = "pi"
		case "opencode", "5":
			selectedKey = "opencode"
		case "local", "agents", "6":
			selectedKey = "local"
		case "copilot", "7":
			selectedKey = "copilot"
		case "hermes", "8":
			selectedKey = "hermes"
		case "codex", "9":
			selectedKey = "codex"
		case "all", "10":
			selectedKey = "all"
		case "none", "11":
			selectedKey = "none"
		default:
			selectedKey = hLower
		}
	} else if autoAll {
		selectedKey = "all"
	} else {
		selectedItem, err := InteractiveSelect("🤖 Selecciona el entorno o Harness de IA que utilizas:", availableHarnesses, 0, 4)
		if err != nil {
			fmt.Println("⏭️ Instalación de Skill cancelada.")
			return
		}
		selectedKey = selectedItem.Key
	}

	var selectedHarnesses []string
	var harnessLabel string

	switch selectedKey {
	case "antigravity":
		selectedHarnesses = []string{"antigravity"}
		harnessLabel = "Gemini Antigravity"
	case "cursor":
		selectedHarnesses = []string{"cursor"}
		harnessLabel = "Cursor IDE"
	case "claude":
		selectedHarnesses = []string{"claude"}
		harnessLabel = "Claude Code / Desktop"
	case "pi":
		selectedHarnesses = []string{"pi"}
		harnessLabel = "Pi Coding Agent"
	case "opencode":
		selectedHarnesses = []string{"opencode"}
		harnessLabel = "OpenCode"
	case "local":
		selectedHarnesses = []string{"local"}
		harnessLabel = "Agentes Estándar (~/.agents/)"
	case "copilot":
		selectedHarnesses = []string{"copilot"}
		harnessLabel = "GitHub Copilot"
	case "hermes":
		selectedHarnesses = []string{"hermes"}
		harnessLabel = "Hermes CLI"
	case "codex":
		selectedHarnesses = []string{"codex"}
		harnessLabel = "OpenAI Codex CLI"
	case "all":
		selectedHarnesses = []string{"local", "antigravity", "cursor", "claude", "pi", "opencode", "copilot", "hermes", "codex"}
		harnessLabel = "Todos los arneses detectados"
	case "none":
		fmt.Println("⏭️ Instalación de Skill omitida.")
		return
	default:
		selectedHarnesses = []string{"local", "antigravity", "cursor", "claude", "pi", "opencode", "copilot", "hermes", "codex"}
		harnessLabel = "Todos los arneses detectados"
	}

	for _, name := range selectedHarnesses {
		if paths, ok := harnesses[name]; ok {
			for _, p := range paths {
				_ = core.InstallSkill(p)
			}
		}
	}

	_ = core.SaveConfig(home, &core.Config{SelectedHarnesses: selectedHarnesses})
	_ = core.InstallRules(home, selectedHarnesses)
	mcpResults := core.ConfigureHarnessMCP(home, selectedHarnesses)
	injectedDirectives := core.InjectAgentDirectives(home, selectedHarnesses)

	dbPath := prettyPath(filepath.Join(core.GetGlobalCogniDir(), "memory.db"))

	fmt.Println("\n⚙️  Configuración completada:")
	fmt.Printf("  ✔ Almacenamiento:   %s\n", dbPath)
	fmt.Printf("  ✔ Skill de IA:      %s\n", harnessLabel)
	fmt.Println("  ✔ Reglas Invariants: Inyectadas")
	if len(mcpResults) > 0 {
		fmt.Printf("  ✔ Servidores MCP:   Configurados en %d destinos\n", len(mcpResults))
	}
	if len(injectedDirectives) > 0 {
		fmt.Printf("  ✔ Directivas AGENTS: Preservadas e inyectadas en %d destinos\n", len(injectedDirectives))
	}

	fmt.Println("\n💡 Uso rápido:")
	fmt.Println("   cogni search \"<query>\"   Busca memorias sintéticas (FTS5 BM25)")
	fmt.Println("   cogni save --title \"..\"  Guarda una firma de conocimiento")
	fmt.Println("   cogni ui                 Abre el dashboard gráfico en el navegador")
}

func handleSave(args []string) int {
	fs := flag.NewFlagSet("save", flag.ExitOnError)
	project := fs.String("project", "", "Nombre del proyecto")
	title := fs.String("title", "", "Título o hito de la memoria")
	topicKey := fs.String("topic-key", "", "Clave temática determinística para posibilitar upserts (ej: 'sdd/auth/spec')")
	summary := fs.String("summary", "", "Resumen sintético de la memoria")
	what := fs.String("what", "", "Qué se hizo (una oración descriptiva)")
	why := fs.String("why", "", "Motivo o causa raíz")
	where := fs.String("where", "", "Archivos o rutas afectadas")
	learned := fs.String("learned", "", "Gotchas o aprendizajes")
	category := fs.String("category", "general", "Categoría")
	tags := fs.String("tags", "", "Tags separados por coma")
	global := fs.Bool("global", false, "Guardar en la base de datos global")
	dbPath := fs.String("db", "", "Ruta personalizada a la base de datos")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	finalSummary := *summary
	if finalSummary == "" && (*what != "" || *why != "" || *where != "" || *learned != "") {
		finalSummary = core.BuildSummarySignature(*what, *why, *where, *learned)
	}

	if *title == "" || finalSummary == "" {
		fmt.Fprintln(os.Stderr, "Error: --title y (--summary o --what/--why/--where/--learned) son obligatorios.")
		return 1
	}

	projectName := *project
	if projectName == "" {
		projectName = core.DetectProjectName()
	}

	if !*global && *dbPath == "" && core.FindLocalCogniDir() == "" {
		fmt.Println("⚠️  No se encontró .cogni/ local en este proyecto.")
		fmt.Println("   Ejecutaré 'cogni init' para inicializar el almacén local del proyecto.")
		localDir := filepath.Join(".", ".cogni")
		if err := os.MkdirAll(localDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creando directorio .cogni: %v\n", err)
			return 1
		}
		fmt.Printf("✅ Almacén local inicializado en: %s\n", localDir)
	}

	formattedTags := core.FormatTags(*tags, projectName)

	s, err := getStorage(*dbPath, *global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	m := &core.Memory{
		ProjectName:      projectName,
		Category:         *category,
		Title:            *title,
		TopicKey:         *topicKey,
		SummarySignature: finalSummary,
		Tags:             formattedTags,
	}

	saved, err := s.SaveMemory(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error guardando memoria: %v\n", err)
		return 1
	}

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(saved)
	} else {
		fmt.Println("💾 **Memoria Guardada con Éxito**")
		fmt.Printf("ID: #%d\n", saved.ID)
		fmt.Printf("Proyecto: [%s]\n", saved.ProjectName)
		fmt.Printf("Título: %s\n", saved.Title)
		if saved.TopicKey != "" {
			fmt.Printf("Topic Key: %s\n", saved.TopicKey)
		}
		fmt.Printf("Categoría: %s\n", saved.Category)
		fmt.Printf("Tags: %s\n", saved.Tags)
		fmt.Printf("Resumen: %s\n", saved.SummarySignature)
		fmt.Printf("Ubicación BD: %s\n", s.DBPath())
	}

	return 0
}

func handleContext(args []string) int {
	fs := flag.NewFlagSet("context", flag.ExitOnError)
	project := fs.String("project", "", "Filtrar por proyecto")
	limit := fs.Int("limit", 5, "Límite de resultados")
	globalOnly := fs.Bool("global", false, "Buscar solo en BD global")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	projectName := *project
	if projectName == "" && !*globalOnly {
		projectName = core.DetectProjectName()
	}

	localStorage, globalStorage := getStorages()
	if localStorage != nil {
		defer localStorage.Close()
	}
	if globalStorage != nil {
		defer globalStorage.Close()
	}

	var memories []core.Memory
	if localStorage != nil && !*globalOnly {
		mems, _ := localStorage.GetRecentContext(projectName, *limit)
		memories = append(memories, mems...)
	}
	if globalStorage != nil && len(memories) < *limit {
		mems, _ := globalStorage.GetRecentContext(projectName, *limit-len(memories))
		memories = append(memories, mems...)
	}

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(memories)
		return 0
	}

	if len(memories) == 0 {
		fmt.Printf("⚡ No hay contexto reciente registrado para '%s'.\n", projectName)
		return 0
	}

	fmt.Printf("⚡ **Contexto Activo Reciente para [%s]** (%d memorias clave):\n\n", projectName, len(memories))
	for _, m := range memories {
		srcBadge := "LOCAL"
		if m.Source == "global" {
			srcBadge = "GLOBAL"
		}
		topicStr := ""
		if m.TopicKey != "" {
			topicStr = fmt.Sprintf(" | Key: %s", m.TopicKey)
		}
		fmt.Printf("▶ [#%d %s] [%s] %s%s\n  %s\n\n",
			m.ID, srcBadge, m.Category, m.Title, topicStr, m.SummarySignature)
	}

	return 0
}

func handleSessionSummary(args []string) int {
	fs := flag.NewFlagSet("session-summary", flag.ExitOnError)
	goal := fs.String("goal", "", "Objetivo principal de la sesión")
	accomplished := fs.String("accomplished", "", "Logros y tareas completadas")
	discoveries := fs.String("discoveries", "", "Hallazgos y decisiones clave")
	nextSteps := fs.String("next-steps", "", "Próximos pasos pendientes")
	where := fs.String("where", "", "Archivos clave modificados")
	instructions := fs.String("instructions", "", "Preferencias o restricciones aprendidas")
	topicKey := fs.String("topic-key", "session/latest", "TopicKey determinístico")
	project := fs.String("project", "", "Nombre del proyecto")
	tags := fs.String("tags", "", "Tags adicionales")
	global := fs.Bool("global", false, "Guardar en BD global")
	dbPath := fs.String("db", "", "Ruta a BD personalizada")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	if *goal == "" || *accomplished == "" {
		fmt.Fprintln(os.Stderr, "Error: --goal y --accomplished son obligatorios para el resumen de sesión.")
		return 1
	}

	projectName := *project
	if projectName == "" {
		projectName = core.DetectProjectName()
	}

	s, err := getStorage(*dbPath, *global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	summaryObj := core.SessionSummary{
		Goal:          *goal,
		Accomplished:  *accomplished,
		Discoveries:   *discoveries,
		NextSteps:     *nextSteps,
		RelevantFiles: *where,
		Instructions:  *instructions,
	}

	saved, err := s.SaveSessionSummary(projectName, *topicKey, summaryObj, *tags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error guardando resumen de sesión: %v\n", err)
		return 1
	}

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(saved)
	} else {
		fmt.Println("📋 **Resumen de Sesión Guardado con Éxito**")
		fmt.Printf("ID: #%d\n", saved.ID)
		fmt.Printf("Proyecto: [%s]\n", saved.ProjectName)
		fmt.Printf("Título: %s\n", saved.Title)
		fmt.Printf("Topic Key: %s\n", saved.TopicKey)
		fmt.Printf("Resumen: %s\n", saved.SummarySignature)
	}

	return 0
}

func getStorages() (*storage.Storage, *storage.Storage) {
	globalPath := core.ResolveDatabasePath("", true)
	globalStorage, _ := storage.NewWithSource(globalPath, "global")

	var localStorage *storage.Storage
	localDir := core.FindLocalCogniDir()
	if localDir != "" {
		localPath := filepath.Join(localDir, "memory.db")
		if filepath.Clean(localPath) != filepath.Clean(globalPath) {
			localStorage, _ = storage.NewWithSource(localPath, "local")
		}
	}

	return localStorage, globalStorage
}

func handleSearch(args []string) int {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	query := fs.String("query", "", "Término de búsqueda")
	project := fs.String("project", "", "Filtrar por proyecto")
	category := fs.String("category", "", "Filtrar por categoría")
	limit := fs.Int("limit", 10, "Límite de resultados")
	full := fs.Bool("full", false, "Mostrar la firma sintética completa en vez de preview compacto")
	globalOnly := fs.Bool("global", false, "Buscar solo en la base de datos global")
	localOnly := fs.Bool("local", false, "Buscar solo en la base de datos local")
	dbPath := fs.String("db", "", "Ruta a la base de datos")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	// If query was passed positionally without --query
	if *query == "" && len(fs.Args()) > 0 {
		*query = strings.Join(fs.Args(), " ")
	}

	projectName := *project
	if projectName == "" && !*globalOnly {
		projectName = core.DetectProjectName()
	}

	localStorage, globalStorage := getStorages()
	if localStorage != nil {
		defer localStorage.Close()
	}
	if globalStorage != nil {
		defer globalStorage.Close()
	}

	var results []core.Memory

	if *dbPath != "" {
		s, err := storage.New(*dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
			return 1
		}
		defer s.Close()
		results, _ = s.SearchMemories(projectName, *query, *category, *limit)
	} else if *globalOnly {
		if globalStorage != nil {
			results, _ = globalStorage.SearchMemories(projectName, *query, *category, *limit)
		}
	} else if *localOnly {
		if localStorage != nil {
			results, _ = localStorage.SearchMemories(projectName, *query, *category, *limit)
		}
	} else {
		// Layered search: Local first, then Global
		if localStorage != nil {
			localRes, _ := localStorage.SearchMemories(projectName, *query, *category, *limit)
			results = append(results, localRes...)
		}
		if globalStorage != nil {
			globalRes, _ := globalStorage.SearchMemories(projectName, *query, *category, *limit)
			results = append(results, globalRes...)
		}
	}

	if *asJSON {
		if results == nil {
			results = []core.Memory{}
		}
		_ = json.NewEncoder(os.Stdout).Encode(results)
		return 0
	}

	if len(results) == 0 {
		fmt.Println("🔍 No se encontraron firmas de memoria coincidentes.")
		return 0
	}

	fmt.Printf("🔍 Se encontraron %d memoria(s) [Usa 'cogni get <id|topic-key>' para ver el contenido completo]:\n\n", len(results))
	for _, m := range results {
		srcBadge := "LOCAL"
		if m.Source == "global" {
			srcBadge = "GLOBAL"
		}
		topicStr := ""
		if m.TopicKey != "" {
			topicStr = fmt.Sprintf(" (Key: %s)", m.TopicKey)
		}

		if *full {
			fmt.Printf("━━━ [%s #%d] [%s] %s%s ━━━\n", srcBadge, m.ID, m.ProjectName, m.Title, topicStr)
			fmt.Printf("🏷️ Tags: %s | 📂 Categoría: %s\n", m.Tags, m.Category)
			fmt.Printf("📝 %s\n\n", m.SummarySignature)
		} else {
			preview := m.SummarySignature
			if idx := strings.Index(preview, "|"); idx != -1 {
				preview = strings.TrimSpace(preview[:idx])
			} else if len(preview) > 100 {
				preview = preview[:97] + "..."
			}
			fmt.Printf("• [#%d %s] [%s] %s%s\n", m.ID, srcBadge, m.Category, m.Title, topicStr)
			fmt.Printf("  Tags: %s | %s\n\n", m.Tags, preview)
		}
	}

	return 0
}

func handleGet(args []string) int {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	id := fs.Int64("id", 0, "ID de la memoria")
	topicKey := fs.String("topic-key", "", "TopicKey determinístico")
	project := fs.String("project", "", "Filtrar por proyecto")
	globalOnly := fs.Bool("global", false, "Buscar solo en BD global")
	localOnly := fs.Bool("local", false, "Buscar solo en BD local")
	dbPath := fs.String("db", "", "Ruta a BD")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	// Check positional argument if neither --id nor --topic-key is provided
	if *id <= 0 && *topicKey == "" && len(fs.Args()) > 0 {
		posArg := fs.Args()[0]
		if parsedID, err := strconv.ParseInt(posArg, 10, 64); err == nil && parsedID > 0 {
			*id = parsedID
		} else {
			*topicKey = posArg
		}
	}

	if *id <= 0 && *topicKey == "" {
		fmt.Fprintln(os.Stderr, "Error: Especifica un ID o TopicKey (ej. 'cogni get 6' o 'cogni get arch/auth/jwt').")
		return 1
	}

	projectName := *project
	if projectName == "" && !*globalOnly {
		projectName = core.DetectProjectName()
	}

	localStorage, globalStorage := getStorages()
	if localStorage != nil {
		defer localStorage.Close()
	}
	if globalStorage != nil {
		defer globalStorage.Close()
	}

	var mem *core.Memory
	var err error

	if *dbPath != "" {
		s, err := storage.New(*dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
			return 1
		}
		defer s.Close()
		if *id > 0 {
			mem, err = s.GetMemoryByID(*id)
		} else {
			mem, err = s.GetMemoryByTopicKey(projectName, *topicKey)
		}
	} else {
		if *id > 0 {
			if !*globalOnly && localStorage != nil {
				mem, err = localStorage.GetMemoryByID(*id)
			}
			if mem == nil && !*localOnly && globalStorage != nil {
				mem, err = globalStorage.GetMemoryByID(*id)
			}
		} else {
			if !*globalOnly && localStorage != nil {
				mem, err = localStorage.GetMemoryByTopicKey(projectName, *topicKey)
			}
			if mem == nil && !*localOnly && globalStorage != nil {
				mem, err = globalStorage.GetMemoryByTopicKey(projectName, *topicKey)
			}
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error recuperando memoria: %v\n", err)
		return 1
	}

	if mem == nil {
		fmt.Println("❌ Memoria no encontrada.")
		return 1
	}

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(mem)
		return 0
	}

	srcBadge := "LOCAL"
	if mem.Source == "global" {
		srcBadge = "GLOBAL"
	}

	fmt.Printf("━━━ [%s #%d] [%s] %s ━━━\n", srcBadge, mem.ID, mem.ProjectName, mem.Title)
	if mem.TopicKey != "" {
		fmt.Printf("🔑 Topic Key: %s\n", mem.TopicKey)
	}
	fmt.Printf("📂 Categoría: %s | 🏷️ Tags: %s\n", mem.Category, mem.Tags)
	fmt.Printf("📅 Actualizado: %s\n\n", mem.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("📝 %s\n", mem.SummarySignature)

	return 0
}

func handlePromote(args []string) int {
	fs := flag.NewFlagSet("promote", flag.ExitOnError)
	id := fs.Int64("id", 0, "ID de la memoria a promover")
	to := fs.String("to", "global", "Destino: global o local")
	_ = fs.Parse(args)

	if *id <= 0 && len(fs.Args()) > 0 {
		parsed, _ := strconv.ParseInt(fs.Args()[0], 10, 64)
		*id = parsed
	}

	if *id <= 0 {
		fmt.Fprintln(os.Stderr, "Error: Especifica el ID de la memoria a promover (ej. cogni promote --id 1).")
		return 1
	}

	localStorage, globalStorage := getStorages()
	if localStorage != nil {
		defer localStorage.Close()
	}
	if globalStorage != nil {
		defer globalStorage.Close()
	}

	if localStorage == nil || globalStorage == nil {
		fmt.Fprintln(os.Stderr, "Error: Se requiere tener tanto base de datos local (.cogni/) como global (~/.cogni/) para promover.")
		return 1
	}

	var src, dst *storage.Storage
	var targetName string
	if *to == "local" {
		src = globalStorage
		dst = localStorage
		targetName = "local (.cogni/)"
	} else {
		src = localStorage
		dst = globalStorage
		targetName = "global (~/.cogni/)"
	}

	promoted, err := storage.PromoteMemory(src, dst, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error promoviendo memoria: %v\n", err)
		return 1
	}

	fmt.Printf("🌐 Memoria #%d (\"%s\") promovida con éxito a %s como #%d.\n", *id, promoted.Title, targetName, promoted.ID)
	return 0
}

func handleUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	id := fs.Int64("id", 0, "ID de la memoria a actualizar")
	title := fs.String("title", "", "Nuevo título")
	topicKey := fs.String("topic-key", "", "Nuevo TopicKey determinístico")
	summary := fs.String("summary", "", "Nuevo resumen")
	category := fs.String("category", "", "Nueva categoría")
	tags := fs.String("tags", "", "Nuevos tags")
	global := fs.Bool("global", false, "Base de datos global")
	dbPath := fs.String("db", "", "Ruta a BD")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	if *id <= 0 {
		fmt.Fprintln(os.Stderr, "Error: --id es obligatorio para actualizar.")
		return 1
	}

	s, err := getStorage(*dbPath, *global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	updated, err := s.UpdateMemory(*id, *title, *summary, *category, *tags, *topicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error actualizando memoria: %v\n", err)
		return 1
	}

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(updated)
	} else {
		fmt.Printf("✅ Memoria #%d actualizada con éxito.\n", updated.ID)
	}

	return 0
}

func handleRemove(args []string) int {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	id := fs.Int64("id", 0, "ID de la memoria a eliminar")
	global := fs.Bool("global", false, "Base de datos global")
	dbPath := fs.String("db", "", "Ruta a BD")

	_ = fs.Parse(args)

	// Fallback to positional argument for ID
	if *id <= 0 && len(fs.Args()) > 0 {
		parsed, _ := strconv.ParseInt(fs.Args()[0], 10, 64)
		*id = parsed
	}

	if *id <= 0 {
		fmt.Fprintln(os.Stderr, "Error: Especifica el ID de la memoria a eliminar (ej. cogni remove --id 6).")
		return 1
	}

	s, err := getStorage(*dbPath, *global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	deleted, err := s.DeleteMemory(*id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error eliminando memoria: %v\n", err)
		return 1
	}

	if !deleted {
		fmt.Fprintf(os.Stderr, "Memoria #%d no encontrada.\n", *id)
		return 1
	}

	fmt.Printf("🗑️ Memoria #%d eliminada correctamente.\n", *id)
	return 0
}

func handleShare(args []string) int {
	fs := flag.NewFlagSet("share", flag.ExitOnError)
	project := fs.String("project", "", "Filtrar por proyecto")
	format := fs.String("format", "", "Formato de exportación directa (markdown, json)")
	outFile := fs.String("out", "", "Guardar paquete cifrado en archivo")
	global := fs.Bool("global", false, "Base de datos global")
	dbPath := fs.String("db", "", "Ruta a BD")
	timeout := fs.Duration("timeout", 5*time.Minute, "Tiempo límite de espera para la sesión")

	_ = fs.Parse(args)

	s, err := getStorage(*dbPath, *global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	projectName := *project
	if projectName == "" && !*global {
		projectName = core.DetectProjectName()
	}

	memories, err := s.ListMemories(projectName, "", 500, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listando memorias: %v\n", err)
		return 1
	}
	if len(memories) == 0 {
		fmt.Fprintf(os.Stderr, "No se encontraron memorias para el proyecto '%s'.\n", projectName)
		return 1
	}

	// 1. Exportación clásica a stdout si se especifica --format
	if *format == "json" {
		_ = json.NewEncoder(os.Stdout).Encode(memories)
		return 0
	} else if *format == "markdown" {
		fmt.Printf("# 🧠 Cogni Memory Export — Proyecto: %s\n\n", projectName)
		fmt.Printf("*Total de firmas: %d*\n\n---\n\n", len(memories))
		for _, m := range memories {
			fmt.Printf("### [%s] %s (#%d)\n", m.ProjectName, m.Title, m.ID)
			fmt.Printf("> **Categoría**: `%s` | **Tags**: `%s` | **Fecha**: %s\n\n", m.Category, m.Tags, m.CreatedAt.Format("2006-01-02 15:04"))
			fmt.Printf("%s\n\n---\n\n", m.SummarySignature)
		}
		return 0
	}

	// 2. Exportación a archivo cifrado si se especifica --out
	if *outFile != "" {
		code := network.GeneratePairCode()
		packet := &network.SyncPacket{
			Version:     "2.4.0",
			ProjectName: projectName,
			Timestamp:   time.Now().UTC(),
			Memories:    memories,
			Code:        code,
		}
		raw, _ := json.Marshal(packet)
		encrypted, err := network.Encrypt(raw, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cifrando paquete: %v\n", err)
			return 1
		}
		if err := os.WriteFile(*outFile, encrypted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error escribiendo archivo: %v\n", err)
			return 1
		}
		fmt.Printf("📦 Paquete cifrado exportado en: %s\n", *outFile)
		fmt.Printf("🔑 Código de descifrado: %s\n", code)
		fmt.Printf("Su compañero puede importar con: cogni sync %s --code %s\n", *outFile, code)
		return 0
	}

	// 3. Cogni Network P2P Sharing efímero + Publicación en Relay
	session, err := network.StartShareSession(projectName, memories, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error iniciando sesión P2P: %v\n", err)
		return 1
	}
	defer session.Close()

	relayURL := os.Getenv("COGNI_RELAY_URL")
	if relayURL == "" {
		relayURL = network.DefaultRelayURL
	}

	relayPublished := false
	if err := network.PublishToRelay(relayURL, session.Code, session.EncryptedData); err == nil {
		relayPublished = true
	}

	fmt.Println()
	fmt.Printf("🌐 Cogni Network — Compartiendo Proyecto: [%s]\n", projectName)
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("🔑 Código de Sincronización:  %s\n", session.Code)
	if relayPublished {
		fmt.Printf("☁️ Servidor de Encuentro:     %s (Acceso mundial E2EE)\n", relayURL)
	}
	fmt.Printf("📡 Direcciones directas:\n")
	primaryAddr := ""
	for _, addr := range session.Addresses {
		fmt.Printf("   • %s\n", addr)
		if primaryAddr == "" && !strings.HasPrefix(addr, "127.0.0.1") {
			primaryAddr = addr
		}
	}
	if primaryAddr == "" && len(session.Addresses) > 0 {
		primaryAddr = session.Addresses[0]
	}
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Println("Pase este código a su compañero de equipo.")
	if relayPublished {
		fmt.Printf("Su compañero en cualquier parte del mundo solo debe ejecutar:\n")
		fmt.Printf("   cogni sync %s\n\n", session.Code)
		fmt.Printf("O por conexión directa local/VPN:\n")
		fmt.Printf("   cogni sync %s --from %s\n\n", session.Code, primaryAddr)
	} else {
		fmt.Printf("Su compañero debe ejecutar en su terminal:\n")
		fmt.Printf("   cogni sync %s --from %s\n\n", session.Code, primaryAddr)
	}
	fmt.Println("⏳ Esperando conexión... (Esta sesión se autodestruirá al completarse)")
	fmt.Println("Presione Ctrl+C para cancelar.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\nSesión de compartir cancelada por el usuario.")
		return 0
	case <-session.Done():
		fmt.Println("\n✔ ¡Transferencia completada con éxito! La sesión ha sido autodestruida.")
		return 0
	case <-time.After(*timeout):
		fmt.Println("\n⏰ Tiempo de espera agotado. Sesión cerrada por seguridad.")
		return 0
	}
}

func handleSync(args []string) int {
	var code string
	var localFile string
	var fromAddr string
	var relayURL string
	var global bool
	var dbPath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if (arg == "--from" || arg == "-f") && i+1 < len(args) {
			fromAddr = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--from=") {
			fromAddr = strings.TrimPrefix(arg, "--from=")
		} else if (arg == "--relay" || arg == "-r") && i+1 < len(args) {
			relayURL = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--relay=") {
			relayURL = strings.TrimPrefix(arg, "--relay=")
		} else if (arg == "--code" || arg == "-c") && i+1 < len(args) {
			code = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--code=") {
			code = strings.TrimPrefix(arg, "--code=")
		} else if arg == "--global" || arg == "-g" {
			global = true
		} else if (arg == "--db") && i+1 < len(args) {
			dbPath = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "--db=") {
			dbPath = strings.TrimPrefix(arg, "--db=")
		} else if !strings.HasPrefix(arg, "-") {
			if _, err := os.Stat(arg); err == nil && localFile == "" {
				localFile = arg
			} else if code == "" {
				code = arg
			}
		}
	}

	if code == "" && localFile == "" {
		fmt.Println("Uso: cogni sync <código> [--from <host:puerto>]")
		fmt.Println("  o: cogni sync <archivo.cogni> --code <código>")
		return 1
	}

	s, err := getStorage(dbPath, global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	if localFile != "" {
		data, err := os.ReadFile(localFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error leyendo archivo: %v\n", err)
			return 1
		}
		decrypted, err := network.Decrypt(data, code)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error descifrando paquete: %v\n", err)
			return 1
		}
		var packet network.SyncPacket
		if err := json.Unmarshal(decrypted, &packet); err != nil {
			fmt.Fprintf(os.Stderr, "Paquete corrupto: %v\n", err)
			return 1
		}
		resp, err := network.MergeMemories(s, packet.Memories, packet.Sender)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error al sincronizar: %v\n", err)
			return 1
		}
		fmt.Printf("✔ Sincronización exitosa desde archivo [%s]:\n", localFile)
		fmt.Printf("  • Memorias añadidas: %d\n", resp.Inserted)
		fmt.Printf("  • Conflictos resguardados: %d\n", resp.Conflicts)
		for _, det := range resp.Details {
			fmt.Printf("    - %s\n", det)
		}
		return 0
	}

	if relayURL != "" {
		_ = os.Setenv("COGNI_RELAY_URL", relayURL)
	}

	if fromAddr != "" {
		fmt.Printf("🔄 Conectando directamente con compañero en %s...\n", fromAddr)
	} else {
		targetRelay := relayURL
		if targetRelay == "" {
			targetRelay = os.Getenv("COGNI_RELAY_URL")
		}
		if targetRelay == "" {
			targetRelay = network.DefaultRelayURL
		}
		fmt.Printf("🔄 Conectando con Cogni Network (%s)...\n", targetRelay)
	}

	resp, err := network.SyncFromPeer(s, code, fromAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error en sincronización: %v\n", err)
		return 1
	}

	fmt.Println("✔ Sincronización Cogni Network completada con éxito:")
	fmt.Printf("  • Nuevas memorias añadidas: %d\n", resp.Inserted)
	fmt.Printf("  • Conflictos resguardados en bifurcaciones seguras: %d\n", resp.Conflicts)
	for _, det := range resp.Details {
		fmt.Printf("    - %s\n", det)
	}
	return 0
}

func handleList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	project := fs.String("project", "", "Filtrar por proyecto")
	category := fs.String("category", "", "Filtrar por categoría")
	limit := fs.Int("limit", 50, "Límite")
	global := fs.Bool("global", false, "Base de datos global")
	dbPath := fs.String("db", "", "Ruta a BD")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	s, err := getStorage(*dbPath, *global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	projectName := *project
	if projectName == "" && !*global {
		projectName = core.DetectProjectName()
	}

	memories, err := s.ListMemories(projectName, *category, *limit, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listando: %v\n", err)
		return 1
	}

	if *asJSON {
		if memories == nil {
			memories = []core.Memory{}
		}
		_ = json.NewEncoder(os.Stdout).Encode(memories)
		return 0
	}

	fmt.Printf("📋 Listado de memorias en [%s] (%d):\n\n", projectName, len(memories))
	for _, m := range memories {
		fmt.Printf("• [#%d] [%s] %s (%s)\n  %s\n\n", m.ID, m.ProjectName, m.Title, m.Category, m.SummarySignature)
	}

	return 0
}

func handleStats(args []string) int {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	global := fs.Bool("global", false, "Base de datos global")
	dbPath := fs.String("db", "", "Ruta a BD")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	s, err := getStorage(*dbPath, *global)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error conectando a BD: %v\n", err)
		return 1
	}
	defer s.Close()

	stats, err := s.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error consultando estadísticas: %v\n", err)
		return 1
	}

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(stats)
		return 0
	}

	fmt.Println("📊 Estadísticas de Memoria en Cogni:")
	fmt.Printf("• Total de Memorias: %d\n", stats.TotalMemories)
	fmt.Printf("• Total de Proyectos: %d\n", stats.TotalProjects)
	fmt.Printf("• Tokens Ahorrados Estimados: ~%d tokens\n", stats.EstimatedTokensSaved)
	fmt.Printf("• Base de Datos: %s\n", s.DBPath())

	return 0
}

func handleOptimize(args []string) int {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	global := fs.Bool("global", false, "Optimizar exclusivamente base de datos global")
	all := fs.Bool("all", true, "Optimizar tanto bases de datos locales como globales")
	asJSON := fs.Bool("json", false, "Salida en formato JSON")
	_ = fs.Parse(args)

	type optResult struct {
		Storage string                 `json:"storage"`
		Stats   *storage.OptimizeStats `json:"stats"`
	}
	results := make([]optResult, 0)

	var localStorage, globalStorage *storage.Storage
	if *global {
		globalPath := core.ResolveDatabasePath("", true)
		s, err := storage.NewWithSource(globalPath, "global")
		if err == nil {
			defer s.Close()
			globalStorage = s
		}
	} else if *all {
		localStorage, globalStorage = getStorages()
		if localStorage != nil {
			defer localStorage.Close()
		}
		if globalStorage != nil {
			defer globalStorage.Close()
		}
	} else {
		localStorage, _ = getStorages()
		if localStorage != nil {
			defer localStorage.Close()
		}
	}

	if localStorage != nil {
		if st, err := localStorage.Optimize(); err == nil {
			results = append(results, optResult{Storage: "local", Stats: st})
		}
	}
	if globalStorage != nil && (localStorage == nil || localStorage.DBPath() != globalStorage.DBPath()) {
		if st, err := globalStorage.Optimize(); err == nil {
			results = append(results, optResult{Storage: "global", Stats: st})
		}
	}

	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(results)
		return 0
	}

	fmt.Println("🧹 Optimización de Base de Datos y FTS5:")
	for _, r := range results {
		fmt.Printf("• [%s] %s\n", strings.ToUpper(r.Storage), r.Stats.DBPath)
		fmt.Printf("  - Total Registros: %d\n", r.Stats.TotalRows)
		fmt.Printf("  - Tamaño Previo:   %d bytes\n", r.Stats.BytesBefore)
		fmt.Printf("  - Tamaño Posterior: %d bytes\n", r.Stats.BytesAfter)
		if r.Stats.SavedBytes > 0 {
			fmt.Printf("  - Espacio Recuperado: %d bytes\n", r.Stats.SavedBytes)
		}
	}
	fmt.Println("✔ Reindexación FTS5, WAL checkpoint y VACUUM completados con éxito.")
	return 0
}

func handleUI(args []string) int {
	fs := flag.NewFlagSet("ui", flag.ExitOnError)
	port := fs.Int("port", 3000, "Puerto inicial para el servidor HTTP")
	host := fs.String("host", "127.0.0.1", "Host para el servidor HTTP")
	noBrowser := fs.Bool("no-browser", false, "No abrir el navegador automáticamente")
	globalOnly := fs.Bool("global", false, "Usar exclusivamente base de datos global")
	dbPath := fs.String("db", "", "Ruta a BD personalizada")

	_ = fs.Parse(args)

	var localStorage, globalStorage *storage.Storage
	if *dbPath != "" {
		s, err := storage.New(*dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error inicializando base de datos: %v\n", err)
			return 1
		}
		defer s.Close()
		globalStorage = s
	} else if *globalOnly {
		globalPath := core.ResolveDatabasePath("", true)
		s, err := storage.NewWithSource(globalPath, "global")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error inicializando base de datos global: %v\n", err)
			return 1
		}
		defer s.Close()
		globalStorage = s
	} else {
		localStorage, globalStorage = getStorages()
		if localStorage != nil {
			defer localStorage.Close()
		}
		if globalStorage != nil {
			defer globalStorage.Close()
		}
	}

	srv := server.New(localStorage, globalStorage, *host, *port)
	if _, err := srv.Start(!*noBrowser); err != nil {
		fmt.Fprintf(os.Stderr, "Error iniciando servidor UI: %v\n", err)
		return 1
	}

	return 0
}

func handleUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ExitOnError)
	purgeDB := fs.Bool("purge-db", false, "Eliminar también las bases de datos globales en ~/.cogni")
	force := fs.Bool("yes", false, "Omitir confirmación interactiva")

	_ = fs.Parse(args)

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error obteniendo directorio home: %v\n", err)
		return 1
	}

	fmt.Println("🗑️ Desinstalando Cogni...")

	// 1. Remove binary
	binPath := filepath.Join(home, ".local", "bin", "cogni")
	if _, err := os.Stat(binPath); err == nil {
		_ = os.Remove(binPath)
		fmt.Printf("  -> Binario eliminado: %s\n", binPath)
	}

	// 2. Remove skills from AI harnesses
	skillPaths := []string{
		filepath.Join(home, ".gemini", "config", "skills", "cogni"),
		filepath.Join(home, ".gemini", "config", "skills", "agent-memory"),
		filepath.Join(home, ".cursor", "skills", "cogni"),
		filepath.Join(home, ".cursor", "skills", "agent-memory"),
		filepath.Join(home, ".claude", "skills", "cogni"),
		filepath.Join(home, ".claude", "skills", "agent-memory"),
		filepath.Join(home, ".config", "opencode", "skills", "cogni"),
		filepath.Join(home, ".agents", "skills", "cogni"),
		filepath.Join(home, ".copilot", "skills", "cogni"),
		filepath.Join(home, ".hermes", "skills", "cogni"),
		filepath.Join(home, ".codex", "skills", "cogni"),
	}

	for _, p := range skillPaths {
		if _, err := os.Stat(p); err == nil {
			_ = os.RemoveAll(p)
			fmt.Printf("  -> Skill eliminada: %s\n", p)
		}
	}

	// 2.1 Remove Copilot instructions installed in VS Code user prompts
	vscodeInstructionPaths := []string{
		filepath.Join(home, ".config", "Code", "User", "prompts", "cogni-copilot.instructions.md"),
	}
	if runtime.GOOS == "darwin" {
		vscodeInstructionPaths = append(vscodeInstructionPaths,
			filepath.Join(home, "Library", "Application Support", "Code", "User", "prompts", "cogni-copilot.instructions.md"),
		)
	}

	for _, p := range vscodeInstructionPaths {
		if _, err := os.Stat(p); err == nil {
			_ = os.Remove(p)
			fmt.Printf("  -> Instrucción Copilot eliminada: %s\n", p)
		}
	}

	// 2.2 Remove Codex MCP entry from ~/.codex/config.toml (preserving other settings)
	codexMCPPaths := []string{
		filepath.Join(home, ".codex", "config.toml"),
	}
	for _, p := range codexMCPPaths {
		if _, err := os.Stat(p); err == nil {
			if err := core.RemoveCodexMCPServer(p, "cogni"); err == nil {
				fmt.Printf("  -> MCP Codex eliminado: %s\n", p)
			}
		}
	}

	// 3. Purge DB if requested or confirmed
	if *purgeDB {
		cogniDir := filepath.Join(home, ".cogni")
		_ = os.RemoveAll(cogniDir)
		fmt.Printf("  -> Base de datos global eliminada: %s\n", cogniDir)
	} else if !*force {
		fmt.Println("\n💡 Nota: Las bases de datos en ~/.cogni/ se conservaron.")
		fmt.Println("   Para eliminarlas ejecuta: cogni uninstall --purge-db")
	}

	fmt.Println("✅ Desinstalación de Cogni completada con éxito.")
	return 0
}

func performAtomicUpgrade(latestTag string) error {
	execPath, err := os.Executable()
	if err != nil || execPath == "" {
		home, _ := os.UserHomeDir()
		execPath = filepath.Join(home, ".local", "bin", "cogni")
	}
	execPath, _ = filepath.EvalSymlinks(execPath)
	binDir := filepath.Dir(execPath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	platform := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	downloadURL := fmt.Sprintf("https://github.com/AdelysAlberto/cogni-memory/releases/download/%s/cogni_%s", latestTag, platform)

	tempFile, err := os.CreateTemp(binDir, "cogni.tmp.*")
	if err != nil {
		return fmt.Errorf("creando archivo temporal: %w", err)
	}
	tempPath := tempFile.Name()

	cleanedUp := false
	cleanup := func() {
		if !cleanedUp {
			cleanedUp = true
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
		}
	}
	defer cleanup()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	done := make(chan error, 1)
	go func() {
		select {
		case <-sigChan:
			cleanup()
			fmt.Println("\n⏭️ Actualización cancelada por el usuario. La versión actual se mantiene intacta.")
			os.Exit(0)
		case <-done:
			return
		}
	}()

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		done <- err
		return fmt.Errorf("descargando release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("código HTTP %d al descargar binario desde GitHub", resp.StatusCode)
		done <- err
		return err
	}

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		done <- err
		return fmt.Errorf("escribiendo binario: %w", err)
	}
	_ = tempFile.Close()

	if err := os.Chmod(tempPath, 0755); err != nil {
		done <- err
		return err
	}

	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-d", "com.apple.quarantine", tempPath).Run()
		_ = exec.Command("codesign", "-s", "-", "-f", tempPath).Run()
	}

	// Atomic rename swap
	oldPath := filepath.Join(binDir, fmt.Sprintf("cogni.old.%d", time.Now().UnixNano()))
	_ = os.Rename(execPath, oldPath)
	if err := os.Rename(tempPath, execPath); err != nil {
		_ = os.Rename(oldPath, execPath) // rollback
		done <- err
		return fmt.Errorf("reemplazando binario: %w", err)
	}
	_ = os.Remove(oldPath)
	cleanedUp = true
	done <- nil

	return nil
}

func handleUpgrade(args []string) int {
	fmt.Println("🔍 Comprobando actualizaciones por tags en GitHub (AdelysAlberto/cogni-memory)...")

	latestTag, releaseURL, err := fetchLatestTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ No se pudo comprobar la última tag remota: %v\n", err)
		return 1
	}

	current := "v" + strings.TrimPrefix(Version, "v")
	current = strings.Fields(current)[0] // Clean extra suffixes if any
	latest := "v" + strings.TrimPrefix(latestTag, "v")

	fmt.Printf("• Versión local:  %s\n", current)
	fmt.Printf("• Versión remota: %s\n", latest)

	cmp := compareSemver(current, latest)
	if cmp == 0 {
		fmt.Printf("✨ Ya estás ejecutando la última versión de Cogni (%s).\n", current)
		return 0
	}

	if cmp > 0 {
		fmt.Printf("✨ Tu versión local (%s) es más nueva que la tag remota (%s).\n", current, latest)
		fmt.Println("⏭️ No se realizará actualización para evitar una posible degradación.")
		return 0
	}

	fmt.Printf("\n🚀 ¡Nueva versión disponible: %s! (%s)\n", latest, releaseURL)
	fmt.Println("📥 Descargando e instalando actualización...")

	if err := performAtomicUpgrade(latest); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Falló la actualización atómica directa: %v\n", err)
		fmt.Println("🔄 Intentando mediante script de instalación como fallback...")

		cmd := exec.Command("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/AdelysAlberto/cogni-memory/main/install.sh | bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error durante la actualización: %v\n", err)
			return 1
		}
	} else {
		// Auto-actualizar skills y reglas de los arneses configurados sin fricción
		home, _ := os.UserHomeDir()
		if home != "" {
			promptAndInstallSkills("", true)
		}
	}

	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		appPath := filepath.Join(home, "Applications", "CogniBar.app")
		if dirExists(appPath) {
			_ = exec.Command("pkill", "-x", "CogniBar").Run()
			_ = exec.Command("open", appPath).Run()
		}
	}

	fmt.Printf("\n🎉 ¡Cogni ha sido actualizado con éxito a la versión %s!\n", latest)
	return 0
}

func compareSemver(local, remote string) int {
	localParts := parseSemver(local)
	remoteParts := parseSemver(remote)

	for i := 0; i < 3; i++ {
		if localParts[i] > remoteParts[i] {
			return 1
		}
		if localParts[i] < remoteParts[i] {
			return -1
		}
	}

	return 0
}

func parseSemver(version string) [3]int {
	v := strings.TrimSpace(version)
	v = strings.TrimPrefix(v, "v")

	// Remove build/prerelease suffixes before splitting.
	for _, sep := range []string{"-", "+", " "} {
		if idx := strings.Index(v, sep); idx >= 0 {
			v = v[:idx]
			break
		}
	}

	parts := strings.Split(v, ".")
	out := [3]int{0, 0, 0}

	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}
		out[i] = n
	}

	return out
}

func isStrictSemverTag(tag string) bool {
	v := strings.TrimSpace(strings.TrimPrefix(tag, "v"))
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}

	for _, p := range parts {
		if p == "" {
			return false
		}
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}

	return true
}

func fetchLatestTag() (string, string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	bestTag := ""

	for page := 1; page <= 3; page++ {
		url := fmt.Sprintf("https://api.github.com/repos/AdelysAlberto/cogni-memory/tags?per_page=100&page=%d", page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", "", err
		}
		req.Header.Set("User-Agent", "Cogni-CLI")

		resp, err := client.Do(req)
		if err != nil {
			return "", "", err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return "", "", fmt.Errorf("código HTTP %d recibido de GitHub", resp.StatusCode)
		}

		var tags []struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			resp.Body.Close()
			return "", "", err
		}
		resp.Body.Close()

		if len(tags) == 0 {
			break
		}

		for _, t := range tags {
			candidate := "v" + strings.TrimPrefix(strings.TrimSpace(t.Name), "v")
			if !isStrictSemverTag(candidate) {
				continue
			}

			if bestTag == "" || compareSemver(candidate, bestTag) > 0 {
				bestTag = candidate
			}
		}
	}

	if bestTag == "" {
		return "", "", fmt.Errorf("no se encontraron tags semánticas válidas")
	}

	releaseURL := fmt.Sprintf("https://github.com/AdelysAlberto/cogni-memory/releases/tag/%s", bestTag)
	return bestTag, releaseURL, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
