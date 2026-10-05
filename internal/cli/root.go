package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AdelysAlberto/cogni/internal/core"
	"github.com/AdelysAlberto/cogni/internal/server"
	"github.com/AdelysAlberto/cogni/internal/storage"
)

const Version = "2.0.1 (Go Core)"

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
	case "search":
		return handleSearch(cmdArgs)
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
	case "list":
		return handleList(cmdArgs)
	case "stats":
		return handleStats(cmdArgs)
	case "promote":
		return handlePromote(cmdArgs)
	case "ui":
		return handleUI(cmdArgs)
	case "skill", "skills":
		promptAndInstallSkills(len(cmdArgs) > 0 && cmdArgs[0] == "--all")
		return 0
	case "uninstall":
		return handleUninstall(cmdArgs)
	case "version", "--version", "-v":
		fmt.Printf("🧠 Cogni v%s\n", Version)
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
  init        Inicializa el directorio local .cogni/ en el proyecto actual
  save        Guarda una firma de memoria sintética
  search      Busca firmas de memoria con FTS5 (local y global federado)
  update      Actualiza una memoria existente por su ID
  promote     Promueve una memoria de local a global (o viceversa)
  remove      Elimina una memoria por su ID
  share       Exporta o comparte firmas de memoria (Markdown / JSON)
  list        Lista las memorias registradas
  stats       Muestra métricas y tokens ahorrados
  ui          Abre el dashboard gráfico interactivo en el navegador
  skill       Instala o actualiza el Skill en tus arneses de IA
  uninstall   Desinstala Cogni, elimina el binario y limpia las skills
  version     Muestra la versión de Cogni

Flags Globales:
  --global  Fuerza el uso de la base de datos global (~/.cogni/memory.db)
  --db      Ruta personalizada al archivo SQLite
  --json    Imprime la salida en formato JSON puro

Ejemplos:
  cogni init
  cogni save --title "Auth JWT" --summary "Firma sintética..." --category auth --tags "jwt,tokens"
  cogni search --query "jwt"
  cogni update --id 6 --summary "Nueva firma..."
  cogni remove --id 6
  cogni share --format markdown > memories.md
  cogni ui
