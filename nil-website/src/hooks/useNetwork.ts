import { useSwitchChain } from 'wagmi'
import { appConfig } from '../config'
import { normalizeWalletError, walletRequest } from '../lib/wallet'

export function useNetwork() {
  const { switchChainAsync } = useSwitchChain()

  const switchNetwork = async () => {
    try {
      await switchChainAsync({ chainId: appConfig.chainId })
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) {
      // Error code 4902 means the chain has not been added to MetaMask.
      // -32603 is an Internal Error that sometimes wraps 4902 in some wallet versions.
      if (e.code === 4902 || e.message?.includes('Unrecognized chain ID') || e.code === -32603) {
         try {
             await walletRequest({
               method: 'wallet_addEthereumChain',
               params: [{
                   chainId: `0x${appConfig.chainId.toString(16)}`,
                   chainName: 'NilChain Local',
                   nativeCurrency: {
                       name: 'NIL',
                       symbol: 'NIL',
                       decimals: 18,
                   },
                   rpcUrls: [appConfig.evmRpc],
                   blockExplorerUrls: ['http://localhost:5173'],
               }],
             })
             // Try switching again after adding
             await switchChainAsync({ chainId: appConfig.chainId })
         } catch (addError) {
             throw new Error(normalizeWalletError(addError))
         }
      } else {
          throw new Error(normalizeWalletError(e))
      }
    }
  }

  return { switchNetwork }
}
