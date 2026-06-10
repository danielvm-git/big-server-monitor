import { useState, useEffect, useCallback } from 'react';
import { activitylog } from '../../wailsjs/go/models';

const getActivityLog = (f: activitylog.ActivityFilter): Promise<activitylog.ActivityEvent[]> =>
  (window as any).go.main.App.GetActivityLog(f);

const clearHistoryCall = (): Promise<void> =>
  (window as any).go.main.App.ClearHistory();

const DEFAULT_FILTER = new activitylog.ActivityFilter({ EventTypes: [], Limit: 100 });

export function useActivityLog(initialFilter?: activitylog.ActivityFilter) {
  const [events, setEvents] = useState<activitylog.ActivityEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [filters, setFiltersState] = useState<activitylog.ActivityFilter>(initialFilter ?? DEFAULT_FILTER);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getActivityLog(filters)
      .then((data: activitylog.ActivityEvent[]) => {
        if (!cancelled) {
          setEvents(data);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [filters]);

  const setFilters = useCallback((f: activitylog.ActivityFilter) => {
    setFiltersState(f);
  }, []);

  const clearHistory = useCallback(async () => {
    try {
      await clearHistoryCall();
      setEvents([]);
    } catch {
      /* noop */
    }
  }, []);

  return { events, loading, filters, setFilters, clearHistory };
}
