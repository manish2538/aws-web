package awscli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/local/aws-local-dashboard/internal/cache"
	"github.com/local/aws-local-dashboard/internal/profiles"
	"github.com/local/aws-local-dashboard/internal/services"
	"github.com/local/aws-local-dashboard/internal/types"
)

type resourceService struct {
	exec Executor
}

// NewResourceService creates a ResourceService implementation backed by the AWS CLI.
func NewResourceService(exec Executor) services.ResourceService {
	return &resourceService{
		exec: exec,
	}
}

// NewCachedResourceService wraps a ResourceService with an in-memory cache so
// repeated calls within a short TTL don't re-hit the AWS CLI.
type cachedResourceService struct {
	inner          services.ResourceService
	cache          *cache.Cache[types.ServiceResources]
	profileManager *profiles.Manager
}

func NewCachedResourceService(inner services.ResourceService, c *cache.Cache[types.ServiceResources], pm *profiles.Manager) services.ResourceService {
	return &cachedResourceService{
		inner:          inner,
		cache:          c,
		profileManager: pm,
	}
}

func (c *cachedResourceService) GetResources(ctx context.Context, service, region string) (types.ServiceResources, error) {
	activeProfile := "system"
	if c.profileManager != nil {
		if id := c.profileManager.ActiveID(); id != "" {
			activeProfile = id
		}
	}

	key := fmt.Sprintf("%s|%s|%s", activeProfile, strings.ToLower(service), strings.ToLower(region))

	if cached, ok := c.cache.Get(key); ok {
		return cached, nil
	}

	res, err := c.inner.GetResources(ctx, service, region)
	if err != nil {
		return types.ServiceResources{}, err
	}

	c.cache.Set(key, res)
	return res, nil
}

func (s *resourceService) GetResources(ctx context.Context, service, region string) (types.ServiceResources, error) {
	key := strings.ToLower(service)

	switch key {
	case "ec2":
		return s.getEC2Instances(ctx, region)
	case "vpc":
		return s.getVPCs(ctx, region)
	case "eip", "elasticip", "elastic-ips":
		return s.getElasticIPs(ctx, region)
	case "s3":
		return s.getS3Buckets(ctx)
	case "rekognition":
		return s.getRekognitionCollections(ctx, region)
	case "rds":
		return s.getRDSInstances(ctx, region)
	default:
		return types.ServiceResources{
			Service: service,
			Message: fmt.Sprintf("Resource drilldown not implemented for service %q", service),
		}, nil
	}
}

// EC2

