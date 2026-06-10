import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import KillConfirmDialog from './KillConfirmDialog'

const mockServer = { port: 3000, processName: 'node', pid: 12345 }

describe('KillConfirmDialog', () => {
  it('renders port and process name in confirmation message', () => {
    render(<KillConfirmDialog server={mockServer} onCancel={vi.fn()} onConfirm={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Kill' })).toBeInTheDocument()
    expect(screen.getByText('node')).toBeInTheDocument()
    expect(screen.getByText(/:3000/)).toBeInTheDocument()
  })

  it('calls onConfirm when Kill button clicked', () => {
    const onConfirm = vi.fn()
    render(<KillConfirmDialog server={mockServer} onCancel={vi.fn()} onConfirm={onConfirm} />)
    fireEvent.click(screen.getByRole('button', { name: 'Kill' }))
    expect(onConfirm).toHaveBeenCalledOnce()
  })

  it('calls onCancel when Cancel button clicked', () => {
    const onCancel = vi.fn()
    render(<KillConfirmDialog server={mockServer} onCancel={onCancel} onConfirm={vi.fn()} />)
    fireEvent.click(screen.getByText('Cancel'))
    expect(onCancel).toHaveBeenCalledOnce()
  })
})
