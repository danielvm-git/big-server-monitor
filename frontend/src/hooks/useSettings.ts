import { useState, useEffect, useCallback } from 'react';
import { GetSettings, SaveSettings, ResetSettings, AddScanDirectory, RemoveScanDirectory } from '../../wailsjs/go/main/App';
import { settings } from '../../wailsjs/go/models';

export function useSettings() {
  const [config, setConfig] = useState<settings.Config>(new settings.Config());
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    GetSettings()
      .then(setConfig)
      .finally(() => setLoading(false));
  }, []);

  const saveSettings = useCallback(async (c: settings.Config) => {
    await SaveSettings(c);
    setConfig(c);
  }, []);

  const resetSettings = useCallback(async () => {
    await ResetSettings();
    const fresh = await GetSettings();
    setConfig(fresh);
  }, []);

  const addScanDirectory = useCallback(async (dir: string) => {
    await AddScanDirectory(dir);
    const fresh = await GetSettings();
    setConfig(fresh);
  }, []);

  const removeScanDirectory = useCallback(async (dir: string) => {
    await RemoveScanDirectory(dir);
    const fresh = await GetSettings();
    setConfig(fresh);
  }, []);

  return { config, loading, saveSettings, resetSettings, addScanDirectory, removeScanDirectory };
}
