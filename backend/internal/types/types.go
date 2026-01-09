package types

type CostOverview struct {
	// Total is the total usage cost before credits/discounts for the period.
	Total float64 `json:"total"`
	// NetTotal is the effective cost after credits/discounts for the period.
	NetTotal float64 `json:"netTotal"`
	// CreditsApplied is the absolute value of credits applied in the period.
	CreditsApplied float64 `json:"creditsApplied"`
	Currency       string  `json:"currency"`
	Start          string  `json:"start"`
	End            string  `json:"end"`
}

// ServiceCost represents the cost of a single AWS service.
type ServiceCost struct {
	Service       string  `json:"service"`
	DisplayName   string  `json:"displayName"`
	DrilldownKey  string  `json:"drilldownKey,omitempty"`
	Cost          float64 `json:"cost"`
	Currency      string  `json:"currency"`
}

// CostResponse is returned from /api/cost.
type CostResponse struct {
	Overview CostOverview `json:"overview"`
}

// ServicesResponse is returned from /api/services.
type ServicesResponse struct {
	Overview CostOverview `json:"overview"`
	Services []ServiceCost `json:"services"`
}

// EC2Instance represents a simplified EC2 instance description.
type EC2Instance struct {
	InstanceID       string `json:"instanceId"`
	Name             string `json:"name"`
	State            string `json:"state"`
	InstanceType     string `json:"instanceType"`
	AvailabilityZone string `json:"availabilityZone"`
	PrivateIP        string `json:"privateIp"`
	PublicIP         string `json:"publicIp"`
	Region           string `json:"region"`
}

// VPC represents a simplified VPC description.
type VPC struct {
	VpcID     string `json:"vpcId"`
	Name      string `json:"name"`
	CIDRBlock string `json:"cidrBlock"`
	State     string `json:"state"`
	IsDefault bool   `json:"isDefault"`
	Region    string `json:"region"`
}

// ElasticIP represents a simplified Elastic IP description.
type ElasticIP struct {
	AllocationID       string `json:"allocationId"`
	PublicIP           string `json:"publicIp"`
	AssociationID      string `json:"associationId"`
	InstanceID         string `json:"instanceId"`
	NetworkInterfaceID string `json:"networkInterfaceId"`
	Domain             string `json:"domain"`
	Region             string `json:"region"`
}

// ServiceResources is returned from /api/services/{service}/resources.
type ServiceResources struct {
	Service                string                  `json:"service"`
	EC2                    []EC2Instance           `json:"ec2Instances,omitempty"`
	VPCs                   []VPC                   `json:"vpcs,omitempty"`
	ElasticIPs             []ElasticIP             `json:"elasticIps,omitempty"`
	S3Buckets              []S3Bucket              `json:"s3Buckets,omitempty"`
	RekognitionCollections []RekognitionCollection `json:"rekognitionCollections,omitempty"`
	RDSInstances           []RDSInstance           `json:"rdsInstances,omitempty"`
	Message                string                  `json:"message,omitempty"`
}

// S3Bucket represents a simplified S3 bucket description.
type S3Bucket struct {
	Name         string `json:"name"`
	CreationDate string `json:"creationDate"`
	Region       string `json:"region"`
}

// RekognitionCollection represents a simplified Rekognition collection.
type RekognitionCollection struct {
	CollectionID     string `json:"collectionId"`
	FaceModelVersion string `json:"faceModelVersion"`
	Region           string `json:"region"`
}

// RDSInstance represents a simplified RDS DB instance.
type RDSInstance struct {
	DBInstanceIdentifier string `json:"dbInstanceIdentifier"`
	Engine               string `json:"engine"`
	Status               string `json:"status"`
	DBInstanceClass      string `json:"dbInstanceClass"`
	AvailabilityZone     string `json:"availabilityZone"`
	Endpoint             string `json:"endpoint"`
	Region               string `json:"region"`
}

// ResourceSummary represents a high-level summary of resources for a service.
type ResourceSummary struct {
	Service      string `json:"service"`
	DisplayName  string `json:"displayName"`
	ResourceType string `json:"resourceType"`
	Count        int    `json:"count"`
}

// ResourcesSummaryResponse is returned from /api/resources/summary.
type ResourcesSummaryResponse struct {
	Summaries []ResourceSummary `json:"summaries"`
}

