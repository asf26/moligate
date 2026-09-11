import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { FAQ } from '../sections/faq'

describe('FAQ', () => {
  it('starts with the first answer expanded', () => {
    render(<FAQ />)

    const question = screen.getByRole('button', {
      name: 'How can I use Claude in China?',
    })
    expect(question).toHaveAttribute('aria-expanded', 'true')
    expect(
      screen.getByText(
        'Create an account, generate an API key, and point Claude Code or the Anthropic SDK at the gateway base URL. The request is routed through an available domestic-compatible provider.'
      )
    ).toBeVisible()
  })

  it('allows multiple answers to stay expanded', async () => {
    const user = userEvent.setup()
    render(<FAQ />)

    const firstQuestion = screen.getByRole('button', {
      name: 'How can I use Claude in China?',
    })
    const secondQuestion = screen.getByRole('button', {
      name: 'What is a Claude proxy or relay?',
    })

    await user.click(secondQuestion)

    expect(firstQuestion).toHaveAttribute('aria-expanded', 'true')
    expect(secondQuestion).toHaveAttribute('aria-expanded', 'true')
    expect(
      screen.getByText(
        'Create an account, generate an API key, and point Claude Code or the Anthropic SDK at the gateway base URL. The request is routed through an available domestic-compatible provider.'
      )
    ).toBeVisible()
    expect(
      screen.getByText(
        'A relay exposes a compatible API endpoint between your application and the upstream model provider. Your integration keeps its existing SDK while the gateway handles routing and billing.'
      )
    ).toBeVisible()
  })
})
