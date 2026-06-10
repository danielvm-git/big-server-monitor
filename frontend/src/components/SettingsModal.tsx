import { useState, useEffect, type FormEvent } from 'react';
import { Plus, Trash2, X } from 'lucide-react';
import { settings } from '../../wailsjs/go/models';
import '../styles/theme.css';
import '../styles/animations.css';

interface Props {
  config: settings.Config;
  onSave: (config: settings.Config) => void;
  onReset: () => void;
  onClose: () => void;
}

export default function SettingsModal({ config, onSave, onReset, onClose }: Props) {
  const [scanDirs, setScanDirs] = useState<string[]>([]);
  const [newDir, setNewDir] = useState('');
  const [pollingInterval, setPollingInterval] = useState(0);
  const [healthInterval, setHealthInterval] = useState(0);
  const [ignoredPorts, setIgnoredPorts] = useState('');
  const [crashAlerts, setCrashAlerts] = useState(false);
  const [showBadge, setShowBadge] = useState(false);
  const [launchAtLogin, setLaunchAtLogin] = useState(false);

  useEffect(() => {
    setScanDirs(config.scanDirectories ?? []);
    setPollingInterval(config.pollingIntervalSeconds ?? 0);
    setHealthInterval(config.healthCheckIntervalSeconds ?? 0);
    setIgnoredPorts((config.ignoredPorts ?? []).join(', '));
    setCrashAlerts(config.notifications?.crashAlerts ?? false);
    setShowBadge(config.notifications?.showBadge ?? false);
    setLaunchAtLogin(config.launchAtLogin ?? false);
  }, [config]);

  function addDir() {
    const trimmed = newDir.trim();
    if (!trimmed || scanDirs.includes(trimmed)) return;
    setScanDirs(prev => [...prev, trimmed]);
    setNewDir('');
  }

  function removeDir(dir: string) {
    setScanDirs(prev => prev.filter(d => d !== dir));
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const ports = ignoredPorts
      .split(',')
      .map(p => parseInt(p.trim(), 10))
      .filter(n => !isNaN(n));

    const updated = new settings.Config({
      ...config,
      scanDirectories: scanDirs,
      pollingIntervalSeconds: pollingInterval,
      healthCheckIntervalSeconds: healthInterval,
      ignoredPorts: ports,
      notifications: { crashAlerts, showBadge },
      launchAtLogin,
    });
    onSave(updated);
  }

  const sectionStyle: React.CSSProperties = {
    marginBottom: 16,
  };

  const labelStyle: React.CSSProperties = {
    display: 'block',
    fontSize: 12,
    fontWeight: 600,
    color: 'var(--text2)',
    marginBottom: 4,
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
  };

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '6px 10px',
    fontSize: 13,
    fontFamily: 'var(--font)',
    color: 'var(--text)',
    background: 'var(--card-bg)',
    border: '1px solid var(--sep)',
    borderRadius: 'var(--radius-sm)',
    boxSizing: 'border-box',
  };

  const toggleRowStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '8px 0',
  };

  const toggleStyle: React.CSSProperties = {
    width: 40,
    height: 22,
    borderRadius: 11,
    border: 'none',
    cursor: 'pointer',
    position: 'relative',
    transition: 'background 0.15s',
    padding: 0,
    flexShrink: 0,
  };

  const toggleKnobStyle = (on: boolean): React.CSSProperties => ({
    width: 16,
    height: 16,
    borderRadius: '50%',
    background: '#fff',
    position: 'absolute',
    top: 3,
    left: on ? 21 : 3,
    transition: 'left 0.15s',
  });

  return (
    <div
      className="pk-fade-in"
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.35)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
        fontFamily: 'var(--font)',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: 'var(--pop-bg)',
          border: `1px solid var(--pop-border)`,
          borderRadius: 'var(--radius-lg)',
          width: 400,
          maxHeight: '80vh',
          overflow: 'auto',
          padding: 24,
          boxShadow: '0 8px 32px rgba(0,0,0,0.18)',
        }}
        onClick={e => e.stopPropagation()}
      >
        <form onSubmit={handleSubmit}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
            <h2 style={{ margin: 0, fontSize: 17, fontWeight: 600, color: 'var(--text)' }}>Settings</h2>
            <button
              type="button"
              onClick={onClose}
              style={{
                background: 'none',
                border: 'none',
                color: 'var(--text2)',
                cursor: 'pointer',
                padding: 4,
                borderRadius: 'var(--radius-sm)',
              }}
            >
              <X size={18} />
            </button>
          </div>

          <div style={sectionStyle}>
            <label style={labelStyle}>Scan Directories</label>
            {scanDirs.map(dir => (
              <div key={dir} style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                <span style={{
                  flex: 1,
                  fontSize: 12,
                  fontFamily: 'var(--mono)',
                  color: 'var(--text)',
                  background: 'var(--card-bg)',
                  padding: '4px 8px',
                  borderRadius: 'var(--radius-sm)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}>{dir}</span>
                <button
                  type="button"
                  onClick={() => removeDir(dir)}
                  style={{
                    background: 'none',
                    border: 'none',
                    color: 'var(--text3)',
                    cursor: 'pointer',
                    padding: 2,
                  }}
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
            <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
              <input
                type="text"
                value={newDir}
                onChange={e => setNewDir(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addDir(); } }}
                placeholder="/path/to/project"
                style={{ ...inputStyle, width: 'auto', flex: 1 }}
              />
              <button
                type="button"
                onClick={addDir}
                style={{
                  background: 'var(--accent)',
                  color: '#fff',
                  border: 'none',
                  borderRadius: 'var(--radius-sm)',
                  padding: '6px 10px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 4,
                  fontSize: 12,
                  fontWeight: 500,
                }}
              >
                <Plus size={14} /> Add
              </button>
            </div>
          </div>

          <div style={sectionStyle}>
            <label style={labelStyle}>Polling Interval (seconds)</label>
            <input
              type="number"
              min={1}
              value={pollingInterval}
              onChange={e => setPollingInterval(parseInt(e.target.value, 10) || 0)}
              style={inputStyle}
            />
          </div>

          <div style={sectionStyle}>
            <label style={labelStyle}>Health Check Interval (seconds)</label>
            <input
              type="number"
              min={1}
              value={healthInterval}
              onChange={e => setHealthInterval(parseInt(e.target.value, 10) || 0)}
              style={inputStyle}
            />
          </div>

          <div style={sectionStyle}>
            <label style={labelStyle}>Ignored Ports (comma-separated)</label>
            <input
              type="text"
              value={ignoredPorts}
              onChange={e => setIgnoredPorts(e.target.value)}
              placeholder="3000, 4000"
              style={inputStyle}
            />
          </div>

          <div style={sectionStyle}>
            <label style={{ ...labelStyle, marginBottom: 0 }}>Notifications</label>
            <div style={toggleRowStyle}>
              <span style={{ fontSize: 13, color: 'var(--text)' }}>Crash Alerts</span>
              <button
                type="button"
                style={{
                  ...toggleStyle,
                  background: crashAlerts ? 'var(--accent)' : 'var(--card-bg)',
                  border: crashAlerts ? 'none' : '1px solid var(--sep)',
                }}
                onClick={() => setCrashAlerts(v => !v)}
                role="switch"
                aria-checked={crashAlerts}
              >
                <div style={toggleKnobStyle(crashAlerts)} />
              </button>
            </div>
            <div style={toggleRowStyle}>
              <span style={{ fontSize: 13, color: 'var(--text)' }}>Show Badge</span>
              <button
                type="button"
                style={{
                  ...toggleStyle,
                  background: showBadge ? 'var(--accent)' : 'var(--card-bg)',
                  border: showBadge ? 'none' : '1px solid var(--sep)',
                }}
                onClick={() => setShowBadge(v => !v)}
                role="switch"
                aria-checked={showBadge}
              >
                <div style={toggleKnobStyle(showBadge)} />
              </button>
            </div>
          </div>

          <div style={sectionStyle}>
            <div style={toggleRowStyle}>
              <span style={{ fontSize: 13, color: 'var(--text)' }}>Launch at Login</span>
              <button
                type="button"
                style={{
                  ...toggleStyle,
                  background: launchAtLogin ? 'var(--accent)' : 'var(--card-bg)',
                  border: launchAtLogin ? 'none' : '1px solid var(--sep)',
                }}
                onClick={() => setLaunchAtLogin(v => !v)}
                role="switch"
                aria-checked={launchAtLogin}
              >
                <div style={toggleKnobStyle(launchAtLogin)} />
              </button>
            </div>
          </div>

          <div style={{ display: 'flex', gap: 8, marginTop: 20 }}>
            <button
              type="button"
              onClick={onReset}
              style={{
                flex: 1,
                padding: '8px 0',
                fontSize: 13,
                fontWeight: 500,
                color: 'var(--text2)',
                background: 'var(--card-bg)',
                border: '1px solid var(--sep)',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
              }}
            >
              Reset to Defaults
            </button>
            <button
              type="submit"
              style={{
                flex: 1,
                padding: '8px 0',
                fontSize: 13,
                fontWeight: 600,
                color: '#fff',
                background: 'var(--accent)',
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
              }}
            >
              Save
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
