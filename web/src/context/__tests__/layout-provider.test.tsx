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
import { beforeEach, describe, expect, test } from 'vitest'

import { LayoutProvider, useLayout } from '../layout-provider'

function LayoutVariantProbe() {
  const { variant } = useLayout()
  return <output data-testid='layout-variant'>{variant}</output>
}

describe('LayoutProvider defaults', () => {
  beforeEach(() => {
    document.cookie = 'layout_variant=; path=/; max-age=0'
    document.cookie = 'layout_collapsible=; path=/; max-age=0'
  })

  test('uses an edge-to-edge sidebar workspace without a saved preference', () => {
    render(
      <LayoutProvider>
        <LayoutVariantProbe />
      </LayoutProvider>
    )

    expect(screen.getByTestId('layout-variant')).toHaveTextContent('sidebar')
  })
})
