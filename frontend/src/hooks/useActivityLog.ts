import { useState, useEffect, useCallback } from 'react';
import { activitylog } from '../../wailsjs/go/models';
import { GetActivityLog, ClearHistory } from '../../wailsjs/go/main/App';

const DEFAULT_FILTER = new activitylog.ActivityFilter({ EventTypes: [], Limit: 100 });

export function useActivityLog(initialFilter?: activitylog.ActivityFilter) {
  const [events, setEvents] = useState<activitylog.ActivityEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [filters, setFiltersState] = useState<activitylog.ActivityFilter>(initialFilter ?? DEFAULT_FILTER);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    GetActivityLog(filters)
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
      await ClearHistory();
      setEvents([]);
    } catch {
      /* noop */
    }
  }, []);

  return { events, loading, filters, setFilters, clearHistory };
}
