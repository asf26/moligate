import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useFAQ } from '@/features/dashboard/hooks/use-status-data'

import { FAQ } from '../sections/faq'

vi.mock('@/features/dashboard/hooks/use-status-data', () => ({
  useFAQ: vi.fn(),
}))

const mockedUseFAQ = vi.mocked(useFAQ)

describe('FAQ', () => {
  beforeEach(() => {
    mockedUseFAQ.mockReturnValue({ items: [], loading: false })
  })

  it('starts with the first answer expanded', () => {
    render(<FAQ />)

    const question = screen.getByRole('button', {
      name: 'What is Moligate?',
    })
    expect(question).toHaveAttribute('aria-expanded', 'true')
    expect(
      screen.getByText(
        'Moligate is a unified AI gateway for developers and teams. One account connects multiple model providers with shared API access, usage tracking, and billing.'
      )
    ).toBeVisible()
  })

  it('allows multiple answers to stay expanded', async () => {
    const user = userEvent.setup()
    render(<FAQ />)

    const firstQuestion = screen.getByRole('button', {
      name: 'What is Moligate?',
    })
    const secondQuestion = screen.getByRole('button', {
      name: 'How do I start using Moligate?',
    })

    await user.click(secondQuestion)

    expect(firstQuestion).toHaveAttribute('aria-expanded', 'true')
    expect(secondQuestion).toHaveAttribute('aria-expanded', 'true')
    expect(
      screen.getByText(
        'Moligate is a unified AI gateway for developers and teams. One account connects multiple model providers with shared API access, usage tracking, and billing.'
      )
    ).toBeVisible()
    expect(
      screen.getByText(
        'Register or sign in, create an API key in the console, choose a supported model, and follow the integration guide. You can also open AI Creation to start without writing code.'
      )
    ).toBeVisible()
  })

  it('renders the FAQ entries configured by the backend', () => {
    mockedUseFAQ.mockReturnValue({
      items: [
        {
          id: 1,
          question: '如何开始使用魔力门？',
          answer: '注册并登录后，在控制台创建 API Key 即可开始调用。',
        },
      ],
      loading: false,
    })

    render(<FAQ />)

    expect(
      screen.getByRole('button', { name: '如何开始使用魔力门？' })
    ).toBeVisible()
    expect(
      screen.getByText('注册并登录后，在控制台创建 API Key 即可开始调用。')
    ).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'What is Moligate?' })
    ).not.toBeInTheDocument()
  })
})
