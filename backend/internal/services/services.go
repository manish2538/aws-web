package services

import (
	"context"
	"errors"

	"github.com/local/aws-local-dashboard/internal/types"
)

// ErrCostExplorerDisabled is returned when AWS Cost Explorer is not enabled for the account.
var ErrCostExplorerDisabled = errors.New("aws cost explorer is not enabled for this account")

type CostService interface {
	// GetCostOverview returns the overall cost for a period. If start/end are
	// empty, the current month is used.
	GetCostOverview(ctx context.Context, start, end string) (types.CostOverview, error)
	GetServiceCosts(ctx context.Context, start, end string) ([]types.ServiceCost, error)
}

// ResourceService provides resource listings for services.
type ResourceService interface {
	// region can be a specific AWS region (e.g. "us-east-1") or "all" to
	// aggregate across all regions. If empty, the AWS CLI default region is used.
	GetResources(ctx context.Context, service, region string) (types.ServiceResources, error)
}


