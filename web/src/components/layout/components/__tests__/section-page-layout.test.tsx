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

import { SectionPageLayout } from '../section-page-layout'

describe('SectionPageLayout workspace frame', () => {
  test('renders the page title and content in separate workspace regions', () => {
    render(
      <SectionPageLayout>
        <SectionPageLayout.Title>Workspace</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          A page description
        </SectionPageLayout.Description>
        <SectionPageLayout.Actions>
          <button type='button'>Create</button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <p>Content</p>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )

    expect(
      screen.getByRole('heading', { level: 2, name: 'Workspace' })
    ).toHaveClass('workspace-page-title')
    expect(screen.getByText('A page description')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create' })).toBeInTheDocument()
    expect(screen.getByText('Content')).toBeInTheDocument()
    expect(document.querySelector('.workspace-page-header')).toBeInTheDocument()
    expect(
      document.querySelector('.workspace-page-content')
    ).toBeInTheDocument()
  })

  test('keeps fixed pages from becoming an additional scrolling surface', () => {
    render(
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>Fixed page</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <p>Table content</p>
        </SectionPageLayout.Content>
      </SectionPageLayout>
    )

    expect(document.querySelector('.workspace-page-content')).toHaveClass(
      'overflow-hidden'
    )
  })
})
