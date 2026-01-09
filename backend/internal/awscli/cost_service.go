package awscli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/local/aws-local-dashboard/internal/cache"
	"github.com/local/aws-local-dashboard/internal/profiles"
	"github.com/local/aws-local-dashboard/internal/services"
	"github.com/local/aws-local-dashboard/internal/types"
)

// CachedCost is used by the cost cache.
type CachedCost struct {
	Overview types.CostOverview
	Services []types.ServiceCost
}

type costService struct {
	exec           Executor
	cache          *cache.Cache[CachedCost]
	profileManager *profiles.Manager
}

// NewCostService creates a CostService implementation backed by the AWS CLI.
func NewCostService(exec Executor, cache *cache.Cache[CachedCost], profileManager *profiles.Manager) services.CostService {
	return &costService{
		exec:           exec,
		cache:          cache,
		profileManager: profileManager,
	}
}

func (s *costService) GetCostOverview(ctx context.Context, start, end string) (types.CostOverview, error) {
	cached, err := s.getOrFetch(ctx, start, end)
	return cached.Overview, err
}

func (s *costService) GetServiceCosts(ctx context.Context, start, end string) ([]types.ServiceCost, error) {
	cached, err := s.getOrFetch(ctx, start, end)
	return cached.Services, err
}

func (s *costService) getOrFetch(ctx context.Context, userStart, userEnd string) (CachedCost, error) {
	activeKey := "system"
	if s.profileManager != nil {
		if id := s.profileManager.ActiveID(); id != "" {
			activeKey = id
		}
	}
	ceStart, ceEnd, displayStart, displayEnd := normalizeDateRange(userStart, userEnd)
	cacheKey := fmt.Sprintf("cost-and-services:%s:%s:%s", activeKey, ceStart, ceEnd)
	if val, ok := s.cache.Get(cacheKey); ok {
		return val, nil
	}

	fetched, err := s.fetchFromAWS(ctx, ceStart, ceEnd, displayStart, displayEnd)
	if err != nil {
		return CachedCost{}, err
	}
	s.cache.Set(cacheKey, fetched)
	return fetched, nil
}

type ceResponse struct {
	ResultsByTime []struct {
		TimePeriod struct {
			Start string `json:"Start"`
			End   string `json:"End"`
		} `json:"TimePeriod"`
		Groups []struct {
			Keys    []string `json:"Keys"`
			Metrics map[string]struct {
				Amount string `json:"Amount"`
				Unit   string `json:"Unit"`
			} `json:"Metrics"`
		} `json:"Groups"`
		Total map[string]struct {
			Amount string `json:"Amount"`
			Unit   string `json:"Unit"`
		} `json:"Total"`
	} `json:"ResultsByTime"`
}

func (s *costService) fetchFromAWS(ctx context.Context, ceStart, ceEnd, displayStart, displayEnd string) (CachedCost, error) {
	args := []string{
		"ce", "get-cost-and-usage",
		"--time-period", fmt.Sprintf("Start=%s,End=%s", ceStart, ceEnd),
		"--granularity", "MONTHLY",
		"--metrics", "UnblendedCost",
		"--group-by", "Type=DIMENSION,Key=SERVICE",
	}

	out, err := s.exec.RunJSON(ctx, args...)
	if err != nil {
		// If Cost Explorer is disabled, surface a friendlier error.
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "cost explorer") && strings.Contains(lower, "enable") {
			return CachedCost{}, services.ErrCostExplorerDisabled
		}
		return CachedCost{}, err
	}

	var resp ceResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return CachedCost{}, fmt.Errorf("failed to parse cost explorer response: %w", err)
	}

	if len(resp.ResultsByTime) == 0 {
		return CachedCost{}, fmt.Errorf("no cost data returned from cost explorer")
	}

	r := resp.ResultsByTime[0]

	currency := "USD"

	var servicesCosts []types.ServiceCost
	for _, g := range r.Groups {
		if len(g.Keys) == 0 {
			continue
		}
		name := g.Keys[0]
		metric, ok := g.Metrics["UnblendedCost"]
		if !ok {
			continue
		}
		amount, err := strconv.ParseFloat(metric.Amount, 64)
		if err != nil {
			continue
		}

		displayName, drillKey := normalizeServiceName(name)

		servicesCosts = append(servicesCosts, types.ServiceCost{
			Service:      name,
			DisplayName:  displayName,
			DrilldownKey: drillKey,
			Cost:         amount,
			Currency:     metric.Unit,
		})
	}

	// Add a synthetic EIP service entry for drilldown convenience if not already present.
	hasEIP := false
	for _, sc := range servicesCosts {
		if sc.DrilldownKey == "eip" {
			hasEIP = true
			break
		}
	}
	if !hasEIP {
		servicesCosts = append(servicesCosts, types.ServiceCost{
			Service:      "Elastic IPs",
			DisplayName:  "Elastic IPs",
			DrilldownKey: "eip",
			Cost:         0,
			Currency:     currency,
		})
	}

	// Derive totals and credits using a second query grouped by RECORD_TYPE, so that
	// we can show "usage before credits", "credits applied", and "net" similar to
	// the AWS console.
	usageTotal, creditsApplied, currencyForTotals, err := s.fetchRecordTypeTotals(ctx, ceStart, ceEnd)
	if err != nil {
		// Fallback to the overall UnblendedCost total if the secondary query fails.
		if t, ok := r.Total["UnblendedCost"]; ok {
			if v, parseErr := strconv.ParseFloat(t.Amount, 64); parseErr == nil {
				usageTotal = v
				currency = t.Unit
			}
		}
	} else {
		currency = currencyForTotals
	}

	netTotal := usageTotal - creditsApplied
	if math.Abs(netTotal) < 0.0000001 {
		netTotal = 0
	}

	overview := types.CostOverview{
		Total:          usageTotal,
		NetTotal:       netTotal,
		CreditsApplied: creditsApplied,
		Currency:       currency,
		Start:          displayStart,
		End:            displayEnd,
	}

	return CachedCost{
		Overview: overview,
		Services: servicesCosts,
	}, nil
}

