import {
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
} from '@tanstack/react-table'
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

import { DataTableView } from '../data-table-view'

type RowData = { name: string }

const columns: ColumnDef<RowData>[] = [
  { accessorKey: 'name', header: 'Name', size: 360 },
]

function EmptyTable() {
  const table = useReactTable({
    data: [],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  return (
    <DataTableView
      table={table}
      isLoading={false}
      applyHeaderSize
      splitHeader
      emptyTitle='No rows'
      emptyDescription='Create a row to get started.'
    />
  )
}

describe('DataTableView empty layout', () => {
  test('keeps an empty table natural-width so its message remains visible', () => {
    render(<EmptyTable />)

    const table = document.querySelector('[data-slot="table"]')
    expect(table).toBeInTheDocument()
    expect(table).not.toHaveAttribute('style')
    expect(table?.querySelector('colgroup')).not.toBeInTheDocument()
    expect(screen.getByText('No rows')).toBeInTheDocument()
  })
})
