import { useState, useCallback, useMemo } from 'react';
import Popover from './components/Popover';
import PopoverHeader from './components/PopoverHeader';
import StatusBanner from './components/StatusBanner';
import ServerList from './components/ServerList';
import HealthCheckSheet from './components/HealthCheckSheet';
import ActivityLogSheet from './components/ActivityLogSheet';
import ServerLogsSheet from './components/ServerLogsSheet';
import SettingsModal from './components/SettingsModal';
import KillConfirmDialog from './components/KillConfirmDialog';
import ToastProvider from './components/ToastProvider';
import { ThemeProvider } from './context/ThemeContext';
import { useServers } from './hooks/useServers';
import { useHealthResults } from './hooks/useHealthResults';
import { useActivityLog } from './hooks/useActivityLog';
import { useSettings } from './hooks/useSettings';
import { useLogs } from './hooks/useLogs';
import type { processmonitor } from '../wailsjs/go/models';
import './styles/theme.css';
import './styles/animations.css';
import './styles/popover.css';
import './styles/sheets.css';

type Sheet = 'health' | 'activity' | 'settings' | null;

// Build a lookup map from servers once (avoids O(n) search on each kill)
function buildServerMap(servers: processmonitor.Server[]): Map<number, processmonitor.Server> {
  const m = new Map<number, processmonitor.Server>();
  for (const s of servers) {
    m.set(s.pid, s);
  }
  return m;
}

function App() {
  const [activeSheet, setActiveSheet] = useState<Sheet>(null);
  const [serverToKill, setServerToKill] = useState<processmonitor.Server | null>(null);
  const [selectedServer, setSelectedServer] = useState<processmonitor.Server | null>(null);

  const { servers, loading: serversLoading, refresh: refreshServers } = useServers();

  const serverPorts = useMemo(() => servers.map(s => s.port), [servers]);
  const serverMap = useMemo(() => buildServerMap(servers), [servers]);

  const { results: healthResults, refresh: refreshHealth } = useHealthResults(serverPorts);
  const { events, filters, setFilters, clearHistory } = useActivityLog();
  const { config, saveSettings, resetSettings } = useSettings();

  const { logs } = useLogs(selectedServer?.port ?? 0);

  const handleKillServer = useCallback((pid: number) => {
    const server = serverMap.get(pid);
    if (server) {
      setServerToKill(server);
    }
  }, [serverMap]);

  const handleConfirmKill = useCallback(async () => {
    if (!serverToKill) return;
    try {
      await (window as any).go?.main?.App?.KillProcess(serverToKill.pid);
      refreshServers();
    } catch (e) {
      console.error('KillProcess failed:', e);
    }
    setServerToKill(null);
  }, [serverToKill, refreshServers]);

  const handleCancelKill = useCallback(() => {
    setServerToKill(null);
  }, []);

  const handleOpenLogs = useCallback((server: processmonitor.Server) => {
    setSelectedServer(server);
  }, []);

  const handleCloseLogs = useCallback(() => {
    setSelectedServer(null);
  }, []);

  return (
    <ThemeProvider>
      <ToastProvider>
        <Popover>
          <PopoverHeader serverCount={servers.length} onRefresh={refreshServers} />
          <StatusBanner />

          {serversLoading && servers.length === 0 ? (
            <div className="pk-loading">Discovering servers...</div>
          ) : (
            <ServerList servers={servers} onKill={handleKillServer} />
          )}

          {/* Footer */}
          <div className="pk-footer">
            <button className="pk-footer-btn" onClick={() => setActiveSheet('health')}>
              Health Check
            </button>
            <button className="pk-footer-btn" onClick={() => setActiveSheet('activity')}>
              Activity Log
            </button>
            <button className="pk-footer-btn" onClick={() => setActiveSheet('settings')}>
              Settings
            </button>
          </div>
        </Popover>

        {/* Sheets */}
        {activeSheet === 'health' && (
          <HealthCheckSheet
            results={healthResults}
            onClose={() => setActiveSheet(null)}
            onRunAll={refreshHealth}
          />
        )}
        {activeSheet === 'activity' && (
          <ActivityLogSheet
            events={events}
            filters={filters}
            onFilterChange={setFilters}
            onClose={() => setActiveSheet(null)}
            onClearHistory={clearHistory}
          />
        )}
        {selectedServer && (
          <ServerLogsSheet
            port={selectedServer.port}
            processName={selectedServer.processName}
            logs={logs}
            onClose={handleCloseLogs}
          />
        )}
        {activeSheet === 'settings' && (
          <SettingsModal
            config={config}
            onSave={saveSettings}
            onReset={resetSettings}
            onClose={() => setActiveSheet(null)}
          />
        )}

        {/* Kill confirm dialog */}
        {serverToKill && (
          <KillConfirmDialog
            server={serverToKill}
            onCancel={handleCancelKill}
            onConfirm={handleConfirmKill}
          />
        )}
      </ToastProvider>
    </ThemeProvider>
  );
}

export default App;