type ec2DescribeInstancesOutput struct {
	Reservations []struct {
		Instances []struct {
			InstanceID   string `json:"InstanceId"`
			InstanceType string `json:"InstanceType"`
			PrivateIP    string `json:"PrivateIpAddress,omitempty"`
			PublicIP     string `json:"PublicIpAddress,omitempty"`
			State        struct {
				Name string `json:"Name"`
			} `json:"State"`
			Placement struct {
				AvailabilityZone string `json:"AvailabilityZone"`
			} `json:"Placement"`
			Tags []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"Tags"`
		} `json:"Instances"`
	} `json:"Reservations"`
}

func (s *resourceService) getEC2Instances(ctx context.Context, region string) (types.ServiceResources, error) {
	if strings.ToLower(region) == "all" {
		return s.getEC2InstancesAllRegions(ctx)
	}

	return s.getEC2InstancesSingleRegion(ctx, region)
}

func (s *resourceService) getEC2InstancesSingleRegion(ctx context.Context, region string) (types.ServiceResources, error) {
	args := []string{"ec2", "describe-instances"}
	if region != "" {
		args = append(args, "--region", region)
	}

	out, err := s.exec.RunJSON(ctx, args...)
	if err != nil {
		return types.ServiceResources{}, err
	}

	var resp ec2DescribeInstancesOutput
	if err := json.Unmarshal(out, &resp); err != nil {
		return types.ServiceResources{}, fmt.Errorf("failed to parse describe-instances output: %w", err)
	}

	var instances []types.EC2Instance
	for _, r := range resp.Reservations {
		for _, inst := range r.Instances {
			name := ""
			for _, t := range inst.Tags {
				if t.Key == "Name" {
					name = t.Value
					break
				}
			}

			instRegion := region
			if instRegion == "" && inst.Placement.AvailabilityZone != "" {
				// Derive region from AZ (e.g. us-east-1a -> us-east-1).
				az := inst.Placement.AvailabilityZone
				if len(az) > 1 {
					instRegion = az[:len(az)-1]
				}
			}

			instances = append(instances, types.EC2Instance{
				InstanceID:       inst.InstanceID,
				Name:             name,
				State:            inst.State.Name,
				InstanceType:     inst.InstanceType,
				AvailabilityZone: inst.Placement.AvailabilityZone,
				PrivateIP:        inst.PrivateIP,
				PublicIP:         inst.PublicIP,
				Region:           instRegion,
			})
		}
	}

	return types.ServiceResources{
		Service: "ec2",
		EC2:     instances,
	}, nil
}

func (s *resourceService) getEC2InstancesAllRegions(ctx context.Context) (types.ServiceResources, error) {
	regions, err := s.listRegions(ctx)
	if err != nil {
		return types.ServiceResources{}, err
	}

	type result struct {
		region    string
		instances []types.EC2Instance
		err       error
	}

	resultsCh := make(chan result, len(regions))
	var wg sync.WaitGroup

	// Limit concurrency to avoid hammering AWS or exhausting local resources.
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	for _, rgn := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := s.getEC2InstancesSingleRegion(ctx, region)
			if err != nil {
				resultsCh <- result{region: region, err: err}
				return
			}
			resultsCh <- result{region: region, instances: res.EC2}
		}(rgn)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var all []types.EC2Instance
	var skipped []string

	for r := range resultsCh {
		if r.err != nil {
			if isAuthError(r.err) {
				skipped = append(skipped, r.region)
				continue
			}
			return types.ServiceResources{}, r.err
		}
		all = append(all, r.instances...)
	}

	msg := ""
	if len(skipped) > 0 {
		msg = fmt.Sprintf("Skipped regions due to authentication errors: %s", strings.Join(skipped, ", "))
	}

	return types.ServiceResources{
		Service: "ec2",
		EC2:     all,
		Message: msg,
	}, nil
}

// VPC

type ec2DescribeVpcsOutput struct {
	VPCs []struct {
		VpcID     string `json:"VpcId"`
		CIDRBlock string `json:"CidrBlock"`
		IsDefault bool   `json:"IsDefault"`
		State     string `json:"State"`
		Tags      []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"Tags"`
	} `json:"Vpcs"`
}

func (s *resourceService) getVPCs(ctx context.Context, region string) (types.ServiceResources, error) {
	if strings.ToLower(region) == "all" {
		return s.getVPCsAllRegions(ctx)
	}

	return s.getVPCsSingleRegion(ctx, region)
}

func (s *resourceService) getVPCsSingleRegion(ctx context.Context, region string) (types.ServiceResources, error) {
	args := []string{"ec2", "describe-vpcs"}
	if region != "" {
		args = append(args, "--region", region)
	}

	out, err := s.exec.RunJSON(ctx, args...)
	if err != nil {
		return types.ServiceResources{}, err
	}

	var resp ec2DescribeVpcsOutput
	if err := json.Unmarshal(out, &resp); err != nil {
		return types.ServiceResources{}, fmt.Errorf("failed to parse describe-vpcs output: %w", err)
	}

	var vpcs []types.VPC
	for _, v := range resp.VPCs {
		name := ""
		for _, t := range v.Tags {
			if t.Key == "Name" {
				name = t.Value
				break
			}
		}

		vpcRegion := region
		vpcs = append(vpcs, types.VPC{
			VpcID:     v.VpcID,
			Name:      name,
			CIDRBlock: v.CIDRBlock,
			State:     v.State,
			IsDefault: v.IsDefault,
			Region:    vpcRegion,
		})
	}

	return types.ServiceResources{
		Service: "vpc",
		VPCs:    vpcs,
	}, nil
}

func (s *resourceService) getVPCsAllRegions(ctx context.Context) (types.ServiceResources, error) {
	regions, err := s.listRegions(ctx)
	if err != nil {
		return types.ServiceResources{}, err
	}

	type result struct {
		region string
		vpcs   []types.VPC
		err    error
	}

	resultsCh := make(chan result, len(regions))
	var wg sync.WaitGroup

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	for _, rgn := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := s.getVPCsSingleRegion(ctx, region)
			if err != nil {
				resultsCh <- result{region: region, err: err}
				return
			}
			resultsCh <- result{region: region, vpcs: res.VPCs}
		}(rgn)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var all []types.VPC
	var skipped []string

	for r := range resultsCh {
		if r.err != nil {
			if isAuthError(r.err) {
				skipped = append(skipped, r.region)
				continue
			}
			return types.ServiceResources{}, r.err
		}
		all = append(all, r.vpcs...)
	}

	msg := ""
	if len(skipped) > 0 {
		msg = fmt.Sprintf("Skipped regions due to authentication errors: %s", strings.Join(skipped, ", "))
	}

	return types.ServiceResources{
		Service: "vpc",
		VPCs:    all,
		Message: msg,
	}, nil
}

