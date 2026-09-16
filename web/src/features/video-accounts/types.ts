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

export type VideoModelPricing = {
  mode?: string
  amount?: number
  currency?: string
}

export type VideoModel = {
  id: string
  display_name?: string
  product_key?: string
  group?: string
  private_group_key?: string
  available?: boolean
  unavailable_reason?: string
  supported_endpoint_types?: string[]
  resolution?: string
  durations_seconds?: number[]
  ratios?: string[]
  sizes?: string[]
  max_images?: number
  max_videos?: number
  max_audios?: number
  audio_requires_image?: boolean
  supports_first_last_frame?: boolean
  pricing?: VideoModelPricing
  billing_pricing?: VideoModelPricing
  group_ratio?: number
}

export type VideoAccount = {
  id: number
  name: string
  base_url: string
  status: number
  groups: string[]
  models: VideoModel[]
  public_key: string
  token_id: string
  api_key_masked: string
  has_api_key: boolean
  last_model_sync_at: number
  proxy?: string
  remark?: string
  created_at: number
  updated_at: number
}

export type VideoAccountListData = {
  items: VideoAccount[]
  total: number
  page: number
  page_size: number
}

export type VideoAccountResponse<T = undefined> = {
  success: boolean
  message?: string
  data?: T
}

export type VideoAccountListParams = {
  p?: number
  page_size?: number
  search?: string
  status?: number
}

export type VideoAccountPayload = {
  name: string
  api_key?: string
  status: number
  groups: string[]
  proxy?: string
  remark?: string
  base_url?: string
  billing_prices?: Record<string, VideoModelPricing | null>
}
