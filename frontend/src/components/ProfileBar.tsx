import { useEffect, useState } from 'react';
import {
  ProfileStatus,
  fetchProfileStatus,
  createProfile,
  selectProfile,
} from '../api/client';

function ProfileBar() {
  const [status, setStatus] = useState<ProfileStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    name: '',
    accessKeyId: '',
    secretAccessKey: '',
    sessionToken: '',
    region: '',
  });

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        setLoading(true);
        const s = await fetchProfileStatus();
        if (!cancelled) setStatus(s);
      } catch (e: any) {
        if (!cancelled) setError(e.message || 'Failed to load profiles');
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const handleSelect = async (id: string) => {
    try {
      setLoading(true);
      setError(null);
      const s = await selectProfile(id);
      setStatus(s);
      window.location.reload();
    } catch (e: any) {
      setError(e.message || 'Failed to switch profile');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      setError(null);
      const s = await createProfile(form);
      setStatus(s);
      setShowForm(false);
      setForm({
        name: '',
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        region: '',
      });
      window.location.reload();
    } catch (err: any) {
      setError(err.message || 'Failed to add profile');
    } finally {
      setLoading(false);
    }
  };

  const profiles = status?.profiles ?? [];
  const hasAnyCreds = !!status?.systemAvailable || profiles.length > 0;

  const activeLabel = status?.activeId
    ? profiles.find((p) => p.id === status.activeId)?.name || status.activeId
    : status?.systemAvailable
    ? 'System default'
    : 'No credentials';

  return (
    <div className="profile-dropdown">
      {error && (
        <span className="text-error" style={{ fontSize: 12 }}>
          {error}
        </span>
      )}

      {status && (
        <select
          disabled={loading || !hasAnyCreds}
          value={status.activeId || (status.systemAvailable ? 'system' : '')}
          onChange={(e) => handleSelect(e.target.value)}
          className="form-select form-input-sm"
          style={{ minWidth: 160 }}
        >
          {status.systemAvailable && <option value="system">System default</option>}
          {profiles.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name} ({p.source})
            </option>
          ))}
          {!hasAnyCreds && <option value="">No credentials</option>}
        </select>
      )}

      <button
        type="button"
        onClick={() => setShowForm((v) => !v)}
        className="btn btn-secondary btn-sm"
      >
        {hasAnyCreds ? 'Add Profile' : 'Configure Credentials'}
      </button>

      {showForm && (
        <div className="modal-overlay" onClick={() => setShowForm(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <span className="modal-title">Add AWS Profile</span>
              <button
                type="button"
                onClick={() => setShowForm(false)}
                className="btn btn-ghost btn-icon btn-sm"
                aria-label="Close"
              >
                ✕
              </button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="modal-body flex flex-col gap-md">
                <div className="form-group">
                  <label className="form-label">Profile name</label>
                  <input
                    type="text"
                    required
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    className="form-input"
                    placeholder="e.g. production"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Access Key ID</label>
                  <input
                    type="text"
                    required
                    value={form.accessKeyId}
                    onChange={(e) => setForm({ ...form, accessKeyId: e.target.value })}
                    className="form-input font-mono"
                    placeholder="AKIAIOSFODNN7EXAMPLE"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Secret Access Key</label>
                  <input
                    type="password"
                    required
                    value={form.secretAccessKey}
                    onChange={(e) => setForm({ ...form, secretAccessKey: e.target.value })}
                    className="form-input font-mono"
                    placeholder="••••••••••••••••••••"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Session Token (optional)</label>
                  <input
                    type="password"
                    value={form.sessionToken}
                    onChange={(e) => setForm({ ...form, sessionToken: e.target.value })}
                    className="form-input font-mono"
                    placeholder="For temporary credentials"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Default Region (optional)</label>
                  <input
                    type="text"
                    value={form.region}
                    onChange={(e) => setForm({ ...form, region: e.target.value })}
                    className="form-input"
                    placeholder="e.g. us-east-1"
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowForm(false)} className="btn btn-ghost">
                  Cancel
                </button>
                <button type="submit" disabled={loading} className="btn btn-primary">
                  {loading ? 'Saving...' : 'Save & Use'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default ProfileBar;
