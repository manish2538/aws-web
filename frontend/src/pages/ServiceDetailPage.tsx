import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  ServiceResources,
  fetchServiceResources,
  EC2Instance,
  VPC,
  ElasticIP,
  S3Bucket,
  RekognitionCollection,
  RDSInstance,
} from '../api/client';

const SERVICE_CONFIG: Record<
  string,
  { title: string; icon: string; resourceType: string }
> = {
  ec2: { title: 'EC2 Instances', icon: '🖥️', resourceType: 'instances' },
  vpc: { title: 'Virtual Private Clouds', icon: '🌐', resourceType: 'VPCs' },
  eip: { title: 'Elastic IP Addresses', icon: '📍', resourceType: 'addresses' },
  s3: { title: 'S3 Buckets', icon: '🪣', resourceType: 'buckets' },
  rekognition: { title: 'Rekognition Collections', icon: '👁️', resourceType: 'collections' },
  rds: { title: 'RDS DB Instances', icon: '🗄️', resourceType: 'instances' },
};

const REGIONS = [
  { value: 'all', label: 'All Regions' },
  { value: 'us-east-1', label: 'US East (N. Virginia)' },
  { value: 'us-east-2', label: 'US East (Ohio)' },
  { value: 'us-west-1', label: 'US West (N. California)' },
  { value: 'us-west-2', label: 'US West (Oregon)' },
  { value: 'eu-west-1', label: 'Europe (Ireland)' },
  { value: 'eu-west-2', label: 'Europe (London)' },
  { value: 'eu-central-1', label: 'Europe (Frankfurt)' },
  { value: 'ap-south-1', label: 'Asia Pacific (Mumbai)' },
  { value: 'ap-southeast-1', label: 'Asia Pacific (Singapore)' },
  { value: 'ap-southeast-2', label: 'Asia Pacific (Sydney)' },
  { value: 'ap-northeast-1', label: 'Asia Pacific (Tokyo)' },
  { value: 'ap-northeast-2', label: 'Asia Pacific (Seoul)' },
  { value: 'ca-central-1', label: 'Canada (Central)' },
  { value: 'sa-east-1', label: 'South America (São Paulo)' },
];

const EC2_STATES = [
  { value: 'all', label: 'All States' },
  { value: 'running', label: 'Running' },
  { value: 'stopped', label: 'Stopped' },
  { value: 'pending', label: 'Pending' },
  { value: 'shutting-down', label: 'Shutting Down' },
  { value: 'terminated', label: 'Terminated' },
];

