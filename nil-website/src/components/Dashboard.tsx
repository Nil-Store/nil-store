import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useAccount, useBalance } from 'wagmi'
import { ArrowDownRight, ArrowUpRight, CheckCircle2, HardDrive, Upload, X } from 'lucide-react'
import { formatUnits } from 'viem'

import { appConfig } from '../config'
import type { LcdDeal as Deal } from '../domain/lcd'
import { lcdFetchDeals } from '../api/lcdClient'
import { ethToNil } from '../lib/address'
import { buildServiceHint, parseServiceHint } from '../lib/serviceHint'

import { StatusBar } from './StatusBar'
import { DealDetail } from './DealDetail'
import { FileSharder } from './FileSharder'

import { useFaucet } from '../hooks/useFaucet'
import { useCreateDeal } from '../hooks/useCreateDeal'
import { useUpdateDealContent } from '../hooks/useUpdateDealContent'
import { useUpload } from '../hooks/useUpload'

interface Provider {
  address: string
  endpoints?: string[]
}

type StagedUpload = {
  cid: string
  sizeBytes: number
  fileSizeBytes: number
  allocatedLength?: number
  filename: string
}

type CommitQueueState =
  | { status: 'idle' }
  | { status: 'queued'; manifestRoot: string; sizeBytes: number }
  | { status: 'pending'; manifestRoot: string; sizeBytes: number }
  | { status: 'confirmed'; manifestRoot: string; sizeBytes: number }
  | { status: 'error'; manifestRoot: string; sizeBytes: number; error: string }

type DrawerProps = {
  open: boolean
  title: string
  description?: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
  testId?: string
}

function Drawer({ open, title, description, onClose, children, footer, testId }: DrawerProps) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50" role="dialog" aria-modal="true" data-testid={testId}>
      <button
        type="button"
        aria-label="Close drawer"
        className="absolute inset-0 bg-black/50"
        onClick={onClose}
      />
      <div className="absolute inset-y-0 right-0 w-full max-w-xl bg-card border-l border-border shadow-xl flex flex-col">
        <div className="px-6 py-4 border-b border-border flex items-start justify-between gap-4">
          <div>
            <div className="text-xs uppercase tracking-widest text-muted-foreground">{title}</div>
            {description && <div className="mt-1 text-sm text-muted-foreground">{description}</div>}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-2 text-muted-foreground hover:text-foreground hover:bg-secondary/60 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto px-6 py-6">{children}</div>
        {footer && <div className="px-6 py-4 border-t border-border bg-muted/30">{footer}</div>}
      </div>
    </div>
  )
}

