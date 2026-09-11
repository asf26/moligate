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
import { api, type ApiRequestConfig } from '@/lib/api'

import type {
  VideoAccount,
  VideoAccountListData,
  VideoAccountListParams,
  VideoAccountPayload,
  VideoAccountResponse,
} from './types'

const videoAccountActionConfig = (
  config: ApiRequestConfig = {}
): ApiRequestConfig => ({
  ...config,
  skipBusinessError: true,
  skipErrorHandler: true,
})

export async function getVideoAccounts(
  params: VideoAccountListParams = {}
): Promise<VideoAccountResponse<VideoAccountListData>> {
  const response = await api.get('/api/video-accounts', { params })
  return response.data
}

export async function createVideoAccount(
  payload: VideoAccountPayload
): Promise<VideoAccountResponse<VideoAccount>> {
  const response = await api.post(
    '/api/video-accounts',
    payload,
    videoAccountActionConfig()
  )
  return response.data
}

export async function updateVideoAccount(
  id: number,
  payload: Partial<VideoAccountPayload>
): Promise<VideoAccountResponse<VideoAccount>> {
  const response = await api.put(
    `/api/video-accounts/${id}`,
    payload,
    videoAccountActionConfig()
  )
  return response.data
}

export async function syncVideoAccount(
  id: number
): Promise<VideoAccountResponse<VideoAccount>> {
  const response = await api.post(
    `/api/video-accounts/${id}/sync`,
    null,
    videoAccountActionConfig()
  )
  return response.data
}

export async function deleteVideoAccount(
  id: number
): Promise<VideoAccountResponse> {
  const response = await api.delete(
    `/api/video-accounts/${id}`,
    videoAccountActionConfig()
  )
  return response.data
}