// Elastic IPs

type ec2DescribeAddressesOutput struct {
	Addresses []struct {
		AllocationID       string `json:"AllocationId"`
		PublicIP           string `json:"PublicIp"`
		AssociationID      string `json:"AssociationId,omitempty"`
		InstanceID         string `json:"InstanceId,omitempty"`
		NetworkInterfaceID string `json:"NetworkInterfaceId,omitempty"`
		Domain             string `json:"Domain,omitempty"`
	} `json:"Addresses"`
}

func (s *resourceService) getElasticIPs(ctx context.Context, region string) (types.ServiceResources, error) {
	if strings.ToLower(region) == "all" {
		return s.getElasticIPsAllRegions(ctx)
	}

	return s.getElasticIPsSingleRegion(ctx, region)
}

func (s *resourceService) getElasticIPsSingleRegion(ctx context.Context, region string) (types.ServiceResources, error) {
	args := []string{"ec2", "describe-addresses"}
	if region != "" {
		args = append(args, "--region", region)
	}

	out, err := s.exec.RunJSON(ctx, args...)
	if err != nil {
		return types.ServiceResources{}, err
	}

	var resp ec2DescribeAddressesOutput
	if err := json.Unmarshal(out, &resp); err != nil {
		return types.ServiceResources{}, fmt.Errorf("failed to parse describe-addresses output: %w", err)
	}

	var eips []types.ElasticIP
	for _, a := range resp.Addresses {
		eips = append(eips, types.ElasticIP{
			AllocationID:       a.AllocationID,
			PublicIP:           a.PublicIP,
			AssociationID:      a.AssociationID,
			InstanceID:         a.InstanceID,
			NetworkInterfaceID: a.NetworkInterfaceID,
			Domain:             a.Domain,
			Region:             region,
		})
	}

	return types.ServiceResources{
		Service:    "eip",
		ElasticIPs: eips,
	}, nil
}

func (s *resourceService) getElasticIPsAllRegions(ctx context.Context) (types.ServiceResources, error) {
	regions, err := s.listRegions(ctx)
	if err != nil {
		return types.ServiceResources{}, err
	}

	type result struct {
		region string
		eips   []types.ElasticIP
		err    error
	}

	resultsCh := make(chan result, len(regions))
	var wg sync.WaitGroup

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	for _, rgn := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := s.getElasticIPsSingleRegion(ctx, region)
			if err != nil {
				resultsCh <- result{region: region, err: err}
				return
			}
			resultsCh <- result{region: region, eips: res.ElasticIPs}
		}(rgn)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var all []types.ElasticIP
	var skipped []string

	for r := range resultsCh {
		if r.err != nil {
			if isAuthError(r.err) {
				skipped = append(skipped, r.region)
				continue
			}
			return types.ServiceResources{}, r.err
		}
		all = append(all, r.eips...)
	}

	msg := ""
	if len(skipped) > 0 {
		msg = fmt.Sprintf("Skipped regions due to authentication errors: %s", strings.Join(skipped, ", "))
	}

	return types.ServiceResources{
		Service:    "eip",
		ElasticIPs: all,
		Message:    msg,
	}, nil
}

// S3

type s3ListBucketsOutput struct {
	Buckets []struct {
		Name         string `json:"Name"`
		CreationDate string `json:"CreationDate"`
	} `json:"Buckets"`
}

