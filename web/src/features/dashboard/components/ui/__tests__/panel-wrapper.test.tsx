/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen } from '@testing-library/react'
import { describe, expect, test } from 'vitest'

import { PanelWrapper } from '../panel-wrapper'

describe('PanelWrapper empty state', () => {
  test('uses a compact empty state while preserving the panel header', () => {
    render(
      <PanelWrapper
        title='API information'
        description='Configured routes'
        empty
        emptyMessage='No routes configured'
      />
    )

    const panel = document.querySelector('[data-panel-state="empty"]')
    expect(panel).toBeInTheDocument()
    expect(panel?.querySelector('.h-32')).toBeInTheDocument()
    expect(screen.getByText('API information')).toBeInTheDocument()
    expect(screen.getByText('No routes configured')).toBeInTheDocument()
  })
})