`
	fmt.Print(usage)
}

func getStorage(customPath string, forceGlobal bool) (*storage.Storage, error) {
	dbPath := core.ResolveDatabasePath(customPath, forceGlobal)
	return storage.New(dbPath)
}

func handleInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	forceGlobal := fs.Bool("global", false, "Inicializa la base de datos global en ~/.cogni")
	noSkills := fs.Bool("no-skills", false, "Omitir instalación de skills de IA")
	allSkills := fs.Bool("all", false, "Instalar automáticamente en todos los arneses de IA")
	_ = fs.Parse(args)

	if *forceGlobal {
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
		fmt.Printf("✅ Cogni global inicializado en: %s\n", dbPath)
	} else {
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
	}

	if !*noSkills {
		promptAndInstallSkills(*allSkills)
	}

	return 0
}

func promptAndInstallSkills(autoAll bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	harnesses := core.GetHarnessSkillPaths(home)

	choice := ""
	if autoAll {
		choice = "8"
	} else {
		fmt.Println("\n🤖 Selecciona el entorno o Harness de IA que utilizas:")
		fmt.Println("1) Proyecto actual (.agents/skills/)")
		fmt.Println("2) Gemini Antigravity (~/.gemini/config/skills/)")
		fmt.Println("3) Cursor IDE (~/.cursor/skills/)")
		fmt.Println("4) Claude Code / Desktop (~/.claude/skills/)")
		fmt.Println("5) OpenCode (~/.config/opencode/skills/ & ~/.agents/skills/)")
		fmt.Println("6) GitHub Copilot (~/.agents/skills/ & ~/.copilot/skills/)")
		fmt.Println("7) Hermes CLI (~/.hermes/skills/)")
		fmt.Println("8) Instalar en TODOS los entornos detectados (Recomendado)")
		fmt.Println("9) Omitir instalación de Skill")
		fmt.Print("\nIngresa tu opción (1-9) [por defecto: 8]: ")

		var input string
		_, _ = fmt.Scanln(&input)
		choice = strings.TrimSpace(input)
		if choice == "" {
			choice = "8"
		}
	}

	switch choice {
	case "1":
		for _, p := range harnesses["local"] {
			_ = core.InstallSkill(p)
			fmt.Printf("  -> Skill instalada en: %s\n", p)
		}
	case "2":
		for _, p := range harnesses["antigravity"] {
			_ = core.InstallSkill(p)
			fmt.Printf("  -> Skill instalada en Antigravity: %s\n", p)
		}
	case "3":
		for _, p := range harnesses["cursor"] {
			_ = core.InstallSkill(p)
			fmt.Printf("  -> Skill instalada en Cursor: %s\n", p)
		}
	case "4":
		for _, p := range harnesses["claude"] {
			_ = core.InstallSkill(p)
			fmt.Printf("  -> Skill instalada en Claude: %s\n", p)
		}
	case "5":
		for _, p := range harnesses["opencode"] {
			_ = core.InstallSkill(p)
			fmt.Printf("  -> Skill instalada en OpenCode: %s\n", p)
		}
	case "6":
		for _, p := range harnesses["copilot"] {
			_ = core.InstallSkill(p)
			fmt.Printf("  -> Skill instalada en Copilot: %s\n", p)
		}
	case "7":
		for _, p := range harnesses["hermes"] {
			_ = core.InstallSkill(p)
			fmt.Printf("  -> Skill instalada en Hermes: %s\n", p)
		}
	case "8":
		fmt.Println("🚀 Registrando Skill de Cogni en todos los arneses de IA...")
		for name, paths := range harnesses {
			for _, p := range paths {
				_ = core.InstallSkill(p)
			}
			fmt.Printf("  -> Configurado para: %s\n", name)
		}
	case "9":
		fmt.Println("⏭️ Instalación de Skill omitida.")
		return
	default:
		fmt.Println("🚀 Opción por defecto: Registrando en todos los arneses...")
		for name, paths := range harnesses {
			for _, p := range paths {
				_ = core.InstallSkill(p)
			}
			fmt.Printf("  -> Configurado para: %s\n", name)
		}
	}

	fmt.Println("✨ Skills de Cogni configuradas y listas para usar con tus Agentes de IA.")
}

func handleSave(args []string) int {
	fs := flag.NewFlagSet("save", flag.ExitOnError)
	project := fs.String("project", "", "Nombre del proyecto")
	title := fs.String("title", "", "Título o hito de la memoria")
	summary := fs.String("summary", "", "Resumen sintético de la memoria")
	category := fs.String("category", "general", "Categoría")
	tags := fs.String("tags", "", "Tags separados por coma")
	global := fs.Bool("global", false, "Guardar en la base de datos global")
	dbPath := fs.String("db", "", "Ruta personalizada a la base de datos")
	asJSON := fs.Bool("json", false, "Salida en JSON")

	_ = fs.Parse(args)

	if *title == "" || *summary == "" {
		fmt.Fprintln(os.Stderr, "Error: --title y --summary son obligatorios.")
		return 1
	}

	projectName := *project
	if projectName == "" {
		projectName = core.DetectProjectName()
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
		SummarySignature: *summary,
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
		fmt.Printf("Categoría: %s\n", saved.Category)
		fmt.Printf("Tags: %s\n", saved.Tags)
		fmt.Printf("Ubicación BD: %s\n", s.DBPath())
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

	fmt.Printf("🔍 Se encontraron %d memoria(s):\n\n", len(results))
	for _, m := range results {
		srcBadge := "LOCAL"
		if m.Source == "global" {
			srcBadge = "GLOBAL"
		}
		fmt.Printf("━━━ [%s #%d] [%s] %s ━━━\n", srcBadge, m.ID, m.ProjectName, m.Title)
		fmt.Printf("🏷️ Tags: %s | 📂 Categoría: %s\n", m.Tags, m.Category)
		fmt.Printf("📝 %s\n\n", m.SummarySignature)
	}

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

	updated, err := s.UpdateMemory(*id, *title, *summary, *category, *tags)
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
	format := fs.String("format", "markdown", "Formato de exportación (markdown, json)")
	global := fs.Bool("global", false, "Base de datos global")
	dbPath := fs.String("db", "", "Ruta a BD")

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

	if *format == "json" {
		if memories == nil {
			memories = []core.Memory{}
		}
		_ = json.NewEncoder(os.Stdout).Encode(memories)
		return 0
	}

	// Markdown Export
	fmt.Printf("# 🧠 Cogni Memory Export — Proyecto: %s\n\n", projectName)
	fmt.Printf("*Total de firmas: %d*\n\n---\n\n", len(memories))

	for _, m := range memories {
		fmt.Printf("### [%s] %s (#%d)\n", m.ProjectName, m.Title, m.ID)
		fmt.Printf("> **Categoría**: `%s` | **Tags**: `%s` | **Fecha**: %s\n\n", m.Category, m.Tags, m.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("%s\n\n---\n\n", m.SummarySignature)
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
		filepath.Join(home, ".config", "opencode", "skills", "cogni"),
		filepath.Join(home, ".agents", "skills", "cogni"),
		filepath.Join(home, ".copilot", "skills", "cogni"),
		filepath.Join(home, ".hermes", "skills", "cogni"),
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

func handleUpgrade(args []string) int {
	fmt.Println("🔍 Comprobando actualizaciones por tags en GitHub (AdelysAlberto/cogni-memory)...")

	latestTag, releaseURL, err := fetchLatestTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ No se pudo comprobar la última tag remota: %v\n", err)
		return 1
	}

	current := "v" + strings.TrimPrefix(Version, "v")
	current = strings.Split(current, " ")[0] // Clean extra suffixes if any
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

	cmd := exec.Command("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/AdelysAlberto/cogni-memory/main/install.sh | bash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error durante la actualización: %v\n", err)
		return 1
	}

	fmt.Printf("🎉 ¡Cogni ha sido actualizado con éxito a la versión %s!\n", latest)
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
