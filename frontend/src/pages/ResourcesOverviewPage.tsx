import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchResourcesSummary, ResourcesSummaryResponse, ResourceSummary } from '../api/client';

const SERVICE_ICONS: Record<string, string> = {
  ec2: '🖥️',
  vpc: '🌐',
  eip: '📍',
  s3: '🪣',
  rekognition: '👁️',
  rds: '🗄️',
};

function ResourcesOverviewPage() {
  const [data, setData] = useState<ResourcesSummaryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        setLoading(true);
        setError(null);
        const resp = await fetchResourcesSummary();
        if (!cancelled) setData(resp);
      } catch (e: any) {
        if (!cancelled) setError(e.message || 'Failed to load resource summary');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const rows: ResourceSummary[] = data?.summaries ?? [];
  const totalResources = rows.reduce((sum, r) => sum + r.count, 0);

  const onRowClick = (svc: ResourceSummary) => {
    navigate(`/services/${svc.service}`);
  };

  return (
    <div className="page">
      {/* Page Header */}
      <div className="page-header">
        <div className="page-breadcrumb">
          <span>Resource Management</span>
          <span>›</span>
          <span>Overview</span>
        </div>
        <h1 className="page-title">Resources Overview</h1>
        <p className="page-subtitle">
          View all active AWS resources across supported services, including free tier resources.
        </p>
      </div>

      {/* Loading */}
      {loading && (
        <div className="card">
          <div className="loading-state">
            <div className="spinner" />
            <span>Scanning resources across services...</span>
          </div>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="alert alert-error">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Content */}
      {!loading && !error && (
        <>
          {/* Metrics */}
          <div className="grid-3">
            <div className="metric-card">
              <div className="metric-label">Total Resources</div>
              <div className="metric-value">{totalResources}</div>
              <div className="metric-detail">Across all services</div>
            </div>
            <div className="metric-card">
              <div className="metric-label">Active Services</div>
              <div className="metric-value metric-value-sm">{rows.filter((r) => r.count > 0).length}</div>
              <div className="metric-detail">With at least one resource</div>
            </div>
            <div className="metric-card">
              <div className="metric-label">Services Scanned</div>
              <div className="metric-value metric-value-sm">{rows.length}</div>
              <div className="metric-detail">EC2, VPC, S3, RDS, and more</div>
            </div>
          </div>

          {/* Resources Table */}
          <div className="card">
            <div className="card-header">
              <span className="card-title">Resources by Service</span>
              <span className="badge badge-info">Click to explore</span>
            </div>
            <div className="table-container">
              <table className="table table-clickable">
                <thead>
                  <tr>
                    <th>Service</th>
                    <th>Resource Type</th>
                    <th className="text-right">Count</th>
                    <th className="text-right">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((svc) => (
                    <tr key={svc.service} onClick={() => onRowClick(svc)}>
                      <td>
                        <span style={{ marginRight: 8 }}>{SERVICE_ICONS[svc.service] || '📦'}</span>
                        <span className="font-medium">{svc.displayName}</span>
                      </td>
                      <td className="text-secondary">{humanizeResourceType(svc.resourceType)}</td>
                      <td className="text-right">
                        <span className="font-mono font-medium">{svc.count}</span>
                      </td>
                      <td className="text-right">
                        {svc.count > 0 ? (
                          <span className="badge badge-success">Active</span>
                        ) : (
                          <span className="badge">None</span>
                        )}
                      </td>
                    </tr>
                  ))}
                  {rows.length === 0 && (
                    <tr>
                      <td colSpan={4}>
                        <div className="empty-state">
                          <div className="empty-state-title">No resources found</div>
                          <div className="empty-state-description">
                            No resources were detected in the scanned services.
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Info Card */}
          <div className="alert alert-info">
            <strong>Note:</strong> This overview includes resources covered by free tier credits that may show $0.00 in Cost Explorer.
            Click on any row to view detailed resource information.
          </div>
        </>
      )}
    </div>
  );
}

function humanizeResourceType(resourceType: string): string {
  switch (resourceType) {
    case 'ec2Instances':
      return 'EC2 Instances';
    case 'vpcs':
      return 'Virtual Private Clouds';
    case 'elasticIps':
      return 'Elastic IP Addresses';
    case 's3Buckets':
      return 'S3 Buckets';
    case 'rekognitionCollections':
      return 'Face Recognition Collections';
    case 'rdsInstances':
      return 'Database Instances';
    default:
      return resourceType;
  }
}

export default ResourcesOverviewPage;
