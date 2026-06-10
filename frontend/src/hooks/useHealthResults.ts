import { useState, useEffect, useCallback } from 'react';
import { healthcheck } from '../../wailsjs/go/models';

const getHealthResults = (): Promise<healthcheck.HealthResult[]> =>
  (window as any).go.main.App.GetHealthResults();

const runHealthCheck = (ports: number[]): Promise<healthcheck.HealthResult[]> =>
  (window as any).go.main.App.RunHealthCheck(ports);

export function useHealthResults(ports: number[]) {
  const [results, setResults] = useState<healthcheck.HealthResult[]>([]);
  const [loading, setLoading] = useState(false);

  const fetch = useCallback(async () => {
    try {
      const r = await getHealthResults();
      setResults(r);
    } catch {
      /* noop */
    }
  }, []);

  useEffect(() => {
    fetch();
    const id = setInterval(fetch, 30_000);
    return () => clearInterval(id);
  }, [fetch]);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const r = await runHealthCheck(ports);
      setResults(r);
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, [ports]);

  return { results, loading, refresh };
}
