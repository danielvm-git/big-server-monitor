import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import PopoverHeader from './PopoverHeader'

describe('PopoverHeader', () => {
  it('renders title and server count', () => {
    render(<PopoverHeader serverCount={3} onRefresh={vi.fn()} />)
    expect(screen.getByText('PortKeeper')).toBeInTheDocument()
    expect(screen.getByText(/3/)).toBeInTheDocument()
    expect(screen.getByText(/servers active/)).toBeInTheDocument()
  })

  it('uses singular "server" when count is 1', () => {
    render(<PopoverHeader serverCount={1} onRefresh={vi.fn()} />)
    expect(screen.getByText(/1 server active/)).toBeInTheDocument()
  })

  it('calls onRefresh when refresh button clicked', () => {
    const onRefresh = vi.fn()
    render(<PopoverHeader serverCount={0} onRefresh={onRefresh} />)
    fireEvent.click(screen.getByLabelText('Refresh servers'))
    expect(onRefresh).toHaveBeenCalledOnce()
  })
})
