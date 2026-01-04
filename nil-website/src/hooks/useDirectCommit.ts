import { useCallback } from 'react';
import { useWriteContract, useWaitForTransactionReceipt } from 'wagmi';
import { NILSTORE_PRECOMPILE_ABI } from '../lib/nilstorePrecompile';
import { appConfig } from '../config';
import { Hex } from 'viem';

interface DirectCommitOptions {
  dealId: string; // The deal ID (string representation of uint64)
  manifestRoot: string; // The canonical 0x-prefixed hex string
  sizeBytes: number; // Total bytes stored in NilFS
  totalMdus: number; // Total MDUs in slab (including MDU0 + witness + user)
  witnessMdus: number; // Witness MDUs in slab (MDU indices 1..witnessMdus)
  onSuccess?: (txHash: string) => void;
  onError?: (error: Error) => void;
}

export function useDirectCommit() {
  const { data: hash, writeContract, isPending, error: writeError } = useWriteContract();
  
  const { isLoading: isConfirming, isSuccess, error: receiptError } = useWaitForTransactionReceipt({
    hash,
  });

  const commitContent = useCallback((options: DirectCommitOptions) => {
    const { dealId, manifestRoot, sizeBytes, totalMdus, witnessMdus } = options;
    
    // Ensure manifestRoot is bytes (0x prefixed)
    const formattedRoot = manifestRoot.startsWith('0x') ? manifestRoot : `0x${manifestRoot}`;

    writeContract({
      address: appConfig.nilstorePrecompile as Hex,
      abi: NILSTORE_PRECOMPILE_ABI,
      functionName: 'updateDealContent',
      args: [BigInt(dealId), formattedRoot as Hex, BigInt(sizeBytes), BigInt(totalMdus), BigInt(witnessMdus)],
    });
  }, [writeContract]);

  return {
    commitContent,
    isPending,      // Waiting for wallet signature
    isConfirming,   // Waiting for block inclusion
    isSuccess,      // Transaction confirmed
    hash,
    error: writeError || receiptError,
  };
}
