export interface BackoffConfig {
  baseMs: number
  maxMs: number
  factor: number
  jitterMs: number
}

export interface BackoffState {
  failures: number
  nextAllowedAt: number
  config: BackoffConfig
}

const DEFAULT_CONFIG: BackoffConfig = {
  baseMs: 4000,
  maxMs: 60000,
  factor: 2,
  jitterMs: 250,
}

export function createBackoff(config?: Partial<BackoffConfig>): BackoffState {
  return {
    failures: 0,
    nextAllowedAt: 0,
    config: { ...DEFAULT_CONFIG, ...(config ?? {}) },
  }
}

export function canAttempt(backoff: BackoffState): boolean {
  return Date.now() >= backoff.nextAllowedAt
}

export function recordSuccess(backoff: BackoffState) {
  backoff.failures = 0
  backoff.nextAllowedAt = 0
}

export function recordFailure(backoff: BackoffState): number {
  backoff.failures += 1
  const exp = Math.min(backoff.failures - 1, 6)
  const baseDelay = backoff.config.baseMs * Math.pow(backoff.config.factor, exp)
  const delay = Math.min(backoff.config.maxMs, baseDelay)
  const jitter = backoff.config.jitterMs
    ? Math.floor(Math.random() * backoff.config.jitterMs)
    : 0
  backoff.nextAllowedAt = Date.now() + delay + jitter
  return delay
}

export function getRetryMs(backoff: BackoffState): number {
  return Math.max(0, backoff.nextAllowedAt - Date.now())
}