// getS3Buckets ignores the region parameter because S3 is global; we
// return all buckets visible to the account and include their region if
// it can be determined cheaply.
func (s *resourceService) getS3Buckets(ctx context.Context) (types.ServiceResources, error) {
	out, err := s.exec.RunJSON(ctx, "s3api", "list-buckets")
	if err != nil {
		return types.ServiceResources{}, err
	}

	var resp s3ListBucketsOutput
	if err := json.Unmarshal(out, &resp); err != nil {
		return types.ServiceResources{}, fmt.Errorf("failed to parse list-buckets output: %w", err)
	}

	var buckets []types.S3Bucket
	for _, b := range resp.Buckets {
		buckets = append(buckets, types.S3Bucket{
			Name:         b.Name,
			CreationDate: b.CreationDate,
			Region:       "", // Region resolution per bucket can be added later if needed.
		})
	}

	return types.ServiceResources{
		Service:   "s3",
		S3Buckets: buckets,
	}, nil
}

// Rekognition

type rekognitionListCollectionsOutput struct {
	CollectionIDs     []string `json:"CollectionIds"`
	FaceModelVersions []string `json:"FaceModelVersions"`
}

func (s *resourceService) getRekognitionCollections(ctx context.Context, region string) (types.ServiceResources, error) {
	if strings.ToLower(region) == "all" {
		return s.getRekognitionCollectionsAllRegions(ctx)
	}

	return s.getRekognitionCollectionsSingleRegion(ctx, region)
}

func (s *resourceService) getRekognitionCollectionsSingleRegion(ctx context.Context, region string) (types.ServiceResources, error) {
	args := []string{"rekognition", "list-collections"}
	if region != "" {
		args = append(args, "--region", region)
	}

	out, err := s.exec.RunJSON(ctx, args...)
	if err != nil {
		return types.ServiceResources{}, err
	}

	var resp rekognitionListCollectionsOutput
	if err := json.Unmarshal(out, &resp); err != nil {
		return types.ServiceResources{}, fmt.Errorf("failed to parse list-collections output: %w", err)
	}

	var collections []types.RekognitionCollection
	for i, id := range resp.CollectionIDs {
		faceModel := ""
		if i < len(resp.FaceModelVersions) {
			faceModel = resp.FaceModelVersions[i]
		}
		collections = append(collections, types.RekognitionCollection{
			CollectionID:     id,
			FaceModelVersion: faceModel,
			Region:           region,
		})
	}

	return types.ServiceResources{
		Service:                "rekognition",
		RekognitionCollections: collections,
	}, nil
}

func (s *resourceService) getRekognitionCollectionsAllRegions(ctx context.Context) (types.ServiceResources, error) {
	regions, err := s.listRegions(ctx)
	if err != nil {
		return types.ServiceResources{}, err
	}

	type result struct {
		region      string
		collections []types.RekognitionCollection
		err         error
	}

	resultsCh := make(chan result, len(regions))
	var wg sync.WaitGroup

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	for _, rgn := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := s.getRekognitionCollectionsSingleRegion(ctx, region)
			if err != nil {
				resultsCh <- result{region: region, err: err}
				return
			}
			resultsCh <- result{region: region, collections: res.RekognitionCollections}
		}(rgn)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var all []types.RekognitionCollection
	var skipped []string

	for r := range resultsCh {
		if r.err != nil {
			if isAuthError(r.err) {
				skipped = append(skipped, r.region)
				continue
			}
			return types.ServiceResources{}, r.err
		}
		all = append(all, r.collections...)
	}

	msg := ""
	if len(skipped) > 0 {
		msg = fmt.Sprintf("Skipped regions due to authentication errors: %s", strings.Join(skipped, ", "))
	}

	return types.ServiceResources{
		Service:                "rekognition",
		RekognitionCollections: all,
		Message:                msg,
	}, nil
}

// RDS

