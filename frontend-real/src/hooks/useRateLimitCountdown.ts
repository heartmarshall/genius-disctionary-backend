import { useState, useEffect, useCallback } from 'react'

export function useRateLimitCountdown() {
  const [secondsLeft, setSecondsLeft] = useState(0)

  useEffect(() => {
    if (secondsLeft <= 0) return
    const timer = setInterval(() => {
      setSecondsLeft((prev) => prev - 1)
    }, 1000)
    return () => clearInterval(timer)
  }, [secondsLeft])

  const startCountdown = useCallback((seconds: number) => {
    setSecondsLeft(seconds)
  }, [])

  return {
    isRateLimited: secondsLeft > 0,
    secondsLeft,
    startCountdown,
  }
}
