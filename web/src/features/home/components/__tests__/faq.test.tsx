import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { FAQ } from '../sections/faq'

describe('FAQ', () => {
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
})
