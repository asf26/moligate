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
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'

import { ModelDirectory } from '../sections/model-directory'

vi.mock('@/features/pricing/hooks/use-pricing-data', () => ({
  usePricingData: vi.fn(),
}))

const mockedUsePricingData = vi.mocked(usePricingData)

describe('ModelDirectory', () => {
  beforeEach(() => {
    mockedUsePricingData.mockReturnValue({
      models: [
        { model_name: 'gpt-live', vendor_name: 'OpenAI' },
        { model_name: 'claude-live', vendor_name: 'Anthropic' },
        { model_name: 'gemini-live', vendor_name: 'Google' },
        { model_name: 'grok-live', vendor_name: 'xAI' },
        { model_name: 'qwen-live', vendor_name: '阿里巴巴' },
      ],
    } as unknown as ReturnType<typeof usePricingData>)
    vi.stubGlobal(
      'IntersectionObserver',
      class IntersectionObserverMock {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    )
  })

  it('shows the model families and models returned by the model catalog', () => {
    render(<ModelDirectory />)

    expect(
      screen.getByRole('heading', { name: 'Models directory' })
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Claude' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'GPT' })).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Chinese models' })
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Gemini' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Grok' })).toBeInTheDocument()
    expect(screen.getByText('gpt-live')).toBeInTheDocument()
    expect(screen.getByText('claude-live')).toBeInTheDocument()
    expect(screen.getByText('gemini-live')).toBeInTheDocument()
    expect(screen.queryByText('claude-opus-5')).not.toBeInTheDocument()
  })

  it('falls back to the curated directory when the model catalog is empty', () => {
    mockedUsePricingData.mockReturnValue({
      models: [],
    } as unknown as ReturnType<typeof usePricingData>)

    render(<ModelDirectory />)

    expect(screen.getByRole('heading', { name: 'Claude' })).toBeInTheDocument()
    expect(screen.getByText('claude-opus-5')).toBeInTheDocument()
  })
})
