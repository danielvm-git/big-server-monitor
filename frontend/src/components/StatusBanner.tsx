import { useEffect, useState } from 'react'
import { GetMonitorStatus } from '../../wailsjs/go/main/App'
import type { processmonitor } from '../../wailsjs/go/models'

export default function StatusBanner() {
  const [status, setStatus] = useState<processmonitor.MonitorStatus | null>(null)

  useEffect(() => {
    const check = () => {
      GetMonitorStatus().then(setStatus).catch(() => {})
    }
    check()
    const id = setInterval(check, 10000)
    return () => clearInterval(id)
  }, [])

  if (!status || status.healthy) return null

  return (
    <div className="pk-status-banner" role="alert">
      Port discovery issue: {status.lastError || 'Unknown error'}
      {status.serverCount > 0 && ` (showing ${status.serverCount} servers from last successful scan)`}
    </div>
  )
}
