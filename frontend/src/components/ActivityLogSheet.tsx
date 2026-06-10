import { Play, AlertTriangle, AlertCircle, X } from 'lucide-react';
import type { ReactNode } from 'react';
import { activitylog } from '../../wailsjs/go/models';
import '../styles/sheets.css';

interface ActivityLogSheetProps {
  events: activitylog.ActivityEvent[];
  filters: activitylog.ActivityFilter;
  onFilterChange: (f: activitylog.ActivityFilter) => void;
  onClose: () => void;
  onClearHistory: () => void;
}

const KNOWN_TYPES = ['start', 'crash', 'unresponsive'] as const;

const typeLabel: Record<string, string> = {
  start: 'Started',
  crash: 'Crashed',
  unresponsive: 'Unresponsive',
};

function eventIcon(type: string): ReactNode {
  switch (type) {
    case 'start':
      return <Play size={18} />;
    case 'crash':
      return <AlertCircle size={18} />;
    case 'unresponsive':
      return <AlertTriangle size={18} />;
    default:
      return <AlertCircle size={18} />;
  }
}

function iconClass(type: string) {
  return `activity-icon ${KNOWN_TYPES.includes(type as never) ? type : 'default'}`;
}

function formatTime(ts: unknown) {
  if (!ts) return '';
  try {
    return new Date(ts as string).toLocaleTimeString();
  } catch {
    return '';
  }
}

export default function ActivityLogSheet({
  events,
  filters,
  onFilterChange,
  onClose,
  onClearHistory,
}: ActivityLogSheetProps) {
  const activeTypes = filters.EventTypes ?? [];

  function toggleType(t: string) {
    const next = activeTypes.includes(t)
      ? activeTypes.filter((x) => x !== t)
      : [...activeTypes, t];
    const updated = new activitylog.ActivityFilter({
      ...filters,
      EventTypes: next,
    });
    onFilterChange(updated);
  }

  return (
    <div className="sheet-backdrop" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-header">
          <h2 className="sheet-title">Activity Log</h2>
          <button className="sheet-close" onClick={onClose} aria-label="Close">
            <X size={16} />
          </button>
        </div>

        <div className="sheet-body">
          <div className="filter-chips">
            {KNOWN_TYPES.map((t) => (
              <button
                key={t}
                className={`filter-chip${activeTypes.includes(t) ? ' active' : ''}`}
                onClick={() => toggleType(t)}
              >
                {eventIcon(t)}
                {typeLabel[t] ?? t}
              </button>
            ))}
          </div>

          {events.length === 0 ? (
            <div className="sheet-empty">
              <span>No activity yet</span>
            </div>
          ) : (
            events.map((e) => (
              <div className="activity-row" key={e.id}>
                <div className={iconClass(e.type)}>{eventIcon(e.type)}</div>
                <div className="activity-details">
                  <div className="activity-header">
                    <span className="activity-port">:{e.port}</span>
                    <span className="activity-process">{e.processName}</span>
                    {e.projectName && (
                      <span className="activity-project" title={e.projectName}>
                        {e.projectName}
                      </span>
                    )}
                  </div>
                  <div className="activity-meta">
                    <span>{formatTime(e.timestamp)}</span>
                    {e.duration && <span>{e.duration}</span>}
                  </div>
                  {e.message && <div className="activity-message">{e.message}</div>}
                </div>
              </div>
            ))
          )}
        </div>

        <div className="sheet-footer">
          <button className="sheet-btn sheet-btn-danger" onClick={onClearHistory}>
            Clear History
          </button>
        </div>
      </div>
    </div>
  );
}
