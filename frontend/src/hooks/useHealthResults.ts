import { useState, useEffect, useCallback } from 'react';
import { healthcheck } from '../../wailsjs/go/models';
import { GetHealthResults, RunHealthCheck } from '../../wailsjs/go/main/App';

export function useHealthResults(ports: number[]) {
  const [results, setResults] = useState<healthcheck.HealthResult[]>([]);
  const [loading, setLoading] = useState(false);

  const fetch = useCallback(async () => {
    try {
      const r = await GetHealthResults();
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
      const r = await RunHealthCheck(ports);
      setResults(r);
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, [ports]);

  return { results, loading, refresh };
}
