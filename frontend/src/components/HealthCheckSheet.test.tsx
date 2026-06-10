import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import HealthCheckSheet from './HealthCheckSheet'

import type { healthcheck } from '../../wailsjs/go/models'

const mockResults = [
  { port: 3000, status: 'ok', statusCode: 200, latencyMs: 42.5, protocol: 'http', checkedAt: new Date() },
  { port: 8080, status: 'error', statusCode: 500, latencyMs: 120, protocol: 'http', checkedAt: new Date() },
] as healthcheck.HealthResult[]

describe('HealthCheckSheet', () => {
  it('renders health results with port numbers', () => {
    render(<HealthCheckSheet results={mockResults} onClose={vi.fn()} onRunAll={vi.fn()} />)
    expect(screen.getByText('3000')).toBeInTheDocument()
    expect(screen.getByText('8080')).toBeInTheDocument()
    expect(screen.getByText('ok')).toBeInTheDocument()
    expect(screen.getByText('error')).toBeInTheDocument()
  })

  it('renders status codes and latencies', () => {
    render(<HealthCheckSheet results={mockResults} onClose={vi.fn()} onRunAll={vi.fn()} />)
    expect(screen.getByText('200')).toBeInTheDocument()
    expect(screen.getByText('500')).toBeInTheDocument()
    expect(screen.getByText('42.5ms')).toBeInTheDocument()
    expect(screen.getByText('120ms')).toBeInTheDocument()
  })

  it('calls onRunAll when Test All Now clicked', () => {
    const onRunAll = vi.fn()
    render(<HealthCheckSheet results={[]} onClose={vi.fn()} onRunAll={onRunAll} />)
    fireEvent.click(screen.getByText('Test All Now'))
    expect(onRunAll).toHaveBeenCalledOnce()
  })

  it('shows empty state when no results', () => {
    render(<HealthCheckSheet results={[]} onClose={vi.fn()} onRunAll={vi.fn()} />)
    expect(screen.getByText('No health check results yet')).toBeInTheDocument()
  })
})
