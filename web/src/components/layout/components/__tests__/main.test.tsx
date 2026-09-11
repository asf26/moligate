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
import { describe, expect, test } from 'vitest'

import { Main } from '../main'

describe('Main accessibility target', () => {
  test('provides the shared skip-link target by default', () => {
    render(
      <>
        <a href='#content'>Skip to Main</a>
        <Main>
          <p>Workspace content</p>
        </Main>
      </>
    )

    expect(screen.getByRole('main')).toHaveAttribute('id', 'content')
    expect(screen.getByRole('link', { name: 'Skip to Main' })).toHaveAttribute(
      'href',
      '#content'
    )
  })

  test('preserves an explicit main id for embedded surfaces', () => {
    render(<Main id='canvas-content'>Canvas</Main>)

    expect(screen.getByRole('main')).toHaveAttribute('id', 'canvas-content')
  })
})
