import { useCallback, useState } from 'react'
import { useAccount } from 'wagmi'
import { encodeFunctionData, numberToHex, type Hex } from 'viem'

import { appConfig } from '../config'
import { NILSTORE_PRECOMPILE_ABI } from '../lib/nilstorePrecompile'
import { waitForTransactionReceipt } from '../lib/evmRpc'
import { walletRequest } from '../lib/wallet'

interface DirectCommitOptions {
  dealId: string
  manifestRoot: string
  fileSize: number
  onSuccess?: (txHash: string) => void
  onError?: (error: Error) => void
}

export function useDirectCommit() {
  const { address } = useAccount()
  const [hash, setHash] = useState<Hex | null>(null)
  const [isPending, setIsPending] = useState(false)
  const [isConfirming, setIsConfirming] = useState(false)
  const [isSuccess, setIsSuccess] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const commitContent = useCallback(
    async (options: DirectCommitOptions) => {
      const { dealId, manifestRoot, fileSize, onSuccess, onError } = options
      setError(null)
      setIsSuccess(false)
      setHash(null)

      if (!address) {
        const err = new Error('Connect a wallet to commit content')
        setError(err)
        onError?.(err)
        return
      }

      try {
        setIsPending(true)
        const formattedRoot = manifestRoot.startsWith('0x') ? manifestRoot : `0x${manifestRoot}`
        const data = encodeFunctionData({
          abi: NILSTORE_PRECOMPILE_ABI,
          functionName: 'updateDealContent',
          args: [BigInt(dealId), formattedRoot as Hex, BigInt(fileSize)],
        })
        const txHash = await walletRequest<Hex>({
          method: 'eth_sendTransaction',
          params: [
            {
              from: address,
              to: appConfig.nilstorePrecompile,
              data,
              gas: numberToHex(5_000_000),
            },
          ],
        })
        setHash(txHash)
        setIsPending(false)
        setIsConfirming(true)
        await waitForTransactionReceipt(txHash)
        setIsConfirming(false)
        setIsSuccess(true)
        onSuccess?.(txHash)
      } catch (e) {
        const err = e instanceof Error ? e : new Error(String(e))
        setIsPending(false)
        setIsConfirming(false)
        setIsSuccess(false)
        setError(err)
        onError?.(err)
      }
    },
    [address],
  )

  return {
    commitContent,
    isPending,
    isConfirming,
    isSuccess,
    hash,
    error,
  }
}

