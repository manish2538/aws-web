export interface CostOverview {
  total: number;
  netTotal: number;
  creditsApplied: number;
  currency: string;
  start: string;
  end: string;
}

export interface CostResponse {
  overview: CostOverview;
}

export interface ServiceCost {
  service: string;
  displayName: string;
  drilldownKey?: string;
  cost: number;
  currency: string;
}

export interface ServicesResponse {
  overview: CostOverview;
  services: ServiceCost[];
}

export interface EC2Instance {
  instanceId: string;
  name: string;
  state: string;
  instanceType: string;
  availabilityZone: string;
  privateIp: string;
  publicIp: string;
  region: string;
}

export interface VPC {
  vpcId: string;
  name: string;
  cidrBlock: string;
  state: string;
  isDefault: boolean;
  region: string;
}

export interface ElasticIP {
  allocationId: string;
  publicIp: string;
  associationId?: string;
  instanceId?: string;
  networkInterfaceId?: string;
  domain?: string;
  region: string;
}

export interface S3Bucket {
  name: string;
  creationDate: string;
  region: string;
}

export interface RekognitionCollection {
  collectionId: string;
  faceModelVersion: string;
  region: string;
}

export interface RDSInstance {
  dbInstanceIdentifier: string;
  engine: string;
  status: string;
  dbInstanceClass: string;
  availabilityZone: string;
  endpoint: string;
  region: string;
}

export interface ServiceResources {
  service: string;
  ec2Instances?: EC2Instance[];
  vpcs?: VPC[];
  elasticIps?: ElasticIP[];
  s3Buckets?: S3Bucket[];
  rekognitionCollections?: RekognitionCollection[];
   rdsInstances?: RDSInstance[];
  message?: string;
}

export interface ApiError {
  error: string;
  details?: string;
}

export interface PublicProfile {
  id: string;
  name: string;
  source: 'system' | 'custom';
}

export interface ProfileStatus {
  systemAvailable: boolean;
  activeId: string;
  profiles: PublicProfile[];
}

export interface ResourceSummary {
  service: string;
  displayName: string;
  resourceType: string;
  count: number;
}

export interface ResourcesSummaryResponse {
  summaries: ResourceSummary[];
}

export interface PublicCommand {
  id: string;
  label: string;
  description: string;
  service: string;
  supportsRegion: boolean;
}

export interface CommandExecutionResult {
  command: string;
  output: any;
}

async function handleResponse<T>(resp: Response): Promise<T> {
  const contentType = resp.headers.get('content-type') || '';
  const isJSON = contentType.includes('application/json');

  if (!resp.ok) {
    if (isJSON) {
      const data = (await resp.json()) as ApiError;
      const errorMessage = data.error || resp.statusText;
      throw new Error(errorMessage + (data.details ? `: ${data.details}` : ''));
    }
    throw new Error(resp.statusText);
  }

  return (await resp.json()) as T;
}

export async function fetchCost(params?: { start?: string; end?: string }): Promise<CostResponse> {
  const qs = new URLSearchParams();
  if (params?.start) qs.set('start', params.start);
  if (params?.end) qs.set('end', params.end);
  const url = qs.toString() ? `/api/cost?${qs.toString()}` : '/api/cost';
  const resp = await fetch(url);
  return handleResponse<CostResponse>(resp);
}

export async function fetchServices(params?: { start?: string; end?: string }): Promise<ServicesResponse> {
  const qs = new URLSearchParams();
  if (params?.start) qs.set('start', params.start);
  if (params?.end) qs.set('end', params.end);
  const url = qs.toString() ? `/api/services?${qs.toString()}` : '/api/services';
  const resp = await fetch(url);
  return handleResponse<ServicesResponse>(resp);
}

export async function fetchServiceResources(serviceKey: string, region?: string): Promise<ServiceResources> {
  const params = new URLSearchParams();
  if (region) {
    params.set('region', region);
  }
  const qs = params.toString();
  const url = `/api/services/${encodeURIComponent(serviceKey)}/resources${qs ? `?${qs}` : ''}`;
  const resp = await fetch(url);
  return handleResponse<ServiceResources>(resp);
}

export async function fetchProfileStatus(): Promise<ProfileStatus> {
  const resp = await fetch('/api/profiles');
  return handleResponse<ProfileStatus>(resp);
}

export async function createProfile(input: {
  name: string;
  accessKeyId: string;
  secretAccessKey: string;
  sessionToken?: string;
  region?: string;
}): Promise<ProfileStatus> {
  const resp = await fetch('/api/profiles', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
  return handleResponse<ProfileStatus>(resp);
}

export async function selectProfile(id: string): Promise<ProfileStatus> {
  const resp = await fetch('/api/profiles/select', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id }),
  });
  return handleResponse<ProfileStatus>(resp);
}

export async function fetchResourcesSummary(): Promise<ResourcesSummaryResponse> {
  const resp = await fetch('/api/resources/summary');
  return handleResponse<ResourcesSummaryResponse>(resp);
}

export async function clearBackendCache(): Promise<void> {
  const resp = await fetch('/api/cache/clear', { method: 'POST' });
  if (!resp.ok && resp.status !== 204) {
    await handleResponse<void>(resp);
  }
}

export async function fetchCommands(): Promise<PublicCommand[]> {
  const resp = await fetch('/api/commands');
  return handleResponse<PublicCommand[]>(resp);
}

export async function executeCommand(id: string, region?: string): Promise<CommandExecutionResult> {
  const resp = await fetch('/api/commands/execute', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id, region }),
  });
  return handleResponse<CommandExecutionResult>(resp);
}

export async function executeRawCommand(args: string): Promise<CommandExecutionResult> {
  const resp = await fetch('/api/commands/execute-raw', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ args }),
  });
  return handleResponse<CommandExecutionResult>(resp);
}


