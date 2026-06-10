import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import ServerRow from './ServerRow'

const mockServer = {
  port: 3000,
  status: 'online',
  pid: 12345,
  processName: 'node',
  runtimeVersion: 'v20.11.0',
  binaryPath: '/usr/local/bin/node',
  projectName: 'bigbase-api',
  projectDir: '/Users/dev/bigbase-api',
  memoryMb: 128,
  uptimeStr: '2h 15m',
  startedAt: new Date(),
  envSnapshot: [],
  localDomain: '',
  tunnelURL: '',
}

describe('ServerRow', () => {
  it('renders port, process name, and uptime', () => {
    render(<ServerRow server={mockServer} onKill={vi.fn()} />)
    expect(screen.getByText(':3000')).toBeInTheDocument()
    expect(screen.getByText('node')).toBeInTheDocument()
    expect(screen.getByText('2h 15m')).toBeInTheDocument()
  })

  it('expands on click showing PID and memory', () => {
    render(<ServerRow server={mockServer} onKill={vi.fn()} />)
    fireEvent.click(screen.getByText(':3000'))
    expect(screen.getByText('PID')).toBeInTheDocument()
    expect(screen.getByText('12345')).toBeInTheDocument()
    expect(screen.getByText('Memory')).toBeInTheDocument()
    expect(screen.getByText('128.0 MB')).toBeInTheDocument()
  })

  it('shows KillConfirmDialog when kill button clicked', () => {
    const onKill = vi.fn()
    render(<ServerRow server={mockServer} onKill={onKill} />)
    fireEvent.click(screen.getByLabelText('Kill node'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('calls onKill after confirming in dialog', () => {
    const onKill = vi.fn()
    render(<ServerRow server={mockServer} onKill={onKill} />)
    fireEvent.click(screen.getByLabelText('Kill node'))
    fireEvent.click(screen.getByRole('button', { name: 'Kill' }))
    expect(onKill).toHaveBeenCalledWith(12345)
  })
})
