import { useCallback } from 'react'
import { usePublicClient, useWaitForTransactionReceipt, useWriteContract } from 'wagmi'
import { encodeFunctionData, type Hex } from 'viem'
import { NILSTORE_PRECOMPILE_ABI } from '../lib/nilstorePrecompile'
import { appConfig } from '../config'

interface DirectCommitOptions {
  dealId: string; // The deal ID (string representation of uint64)
  manifestRoot: string; // The canonical 0x-prefixed hex string
  fileSize: number; // Size in bytes
  totalMdus?: number
  witnessMdus?: number
  onSuccess?: (txHash: string) => void;
  onError?: (error: Error) => void;
}

export function useDirectCommit() {
  const publicClient = usePublicClient()
  const { data: hash, writeContractAsync, isPending, error: writeError } = useWriteContract()
  
  const { isLoading: isConfirming, isSuccess, error: receiptError } = useWaitForTransactionReceipt({
    hash,
  });

  const commitContent = useCallback(async (options: DirectCommitOptions) => {
    const { dealId, manifestRoot, fileSize } = options;
    
    // Ensure manifestRoot is bytes (0x prefixed)
    const formattedRoot = manifestRoot.startsWith('0x') ? manifestRoot : `0x${manifestRoot}`;

    const hasLayout = Number.isFinite(options.totalMdus) && Number.isFinite(options.witnessMdus)
    const totalMdus = hasLayout ? Math.max(0, Number(options.totalMdus)) : 0
    const witnessMdus = hasLayout ? Math.max(0, Number(options.witnessMdus)) : 0

    if (hasLayout && totalMdus <= 1 + witnessMdus) {
      throw new Error('Commit requires totalMdus > 1 + witnessMdus')
    }

    const dataV2 = hasLayout
      ? encodeFunctionData({
          abi: NILSTORE_PRECOMPILE_ABI,
          functionName: 'updateDealContent',
          args: [BigInt(dealId), formattedRoot as Hex, BigInt(fileSize), BigInt(totalMdus), BigInt(witnessMdus)],
        })
      : null

    let useV2 = false
    if (publicClient && dataV2) {
      try {
        await publicClient.call({
          account: undefined,
          to: appConfig.nilstorePrecompile as Hex,
          data: dataV2,
        })
        useV2 = true
      } catch {
        useV2 = false
      }
    }

    await writeContractAsync({
      address: appConfig.nilstorePrecompile as Hex,
      abi: NILSTORE_PRECOMPILE_ABI,
      functionName: 'updateDealContent',
      args: useV2 && dataV2
        ? [BigInt(dealId), formattedRoot as Hex, BigInt(fileSize), BigInt(totalMdus), BigInt(witnessMdus)]
        : [BigInt(dealId), formattedRoot as Hex, BigInt(fileSize)],
    })
  }, [publicClient, writeContractAsync]);

  return {
    commitContent,
    isPending,      // Waiting for wallet signature
    isConfirming,   // Waiting for block inclusion
    isSuccess,      // Transaction confirmed
    hash,
    error: writeError || receiptError,
  };
}
