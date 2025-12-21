import { useState } from 'react'
import { appConfig } from '../config'
import { encodeFunctionData, decodeEventLog, numberToHex, type Hex } from 'viem'
import { NILSTORE_PRECOMPILE_ABI } from '../lib/nilstorePrecompile'
import { waitForTransactionReceipt } from '../lib/evmRpc'
import { buildServiceHint } from '../lib/serviceHint'
import { walletRequest } from '../lib/wallet'

export interface CreateDealInput {
  creator: string
  duration: number
  initialEscrow: string
  maxMonthlySpend: string
  replication?: number
  serviceHint?: string
}

export function useCreateDeal() {
  const [loading, setLoading] = useState(false)
  const [lastTx, setLastTx] = useState<string | null>(null)

  async function submitDeal(input: CreateDealInput) {
    setLoading(true)
    setLastTx(null)
    try {
      const evmAddress = String(input.creator || '')
      if (!evmAddress.startsWith('0x')) throw new Error('EVM address required')
      const replicas = Number.isFinite(input.replication) && (input.replication ?? 0) > 0 ? input.replication : 1
      const serviceHint = input.serviceHint && input.serviceHint.trim().length > 0
        ? input.serviceHint.trim()
        : buildServiceHint('General', { replicas })

      const data = encodeFunctionData({
        abi: NILSTORE_PRECOMPILE_ABI,
        functionName: 'createDeal',
        args: [
          BigInt(Math.max(1, Number(input.duration) || 0)),
          serviceHint,
          BigInt(String(input.initialEscrow || '0')),
          BigInt(String(input.maxMonthlySpend || '0')),
        ],
      })

      const txHash = await walletRequest<Hex>({
        method: 'eth_sendTransaction',
        params: [{ from: evmAddress, to: appConfig.nilstorePrecompile, data, gas: numberToHex(5_000_000) }],
      })
      setLastTx(txHash)

      const receipt = await waitForTransactionReceipt(txHash)
      const logs = receipt.logs || []
      for (const log of logs) {
        if (String(log.address || '').toLowerCase() !== appConfig.nilstorePrecompile.toLowerCase()) continue
        try {
          const decoded = decodeEventLog({
            abi: NILSTORE_PRECOMPILE_ABI,
            eventName: 'DealCreated',
            topics: log.topics as [signature: `0x${string}`, ...args: `0x${string}`[]],
            data: log.data,
          })
          const dealId = (decoded.args as { dealId: bigint }).dealId
          return { status: 'success', tx_hash: txHash, deal_id: String(dealId) }
        } catch {
          continue
        }
      }
      throw new Error('createDeal tx confirmed but DealCreated event not found')
    } finally {
      setLoading(false)
    }
  }

  return { submitDeal, loading, lastTx }
}