// currentMonthRange returns the start and end dates (YYYY-MM-DD) for the current month in UTC.
func currentMonthRange() (string, string) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	// Cost Explorer expects the end date to be exclusive, so use tomorrow.
	end := now.AddDate(0, 0, 1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

// normalizeDateRange takes optional user-provided inclusive start/end dates and
// returns (ceStart, ceEndExclusive, displayStart, displayEnd). If the input is
// empty or invalid, it falls back to the current month.
func normalizeDateRange(userStart, userEnd string) (string, string, string, string) {
	const layout = "2006-01-02"

	s := strings.TrimSpace(userStart)
	e := strings.TrimSpace(userEnd)

	// Fallback: current month
	useCurrentMonth := func() (string, string, string, string) {
		cs, ce := currentMonthRange()
		displayStart := cs
		displayEnd := ce
		if t, err := time.Parse(layout, ce); err == nil {
			displayEnd = t.AddDate(0, 0, -1).Format(layout)
		}
		return cs, ce, displayStart, displayEnd
	}

	if s == "" || e == "" {
		return useCurrentMonth()
	}

	startTime, err1 := time.Parse(layout, s)
	endTime, err2 := time.Parse(layout, e)
	if err1 != nil || err2 != nil || endTime.Before(startTime) {
		return useCurrentMonth()
	}

	ceStart := startTime.Format(layout)
	ceEndExclusive := endTime.AddDate(0, 0, 1).Format(layout)

	displayStart := startTime.Format(layout)
	displayEnd := endTime.Format(layout)

	return ceStart, ceEndExclusive, displayStart, displayEnd
}

// normalizeServiceName maps verbose Cost Explorer service names to friendly display names and drilldown keys.
func normalizeServiceName(name string) (displayName string, drilldownKey string) {
	lower := strings.ToLower(name)

	switch {
	case strings.Contains(lower, "elastic compute cloud") || strings.HasPrefix(lower, "ec2"):
		return "EC2", "ec2"
	case strings.Contains(lower, "virtual private cloud"):
		return "VPC", "vpc"
	case strings.Contains(lower, "elastic ip"):
		return "Elastic IPs", "eip"
	case strings.Contains(lower, "rekognition"):
		return "Rekognition", "rekognition"
	case strings.Contains(lower, "simple storage service") || strings.Contains(lower, "s3"):
		return "Amazon S3", "s3"
	case strings.Contains(lower, "relational database service"):
		return "RDS", "rds"
	default:
		return name, ""
	}
}

// fetchRecordTypeTotals queries Cost Explorer grouped by RECORD_TYPE so we can
// distinguish usage from credits and compute net totals.
func (s *costService) fetchRecordTypeTotals(ctx context.Context, start, end string) (usageTotal float64, creditsApplied float64, currency string, err error) {
	args := []string{
		"ce", "get-cost-and-usage",
		"--time-period", fmt.Sprintf("Start=%s,End=%s", start, end),
		"--granularity", "MONTHLY",
		"--metrics", "UnblendedCost",
		"--group-by", "Type=DIMENSION,Key=RECORD_TYPE",
	}

	out, execErr := s.exec.RunJSON(ctx, args...)
	if execErr != nil {
		return 0, 0, "", execErr
	}

	var resp struct {
		ResultsByTime []struct {
			Groups []struct {
				Keys    []string `json:"Keys"`
				Metrics map[string]struct {
					Amount string `json:"Amount"`
					Unit   string `json:"Unit"`
				} `json:"Metrics"`
			} `json:"Groups"`
		} `json:"ResultsByTime"`
	}

	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, 0, "", fmt.Errorf("failed to parse cost explorer RECORD_TYPE response: %w", err)
	}

	if len(resp.ResultsByTime) == 0 {
		return 0, 0, "", fmt.Errorf("no cost data returned from cost explorer for RECORD_TYPE breakdown")
	}

	r := resp.ResultsByTime[0]

	currency = "USD"
	var usage, credits float64

	for _, g := range r.Groups {
		if len(g.Keys) == 0 {
			continue
		}
		recordType := strings.ToLower(g.Keys[0])
		metric, ok := g.Metrics["UnblendedCost"]
		if !ok {
			continue
		}
		amount, parseErr := strconv.ParseFloat(metric.Amount, 64)
		if parseErr != nil {
			continue
		}
		currency = metric.Unit

		switch recordType {
		case "usage":
			usage += amount
		case "credit":
			// Credits are represented as negative amounts in Cost Explorer.
			if amount < 0 {
				credits += -amount
			} else {
				credits += amount
			}
		}
	}

	return usage, credits, currency, nil
}
