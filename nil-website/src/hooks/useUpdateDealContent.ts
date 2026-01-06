import { useState } from 'react'
import { appConfig } from '../config'
import { encodeFunctionData, numberToHex, type Hex } from 'viem'
import { NILSTORE_PRECOMPILE_ABI } from '../lib/nilstorePrecompile'
import { waitForTransactionReceipt } from '../lib/evmRpc'

export interface UpdateDealContentInput {
  creator: string
  dealId: number
  cid: string
  sizeBytes: number
  totalMdus?: number
  witnessMdus?: number
}

export function useUpdateDealContent() {
  const [loading, setLoading] = useState(false)
  const [lastTx, setLastTx] = useState<string | null>(null)

  async function submitUpdate(input: UpdateDealContentInput) {
    setLoading(true)
    setLastTx(null)
    try {
      const evmAddress = String(input.creator || '')
      if (!evmAddress.startsWith('0x')) throw new Error('EVM address required')
      const ethereum = window.ethereum
      if (!ethereum || typeof ethereum.request !== 'function') {
        throw new Error('Ethereum provider (MetaMask) not available')
      }
      const manifestRoot = String(input.cid || '').trim() as Hex

      const hasLayout = Number.isFinite(input.totalMdus) && Number.isFinite(input.witnessMdus)
      const totalMdus = hasLayout ? Math.max(0, Number(input.totalMdus)) : 0
      const witnessMdus = hasLayout ? Math.max(0, Number(input.witnessMdus)) : 0

      if (hasLayout && totalMdus <= 1 + witnessMdus) {
        throw new Error('Commit requires totalMdus > 1 + witnessMdus')
      }

      const dataV1 = encodeFunctionData({
        abi: NILSTORE_PRECOMPILE_ABI,
        functionName: 'updateDealContent',
        args: [BigInt(input.dealId), manifestRoot, BigInt(input.sizeBytes)],
      })

      const dataV2 = hasLayout
        ? encodeFunctionData({
            abi: NILSTORE_PRECOMPILE_ABI,
            functionName: 'updateDealContent',
            args: [BigInt(input.dealId), manifestRoot, BigInt(input.sizeBytes), BigInt(totalMdus), BigInt(witnessMdus)],
          })
        : null

      let data = dataV1
      if (dataV2) {
        try {
          await ethereum.request({
            method: 'eth_call',
            params: [{ from: evmAddress, to: appConfig.nilstorePrecompile, data: dataV2 }, 'latest'],
          })
          data = dataV2
        } catch {
          data = dataV1
        }
      }

      const txHash = (await ethereum.request({
        method: 'eth_sendTransaction',
        params: [{ from: evmAddress, to: appConfig.nilstorePrecompile, data, gas: numberToHex(3_000_000) }],
      })) as Hex
      setLastTx(txHash)
      await waitForTransactionReceipt(txHash)
      return { status: 'success', tx_hash: txHash }
    } finally {
      setLoading(false)
    }
  }

  return { submitUpdate, loading, lastTx }
}
