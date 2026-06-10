import { useState, useEffect, useCallback } from "react";
import { GetServers } from "../../wailsjs/go/main/App";
import type { processmonitor } from "../../wailsjs/go/models";

interface UseServersResult {
  servers: processmonitor.Server[];
  loading: boolean;
  error: string | null;
}

export function useServers(): UseServersResult & { refresh: () => void } {
  const [servers, setServers] = useState<processmonitor.Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchServers = useCallback(() => {
    GetServers()
      .then((data) => {
        setServers(data ?? []);
        setError(null);
      })
      .catch((err) => {
        setError(String(err));
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    fetchServers();
    const interval = setInterval(fetchServers, 5000);
    return () => clearInterval(interval);
  }, [fetchServers]);

  return { servers, loading, error, refresh: fetchServers };
}
