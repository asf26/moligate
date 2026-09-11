/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { Hero } from '../sections/hero'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: {
    children?: ReactNode
    search?: { standalone?: boolean }
    to?: string
  }) => (
    <a
      href={`${props.to}${props.search?.standalone ? '?standalone=true' : ''}`}
    >
      {props.children}
    </a>
  ),
}))

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: { docs_link: 'https://docs.example.com' } }),
}))

describe('Hero', () => {
  it('sends unauthenticated visitors to sign in from the primary action', () => {
    render(<Hero />)

    expect(screen.getByRole('link', { name: /Connect now/ })).toHaveAttribute(
      'href',
      '/sign-in'
    )
  })

  it('sends authenticated visitors to the dashboard from the primary action', () => {
    render(<Hero isAuthenticated />)

    expect(screen.getByRole('link', { name: /Connect now/ })).toHaveAttribute(
      'href',
      '/dashboard'
    )
  })
})
