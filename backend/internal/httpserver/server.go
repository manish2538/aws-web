package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/local/aws-local-dashboard/internal/commands"
	"github.com/local/aws-local-dashboard/internal/profiles"
	"github.com/local/aws-local-dashboard/internal/services"
	"github.com/local/aws-local-dashboard/internal/types"
)

type Server struct {
	costService     services.CostService
	resourceService services.ResourceService
	profileManager  *profiles.Manager
	commandManager  *commands.Manager
	staticDir       string
	clearCaches     func()
}

// NewServer wires HTTP routes for the API and static frontend.
func NewServer(costService services.CostService, resourceService services.ResourceService, profileManager *profiles.Manager, commandManager *commands.Manager, staticDir string, clearCaches func()) http.Handler {
	s := &Server{
		costService:     costService,
		resourceService: resourceService,
		profileManager:  profileManager,
		commandManager:  commandManager,
		staticDir:       staticDir,
		clearCaches:     clearCaches,
	}

	mux := http.NewServeMux()

	mux.Handle("/api/cost", loggingMiddleware(http.HandlerFunc(s.handleCost)))
	mux.Handle("/api/services", loggingMiddleware(http.HandlerFunc(s.handleServices)))
	mux.Handle("/api/services/", loggingMiddleware(http.HandlerFunc(s.handleServiceResources)))
	mux.Handle("/api/resources/summary", loggingMiddleware(http.HandlerFunc(s.handleResourcesSummary)))
	mux.Handle("/api/profiles", loggingMiddleware(http.HandlerFunc(s.handleProfiles)))
	mux.Handle("/api/profiles/select", loggingMiddleware(http.HandlerFunc(s.handleSelectProfile)))
	mux.Handle("/api/cache/clear", loggingMiddleware(http.HandlerFunc(s.handleCacheClear)))
	mux.Handle("/api/commands", loggingMiddleware(http.HandlerFunc(s.handleCommands)))
	mux.Handle("/api/commands/execute", loggingMiddleware(http.HandlerFunc(s.handleExecuteCommand)))
	mux.Handle("/api/commands/execute-raw", loggingMiddleware(http.HandlerFunc(s.handleExecuteRawCommand)))

	// SPA handler for React build output
	mux.Handle("/", loggingMiddleware(spaHandler(staticDir, "index.html")))

	return mux
}

type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleCost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	start := q.Get("start")
	end := q.Get("end")

	overview, err := s.costService.GetCostOverview(r.Context(), start, end)
	if err != nil {
		if err == services.ErrCostExplorerDisabled {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{
				Error:   "Cost Explorer not enabled",
				Details: "AWS Cost Explorer is not enabled for this account. Enable it in the AWS console to view cost data.",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error:   "Failed to fetch cost overview",
			Details: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, types.CostResponse{
		Overview: overview,
	})
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	start := q.Get("start")
	end := q.Get("end")

	overview, err := s.costService.GetCostOverview(r.Context(), start, end)
	if err != nil {
		if err == services.ErrCostExplorerDisabled {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{
				Error:   "Cost Explorer not enabled",
				Details: "AWS Cost Explorer is not enabled for this account. Enable it in the AWS console to view cost data.",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error:   "Failed to fetch cost overview",
			Details: err.Error(),
		})
		return
	}

	svcCosts, err := s.costService.GetServiceCosts(r.Context(), start, end)
	if err != nil {
		if err == services.ErrCostExplorerDisabled {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{
				Error:   "Cost Explorer not enabled",
				Details: "AWS Cost Explorer is not enabled for this account. Enable it in the AWS console to view cost data.",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error:   "Failed to fetch service costs",
			Details: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, types.ServicesResponse{
		Overview: overview,
		Services: svcCosts,
	})
}

func (s *Server) handleServiceResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/services/")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "Service name is required",
		})
		return
	}

	// Path format: /api/services/{service}/resources
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[1] != "resources" {
		writeJSON(w, http.StatusNotFound, errorResponse{
			Error: "Not found",
		})
		return
	}

	service := parts[0]

	region := r.URL.Query().Get("region")

	resources, err := s.resourceService.GetResources(r.Context(), service, region)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error:   "Failed to fetch resources",
			Details: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, resources)
}