type rdsDescribeDBInstancesOutput struct {
	DBInstances []struct {
		DBInstanceIdentifier string `json:"DBInstanceIdentifier"`
		DBInstanceClass      string `json:"DBInstanceClass"`
		Engine               string `json:"Engine"`
		DBInstanceStatus     string `json:"DBInstanceStatus"`
		AvailabilityZone     string `json:"AvailabilityZone"`
		Endpoint             struct {
			Address string `json:"Address"`
		} `json:"Endpoint"`
		MultiAZ bool `json:"MultiAZ"`
	} `json:"DBInstances"`
}

func (s *resourceService) getRDSInstances(ctx context.Context, region string) (types.ServiceResources, error) {
	if strings.ToLower(region) == "all" {
		return s.getRDSInstancesAllRegions(ctx)
	}
	return s.getRDSInstancesSingleRegion(ctx, region)
}

func (s *resourceService) getRDSInstancesSingleRegion(ctx context.Context, region string) (types.ServiceResources, error) {
	args := []string{"rds", "describe-db-instances"}
	if region != "" {
		args = append(args, "--region", region)
	}

	out, err := s.exec.RunJSON(ctx, args...)
	if err != nil {
		return types.ServiceResources{}, err
	}

	var resp rdsDescribeDBInstancesOutput
	if err := json.Unmarshal(out, &resp); err != nil {
		return types.ServiceResources{}, fmt.Errorf("failed to parse describe-db-instances output: %w", err)
	}

	var dbs []types.RDSInstance
	for _, db := range resp.DBInstances {
		dbs = append(dbs, types.RDSInstance{
			DBInstanceIdentifier: db.DBInstanceIdentifier,
			Engine:               db.Engine,
			Status:               db.DBInstanceStatus,
			DBInstanceClass:      db.DBInstanceClass,
			AvailabilityZone:     db.AvailabilityZone,
			Endpoint:             db.Endpoint.Address,
			Region:               region,
		})
	}

	return types.ServiceResources{
		Service:      "rds",
		RDSInstances: dbs,
	}, nil
}

func (s *resourceService) getRDSInstancesAllRegions(ctx context.Context) (types.ServiceResources, error) {
	regions, err := s.listRegions(ctx)
	if err != nil {
		return types.ServiceResources{}, err
	}

	type result struct {
		region string
		dbs    []types.RDSInstance
		err    error
	}

	resultsCh := make(chan result, len(regions))
	var wg sync.WaitGroup

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	for _, rgn := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := s.getRDSInstancesSingleRegion(ctx, region)
			if err != nil {
				resultsCh <- result{region: region, err: err}
				return
			}
			resultsCh <- result{region: region, dbs: res.RDSInstances}
		}(rgn)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var all []types.RDSInstance
	var skipped []string

	for r := range resultsCh {
		if r.err != nil {
			if isAuthError(r.err) {
				skipped = append(skipped, r.region)
				continue
			}
			return types.ServiceResources{}, r.err
		}
		all = append(all, r.dbs...)
	}

	msg := ""
	if len(skipped) > 0 {
		msg = fmt.Sprintf("Skipped regions due to authentication errors: %s", strings.Join(skipped, ", "))
	}

	return types.ServiceResources{
		Service:      "rds",
		RDSInstances: all,
		Message:      msg,
	}, nil
}

// listRegions returns the list of region names for the account.
func (s *resourceService) listRegions(ctx context.Context) ([]string, error) {
	out, err := s.exec.RunJSON(ctx, "ec2", "describe-regions", "--all-regions")
	if err != nil {
		return nil, err
	}

	var payload struct {
		Regions []struct {
			RegionName  string `json:"RegionName"`
			OptInStatus string `json:"OptInStatus"`
		} `json:"Regions"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse describe-regions output: %w", err)
	}

	var regions []string
	for _, r := range payload.Regions {
		if r.RegionName == "" {
			continue
		}
		// Skip regions that are not opted in for this account.
		if strings.EqualFold(r.OptInStatus, "not-opted-in") {
			continue
		}
		regions = append(regions, r.RegionName)
	}
	return regions, nil
}

// isAuthError returns true if the error looks like an AWS auth/credential error
// or a region/endpoint that is not available for this service. In both cases
// we treat the region as skippable when aggregating across regions.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "authfailure") ||
		strings.Contains(msg, "not able to validate the provided access credentials") ||
		strings.Contains(msg, "invalidclienttokenid") ||
		strings.Contains(msg, "could not connect to the endpoint url")
}
