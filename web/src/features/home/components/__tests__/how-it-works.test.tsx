import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { HowItWorks } from '../sections/how-it-works'

describe('HowItWorks', () => {
  it('switches the active integration example when a tab is selected', async () => {
    const user = userEvent.setup()
    render(<HowItWorks />)

    const claudeTab = screen.getByRole('tab', { name: 'Claude Code' })
    const openAiTab = screen.getByRole('tab', { name: 'OpenAI SDK' })
    expect(claudeTab).toHaveAttribute('aria-selected', 'true')

    await user.click(openAiTab)

    expect(openAiTab).toHaveAttribute('aria-selected', 'true')
    expect(claudeTab).toHaveAttribute('aria-selected', 'false')
    expect(screen.getByRole('tabpanel')).toHaveTextContent('client = OpenAI(')
  })
})