export function Dashboard() {
  const { address, isConnected } = useAccount()
  const { data: evmBalance, refetch: refetchEvm } = useBalance({
    address,
    chainId: appConfig.chainId,
    query: { enabled: Boolean(address) },
  })

  const { requestFunds, loading: faucetLoading, lastTx: faucetTx, txStatus: faucetTxStatus } = useFaucet()
  const { submitDeal, loading: dealLoading, lastTx: createTx } = useCreateDeal()
  const { submitUpdate, loading: updateLoading, lastTx: updateTx } = useUpdateDealContent()
  const { upload, loading: uploadLoading } = useUpload()

  const [nilAddress, setNilAddress] = useState('')
  const [providers, setProviders] = useState<Provider[]>([])
  const providerCount = providers.length

  const [deals, setDeals] = useState<Deal[]>([])
  const [loadingDeals, setLoadingDeals] = useState(false)

  const [selectedDealId, setSelectedDealId] = useState<string>('')
  const selectedDeal = useMemo(() => (selectedDealId ? deals.find((d) => d.id === selectedDealId) || null : null), [deals, selectedDealId])
  const selectedDealService = useMemo(() => parseServiceHint(selectedDeal?.service_hint), [selectedDeal?.service_hint])
  const selectedDealIsMode2 = selectedDealService.mode === 'mode2'

  const [exploringDeal, setExploringDeal] = useState<Deal | null>(null)

  const [createDrawerOpen, setCreateDrawerOpen] = useState(false)
  const [uploadDrawerOpen, setUploadDrawerOpen] = useState(false)
  const [uploadStep, setUploadStep] = useState<1 | 2>(1)
  const [uploadPath, setUploadPath] = useState<'mode2_local' | 'gateway_legacy'>('mode2_local')

  const [statusMsg, setStatusMsg] = useState<string | null>(null)
  const [statusTone, setStatusTone] = useState<'neutral' | 'error' | 'success'>('neutral')

  const [bankBalances, setBankBalances] = useState<{ atom?: string; stake?: string }>({})

  const [duration, setDuration] = useState('100')
  const [initialEscrow, setInitialEscrow] = useState('1000000')
  const [maxMonthlySpend, setMaxMonthlySpend] = useState('5000000')
  const [redundancyMode, setRedundancyMode] = useState<'mode1' | 'mode2'>('mode2')
  const [replication, setReplication] = useState('1')
  const [rsK, setRsK] = useState('8')
  const [rsM, setRsM] = useState('4')

  const [stagedUpload, setStagedUpload] = useState<StagedUpload | null>(null)
  const [commitQueue, setCommitQueue] = useState<CommitQueueState>({ status: 'idle' })

  useEffect(() => {
    if (!address) {
      setNilAddress('')
      return
    }
    setNilAddress(ethToNil(address))
  }, [address])

  useEffect(() => {
    let cancelled = false
    async function refreshProviders() {
      try {
        const res = await fetch(`${appConfig.lcdBase}/nilchain/nilchain/v1/providers`)
        if (!res.ok) return
        const json = (await res.json().catch(() => null)) as { providers?: Provider[] } | null
        const next = Array.isArray(json?.providers) ? json!.providers : []
        if (!cancelled) setProviders(next)
      } catch {
        if (!cancelled) setProviders([])
      }
    }
    refreshProviders()
    const t = window.setInterval(refreshProviders, 10_000)
    return () => {
      cancelled = true
      window.clearInterval(t)
    }
  }, [])

  const fetchDeals = useCallback(async (owner?: string) => {
    setLoadingDeals(true)
    try {
      const all = await lcdFetchDeals(appConfig.lcdBase)
      let filtered = owner ? all.filter((d) => d.owner === owner) : all
      if (owner && filtered.length === 0 && all.length > 0) filtered = all
      setDeals(filtered)
      return filtered
    } finally {
      setLoadingDeals(false)
    }
  }, [])

  useEffect(() => {
    if (!nilAddress) return
    let cancelled = false
    const refresh = async () => {
      if (cancelled) return
      try {
        await fetchDeals(nilAddress)
      } catch {
        // silent: offline stack is common while browsing
      }
    }
    refresh()
    const t = window.setInterval(refresh, 5_000)
    return () => {
      cancelled = true
      window.clearInterval(t)
    }
  }, [fetchDeals, nilAddress])

  const fetchBalances = useCallback(async (owner: string) => {
    try {
      const res = await fetch(`${appConfig.lcdBase}/cosmos/bank/v1beta1/balances/${owner}`)
      if (!res.ok) return null
      const json = await res.json().catch(() => null)
      const bal = Array.isArray((json as { balances?: unknown[] } | null)?.balances) ? (json as { balances: { denom: string; amount: string }[] }).balances : []
      const getAmt = (denom: string) => {
        const hit = bal.find((b) => b.denom === denom)
        return hit ? hit.amount : undefined
      }
      const next = { atom: getAmt('atom'), stake: getAmt('stake') }
      setBankBalances(next)
      return next
    } catch {
      return null
    }
  }, [])

  useEffect(() => {
    if (!nilAddress) return
    fetchBalances(nilAddress).catch(() => undefined)
  }, [fetchBalances, nilAddress])

  const dealSummary = useMemo(() => {
    let active = 0
    let totalBytes = 0
    for (const deal of deals) {
      if (String(deal.cid || '').trim()) active += 1
      const sizeNum = Number(deal.size)
      if (Number.isFinite(sizeNum) && sizeNum > 0) totalBytes += sizeNum
    }
    return { total: deals.length, active, totalBytes }
  }, [deals])

  const mode2Config = useMemo(() => {
    if (redundancyMode !== 'mode2') {
      return { slots: null as number | null, error: null as string | null, warning: null as string | null }
    }
    const k = Number(rsK)
    const m = Number(rsM)
    if (!Number.isFinite(k) || !Number.isFinite(m) || k <= 0 || m <= 0) {
      return { slots: null, error: 'Enter numeric K and M values.', warning: null }
    }
    const slots = k + m
    if (64 % k !== 0) {
      return { slots, error: 'K must divide 64.', warning: null }
    }
    if (providerCount > 0 && slots > providerCount) {
      return { slots, error: null, warning: `Need ${slots} providers (K+M); only ${providerCount} available.` }
    }
    return { slots, error: null, warning: null }
  }, [providerCount, redundancyMode, rsK, rsM])

  const handleRequestFaucet = async () => {
    if (!address) return
    try {
      setStatusTone('neutral')
      setStatusMsg('Requesting faucet...')
      await requestFunds(address)
      if (nilAddress) {
        setStatusMsg('Faucet requested. Waiting for balance...')
        await fetchBalances(nilAddress)
      }
      refetchEvm?.()
    } catch {
      setStatusTone('error')
      setStatusMsg('Faucet request failed. Is the faucet running?')
    }
  }

  const handleCreateDeal = async () => {
    if (!address) {
      setStatusTone('error')
      setStatusMsg('Connect wallet first.')
      return
    }
    if (!bankBalances.stake && !bankBalances.atom) {
      setStatusTone('error')
      setStatusMsg('Request testnet tokens from the faucet before creating a deal.')
      return
    }

    try {
      let serviceHint = ''
      if (redundancyMode === 'mode2') {
        const k = Number(rsK)
        const m = Number(rsM)
        if (!Number.isFinite(k) || !Number.isFinite(m) || k <= 0 || m <= 0) throw new Error('Mode 2 requires numeric K and M values.')
        if (64 % k !== 0) throw new Error('Mode 2 requires K to divide 64.')
        serviceHint = buildServiceHint('General', { replicas: k + m, rsK: k, rsM: m })
      } else {
        const replicas = Math.max(1, Number(replication) || 1)
        serviceHint = buildServiceHint('General', { replicas })
      }

      const res = await submitDeal({
        creator: address,
        duration: Math.max(1, Number(duration) || 1),
        initialEscrow,
        maxMonthlySpend,
        replication: redundancyMode === 'mode1' ? Math.max(1, Number(replication) || 1) : undefined,
        serviceHint,
      })

      setStatusTone('success')
      setStatusMsg(`Capacity Allocated (Deal ID: ${res.deal_id}).`)
      if (nilAddress) {
        await fetchDeals(nilAddress)
        await fetchBalances(nilAddress)
      }
      setSelectedDealId(String(res.deal_id))
      setCreateDrawerOpen(false)
      setUploadDrawerOpen(true)
      setUploadStep(1)
      setUploadPath('mode2_local')
    } catch (e) {
      setStatusTone('error')
      setStatusMsg(e instanceof Error ? e.message : 'Deal allocation failed.')
    }
  }

  const handleLegacyGatewayFileChange = async (file: File | null) => {
    if (!file) return
    if (!address) throw new Error('Wallet not connected')
    if (!selectedDealId) throw new Error('Select a deal first')
    setStagedUpload(null)
    try {
      const res = await upload(file, address, { dealId: selectedDealId })
      setStagedUpload(res)
      setCommitQueue({ status: 'idle' })
    } catch (e) {
      setStatusTone('error')
      setStatusMsg(e instanceof Error ? e.message : 'Upload failed')
    }
  }

  const triggerCommit = async (manifestRoot: string, sizeBytes: number) => {
    if (!address) throw new Error('Wallet not connected')
    const dealIdNum = Number(selectedDealId)
    if (!Number.isFinite(dealIdNum) || dealIdNum < 0) throw new Error('Invalid deal id')
    setCommitQueue({ status: 'pending', manifestRoot, sizeBytes })
    try {
      await submitUpdate({
        creator: address,
        dealId: dealIdNum,
        cid: manifestRoot,
        sizeBytes,
      })
      setCommitQueue({ status: 'confirmed', manifestRoot, sizeBytes })
      setStatusTone('success')
      setStatusMsg('Content committed on-chain.')
      if (nilAddress) await fetchDeals(nilAddress)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Commit failed'
      setCommitQueue({ status: 'error', manifestRoot, sizeBytes, error: msg })
      setStatusTone('error')
      setStatusMsg(msg)
    }
  }

  const handleCommitLegacy = async () => {
    if (!stagedUpload) return
    setCommitQueue({ status: 'queued', manifestRoot: stagedUpload.cid, sizeBytes: stagedUpload.sizeBytes })
    await triggerCommit(stagedUpload.cid, stagedUpload.sizeBytes)
  }

  const selectedDealLabel = selectedDealId ? `#${selectedDealId}` : '—'

  const shortEvmBalance = useMemo(() => {
    if (!evmBalance) return '—'
    const symbol = evmBalance.symbol || 'NIL'
    const formatted = formatUnits(evmBalance.value, evmBalance.decimals)
    const [whole, frac] = formatted.split('.')
    const trimmed = frac ? `${whole}.${frac.slice(0, 4)}` : whole
    return `${trimmed} ${symbol}`
  }, [evmBalance])

  return (
    <div className="space-y-6">
      <StatusBar />

      <div className="bg-card rounded-xl border border-border overflow-hidden shadow-sm">
        <div className="px-6 py-4 border-b border-border flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Wallet</div>
            <div className="mt-1 text-sm text-muted-foreground">
              {isConnected ? 'Connected (see navbar)' : 'Not connected (use navbar)'}
            </div>
            {isConnected && (
              <div className="mt-2 text-xs text-muted-foreground">
                Cosmos: <span className="font-mono text-primary" data-testid="cosmos-identity">{nilAddress}</span>
              </div>
            )}
          </div>

          <div className="flex flex-col items-end gap-2">
            <button
              type="button"
              onClick={handleRequestFaucet}
              disabled={!isConnected || faucetLoading}
              data-testid="faucet-request"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-secondary/60 hover:bg-secondary border border-border text-foreground text-sm font-semibold transition-colors disabled:opacity-60"
            >
              <ArrowDownRight className="w-4 h-4" />
              Get Testnet NIL
            </button>
            <div className="grid grid-cols-2 gap-3 text-xs w-full max-w-[360px]">
              <div className="bg-secondary/50 border border-border rounded p-2">
                <div className="text-muted-foreground uppercase tracking-wide">EVM (NIL)</div>
                <div className="font-mono text-green-600 dark:text-green-400">{shortEvmBalance}</div>
              </div>
              <div className="bg-secondary/50 border border-border rounded p-2">
                <div className="text-muted-foreground uppercase tracking-wide">Cosmos stake</div>
                <div className="font-mono text-blue-600 dark:text-blue-400" data-testid="cosmos-stake-balance">
                  {bankBalances.stake ? `${bankBalances.stake} stake` : '—'}
                </div>
              </div>
            </div>

            {faucetTx && (
              <div className="flex flex-wrap items-center gap-2 text-xs text-green-600 dark:text-green-400 bg-green-500/10 px-2 py-1 rounded border border-green-500/20">
                <ArrowDownRight className="w-3 h-3 flex-shrink-0" />
                <span className="truncate max-w-[180px]" title={faucetTx}>
                  Tx: <span className="font-mono">{faucetTx.slice(0, 10)}...{faucetTx.slice(-8)}</span>
                </span>
                <span className="opacity-75">({faucetTxStatus})</span>
              </div>
            )}
          </div>
        </div>

        {statusMsg && (
          <div
            className={`px-6 py-3 text-sm border-b border-border ${
              statusTone === 'error'
                ? 'bg-destructive/10 text-destructive'
                : statusTone === 'success'
                  ? 'bg-green-500/10 text-green-600 dark:text-green-400'
                  : 'bg-secondary/40 text-muted-foreground'
            }`}
          >
            {statusMsg}
          </div>
        )}
      </div>

      <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
        <div className="px-6 py-4 border-b border-border flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Deals</div>
            <h3 className="text-lg font-semibold text-foreground">Deal Library</h3>
            <div className="mt-1 text-xs text-muted-foreground">
              Select a deal, then upload a file or explore its contents.
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              data-testid="create-deal-open"
              onClick={() => setCreateDrawerOpen(true)}
              className="px-3 py-2 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              Create Deal
            </button>
            <button
              type="button"
              data-testid="upload-open"
              onClick={() => {
                if (!selectedDealId) return
                setUploadDrawerOpen(true)
                setUploadStep(1)
                setUploadPath('mode2_local')
              }}
              disabled={!selectedDealId}
              className="px-3 py-2 text-sm rounded-md border border-primary/30 text-primary hover:bg-primary/10 disabled:opacity-50 transition-colors"
            >
              Upload File
            </button>
            <button
              type="button"
              data-testid="explore-open"
              onClick={() => selectedDeal && setExploringDeal(selectedDeal)}
              disabled={!selectedDealId || !selectedDeal}
              className="px-3 py-2 text-sm rounded-md border border-border text-foreground hover:bg-secondary/60 disabled:opacity-50 transition-colors"
            >
              Explore
            </button>
          </div>
        </div>

        <div className="px-6 py-3 bg-muted/30 border-b border-border flex flex-wrap items-center justify-between gap-3">
          <div className="text-xs text-muted-foreground">
            Selected:{' '}
            <span className="font-mono text-foreground" data-testid="selected-deal-id">
              {selectedDealLabel}
            </span>
            {' '}• Active:{' '}
            <span className="font-mono text-foreground">{dealSummary.active}</span>
            {' '}• Total:{' '}
            <span className="font-mono text-foreground">{dealSummary.total}</span>
          </div>
          <div className="text-xs text-muted-foreground">
            Providers known: <span className="font-mono text-foreground">{providerCount || '—'}</span>
          </div>
        </div>

        {loadingDeals && deals.length === 0 ? (
          <div className="text-center py-24">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
            <p className="text-muted-foreground">Syncing with NilChain...</p>
          </div>
        ) : deals.length === 0 ? (
          <div className="p-10 text-center">
            <div className="w-14 h-14 bg-secondary rounded-full flex items-center justify-center mx-auto mb-4">
              <HardDrive className="w-7 h-7 text-muted-foreground" />
            </div>
            <h3 className="text-base font-medium text-foreground mb-2">No deals yet</h3>
            <p className="text-muted-foreground text-sm">Create a deal to start storing files.</p>
          </div>
        ) : (
          <table className="min-w-full divide-y divide-border" data-testid="deals-table">
            <thead className="bg-muted/50">
              <tr>
                <th className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Deal ID</th>
                <th className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Manifest Root</th>
                <th className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Size</th>
                <th className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Status</th>
                <th className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Provider</th>
                <th className="px-6 py-4 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {deals.map((deal) => (
                <tr
                  key={deal.id}
                  data-testid={`deal-row-${deal.id}`}
                  className={`hover:bg-muted/50 transition-colors cursor-pointer ${selectedDealId === deal.id ? 'bg-secondary/40' : ''}`}
                  onClick={() => setSelectedDealId(String(deal.id ?? ''))}
                >
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-foreground">#{deal.id}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-primary" title={deal.cid} data-testid={`deal-manifest-${deal.id}`}>
                    {deal.cid ? `${deal.cid.slice(0, 18)}...` : <span className="text-muted-foreground italic">Empty</span>}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground" data-testid={`deal-size-${deal.id}`}>
                    {(() => {
                      const sizeNum = Number(deal.size)
                      if (!Number.isFinite(sizeNum) || sizeNum <= 0) return '—'
                      return `${(sizeNum / 1024 / 1024).toFixed(2)} MB`
                    })()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    {deal.cid ? (
                      <span className="px-2 py-1 text-xs rounded-full bg-green-500/10 text-green-600 dark:text-green-400 border border-green-500/20">
                        Active
                      </span>
                    ) : (
                      <span className="px-2 py-1 text-xs rounded-full bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-500/20">
                        Allocated
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-muted-foreground">
                    {deal.providers && deal.providers.length > 0 ? `${deal.providers[0].slice(0, 10)}...${deal.providers[0].slice(-4)}` : '—'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation()
                          setSelectedDealId(String(deal.id ?? ''))
                          setUploadDrawerOpen(true)
                          setUploadStep(1)
                          setUploadPath('mode2_local')
                        }}
                        className="px-3 py-1.5 text-xs rounded-md border border-primary/30 text-primary hover:bg-primary/10"
                      >
                        Upload
                      </button>
                      <button
                        type="button"
                        data-testid={`deal-explore-${deal.id}`}
                        onClick={(e) => {
                          e.stopPropagation()
                          setExploringDeal(deal)
                        }}
                        className="px-3 py-1.5 text-xs rounded-md border border-border text-foreground hover:bg-secondary/60"
                      >
                        Explore
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {exploringDeal && <DealDetail deal={exploringDeal} onClose={() => setExploringDeal(null)} nilAddress={nilAddress} />}

      <Drawer
        open={createDrawerOpen}
        onClose={() => setCreateDrawerOpen(false)}
        title="Create Deal"
        description="Create a new container deal. Mode 2 (StripeReplica) is the default; Mode 1 is legacy/advanced."
        testId="create-deal-drawer"
        footer={
          <div className="flex items-center justify-between gap-3">
            <div className="text-xs text-muted-foreground">
              {createTx && (
                <span className="text-green-600 dark:text-green-400 flex items-center gap-1">
                  <CheckCircle2 className="w-3 h-3" /> Tx: {createTx.slice(0, 10)}...
                </span>
              )}
            </div>
            <button
              onClick={handleCreateDeal}
              disabled={!isConnected || dealLoading || (redundancyMode === 'mode2' && Boolean(mode2Config.error))}
              data-testid="alloc-submit"
              className="px-4 py-2 bg-primary hover:bg-primary/90 text-primary-foreground text-sm font-medium rounded-md disabled:opacity-50 transition-colors"
            >
              {dealLoading ? 'Creating...' : 'Create Deal'}
            </button>
          </div>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3 text-sm">
            <label className="space-y-1">
              <span className="text-xs uppercase tracking-wide text-muted-foreground">Duration (blocks)</span>
              <input
                type="number"
                value={duration}
                onChange={(e) => setDuration(e.target.value)}
                data-testid="alloc-duration"
                className="w-full bg-background border border-border rounded px-3 py-2 text-foreground text-sm focus:outline-none focus:border-primary"
              />
            </label>
            <label className="space-y-1">
              <span className="text-xs uppercase tracking-wide text-muted-foreground">Initial Escrow</span>
              <input
                type="number"
                value={initialEscrow}
                onChange={(e) => setInitialEscrow(e.target.value)}
                data-testid="alloc-escrow"
                className="w-full bg-background border border-border rounded px-3 py-2 text-foreground text-sm focus:outline-none focus:border-primary"
              />
            </label>
            <label className="space-y-1">
              <span className="text-xs uppercase tracking-wide text-muted-foreground">Max Monthly Spend</span>
              <input
                type="number"
                value={maxMonthlySpend}
                onChange={(e) => setMaxMonthlySpend(e.target.value)}
                data-testid="alloc-monthly"
                className="w-full bg-background border border-border rounded px-3 py-2 text-foreground text-sm focus:outline-none focus:border-primary"
              />
            </label>
            <label className="space-y-1">
              <span className="text-xs uppercase tracking-wide text-muted-foreground">Redundancy Mode</span>
              <select
                value={redundancyMode}
                onChange={(e) => setRedundancyMode(e.target.value === 'mode1' ? 'mode1' : 'mode2')}
                data-testid="alloc-redundancy-mode"
                className="w-full bg-background border border-border rounded px-3 py-2 text-foreground text-sm focus:outline-none focus:border-primary"
              >
                <option value="mode2">Mode 2 (StripeReplica, RS)</option>
                <option value="mode1">Mode 1 (Replication) — Advanced</option>
              </select>
            </label>
          </div>

          {redundancyMode === 'mode2' ? (
            <div className="space-y-2">
              <div className="grid grid-cols-2 gap-3">
                <label className="space-y-1">
                  <span className="text-xs uppercase tracking-wide text-muted-foreground">RS K (Data)</span>
                  <input
                    type="number"
                    min={1}
                    max={64}
                    value={rsK}
                    onChange={(e) => setRsK(e.target.value)}
                    data-testid="alloc-rs-k"
                    className="w-full bg-background border border-border rounded px-3 py-2 text-foreground text-sm focus:outline-none focus:border-primary"
                  />
                </label>
                <label className="space-y-1">
                  <span className="text-xs uppercase tracking-wide text-muted-foreground">RS M (Parity)</span>
                  <input
                    type="number"
                    min={1}
                    max={64}
                    value={rsM}
                    onChange={(e) => setRsM(e.target.value)}
                    data-testid="alloc-rs-m"
                    className="w-full bg-background border border-border rounded px-3 py-2 text-foreground text-sm focus:outline-none focus:border-primary"
                  />
                </label>
              </div>
              <div className="text-[11px] text-muted-foreground">
                Slots required: <span className="font-mono text-foreground">{mode2Config.slots ?? '—'}</span>
                {' '}• Providers available:{' '}
                <span className="font-mono text-foreground">{providerCount || '—'}</span>
                {mode2Config.error && <div className="text-[11px] text-red-500 mt-1">{mode2Config.error}</div>}
                {mode2Config.warning && (
                  <div className="text-[11px] text-yellow-600 dark:text-yellow-400 mt-1">{mode2Config.warning}</div>
                )}
              </div>
            </div>
          ) : (
            <div className="space-y-2">
              <label className="space-y-1">
                <span className="text-xs uppercase tracking-wide text-muted-foreground">Replication</span>
                <input
                  type="number"
                  min={1}
                  max={12}
                  value={replication}
                  onChange={(e) => setReplication(e.target.value)}
                  data-testid="alloc-replication"
                  className="w-full bg-background border border-border rounded px-3 py-2 text-foreground text-sm focus:outline-none focus:border-primary"
                />
              </label>
              <div className="text-[11px] text-muted-foreground">
                Legacy path: full replication. Mode 2 is recommended for new deals.
              </div>
            </div>
          )}
        </div>
      </Drawer>

      <Drawer
        open={uploadDrawerOpen}
        onClose={() => setUploadDrawerOpen(false)}
        title="Upload File"
        description={selectedDealId ? `Upload into deal ${selectedDealLabel}.` : 'Select a deal first.'}
        testId="upload-drawer"
        footer={
          <div className="flex items-center justify-between gap-3">
            <div className="text-xs text-muted-foreground">Step {uploadStep} of 2</div>
            {uploadStep === 1 ? (
              <button
                type="button"
                data-testid="upload-continue"
                disabled={!selectedDealId}
                onClick={() => setUploadStep(2)}
                className="px-4 py-2 bg-primary hover:bg-primary/90 text-primary-foreground text-sm font-medium rounded-md disabled:opacity-50 transition-colors"
              >
                Continue
              </button>
            ) : (
              <button
                type="button"
                onClick={() => setUploadStep(1)}
                className="px-4 py-2 border border-border text-foreground text-sm font-medium rounded-md hover:bg-secondary/60 transition-colors"
              >
                Back
              </button>
            )}
          </div>
        }
      >
        {!selectedDealId ? (
          <div className="p-8 text-center border border-dashed border-border rounded-xl">
            <p className="text-muted-foreground text-sm">Select a deal from the Deal Library first.</p>
          </div>
        ) : uploadStep === 1 ? (
          <div className="space-y-4">
            <div className="text-sm text-muted-foreground">
              Choose how to upload. Mode 2 (local sharding) is recommended; gateway upload is legacy.
            </div>

            <div className="grid grid-cols-1 gap-3">
              <button
                type="button"
                data-testid="upload-path-mode2"
                onClick={() => setUploadPath('mode2_local')}
                className={`rounded-lg border px-4 py-3 text-left transition-colors ${
                  uploadPath === 'mode2_local'
                    ? 'border-primary/40 bg-primary/5'
                    : 'border-border hover:bg-secondary/40'
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="font-medium text-foreground">Mode 2 (Local WASM) — Recommended</div>
                  <span className="text-[11px] text-muted-foreground">Fast + verifiable</span>
                </div>
                <div className="mt-1 text-xs text-muted-foreground flex items-center gap-1">
                  <ArrowUpRight className="w-3 h-3" /> Shards locally, uploads stripes to providers, then commits on-chain.
                </div>
              </button>

              <button
                type="button"
                data-testid="upload-path-gateway"
                onClick={() => setUploadPath('gateway_legacy')}
                disabled={selectedDealIsMode2}
                className={`rounded-lg border px-4 py-3 text-left transition-colors disabled:opacity-50 ${
                  uploadPath === 'gateway_legacy'
                    ? 'border-primary/40 bg-primary/5'
                    : 'border-border hover:bg-secondary/40'
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <div className="font-medium text-foreground">Gateway Upload (Legacy)</div>
                  <span className="text-[11px] text-muted-foreground">Mode 1 only</span>
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  Upload through the gateway for legacy deals. Disabled for Mode 2 deals.
                </div>
              </button>
            </div>
          </div>
        ) : uploadPath === 'mode2_local' ? (
          <div className="space-y-4">
            <div className="text-xs text-muted-foreground">
              Deal selected: <span className="font-mono text-foreground">{selectedDealLabel}</span>
            </div>
            <FileSharder dealId={selectedDealId} />
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-xs text-muted-foreground">
              Legacy gateway flow. Upload a file, then finalize the on-chain commit.
            </p>
            <div className="grid grid-cols-1 gap-3 text-sm">
              <div className="text-xs text-muted-foreground">
                Deal selected: <span className="font-mono text-foreground">{selectedDealLabel}</span>
              </div>

              <label className="space-y-1">
                <span className="text-xs uppercase tracking-wide text-muted-foreground flex items-center gap-2">
                  <Upload className="w-3 h-3 text-primary" />
                  Upload &amp; Shard (gateway)
                </span>
                <input
                  type="file"
                  onChange={(e) => handleLegacyGatewayFileChange(e.target.files?.[0] ?? null)}
                  disabled={uploadLoading || !selectedDealId || selectedDealIsMode2}
                  data-testid="content-file-input"
                  className="w-full text-xs text-muted-foreground file:mr-2 file:py-1.5 file:px-3 file:rounded-md file:border-0 file:text-xs file:font-medium file:bg-primary file:text-primary-foreground hover:file:bg-primary/90 file:cursor-pointer cursor-pointer"
                />
              </label>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <span className="text-xs uppercase tracking-wide text-muted-foreground">Staged Manifest Root</span>
                  <div
                    className="w-full bg-secondary border border-border rounded px-3 py-2 text-foreground text-sm font-mono text-xs min-h-[40px] flex items-center"
                    data-testid="staged-manifest-root"
                  >
                    {stagedUpload?.cid ? (
                      <span className="break-all">{stagedUpload.cid}</span>
                    ) : (
                      <span className="text-muted-foreground">Upload a file to populate</span>
                    )}
                  </div>
                </div>
                <div className="space-y-1">
                  <span className="text-xs uppercase tracking-wide text-muted-foreground">Staged Total Size (bytes)</span>
                  <div
                    className="w-full bg-secondary border border-border rounded px-3 py-2 text-foreground text-sm font-mono text-xs min-h-[40px] flex items-center"
                    data-testid="staged-total-size"
                  >
                    {stagedUpload?.sizeBytes ? <span>{stagedUpload.sizeBytes}</span> : <span className="text-muted-foreground">Upload a file</span>}
                  </div>
                </div>
              </div>

              {stagedUpload && (
                <div className="rounded-md border border-border bg-secondary/30 px-3 py-2 text-[11px] text-muted-foreground">
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate">
                      Staged: <span className="font-mono text-foreground">{stagedUpload.filename}</span> ({stagedUpload.fileSizeBytes} bytes)
                    </span>
                    {updateTx && (
                      <span className="text-green-600 dark:text-green-400 flex items-center gap-1">
                        <CheckCircle2 className="w-3 h-3" /> Tx: {updateTx.slice(0, 10)}...
                      </span>
                    )}
                  </div>
                  <div className="mt-1">
                    {commitQueue.status === 'queued' && 'Queued — confirm the transaction to finalize.'}
                    {commitQueue.status === 'pending' && 'Waiting for wallet confirmation...'}
                    {commitQueue.status === 'confirmed' && 'Committed on-chain.'}
                    {commitQueue.status === 'error' && <span className="text-red-500">Commit failed: {commitQueue.error}</span>}
                  </div>
                </div>
              )}

              <button
                onClick={handleCommitLegacy}
                disabled={updateLoading || commitQueue.status === 'pending' || commitQueue.status === 'confirmed' || !stagedUpload || selectedDealIsMode2}
                data-testid="content-commit"
                className="px-4 py-2 bg-primary hover:bg-primary/90 text-primary-foreground text-sm font-medium rounded-md disabled:opacity-50 transition-colors"
              >
                {commitQueue.status === 'confirmed' ? 'Committed' : commitQueue.status === 'pending' || updateLoading ? 'Check Wallet...' : 'Finalize Commit'}
              </button>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  )
}
