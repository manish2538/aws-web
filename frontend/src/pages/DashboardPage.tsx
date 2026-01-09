import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  CostResponse,
  ServicesResponse,
  fetchCost,
  fetchServices,
  ServiceCost,
  clearBackendCache,
} from '../api/client';
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  Cell,
} from 'recharts';
import { useCurrency } from '../context/CurrencyContext';

const CHART_COLORS = ['#388bfd', '#58a6ff', '#79c0ff', '#a5d6ff', '#c6e6ff'];

function DashboardPage() {
  const [costData, setCostData] = useState<CostResponse | null>(null);
  const [servicesData, setServicesData] = useState<ServicesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [minCost, setMinCost] = useState<string>('');
  const [maxCost, setMaxCost] = useState<string>('');
  const [startDate, setStartDate] = useState<string>(() => {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1).toISOString().split('T')[0];
  });
  const [endDate, setEndDate] = useState<string>(() => new Date().toISOString().split('T')[0]);

  const navigate = useNavigate();
  const { formatCost, currency } = useCurrency();

  const load = async (signal?: { cancelled: boolean }) => {
    try {
      setLoading(true);
      setError(null);

      const params =
        startDate || endDate
          ? { start: startDate || undefined, end: endDate || undefined }
          : undefined;

      const [cost, services] = await Promise.all([fetchCost(params), fetchServices(params)]);
      if (signal?.cancelled) return;
      setCostData(cost);
      setServicesData(services);
    } catch (e: any) {
      if (signal?.cancelled) return;
      setError(e.message || 'Failed to load data');
    } finally {
      if (!signal?.cancelled) setLoading(false);
    }
  };

  useEffect(() => {
    let cancelled = false;
    load({ cancelled });
    return () => {
      cancelled = true;
    };
  }, []);

  const handleApplyFilters = async () => {
    await load();
  };

  const handleHardRefresh = async () => {
    try {
      await clearBackendCache();
    } catch {
      // ignore
    }
    await load();
  };

  const overview = costData?.overview;
  const allServices = servicesData?.services ?? [];

  const min = minCost ? parseFloat(minCost) : undefined;
  const max = maxCost ? parseFloat(maxCost) : undefined;

  const services = allServices.filter((s) => {
    const v = s.cost;
    if (!isNaN(min as number) && min !== undefined && v < min) return false;
    if (!isNaN(max as number) && max !== undefined && v > max) return false;
    return true;
  });

  // Create chart data with converted values for display
  const chartData = services
    .filter((s) => s.cost > 0)
    .sort((a, b) => b.cost - a.cost)
    .slice(0, 10)
    .map((s) => ({
      name: s.displayName || s.service,
      cost: s.cost * currency.rate, // Convert for chart display
      costUSD: s.cost,
      drilldownKey: s.drilldownKey,
    }));

  const onRowClick = (service: ServiceCost) => {
    if (!service.drilldownKey) return;
    navigate(`/services/${service.drilldownKey}`);
  };

  return (
    <div className="page">
      {/* Page Header */}
      <div className="page-header">
        <div className="page-breadcrumb">
          <span>Billing &amp; Cost Management</span>
          <span>›</span>
          <span>Cost Explorer</span>
        </div>
        <h1 className="page-title">Cost Explorer</h1>
        <p className="page-subtitle">
          Analyze your AWS spending and identify cost optimization opportunities.
          {currency.code !== 'USD' && (
            <span className="text-muted" style={{ marginLeft: 8 }}>
              (Showing in {currency.name} @ 1 USD = {currency.rate} {currency.code})
            </span>
          )}
        </p>
      </div>

      {/* Toolbar */}
      <div className="toolbar">
        <div className="toolbar-section">
          <div className="toolbar-group">
            <span className="toolbar-label">Date Range</span>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="form-input form-input-sm"
            />
            <span className="text-muted">to</span>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              className="form-input form-input-sm"
            />
          </div>
          <div className="toolbar-divider" />
          <div className="toolbar-group">
            <span className="toolbar-label">Cost Filter (USD)</span>
            <input
              type="number"
              placeholder="Min"
              value={minCost}
              onChange={(e) => setMinCost(e.target.value)}
              className="form-input form-input-sm"
              style={{ width: 80 }}
            />
            <span className="text-muted">–</span>
            <input
              type="number"
              placeholder="Max"
              value={maxCost}
              onChange={(e) => setMaxCost(e.target.value)}
              className="form-input form-input-sm"
              style={{ width: 80 }}
            />
          </div>
        </div>
        <div className="toolbar-section">
          <button type="button" onClick={handleApplyFilters} className="btn btn-primary btn-sm">
            Apply Filters
          </button>
          <button type="button" onClick={handleHardRefresh} className="btn btn-ghost btn-sm">
            Refresh (Clear Cache)
          </button>
        </div>
      </div>

      {/* Loading State */}
      {loading && (
        <div className="card">
          <div className="loading-state">
            <div className="spinner" />
            <span>Loading cost data...</span>
          </div>
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="alert alert-error">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Main Content */}
      {!loading && !error && overview && (
        <>
          {/* Metrics */}
          <div className="grid-4">
            <div className="metric-card">
              <div className="metric-label">Total Spend</div>
              <div className="metric-value">{formatCost(overview.total)}</div>
              <div className="metric-detail">
                {overview.start} — {overview.end}
              </div>
            </div>
            <div className="metric-card">
              <div className="metric-label">Credits Applied</div>
              <div className="metric-value metric-value-sm text-success">
                {formatCost(overview.creditsApplied || 0)}
              </div>
              <div className="metric-detail">Free tier &amp; promotional credits</div>
            </div>
            <div className="metric-card">
              <div className="metric-label">Net Cost</div>
              <div className="metric-value metric-value-sm">
                {formatCost(overview.netTotal || overview.total)}
              </div>
              <div className="metric-detail">After credits applied</div>
            </div>
            <div className="metric-card">
              <div className="metric-label">Active Services</div>
              <div className="metric-value metric-value-sm">{services.length}</div>
              <div className="metric-detail">
                {services.filter((s) => s.cost > 0).length} with charges
              </div>
            </div>
          </div>

          {/* Charts and Table */}
          <div className="grid-2">
            {/* Chart */}
            <div className="card">
              <div className="card-header">
                <span className="card-title">Top Services by Cost</span>
                <span className="badge badge-info">{chartData.length} services</span>
              </div>
              <div className="card-body">
                {chartData.length === 0 ? (
                  <div className="empty-state">
                    <div className="empty-state-title">No cost data</div>
                    <div className="empty-state-description">
                      No services have incurred charges in the selected period.
                    </div>
                  </div>
                ) : (
                  <div className="chart-container">
                    <ResponsiveContainer>
                      <BarChart data={chartData} layout="vertical" margin={{ left: 20, right: 20 }}>
                        <CartesianGrid strokeDasharray="3 3" horizontal={true} vertical={false} />
                        <XAxis
                          type="number"
                          tickFormatter={(v) => `${currency.symbol}${v.toFixed(currency.rate > 100 ? 0 : 2)}`}
                        />
                        <YAxis
                          type="category"
                          dataKey="name"
                          width={120}
                          tick={{ fontSize: 11 }}
                        />
                        <Tooltip
                          formatter={(value: any, name: any, props: any) => {
                            if (name === 'cost') {
                              return [formatCost(props.payload.costUSD), 'Cost'];
                            }
                            return [value, name];
                          }}
                          contentStyle={{
                            background: '#1c2128',
                            border: '1px solid #30363d',
                            borderRadius: 6,
                          }}
                        />
                        <Bar dataKey="cost" radius={[0, 4, 4, 0]}>
                          {chartData.map((_, index) => (
                            <Cell
                              key={`cell-${index}`}
                              fill={CHART_COLORS[index % CHART_COLORS.length]}
                            />
                          ))}
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </div>
            </div>

            {/* Services Table */}
            <div className="card">
              <div className="card-header">
                <span className="card-title">All Services</span>
                <span className="badge">{services.length} total</span>
              </div>
              <div className="table-container">
                <table className="table table-clickable">
                  <thead>
                    <tr>
                      <th>Service</th>
                      <th className="text-right">Cost ({currency.code})</th>
                    </tr>
                  </thead>
                  <tbody>
                    {services.map((svc) => (
                      <tr
                        key={svc.service + svc.displayName}
                        onClick={() => onRowClick(svc)}
                        style={{ cursor: svc.drilldownKey ? 'pointer' : 'default' }}
                      >
                        <td>
                          <span className="font-medium">
                            {svc.displayName || svc.service}
                          </span>
                          {svc.drilldownKey && (
                            <span className="badge badge-purple" style={{ marginLeft: 8 }}>
                              View Details
                            </span>
                          )}
                        </td>
                        <td className="text-right text-mono">
                          {formatCost(svc.cost)}
                        </td>
                      </tr>
                    ))}
                    {services.length === 0 && (
                      <tr>
                        <td colSpan={2} className="text-center text-muted" style={{ padding: 32 }}>
                          No services match the current filters.
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default DashboardPage;
