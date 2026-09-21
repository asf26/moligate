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
export interface OrderRecord {
  id: number
  kind: string
  user_id: number
  username: string
  plan_id: number
  plan_title: string
  trade_no: string
  money: number
  amount: number
  payment_method: string
  payment_provider: string
  status: string
  create_time: number
  complete_time: number
}

export interface OrdersQuery {
  p: number
  page_size: number
  trade_no?: string
  username?: string
  user_id?: number
  status?: string
  kind?: string
  start_time?: number
  end_time?: number
}

export interface OrdersPageData {
  page: number
  page_size: number
  total: number
  items: OrderRecord[]
}

export interface OrdersResponse {
  success: boolean
  message?: string
  data?: OrdersPageData
}
