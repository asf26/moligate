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
import { render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { CreationShowcase } from '../sections/creation-showcase'

describe('CreationShowcase', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'IntersectionObserver',
      class IntersectionObserverMock {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    )
  })

  it('opens the creation gallery and CTA in a standalone new tab', () => {
    render(<CreationShowcase />)

    expect(
      screen.getByRole('heading', {
        name: 'Creative Canvas',
        level: 2,
      })
    ).toBeInTheDocument()
    const createButton = screen.getByRole('button', {
      name: 'Start creating now',
    })
    expect(createButton).toHaveAttribute('href', '/canvas?standalone=true')
    expect(createButton).toHaveAttribute('target', '_blank')
    expect(createButton).toHaveAttribute('rel', 'noopener noreferrer')

    const gallery = screen.getByTestId('creation-showcase-grid')
    expect(
      gallery.compareDocumentPosition(createButton) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy()
    const galleryLinks = within(gallery).getAllByRole('link')
    expect(galleryLinks).toHaveLength(8)
    expect(
      galleryLinks.every((link) => {
        return (
          link.getAttribute('href') === '/canvas?standalone=true' &&
          link.getAttribute('target') === '_blank' &&
          link.getAttribute('rel') === 'noopener noreferrer'
        )
      })
    ).toBe(true)
    expect(
      within(gallery).getByRole('img', { name: '苹果风格海报' })
    ).toBeInTheDocument()
    expect(
      within(gallery).getByText(/充分参考图片的设计风格/)
    ).toBeInTheDocument()
  })
})
