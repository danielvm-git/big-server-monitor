import { X } from 'lucide-react';
import { healthcheck } from '../../wailsjs/go/models';
import '../styles/sheets.css';

interface HealthCheckSheetProps {
  results: healthcheck.HealthResult[];
  onClose: () => void;
  onRunAll: () => void;
}

function statusClass(s: string) {
  switch (s) {
    case 'ok':
      return 'ok';
    case 'warn':
      return 'warn';
    case 'error':
      return 'error';
    case 'timeout':
      return 'timeout';
    default:
      return '';
  }
}

export default function HealthCheckSheet({ results, onClose, onRunAll }: HealthCheckSheetProps) {
  return (
    <div className="sheet-backdrop" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-header">
          <h2 className="sheet-title">Health Check</h2>
          <button className="sheet-close" onClick={onClose} aria-label="Close">
            <X size={16} />
          </button>
        </div>

        <div className="sheet-body">
          {results.length === 0 ? (
            <div className="sheet-empty">
              <span>No health check results yet</span>
              <span>Tap "Test All Now" to run checks</span>
            </div>
          ) : (
            results.map((r, i) => (
              <div className="health-row" key={`${r.port}-${i}`}>
                <span className="health-port">{r.port}</span>
                <span className={`status-badge ${statusClass(r.status)}`}>{r.status}</span>
                <span className="protocol-pill">{r.protocol}</span>
                <span className="health-code">{r.statusCode > 0 ? r.statusCode : '\u2014'}</span>
                <span className="health-latency">{r.latencyMs}ms</span>
              </div>
            ))
          )}
        </div>

        <div className="sheet-footer">
          <button className="sheet-btn sheet-btn-primary" onClick={onRunAll}>
            Test All Now
          </button>
        </div>
      </div>
    </div>
  );
}
