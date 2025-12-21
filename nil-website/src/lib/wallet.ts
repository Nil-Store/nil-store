export interface EthereumProvider {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>
  on?: (event: string, handler: (...args: unknown[]) => void) => void
  removeListener?: (event: string, handler: (...args: unknown[]) => void) => void
}

export function getEthereumProvider(): EthereumProvider {
  const provider = typeof window !== 'undefined' ? (window as { ethereum?: EthereumProvider }).ethereum : undefined
  if (!provider || typeof provider.request !== 'function') {
    throw new Error('MetaMask not detected. Install a wallet to continue.')
  }
  return provider
}

export function normalizeWalletError(error: unknown): string {
  if (error && typeof error === 'object') {
    const code = (error as { code?: number }).code
    const message = String((error as { message?: string }).message || '')
    if (code === 4001 || message.toLowerCase().includes('user rejected')) {
      return 'Wallet request was rejected.'
    }
    if (code === 4902 || message.toLowerCase().includes('unrecognized chain id')) {
      return 'Wallet does not recognize this chain. Add the NilChain network and try again.'
    }
    if (message) return message
  }
  return 'Wallet request failed.'
}

export async function walletRequest<T = unknown>(args: { method: string; params?: unknown[] }): Promise<T> {
  const provider = getEthereumProvider()
  try {
    return (await provider.request(args)) as T
  } catch (e) {
    throw new Error(normalizeWalletError(e))
  }
}
