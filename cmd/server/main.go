package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"cookie-router/internal/config"
	"cookie-router/internal/web"

	// Register all providers
	_ "cookie-router/internal/providers/chatgpt"
	_ "cookie-router/internal/providers/claude"
	_ "cookie-router/internal/providers/custom"
	_ "cookie-router/internal/providers/deepseek"
	_ "cookie-router/internal/providers/gemini"
	_ "cookie-router/internal/providers/grok"
	_ "cookie-router/internal/providers/kimi"
	_ "cookie-router/internal/providers/perplexity"
	_ "cookie-router/internal/providers/poe"
	_ "cookie-router/internal/providers/qwen"
)

var (
	version   = "1.0.0"
	buildDate = "dev"
)

func main() {
	configPath := flag.String("config", "", "Path to config.json (default: auto-detect)")
	port := flag.Int("port", 0, "Override listen port (default: from config or 8080)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("cookie-router v%s (built %s)\n", version, buildDate)
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Banner
	fmt.Println("┌──────────────────────────────────────┐")
	fmt.Println("│  🍪 Cookie Router v" + version + "              │")
	fmt.Println("│  Lightweight Web Cookie LLM Gateway   │")
	fmt.Println("│  Built for STB Armbian (ARM64)        │")
	fmt.Println("└──────────────────────────────────────┘")

	cfgMgr, err := config.NewManager(*configPath)
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	cfg := cfgMgr.GetConfig()
	listenPort := cfg.Port
	if *port > 0 {
		listenPort = *port
	}
	if listenPort <= 0 {
		listenPort = 8080
	}

	srv := web.NewServer(cfgMgr)

	addr := fmt.Sprintf("0.0.0.0:%d", listenPort)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received %v, shutting down...", sig)
		httpServer.Close()
	}()

	log.Printf("🚀 Listening on http://%s", addr)
	log.Printf("📊 Dashboard: http://localhost:%d/dashboard", listenPort)
	log.Printf("🔌 API: http://localhost:%d/v1/chat/completions", listenPort)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped.")
}
