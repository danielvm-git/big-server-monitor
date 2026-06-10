import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { useServers } from './useServers'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetServers: vi.fn().mockResolvedValue([{ port: 3000, processName: 'node' }]),
}))

describe('useServers', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('returns servers after fetch', async () => {
    const { result } = renderHook(() => useServers())

    await waitFor(() => {
      expect(result.current.servers.length).toBeGreaterThan(0)
    }, { timeout: 2000 })

    expect(result.current.servers[0].port).toBe(3000)
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  it('exposes a refresh function', () => {
    const { result } = renderHook(() => useServers())
    expect(typeof result.current.refresh).toBe('function')
  })
})
