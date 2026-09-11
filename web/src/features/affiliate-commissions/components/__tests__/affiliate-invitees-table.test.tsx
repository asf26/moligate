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

import { AffiliateInviteesTable } from '../affiliate-invitees-table'

describe('AffiliateInviteesTable', () => {
  test('shows exact invitee usernames and their point contribution', () => {
    render(
      <AffiliateInviteesTable
        total={1}
        items={[
          {
            user_id: 42,
            username: 'alice@example.com',
            display_name: 'Alice',
            created_at: 1710000000,
            top_up_count: 2,
            base_quota: 1000000,
            reward_points: 120,
            pending_points: 20,
            settled_points: 100,
            last_contribution_at: 1711000000,
          },
        ]}
      />
    )

    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    expect(screen.getByText('120')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.queryByText('a***@example.com')).not.toBeInTheDocument()
  })

  test('renders a clear empty state', () => {
    render(<AffiliateInviteesTable items={[]} />)

    expect(screen.getByText('No invited users yet')).toBeInTheDocument()
  })
})