function ServiceDetailPage() {
  const { serviceKey } = useParams<{ serviceKey: string }>();
  const [data, setData] = useState<ServiceResources | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [region, setRegion] = useState<string>('all');
  const [ec2StateFilter, setEc2StateFilter] = useState<string>('all');

  useEffect(() => {
    if (!serviceKey) return;
    let cancelled = false;
    const key = serviceKey; // Capture for closure

    async function load() {
      try {
        setLoading(true);
        setError(null);
        const resp = await fetchServiceResources(key, region === 'all' ? 'all' : region);
        if (cancelled) return;
        setData(resp);
      } catch (e: any) {
        if (cancelled) return;
        setError(e.message || 'Failed to load resources');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [serviceKey, region]);

  const config = serviceKey ? SERVICE_CONFIG[serviceKey] : null;
  const title = config?.title || serviceKey || 'Service';

  const ec2Instances = data?.ec2Instances ?? [];
  const vpcs = data?.vpcs ?? [];
  const eips = data?.elasticIps ?? [];
  const s3Buckets = data?.s3Buckets ?? [];
  const collections = data?.rekognitionCollections ?? [];
  const rdsInstances = data?.rdsInstances ?? [];

  const filteredEc2 =
    serviceKey === 'ec2'
      ? ec2Instances.filter((inst) =>
          ec2StateFilter === 'all' ? true : inst.state === ec2StateFilter,
        )
      : ec2Instances;

  const getResourceCount = () => {
    if (serviceKey === 'ec2') return filteredEc2.length;
    if (serviceKey === 'vpc') return vpcs.length;
    if (serviceKey === 'eip') return eips.length;
    if (serviceKey === 's3') return s3Buckets.length;
    if (serviceKey === 'rekognition') return collections.length;
    if (serviceKey === 'rds') return rdsInstances.length;
    return 0;
  };

  return (
    <div className="page">
      {/* Page Header */}
      <div className="page-header">
        <div className="page-breadcrumb">
          <Link to="/">Cost Explorer</Link>
          <span>›</span>
          <span>{title}</span>
        </div>
        <h1 className="page-title">
          {config?.icon && <span style={{ marginRight: 12 }}>{config.icon}</span>}
          {title}
        </h1>
        <p className="page-subtitle">
          View and manage your {config?.resourceType || 'resources'} across regions.
        </p>
      </div>

      {/* Toolbar */}
      <div className="toolbar">
        <div className="toolbar-section">
          <div className="toolbar-group">
            <span className="toolbar-label">Region</span>
            <select
              value={region}
              onChange={(e) => setRegion(e.target.value)}
              className="form-select form-input-sm"
              style={{ minWidth: 200 }}
            >
              {REGIONS.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </select>
          </div>
          {serviceKey === 'ec2' && (
            <>
              <div className="toolbar-divider" />
              <div className="toolbar-group">
                <span className="toolbar-label">State</span>
                <select
                  value={ec2StateFilter}
                  onChange={(e) => setEc2StateFilter(e.target.value)}
                  className="form-select form-input-sm"
                  style={{ minWidth: 140 }}
                >
                  {EC2_STATES.map((s) => (
                    <option key={s.value} value={s.value}>
                      {s.label}
                    </option>
                  ))}
                </select>
              </div>
            </>
          )}
        </div>
        <div className="toolbar-section">
          <Link to="/" className="btn btn-secondary btn-sm">
            ← Back to Cost Explorer
          </Link>
        </div>
      </div>

      {/* Loading */}
      {loading && (
        <div className="card">
          <div className="loading-state">
            <div className="spinner" />
            <span>Loading resources...</span>
          </div>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="alert alert-error">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Message */}
      {!loading && !error && data?.message && (
        <div className="alert alert-info">{data.message}</div>
      )}

      {/* Resource Count */}
      {!loading && !error && (
        <div className="metric-card" style={{ maxWidth: 280 }}>
          <div className="metric-label">Total Resources</div>
          <div className="metric-value">{getResourceCount()}</div>
          <div className="metric-detail">
            {region === 'all' ? 'Across all regions' : `In ${region}`}
          </div>
        </div>
      )}

      {/* EC2 */}
      {!loading && !error && serviceKey === 'ec2' && filteredEc2.length > 0 && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">EC2 Instances</span>
            <span className="badge">{filteredEc2.length} instances</span>
          </div>
          <div className="table-container">
            <EC2Table instances={filteredEc2} />
          </div>
        </div>
      )}

      {/* VPC */}
      {!loading && !error && serviceKey === 'vpc' && vpcs.length > 0 && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">VPCs</span>
            <span className="badge">{vpcs.length} VPCs</span>
          </div>
          <div className="table-container">
            <VPCTable vpcs={vpcs} />
          </div>
        </div>
      )}

      {/* EIP */}
      {!loading && !error && serviceKey === 'eip' && eips.length > 0 && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">Elastic IPs</span>
            <span className="badge">{eips.length} addresses</span>
          </div>
          <div className="table-container">
            <EIPTable eips={eips} />
          </div>
        </div>
      )}

      {/* S3 */}
      {!loading && !error && serviceKey === 's3' && s3Buckets.length > 0 && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">S3 Buckets</span>
            <span className="badge">{s3Buckets.length} buckets</span>
          </div>
          <div className="table-container">
            <S3Table buckets={s3Buckets} />
          </div>
        </div>
      )}

      {/* Rekognition */}
      {!loading && !error && serviceKey === 'rekognition' && collections.length > 0 && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">Rekognition Collections</span>
            <span className="badge">{collections.length} collections</span>
          </div>
          <div className="table-container">
            <RekognitionTable collections={collections} />
          </div>
        </div>
      )}

      {/* RDS */}
      {!loading && !error && serviceKey === 'rds' && rdsInstances.length > 0 && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">RDS DB Instances</span>
            <span className="badge">{rdsInstances.length} instances</span>
          </div>
          <div className="table-container">
            <RDSTable instances={rdsInstances} />
          </div>
        </div>
      )}

      {/* Empty State */}
      {!loading &&
        !error &&
        !data?.message &&
        serviceKey &&
        getResourceCount() === 0 && (
          <div className="card">
            <div className="empty-state">
              <div className="empty-state-title">No resources found</div>
              <div className="empty-state-description">
                No {config?.resourceType || 'resources'} were found in the selected region.
              </div>
            </div>
          </div>
        )}
    </div>
  );
}

function StateTag({ state }: { state: string }) {
  const className =
    state === 'running'
      ? 'badge-success'
      : state === 'stopped'
      ? 'badge-error'
      : state === 'pending' || state === 'shutting-down'
      ? 'badge-warning'
      : '';
  return <span className={`badge ${className}`}>{state}</span>;
}

