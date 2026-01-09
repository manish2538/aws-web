import { useEffect, useState } from 'react';
import {
  fetchCommands,
  executeCommand,
  executeRawCommand,
  PublicCommand,
  CommandExecutionResult,
} from '../api/client';

function CommandRunnerPage() {
  const [commands, setCommands] = useState<PublicCommand[]>([]);
  const [selectedId, setSelectedId] = useState<string>('');
  const [region, setRegion] = useState<string>('');
  const [result, setResult] = useState<CommandExecutionResult | null>(null);
  const [rawArgs, setRawArgs] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [loadingRaw, setLoadingRaw] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const list = await fetchCommands();
        if (cancelled) return;
        setCommands(list);
        if (list.length > 0) {
          setSelectedId(list[0].id);
        }
      } catch (e: any) {
        if (cancelled) return;
        setError(e.message || 'Failed to load commands');
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const selected = commands.find((c) => c.id === selectedId);

  const handleExecute = async () => {
    if (!selectedId) return;
    try {
      setLoading(true);
      setError(null);
      setResult(null);
      const res = await executeCommand(
        selectedId,
        selected?.supportsRegion ? region || undefined : undefined,
      );
      setResult(res);
    } catch (e: any) {
      setError(e.message || 'Failed to execute command');
    } finally {
      setLoading(false);
    }
  };

  const handleExecuteRaw = async () => {
    if (!rawArgs.trim()) return;
    try {
      setLoadingRaw(true);
      setError(null);
      setResult(null);
      const res = await executeRawCommand(rawArgs);
      setResult(res);
    } catch (e: any) {
      setError(e.message || 'Failed to execute raw command');
    } finally {
      setLoadingRaw(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleExecuteRaw();
    }
  };

  return (
    <div className="page">
      {/* Page Header */}
      <div className="page-header">
        <div className="page-breadcrumb">
          <span>Tools</span>
          <span>›</span>
          <span>CLI Runner</span>
        </div>
        <h1 className="page-title">AWS CLI Runner</h1>
        <p className="page-subtitle">
          Execute read-only AWS CLI commands directly from the dashboard. Only describe, list, and get operations are allowed.
        </p>
      </div>

      {/* Error */}
      {error && (
        <div className="alert alert-error">
          <strong>Error:</strong> {error}
        </div>
      )}

      <div className="grid-2">
        {/* Predefined Commands */}
        <div className="card">
          <div className="card-header">
            <span className="card-title">Predefined Commands</span>
            <span className="badge badge-info">{commands.length} available</span>
          </div>
          <div className="card-body flex flex-col gap-md">
            <div className="form-group">
              <label className="form-label">Select Command</label>
              <select
                value={selectedId}
                onChange={(e) => setSelectedId(e.target.value)}
                className="form-select"
              >
                {commands.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.label}
                  </option>
                ))}
              </select>
            </div>

            {selected?.supportsRegion && (
              <div className="form-group">
                <label className="form-label">Region (optional)</label>
                <input
                  type="text"
                  placeholder="e.g. ap-south-1"
                  value={region}
                  onChange={(e) => setRegion(e.target.value)}
                  className="form-input"
                />
              </div>
            )}

            {selected && (
              <div className="text-secondary" style={{ fontSize: 13 }}>
                <strong>Description:</strong> {selected.description}
                <br />
                <strong>Service:</strong> <code>{selected.service}</code>
              </div>
            )}

            <button
              type="button"
              onClick={handleExecute}
              disabled={loading || !selectedId}
              className="btn btn-primary"
            >
              {loading ? (
                <>
                  <span className="spinner" style={{ width: 14, height: 14 }} />
                  Running...
                </>
              ) : (
                'Execute Command'
              )}
            </button>
          </div>
        </div>

        {/* Raw Command */}
        <div className="card">
          <div className="card-header">
            <span className="card-title">Raw Command</span>
            <span className="badge badge-warning">Advanced</span>
          </div>
          <div className="card-body flex flex-col gap-md">
            <div className="form-group">
              <label className="form-label">AWS CLI Arguments</label>
              <input
                type="text"
                value={rawArgs}
                onChange={(e) => setRawArgs(e.target.value)}
                onKeyDown={handleKeyDown}
                className="form-input font-mono"
                placeholder="ec2 describe-instances --region ap-south-1"
              />
            </div>

            <div className="text-secondary" style={{ fontSize: 13 }}>
              Enter AWS CLI arguments without the leading <code>aws</code>.
              <br />
              Only read-only operations (describe, list, get) are allowed.
            </div>

            <button
              type="button"
              onClick={handleExecuteRaw}
              disabled={loadingRaw || !rawArgs.trim()}
              className="btn btn-secondary"
            >
              {loadingRaw ? (
                <>
                  <span className="spinner" style={{ width: 14, height: 14 }} />
                  Running...
                </>
              ) : (
                'Execute Raw Command'
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Result */}
      {result && (
        <div className="card">
          <div className="card-header">
            <span className="card-title">Command Output</span>
            <span className="badge badge-success">Success</span>
          </div>
          <div className="card-body flex flex-col gap-md">
            <div className="form-group">
              <label className="form-label">Executed Command</label>
              <pre style={{ margin: 0, padding: '12px 16px' }}>
                <code>{result.command}</code>
              </pre>
            </div>

            <div className="form-group">
              <label className="form-label">Output</label>
              <pre style={{ maxHeight: 400, overflow: 'auto', margin: 0 }}>
                {JSON.stringify(result.output, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}

      {/* Help Card */}
      <div className="alert alert-info">
        <strong>Security Note:</strong> This CLI runner is configured to only allow read-only AWS operations.
        Commands that modify, create, or delete resources are blocked for safety.
      </div>
    </div>
  );
}

export default CommandRunnerPage;
