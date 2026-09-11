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

import { ModelDirectory } from '../sections/model-directory'

describe('ModelDirectory', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'IntersectionObserver',
      class IntersectionObserverMock {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    )
  })

  it('shows each supported model family and its key routes', () => {
    render(<ModelDirectory />)

    expect(
      screen.getByRole('heading', { name: 'Models directory' })
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Claude' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'GPT' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Gemini' })).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Chinese models' })
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: 'Codex' })
    ).not.toBeInTheDocument()
    expect(screen.getByText('claude-opus-5')).toBeInTheDocument()
    expect(screen.getByText('gpt-5.6-sol')).toBeInTheDocument()
    expect(screen.getByText('gemini-2.5-pro')).toBeInTheDocument()
    expect(screen.getByText('deepseek-v4-pro')).toBeInTheDocument()
    expect(screen.getByText('qwen3.8-max')).toBeInTheDocument()
    expect(screen.getByText('glm-5.3')).toBeInTheDocument()
    expect(screen.getByText('kimi-k3')).toBeInTheDocument()
    expect(screen.getByText('doubao-seed-2.0-pro')).toBeInTheDocument()
    expect(screen.getByText('minimax-m3')).toBeInTheDocument()
    expect(screen.queryByText('qwen-max')).not.toBeInTheDocument()
    expect(screen.queryByText('GLM-4.5')).not.toBeInTheDocument()
    expect(screen.queryByText('MiniMax-M2.5')).not.toBeInTheDocument()
    expect(screen.queryByText('SparkDesk-v4.0')).not.toBeInTheDocument()
    expect(
      screen.queryByText(/ernie|hy4|hunyuan|stepfun/i)
    ).not.toBeInTheDocument()
    expect(screen.queryByText(/mimo|xiaomi/i)).not.toBeInTheDocument()
  })
})