function EC2Table({ instances }: { instances: EC2Instance[] }) {
  return (
    <table className="table">
      <thead>
        <tr>
          <th>Instance ID</th>
          <th>Name</th>
          <th>State</th>
          <th>Type</th>
          <th>Availability Zone</th>
          <th>Region</th>
          <th>Private IP</th>
          <th>Public IP</th>
        </tr>
      </thead>
      <tbody>
        {instances.map((inst) => (
          <tr key={inst.instanceId}>
            <td className="text-mono">{inst.instanceId}</td>
            <td>{inst.name || <span className="text-muted">—</span>}</td>
            <td>
              <StateTag state={inst.state} />
            </td>
            <td className="text-mono">{inst.instanceType}</td>
            <td>{inst.availabilityZone}</td>
            <td>{inst.region || <span className="text-muted">—</span>}</td>
            <td className="text-mono">{inst.privateIp || <span className="text-muted">—</span>}</td>
            <td className="text-mono">{inst.publicIp || <span className="text-muted">—</span>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function VPCTable({ vpcs }: { vpcs: VPC[] }) {
  return (
    <table className="table">
      <thead>
        <tr>
          <th>VPC ID</th>
          <th>Name</th>
          <th>CIDR Block</th>
          <th>State</th>
          <th>Default</th>
          <th>Region</th>
        </tr>
      </thead>
      <tbody>
        {vpcs.map((vpc) => (
          <tr key={vpc.vpcId}>
            <td className="text-mono">{vpc.vpcId}</td>
            <td>{vpc.name || <span className="text-muted">—</span>}</td>
            <td className="text-mono">{vpc.cidrBlock}</td>
            <td>
              <span className={`badge ${vpc.state === 'available' ? 'badge-success' : ''}`}>
                {vpc.state}
              </span>
            </td>
            <td>{vpc.isDefault ? <span className="badge badge-info">Default</span> : 'No'}</td>
            <td>{vpc.region || <span className="text-muted">—</span>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function EIPTable({ eips }: { eips: ElasticIP[] }) {
  return (
    <table className="table">
      <thead>
        <tr>
          <th>Allocation ID</th>
          <th>Public IP</th>
          <th>Association ID</th>
          <th>Instance ID</th>
          <th>Network Interface</th>
          <th>Domain</th>
          <th>Region</th>
        </tr>
      </thead>
      <tbody>
        {eips.map((eip) => (
          <tr key={eip.allocationId}>
            <td className="text-mono">{eip.allocationId}</td>
            <td className="text-mono">{eip.publicIp}</td>
            <td className="text-mono">{eip.associationId || <span className="text-muted">—</span>}</td>
            <td className="text-mono">{eip.instanceId || <span className="text-muted">—</span>}</td>
            <td className="text-mono">
              {eip.networkInterfaceId || <span className="text-muted">—</span>}
            </td>
            <td>{eip.domain || <span className="text-muted">—</span>}</td>
            <td>{eip.region || <span className="text-muted">—</span>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function S3Table({ buckets }: { buckets: S3Bucket[] }) {
  return (
    <table className="table">
      <thead>
        <tr>
          <th>Bucket Name</th>
          <th>Creation Date</th>
          <th>Region</th>
        </tr>
      </thead>
      <tbody>
        {buckets.map((b) => (
          <tr key={b.name}>
            <td className="text-mono">{b.name}</td>
            <td>{b.creationDate}</td>
            <td>{b.region || <span className="text-muted">—</span>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function RekognitionTable({ collections }: { collections: RekognitionCollection[] }) {
  return (
    <table className="table">
      <thead>
        <tr>
          <th>Collection ID</th>
          <th>Face Model Version</th>
          <th>Region</th>
        </tr>
      </thead>
      <tbody>
        {collections.map((c) => (
          <tr key={c.collectionId}>
            <td className="text-mono">{c.collectionId}</td>
            <td>{c.faceModelVersion || <span className="text-muted">—</span>}</td>
            <td>{c.region || <span className="text-muted">—</span>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function RDSTable({ instances }: { instances: RDSInstance[] }) {
  return (
    <table className="table">
      <thead>
        <tr>
          <th>DB Identifier</th>
          <th>Engine</th>
          <th>Status</th>
          <th>Class</th>
          <th>Availability Zone</th>
          <th>Endpoint</th>
          <th>Region</th>
        </tr>
      </thead>
      <tbody>
        {instances.map((db) => (
          <tr key={db.dbInstanceIdentifier}>
            <td className="text-mono">{db.dbInstanceIdentifier}</td>
            <td>{db.engine}</td>
            <td>
              <span
                className={`badge ${
                  db.status === 'available' ? 'badge-success' : 'badge-warning'
                }`}
              >
                {db.status}
              </span>
            </td>
            <td className="text-mono">{db.dbInstanceClass}</td>
            <td>{db.availabilityZone}</td>
            <td className="text-mono truncate" style={{ maxWidth: 200 }}>
              {db.endpoint || <span className="text-muted">—</span>}
            </td>
            <td>{db.region || <span className="text-muted">—</span>}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default ServiceDetailPage;
