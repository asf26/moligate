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
import { useCallback, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

interface CounterProps {
  end: number
  suffix?: string
}

function Counter(props: CounterProps) {
  const ref = useRef<HTMLSpanElement>(null)
  const hasStarted = useRef(false)

  const renderValue = useCallback(
    (value: number) => {
      if (ref.current) {
        ref.current.textContent = `${Math.round(value).toLocaleString()}${props.suffix ?? ''}`
      }
    },
    [props.suffix]
  )

  useEffect(() => {
    const element = ref.current
    if (!element) return

    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
    if (reducedMotion.matches) {
      renderValue(props.end)
      return
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry.isIntersecting || hasStarted.current) return
        hasStarted.current = true
        const start = performance.now()

        const animate = (now: number) => {
          const progress = Math.min((now - start) / 1000, 1)
          const eased = 1 - Math.pow(1 - progress, 3)
          renderValue(eased * props.end)
          if (progress < 1) requestAnimationFrame(animate)
        }

        requestAnimationFrame(animate)
        observer.unobserve(element)
      },
      { threshold: 0.5 }
    )

    observer.observe(element)
    return () => observer.disconnect()
  }, [props.end, renderValue])

  return <span ref={ref}>0{props.suffix}</span>
}

interface StatsProps {
  className?: string
}

export function Stats(_props: StatsProps) {
  const { t } = useTranslation()
  const stats = [
    { end: 50, suffix: '+', label: t('upstream services integrated') },
    { end: 100, suffix: '+', label: t('model billing support') },
    { end: 50, suffix: '+', label: t('compatible API routes') },
    { end: 10, suffix: '+', label: t('scheduling controls') },
  ]

  return (
    <section
      className='home-stats border-b'
      aria-label={t('API usage records')}
    >
      <div className='mx-auto grid max-w-7xl grid-cols-2 md:grid-cols-4'>
        {stats.map((stat) => (
          <div key={stat.label} className='home-stat-cell'>
            <span className='home-stat-value font-mono'>
              <Counter end={stat.end} suffix={stat.suffix} />
            </span>
            <span className='text-muted-foreground mt-2 text-xs'>
              {stat.label}
            </span>
          </div>
        ))}
      </div>
    </section>
  )
}
