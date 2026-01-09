package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/local/aws-local-dashboard/internal/awscli"
	"github.com/local/aws-local-dashboard/internal/cache"
	"github.com/local/aws-local-dashboard/internal/commands"
	"github.com/local/aws-local-dashboard/internal/httpserver"
	"github.com/local/aws-local-dashboard/internal/profiles"
	"github.com/local/aws-local-dashboard/internal/types"
)

func main() {
	ctx := context.Background()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static"
	}

	cacheTTLSeconds := 60
	if v := os.Getenv("CACHE_TTL_SECONDS"); v != "" {
		if parsed, err := time.ParseDuration(v + "s"); err == nil {
			cacheTTLSeconds = int(parsed.Seconds())
		}
	}

	cacheTTL := time.Duration(cacheTTLSeconds) * time.Second

	// Profile manager handles system vs custom AWS credentials without
	// mutating the user's ~/.aws configuration.
	profileManager := profiles.NewManager(ctx)

	executor := awscli.NewCLIExecutor(profileManager)

	cmdManager, err := commands.LoadManager(executor, os.Getenv("COMMAND_CONFIG_PATH"))
	if err != nil {
		log.Printf("warning: failed to load command config: %v", err)
	}

	costCache := cache.New[awscli.CachedCost](cacheTTL)
	costService := awscli.NewCostService(executor, costCache, profileManager)

	resourceCLI := awscli.NewResourceService(executor)
	resourceCache := cache.New[types.ServiceResources](cacheTTL)
	resourceService := awscli.NewCachedResourceService(resourceCLI, resourceCache, profileManager)

	clearCaches := func() {
		costCache.Clear()
		resourceCache.Clear()
	}

	handler := httpserver.NewServer(costService, resourceService, profileManager, cmdManager, staticDir, clearCaches)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on :%s (static dir: %s)", port, staticDir)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}

	<-ctx.Done()
}