// handleResourcesSummary aggregates a lightweight summary of resources for each
// supported service so the UI can show which services are in use, even when
// cost is zero.
func (s *Server) handleResourcesSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type svcDef struct {
		Key         string
		DisplayName string
		ResourceKey string
	}

	servicesToCheck := []svcDef{
		{Key: "ec2", DisplayName: "EC2", ResourceKey: "ec2Instances"},
		{Key: "vpc", DisplayName: "VPC", ResourceKey: "vpcs"},
		{Key: "eip", DisplayName: "Elastic IPs", ResourceKey: "elasticIps"},
		{Key: "s3", DisplayName: "S3", ResourceKey: "s3Buckets"},
		{Key: "rekognition", DisplayName: "Rekognition", ResourceKey: "rekognitionCollections"},
		{Key: "rds", DisplayName: "RDS", ResourceKey: "rdsInstances"},
	}

	ctx := r.Context()

	type result struct {
		Svc   svcDef
		Count int
		Err   error
	}

	resultsCh := make(chan result, len(servicesToCheck))

	for _, svc := range servicesToCheck {
		svc := svc
		go func() {
			res, err := s.resourceService.GetResources(ctx, svc.Key, "all")
			if err != nil {
				resultsCh <- result{Svc: svc, Err: err}
				return
			}

			count := 0
			switch svc.ResourceKey {
			case "ec2Instances":
				count = len(res.EC2)
			case "vpcs":
				count = len(res.VPCs)
			case "elasticIps":
				count = len(res.ElasticIPs)
			case "s3Buckets":
				count = len(res.S3Buckets)
			case "rekognitionCollections":
				count = len(res.RekognitionCollections)
			case "rdsInstances":
				count = len(res.RDSInstances)
			}

			resultsCh <- result{Svc: svc, Count: count}
		}()
	}

	var summaries []types.ResourceSummary

	for i := 0; i < len(servicesToCheck); i++ {
		r := <-resultsCh
		if r.Err != nil {
			// For now, we ignore individual service errors so one failing
			// call doesn't break the whole summary.
			log.Printf("resources summary: error fetching %s: %v", r.Svc.Key, r.Err)
			continue
		}
		summaries = append(summaries, types.ResourceSummary{
			Service:      r.Svc.Key,
			DisplayName:  r.Svc.DisplayName,
			ResourceType: r.Svc.ResourceKey,
			Count:        r.Count,
		})
	}

	writeJSON(w, http.StatusOK, types.ResourcesSummaryResponse{
		Summaries: summaries,
	})
}

