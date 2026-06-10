import { renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { useSettings } from './useSettings'

vi.mock('../../wailsjs/go/main/App', () => ({
  GetSettings: vi.fn().mockResolvedValue({
    pollingIntervalSeconds: 10,
  }),
  SaveSettings: vi.fn().mockResolvedValue(undefined),
  ResetSettings: vi.fn().mockResolvedValue(undefined),
  AddScanDirectory: vi.fn().mockResolvedValue(undefined),
  RemoveScanDirectory: vi.fn().mockResolvedValue(undefined),
}))

describe('useSettings', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('returns config after fetch', async () => {
    const { result } = renderHook(() => useSettings())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    }, { timeout: 2000 })

    expect(result.current.config.pollingIntervalSeconds).toBe(10)
  })

  it('provides saveSettings function', () => {
    const { result } = renderHook(() => useSettings())
    expect(typeof result.current.saveSettings).toBe('function')
  })
})
