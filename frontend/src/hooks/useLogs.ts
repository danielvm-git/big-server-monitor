import { useState, useEffect } from 'react';
import { logcapture } from '../../wailsjs/go/models';

const getLogs = (f: logcapture.LogFilter): Promise<logcapture.LogLine[]> =>
  (window as any).go.main.App.GetLogs(f);

export function useLogs(port: number) {
  const [logs, setLogs] = useState<logcapture.LogLine[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const filter = new logcapture.LogFilter({ Port: port, Levels: [], Limit: 500 });
    getLogs(filter)
      .then((data: logcapture.LogLine[]) => {
        if (!cancelled) {
          setLogs(data);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [port]);

  return { logs, loading };
}
