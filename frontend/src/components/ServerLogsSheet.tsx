import { useState, useRef, useCallback } from 'react';
import { Copy, Check, X } from 'lucide-react';
import { logcapture } from '../../wailsjs/go/models';
import '../styles/sheets.css';

const getLogsForAI = (port: number): Promise<string> =>
  (window as any).go.main.App.GetLogsForAI(port);

interface ServerLogsSheetProps {
  port: number;
  processName: string;
  logs: logcapture.LogLine[];
  onClose: () => void;
}

type LogTab = 'all' | 'errors' | 'warnings';

function formatTime(ts: unknown) {
  if (!ts) return '';
  try {
    return new Date(ts as string).toLocaleTimeString();
  } catch {
    return '';
  }
}

function filteredLogs(logs: logcapture.LogLine[], tab: LogTab) {
  if (tab === 'all') return logs;
  if (tab === 'errors') return logs.filter((l) => l.level === 'error');
  return logs.filter((l) => l.level === 'warn');
}

const TABS: { key: LogTab; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'errors', label: 'Errors' },
  { key: 'warnings', label: 'Warnings' },
];

export default function ServerLogsSheet({ port, processName, logs, onClose }: ServerLogsSheetProps) {
  const [tab, setTab] = useState<LogTab>('all');
  const [copied, setCopied] = useState(false);
  const copyTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const displayed = filteredLogs(logs, tab);

  const handleCopyAll = useCallback(() => {
    const text = logs.map((l) => `[${formatTime(l.timestamp)}] [${l.level}] (${l.stream}) ${l.text}`).join('\n');
    navigator.clipboard.writeText(text).catch(() => {});
  }, [logs]);

  const handleCopyForAI = useCallback(async () => {
    try {
      const text = await getLogsForAI(port);
      await navigator.clipboard.writeText(text);
      setCopied(true);
      if (copyTimerRef.current) clearTimeout(copyTimerRef.current);
      copyTimerRef.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      /* noop */
    }
  }, [port]);

  return (
    <div className="sheet-backdrop" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-header">
          <h2 className="sheet-title">
            Server Logs &mdash; {processName}:{port}
          </h2>
          <button className="sheet-close" onClick={onClose} aria-label="Close">
            <X size={16} />
          </button>
        </div>

        <div className="log-tabs">
          {TABS.map((t) => (
            <button
              key={t.key}
              className={`log-tab${tab === t.key ? ' active' : ''}`}
              onClick={() => setTab(t.key)}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className="sheet-body">
          {displayed.length === 0 ? (
            <div className="sheet-empty">
              <span>No logs captured</span>
            </div>
          ) : (
            displayed.map((l) => (
              <div className="log-line" key={l.seq}>
                <span className="log-line-time">{formatTime(l.timestamp)}</span>
                <span className={`log-line-level ${l.level}`}>{l.level}</span>
                <span className="log-line-stream">{l.stream}</span>
                <span className="log-line-text">{l.text}</span>
              </div>
            ))
          )}
        </div>

        <div className="sheet-footer">
          <button className="sheet-btn sheet-btn-secondary" onClick={handleCopyAll}>
            <Copy size={15} className="sheet-btn-icon" />
            Copy
          </button>
          <button className="sheet-btn sheet-btn-secondary" onClick={handleCopyForAI}>
            <Copy size={15} className="sheet-btn-icon" />
            Copy for AI
          </button>
          {copied && (
            <span className="sheet-copied">
              <Check size={15} />
              Copied
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