// handleProfiles handles:
// - GET /api/profiles : returns current profile status
// - POST /api/profiles : creates and activates a new custom profile
func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if s.profileManager == nil {
			writeJSON(w, http.StatusOK, profiles.Status{})
			return
		}
		writeJSON(w, http.StatusOK, s.profileManager.Status())
		return
	}

	if r.Method == http.MethodPost {
		if s.profileManager == nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error: "Profile management not configured on server",
			})
			return
		}

		var body struct {
			Name            string `json:"name"`
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
			SessionToken    string `json:"sessionToken"`
			Region          string `json:"region"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "Invalid request body",
				Details: err.Error(),
			})
			return
		}

		_, err := s.profileManager.AddAndActivateProfile(r.Context(), body.Name, body.AccessKeyID, body.SecretAccessKey, body.SessionToken, body.Region)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "Failed to add profile",
				Details: err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, s.profileManager.Status())
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// handleSelectProfile handles POST /api/profiles/select to switch active profile.
func (s *Server) handleSelectProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.profileManager == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "Profile management not configured on server",
		})
		return
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	if err := s.profileManager.SetActiveProfile(body.ID); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "Failed to select profile",
			Details: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, s.profileManager.Status())
}

// handleCacheClear clears in-memory caches so subsequent requests refetch data.
func (s *Server) handleCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.clearCaches != nil {
		s.clearCaches()
	}
	w.WriteHeader(http.StatusNoContent)
}

// basic safety filter to reject obviously destructive raw AWS CLI operations.
func isSafeAWSArgs(args []string) bool {
	if len(args) == 0 {
		return false
	}
	joined := strings.ToLower(strings.Join(args, " "))
	// Very conservative blocklist – focuses on irreversible or mutating verbs.
	block := []string{
		" delete-", " delete",
		" terminate-", " terminate",
		" stop-", " stop ",
		" start-", " start ",
		" reboot-", " reboot",
		" destroy", " drop-",
		" modify-", " update-",
		" put-", " create-",
		" attach-", " detach-",
	}
	for _, b := range block {
		if strings.Contains(joined, b) {
			return false
		}
	}
	return true
}

// handleCommands returns the list of configured read-only AWS CLI commands.
func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.commandManager == nil {
		writeJSON(w, http.StatusOK, []commands.PublicCommand{})
		return
	}
	writeJSON(w, http.StatusOK, s.commandManager.List())
}

// handleExecuteCommand executes a configured read-only AWS CLI command.
func (s *Server) handleExecuteCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.commandManager == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "Command execution is not configured on server",
		})
		return
	}

	var body struct {
		ID     string `json:"id"`
		Region string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	out, args, err := s.commandManager.Execute(r.Context(), body.ID, body.Region)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "usage: aws") || strings.Contains(msg, "argument command: Invalid choice") {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "Invalid AWS command configuration",
				Details: "The configured command is not a valid aws CLI command. Please check command-config.json.",
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "Failed to execute command",
			Details: msg,
		})
		return
	}

	res := struct {
		Command string          `json:"command"`
		Output  json.RawMessage `json:"output"`
	}{
		Command: "aws " + strings.Join(args, " "),
		Output:  json.RawMessage(out),
	}

	writeJSON(w, http.StatusOK, res)
}

// handleExecuteRawCommand executes arbitrary read-only AWS CLI commands as entered
// by the user, after passing a conservative safety filter.
func (s *Server) handleExecuteRawCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.commandManager == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Error: "Command execution is not configured on server",
		})
		return
	}

	var body struct {
		Args string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	fields := strings.Fields(body.Args)
	if len(fields) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "No command provided",
		})
		return
	}

	if !isSafeAWSArgs(fields) {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "Command blocked by safety filter",
			Details: "Only read/list/describe operations are allowed from the dashboard.",
		})
		return
	}

	out, args, err := s.commandManager.ExecuteRaw(r.Context(), fields)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "usage: aws") || strings.Contains(msg, "argument command: Invalid choice") {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "Invalid AWS CLI syntax",
				Details: "Use: <service> <operation> [parameters], e.g. 'ec2 describe-instances --region ap-south-1'.",
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error:   "Failed to execute command",
			Details: msg,
		})
		return
	}

	res := struct {
		Command string          `json:"command"`
		Output  json.RawMessage `json:"output"`
	}{
		Command: "aws " + strings.Join(args, " "),
		Output:  json.RawMessage(out),
	}

	writeJSON(w, http.StatusOK, res)
}

// spaHandler serves a built SPA from a static directory, falling back to index.html
// for unknown routes (for client-side routing).
func spaHandler(staticDir, indexFile string) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API routes are handled separately.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the requested file.
		path := filepath.Join(staticDir, filepath.Clean(r.URL.Path))

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// File exists, serve it.
			fs.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA.
		http.ServeFile(w, r, filepath.Join(staticDir, indexFile))
	})
}

// loggingMiddleware logs basic request information.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
