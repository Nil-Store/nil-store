import { useState } from 'react'
import { useAccount } from 'wagmi'
import { numberToHex, type Hex } from 'viem'

import { appConfig } from '../config'
import { normalizeDealId } from '../lib/dealId'
import { buildRetrievalRequestTypedData } from '../lib/eip712'
import { waitForTransactionReceipt } from '../lib/evmRpc'
import {
  decodeComputeRetrievalSessionIdsResult,
  encodeComputeRetrievalSessionIdsData,
  encodeConfirmRetrievalSessionsData,
  encodeOpenRetrievalSessionsData,
} from '../lib/nilstorePrecompile'
import { planNilfsFileRangeChunks } from '../lib/rangeChunker'
import {
  resolveProviderEndpoint,
  resolveProviderEndpointByAddress,
  resolveProviderP2pEndpoint,
  resolveProviderP2pEndpointByAddress,
} from '../lib/providerDiscovery'
import { fetchGatewayP2pAddrs } from '../lib/gatewayStatus'
import { multiaddrToP2pTarget, type P2pTarget } from '../lib/multiaddr'
import { useTransportRouter } from './useTransportRouter'
import type { RoutePreference } from '../lib/transport/types'

export interface FetchInput {
  dealId: string
  manifestRoot: string
  owner: string
  filePath: string
  /**
   * When true, performs an on-chain RetrievalSession flow and submits proofs (slow, costs escrow).
   * When false (default), downloads via a fast path (no receipts).
   */
  withReceipt?: boolean
  /**
   * Base URL for the service hosting `/gateway/*` retrieval endpoints.
   * Defaults to `appConfig.gatewayBase`.
   *
   * In thick-client flows, this often needs to point at the Storage Provider (`appConfig.spBase`)
   * because the local gateway may not have the slab on disk.
   */
  serviceBase?: string
  rangeStart?: number
  rangeLen?: number
  fileStartOffset?: number
  fileSizeBytes?: number
  mduSizeBytes?: number
  blobSizeBytes?: number
}

export type FetchPhase =
  | 'idle'
  | 'opening_session_tx'
  | 'fetching'
  | 'confirming_session_tx'
  | 'submitting_proof_request'
  | 'done'
  | 'error'

export interface FetchProgress {
  phase: FetchPhase
  filePath: string
  chunksFetched: number
  chunkCount: number
  bytesFetched: number
  bytesTotal: number
  receiptsSubmitted: number
  receiptsTotal: number
  message?: string
}

export interface FetchResult {
  url: string
  blob: Blob
}

function decodeHttpError(bodyText: string): string {
  const trimmed = bodyText?.trim?.() ? bodyText.trim() : String(bodyText ?? '')
  if (!trimmed) return 'request failed'
  try {
    const json = JSON.parse(trimmed)
    if (json && typeof json === 'object') {
      if (typeof json.error === 'string' && json.error.trim()) {
        const hint = typeof json.hint === 'string' && json.hint.trim() ? ` (${json.hint.trim()})` : ''
        return `${json.error.trim()}${hint}`
      }
      if (typeof json.message === 'string' && json.message.trim()) {
        return json.message.trim()
      }
    }
  } catch (e) {
    void e
  }
  return trimmed
}

function rawFetchUrl(base: string, req: { manifestRoot: Hex; dealId: string; owner: string; filePath: string; rangeStart: number; rangeLen: number }): string {
  const normalizeBase = (s: string) => s.replace(/\/$/, '')
  const q = new URLSearchParams()
  q.set('deal_id', req.dealId)
  q.set('owner', req.owner)
  q.set('file_path', req.filePath)
  q.set('range_start', String(req.rangeStart))
  q.set('range_len', String(req.rangeLen))
  return `${normalizeBase(base)}/gateway/debug/raw-fetch/${encodeURIComponent(req.manifestRoot)}?${q.toString()}`
}

export function useFetch() {
  const { address } = useAccount()
  const transport = useTransportRouter()
  const [loading, setLoading] = useState(false)
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null)
  const [receiptStatus, setReceiptStatus] = useState<'idle' | 'submitted' | 'failed'>('idle')
  const [receiptError, setReceiptError] = useState<string | null>(null)
  const [progress, setProgress] = useState<FetchProgress>({
    phase: 'idle',
    filePath: '',
    chunksFetched: 0,
    chunkCount: 0,
    bytesFetched: 0,
    bytesTotal: 0,
    receiptsSubmitted: 0,
    receiptsTotal: 0,
  })

  async function fetchFile(input: FetchInput): Promise<FetchResult | null> {
    setLoading(true)
    setDownloadUrl(null)
    setReceiptStatus('idle')
    setReceiptError(null)
    setProgress({
      phase: 'idle',
      filePath: String(input.filePath || ''),
      chunksFetched: 0,
      chunkCount: 0,
      bytesFetched: 0,
      bytesTotal: 0,
      receiptsSubmitted: 0,
      receiptsTotal: 0,
    })

    try {
      const withReceipt = Boolean(input.withReceipt)
      const ethereum = window.ethereum
      if (withReceipt) {
        if (!address) throw new Error('Connect a wallet to submit retrieval proofs')
        if (!ethereum || typeof ethereum.request !== 'function') {
          throw new Error('Ethereum provider (MetaMask) not available')
        }
      }

      const dealId = normalizeDealId(input.dealId)
      const owner = String(input.owner ?? '').trim()
      if (!owner) throw new Error('owner is required')
      const filePath = String(input.filePath || '').trim()
      if (!filePath) throw new Error('filePath is required')

      const manifestRoot = String(input.manifestRoot || '').trim() as Hex
      if (!manifestRoot.startsWith('0x')) throw new Error('manifestRoot must be 0x-prefixed hex bytes')

      const blobSizeBytes = Number(input.blobSizeBytes || 128 * 1024)
      const mduSizeBytes = Number(input.mduSizeBytes || 8 * 1024 * 1024)
      const wantRangeStart = Math.max(0, Number(input.rangeStart ?? 0))
      const wantRangeLen = Math.max(0, Number(input.rangeLen ?? 0))
      const wantFileSize = typeof input.fileSizeBytes === 'number' ? Number(input.fileSizeBytes) : 0

      let effectiveRangeLen = wantRangeLen
      if (effectiveRangeLen === 0) {
        if (!wantFileSize) throw new Error('fileSizeBytes is required for full downloads (rangeLen=0)')
        if (wantRangeStart >= wantFileSize) throw new Error('rangeStart beyond EOF')
        effectiveRangeLen = wantFileSize - wantRangeStart
      }

      const hasMeta =
        typeof input.fileStartOffset === 'number' &&
        typeof input.fileSizeBytes === 'number' &&
        typeof input.mduSizeBytes === 'number' &&
        typeof input.blobSizeBytes === 'number'

      const chunks = hasMeta
        ? planNilfsFileRangeChunks({
            fileStartOffset: input.fileStartOffset!,
            fileSizeBytes: input.fileSizeBytes!,
            rangeStart: wantRangeStart,
            rangeLen: effectiveRangeLen,
            mduSizeBytes: input.mduSizeBytes!,
            blobSizeBytes: input.blobSizeBytes!,
          })
        : [{ rangeStart: wantRangeStart, rangeLen: effectiveRangeLen }]

      if (withReceipt && !hasMeta && effectiveRangeLen > blobSizeBytes) {
        throw new Error('range fetch > blob size requires fileStartOffset/fileSizeBytes/mduSizeBytes/blobSizeBytes')
      }

      const serviceOverride = String(input.serviceBase ?? '').trim().replace(/\/$/, '')
      const preferenceOverride: RoutePreference | undefined =
        serviceOverride && serviceOverride !== appConfig.gatewayBase && transport.preference !== 'prefer_p2p'
          ? 'prefer_direct_sp'
          : undefined
      const directEndpoint = await resolveProviderEndpoint(appConfig.lcdBase, dealId).catch(() => null)
      const p2pEndpoint = await resolveProviderP2pEndpoint(appConfig.lcdBase, dealId).catch(() => null)
      const directBase = serviceOverride || directEndpoint?.baseUrl || appConfig.spBase

      let gatewayP2pTarget: P2pTarget | undefined
      if (appConfig.p2pEnabled && !appConfig.gatewayDisabled && !p2pEndpoint?.target) {
        const addrs = await fetchGatewayP2pAddrs(appConfig.gatewayBase)
        for (const addr of addrs) {
          const target = multiaddrToP2pTarget(addr)
          if (target) {
            gatewayP2pTarget = target
            break
          }
        }
      }

      const planP2pTarget = p2pEndpoint?.target || gatewayP2pTarget || undefined
      const providerEndpointCache = new Map<string, Awaited<ReturnType<typeof resolveProviderEndpointByAddress>> | null>()
      const providerP2pCache = new Map<string, Awaited<ReturnType<typeof resolveProviderP2pEndpointByAddress>> | null>()

      const getProviderEndpoint = async (provider: string) => {
        if (providerEndpointCache.has(provider)) return providerEndpointCache.get(provider) ?? null
        const endpoint = await resolveProviderEndpointByAddress(appConfig.lcdBase, provider).catch(() => null)
        providerEndpointCache.set(provider, endpoint)
        return endpoint
      }
      const getProviderP2pEndpoint = async (provider: string) => {
        if (providerP2pCache.has(provider)) return providerP2pCache.get(provider) ?? null
        const endpoint = await resolveProviderP2pEndpointByAddress(appConfig.lcdBase, provider).catch(() => null)
        providerP2pCache.set(provider, endpoint)
        return endpoint
      }

      if (!withReceipt) {
        const shouldTryRawFetch = effectiveRangeLen > blobSizeBytes
        const rawFetchBases = serviceOverride
          ? [serviceOverride]
          : [
              ...(!appConfig.gatewayDisabled ? [appConfig.gatewayBase] : []),
              ...(directBase && directBase !== appConfig.gatewayBase ? [directBase] : []),
            ]

        const tryRawFetch = async (): Promise<Uint8Array | null> => {
          if (!shouldTryRawFetch) return null
          for (const base of rawFetchBases) {
            const url = rawFetchUrl(base, { manifestRoot, dealId, owner, filePath, rangeStart: wantRangeStart, rangeLen: effectiveRangeLen })
            try {
              const res = await fetch(url)
              if (!res.ok) {
                const txt = await res.text().catch(() => '')
                throw new Error(decodeHttpError(txt) || `raw fetch failed (${res.status})`)
              }
              return new Uint8Array(await res.arrayBuffer())
            } catch (e) {
              const msg = e instanceof Error ? decodeHttpError(e.message) : String(e)
              // Only treat "slab missing" as a recoverable raw-fetch failure; other errors should surface.
              if (/slab not found on disk/i.test(msg) || /file not found in deal/i.test(msg)) continue
              throw e
            }
          }
          return null
        }

        setProgress((p) => ({
          ...p,
          phase: 'fetching',
          filePath,
          chunkCount: chunks.length,
          chunksFetched: 0,
          bytesTotal: effectiveRangeLen,
          bytesFetched: 0,
          receiptsSubmitted: 0,
          receiptsTotal: 0,
        }))

        const rawBytes = await tryRawFetch()
        if (rawBytes) {
          const blob = new Blob([rawBytes] as BlobPart[], { type: 'application/octet-stream' })
          const url = URL.createObjectURL(blob)
          setDownloadUrl(url)
          setProgress((p) => ({
            ...p,
            phase: 'done',
            chunksFetched: chunks.length,
            bytesFetched: rawBytes.byteLength,
          }))
          return { url, blob }
        }

        // Fallback: open a download session (1 signature) then fetch blob-sized chunks with X-Nil-Download-Session.
        if (!ethereum || typeof ethereum.request !== 'function') {
          throw new Error('Ethereum provider (MetaMask) not available')
        }
        if (!address) throw new Error('Connect a wallet to authorize download')
        if (!hasMeta && effectiveRangeLen > blobSizeBytes) {
          throw new Error('range fetch > blob size requires fileStartOffset/fileSizeBytes/mduSizeBytes/blobSizeBytes')
        }

        let metaAuth:
          | {
              reqSig: string
              reqNonce: number
              reqExpiresAt: number
              signedRangeStart: number
              signedRangeLen: number
            }
          | undefined

        const shouldSignMetaAuth = (err: unknown): boolean => {
          if (!(err instanceof Error)) return false
          const msg = decodeHttpError(err.message)
          return /req_sig is required/i.test(msg) || /invalid req_nonce/i.test(msg) || /invalid req_expires_at/i.test(msg)
        }

        const signMetaAuth = async () => {
          const now = Math.floor(Date.now() / 1000)
          const reqNonce = Math.floor(Math.random() * 1_000_000_000) + Date.now()
          const reqExpiresAt = now + 9 * 60
          const typedData = buildRetrievalRequestTypedData(
            {
              deal_id: Number(dealId),
              file_path: filePath,
              range_start: wantRangeStart,
              range_len: effectiveRangeLen,
              nonce: reqNonce,
              expires_at: reqExpiresAt,
            },
            appConfig.chainId,
          )
          const reqSig = (await ethereum.request({
            method: 'eth_signTypedData_v4',
            params: [address, JSON.stringify(typedData)],
          })) as string

          metaAuth = {
            reqSig,
            reqNonce,
            reqExpiresAt,
            signedRangeStart: wantRangeStart,
            signedRangeLen: effectiveRangeLen,
          }
          return metaAuth
        }

        const openDownloadSession = async (base: string, auth?: typeof metaAuth): Promise<string> => {
          const normalizeBase = (s: string) => s.replace(/\/$/, '')
          const q = new URLSearchParams()
          q.set('deal_id', dealId)
          q.set('owner', owner)
          q.set('file_path', filePath)
          const url = `${normalizeBase(base)}/gateway/open-session/${encodeURIComponent(manifestRoot)}?${q.toString()}`
          const res = await fetch(url, {
            method: 'POST',
            headers: {
              ...(auth
                ? {
                    'X-Nil-Req-Sig': auth.reqSig,
                    'X-Nil-Req-Nonce': String(auth.reqNonce),
                    'X-Nil-Req-Expires-At': String(auth.reqExpiresAt),
                    'X-Nil-Req-Range-Start': String(auth.signedRangeStart),
                    'X-Nil-Req-Range-Len': String(auth.signedRangeLen),
                  }
                : {}),
            },
          })
          if (!res.ok) {
            const txt = await res.text().catch(() => '')
            throw new Error(decodeHttpError(txt) || `open-session failed (${res.status})`)
          }
          const json = (await res.json().catch(() => null)) as { download_session?: string } | null
          const id = String(json?.download_session || '').trim()
          if (!id) throw new Error('open-session returned an invalid download_session')
          return id
        }

        const openBases = serviceOverride
          ? [serviceOverride]
          : [
              ...(!appConfig.gatewayDisabled ? [appConfig.gatewayBase] : []),
              ...(directBase && directBase !== appConfig.gatewayBase ? [directBase] : []),
            ]

        let downloadSessionId: string | null = null
        let sessionPreference: RoutePreference | undefined = undefined
        for (const base of openBases) {
          try {
            downloadSessionId = await openDownloadSession(base, metaAuth)
            sessionPreference = base === appConfig.gatewayBase ? 'prefer_gateway' : 'prefer_direct_sp'
            break
          } catch (err) {
            if (!metaAuth && shouldSignMetaAuth(err)) {
              setProgress((p) => ({ ...p, message: 'Sign the download request to authorize retrieval' }))
              metaAuth = await signMetaAuth()
              downloadSessionId = await openDownloadSession(base, metaAuth)
              sessionPreference = base === appConfig.gatewayBase ? 'prefer_gateway' : 'prefer_direct_sp'
              break
            }
          }
        }
        if (!downloadSessionId) {
          throw new Error('failed to open download session')
        }

        const parts: Uint8Array[] = new Array(chunks.length)
        let bytesFetched = 0
        for (let i = 0; i < chunks.length; i++) {
          const c = chunks[i]
          const rangeResult = await transport.fetchRange({
            manifestRoot,
            owner,
            dealId,
            filePath,
            rangeStart: c.rangeStart,
            rangeLen: c.rangeLen,
            downloadSessionId,
            directBase,
            p2pTarget: planP2pTarget,
            preference: sessionPreference ?? preferenceOverride,
          })
          const buf = rangeResult.data.bytes
          parts[i] = buf
          bytesFetched += buf.byteLength
          setProgress((p) => ({
            ...p,
            phase: 'fetching',
            chunksFetched: Math.min(p.chunkCount || chunks.length, i + 1),
            bytesFetched: Math.min(p.bytesTotal || bytesFetched, bytesFetched),
          }))
        }

        const blob = new Blob(parts as BlobPart[], { type: 'application/octet-stream' })
        const url = URL.createObjectURL(blob)
        setDownloadUrl(url)
        setProgress((p) => ({
          ...p,
          phase: 'done',
          chunksFetched: chunks.length,
          bytesFetched,
        }))
        return { url, blob }
      }

      if (!ethereum || typeof ethereum.request !== 'function') {
        throw new Error('Ethereum provider (MetaMask) not available')
      }

      const parts: Uint8Array[] = new Array(chunks.length)
      let bytesFetched = 0
      let receiptsSubmitted = 0
      let chunksFetched = 0

      type PlannedChunk = {
        chunkIndex: number
        rangeStart: number
        rangeLen: number
        provider: string
        startMduIndex: bigint
        startBlobIndex: number
        blobCount: bigint
        planBackend: string
        planEndpoint?: string
      }

      const plannedChunks: PlannedChunk[] = []
      for (let chunkIndex = 0; chunkIndex < chunks.length; chunkIndex++) {
        const c = chunks[chunkIndex]
        const planResult = await transport.plan({
          manifestRoot,
          owner,
          dealId,
          filePath,
          rangeStart: c.rangeStart,
          rangeLen: c.rangeLen,
          directBase,
          p2pTarget: planP2pTarget,
          preference: preferenceOverride,
        })

        const planJson = planResult.data
        const provider = String(planJson.provider || '').trim()
        if (!provider) throw new Error('gateway plan did not return provider')
        const startMduIndex = BigInt(Number(planJson.start_mdu_index || 0))
        const startBlobIndex = Number(planJson.start_blob_index || 0)
        const blobCount = BigInt(Number(planJson.blob_count || 0))
        if (startMduIndex <= 0n) throw new Error('gateway plan did not return start_mdu_index')
        if (!Number.isFinite(startBlobIndex) || startBlobIndex < 0) throw new Error('gateway plan did not return start_blob_index')
        if (blobCount <= 0n) throw new Error('gateway plan did not return blob_count')

        plannedChunks.push({
          chunkIndex,
          rangeStart: c.rangeStart,
          rangeLen: c.rangeLen,
          provider,
          startMduIndex,
          startBlobIndex,
          blobCount,
          planBackend: planResult.backend,
          planEndpoint: planResult.trace?.chosen?.endpoint,
        })
      }

      const leafCount = BigInt(Math.max(1, Math.floor(mduSizeBytes / blobSizeBytes)))
      const providerGroups = new Map<string, {
        provider: string
        chunks: PlannedChunk[]
        globalStart: bigint
        globalEnd: bigint
      }>()

      for (const chunk of plannedChunks) {
        const globalStart = chunk.startMduIndex * leafCount + BigInt(chunk.startBlobIndex)
        const globalEnd = globalStart + chunk.blobCount - 1n
        const existing = providerGroups.get(chunk.provider)
        if (existing) {
          existing.chunks.push(chunk)
          if (globalStart < existing.globalStart) existing.globalStart = globalStart
          if (globalEnd > existing.globalEnd) existing.globalEnd = globalEnd
        } else {
          providerGroups.set(chunk.provider, {
            provider: chunk.provider,
            chunks: [chunk],
            globalStart,
            globalEnd,
          })
        }
      }

      setProgress((p) => ({
        ...p,
        phase: 'opening_session_tx',
        filePath,
        chunkCount: chunks.length,
        bytesTotal: effectiveRangeLen,
        receiptsSubmitted: 0,
        receiptsTotal: providerGroups.size > 0 ? 2 : 0,
      }))

      const groups = Array.from(providerGroups.values())
      const openBaseNonce = BigInt(Date.now())
      const openRequests = groups.map((group, index) => {
        const groupStartMdu = group.globalStart / leafCount
        const groupStartBlob = Number(group.globalStart % leafCount)
        const groupBlobCount = group.globalEnd - group.globalStart + 1n
        return {
          dealId: BigInt(dealId),
          provider: group.provider,
          manifestRoot,
          startMduIndex: groupStartMdu,
          startBlobIndex: groupStartBlob,
          blobCount: groupBlobCount,
          nonce: openBaseNonce + BigInt(index),
          expiresAt: 0n,
        }
      })

      const computeData = encodeComputeRetrievalSessionIdsData(openRequests)
      const computeResult = (await ethereum.request({
        method: 'eth_call',
        params: [{ from: address, to: appConfig.nilstorePrecompile, data: computeData }, 'latest'],
      })) as Hex
      const { providers: computedProviders, sessionIds: computedSessionIds } =
        decodeComputeRetrievalSessionIdsResult(computeResult)
      const sessionsByProvider = new Map<string, Hex>()
      for (let i = 0; i < computedProviders.length; i++) {
        const provider = String(computedProviders[i] || '').trim()
        const sessionId = computedSessionIds[i]
        if (!provider || !sessionId) continue
        sessionsByProvider.set(provider, sessionId)
      }
      for (const group of groups) {
        if (!sessionsByProvider.has(group.provider)) {
          throw new Error(`computeRetrievalSessionIds did not return session for ${group.provider}`)
        }
      }

      const openTxData = encodeOpenRetrievalSessionsData(openRequests)
      const openTxHash = (await ethereum.request({
        method: 'eth_sendTransaction',
        params: [{ from: address, to: appConfig.nilstorePrecompile, data: openTxData, gas: numberToHex(7_000_000) }],
      })) as Hex

      await waitForTransactionReceipt(openTxHash)

      receiptsSubmitted = 1
      setProgress((p) => ({
        ...p,
        phase: 'fetching',
        chunkCount: chunks.length,
        bytesTotal: effectiveRangeLen,
        receiptsSubmitted,
      }))

      let metaAuth:
        | {
            reqSig: string
            reqNonce: number
            reqExpiresAt: number
            signedRangeStart: number
            signedRangeLen: number
          }
        | undefined

      const shouldSignMetaAuth = (err: unknown): boolean => {
        if (!(err instanceof Error)) return false
        const msg = decodeHttpError(err.message)
        return /req_sig is required/i.test(msg) || /range must be signed/i.test(msg)
      }

      const signMetaAuth = async () => {
        const now = Math.floor(Date.now() / 1000)
        const reqNonce = Math.floor(Math.random() * 1_000_000_000) + Date.now()
        const reqExpiresAt = now + 9 * 60
        const typedData = buildRetrievalRequestTypedData(
          {
            deal_id: Number(dealId),
            file_path: filePath,
            range_start: wantRangeStart,
            range_len: effectiveRangeLen,
            nonce: reqNonce,
            expires_at: reqExpiresAt,
          },
          appConfig.chainId,
        )
        const reqSig = (await ethereum.request({
          method: 'eth_signTypedData_v4',
          params: [address, JSON.stringify(typedData)],
        })) as string

        metaAuth = {
          reqSig,
          reqNonce,
          reqExpiresAt,
          signedRangeStart: wantRangeStart,
          signedRangeLen: effectiveRangeLen,
        }
        return metaAuth
      }

      for (const group of groups) {
        const provider = group.provider
        const sessionId = sessionsByProvider.get(provider)
        if (!sessionId) {
          throw new Error(`missing session for provider ${provider}`)
        }

        const providerEndpoint = await getProviderEndpoint(provider)
        const providerP2pEndpoint = await getProviderP2pEndpoint(provider)
        const fetchP2pTarget =
          providerP2pEndpoint?.target ||
          (p2pEndpoint && p2pEndpoint.provider === provider ? p2pEndpoint.target : undefined) ||
          gatewayP2pTarget

        let fetchDirectBase =
          providerEndpoint?.baseUrl ||
          group.chunks.find((c) => c.planBackend === 'direct_sp')?.planEndpoint ||
          (serviceOverride && serviceOverride !== appConfig.gatewayBase ? serviceOverride : undefined) ||
          (directBase && directBase !== appConfig.gatewayBase ? directBase : undefined)
        if (!providerEndpoint && group.chunks.every((c) => c.planBackend !== 'direct_sp')) {
          fetchDirectBase = undefined
        }

        for (const c of group.chunks) {
          const fetchReq = {
            manifestRoot,
            owner,
            dealId,
            filePath,
            rangeStart: c.rangeStart,
            rangeLen: c.rangeLen,
            sessionId,
            expectedProvider: provider,
            directBase: fetchDirectBase,
            p2pTarget: fetchP2pTarget,
            preference: preferenceOverride,
          }

          let rangeResult: Awaited<ReturnType<typeof transport.fetchRange>>
          try {
            rangeResult = await transport.fetchRange({ ...fetchReq, auth: metaAuth })
          } catch (err) {
            if (!metaAuth && shouldSignMetaAuth(err)) {
              setProgress((p) => ({ ...p, message: 'Sign the download request to authorize retrieval' }))
              metaAuth = await signMetaAuth()
              rangeResult = await transport.fetchRange({ ...fetchReq, auth: metaAuth })
            } else {
              throw err
            }
          }

          const buf = rangeResult.data.bytes
          parts[c.chunkIndex] = buf
          bytesFetched += buf.byteLength
          chunksFetched += 1

          setProgress((p) => ({
            ...p,
            phase: 'fetching',
            chunksFetched: Math.min(p.chunkCount || chunks.length, chunksFetched),
            bytesFetched: Math.min(p.bytesTotal || bytesFetched, bytesFetched),
          }))
        }
      }

      setProgress((p) => ({
        ...p,
        phase: 'confirming_session_tx',
        receiptsSubmitted,
      }))

      const sessionIds = groups.map((group) => sessionsByProvider.get(group.provider) as Hex)
      const confirmTxData = encodeConfirmRetrievalSessionsData(sessionIds)
      const confirmTxHash = (await ethereum.request({
        method: 'eth_sendTransaction',
        params: [{ from: address, to: appConfig.nilstorePrecompile, data: confirmTxData, gas: numberToHex(3_000_000) }],
      })) as Hex
      await waitForTransactionReceipt(confirmTxHash)
      receiptsSubmitted = 2

      setProgress((p) => ({
        ...p,
        phase: 'submitting_proof_request',
        receiptsSubmitted,
      }))

      for (const group of groups) {
        const provider = group.provider
        const sessionId = sessionsByProvider.get(provider)
        if (!sessionId) {
          throw new Error(`missing session for provider ${provider}`)
        }
        // `session-proof` is an internal "user daemon -> provider" forward and requires gateway auth.
        // Even when `serviceBase` points at the provider (direct fetch flows), proof submission must go
        // through the local gateway.
        const proofBase = appConfig.gatewayBase
        const proofRes = await fetch(`${proofBase}/gateway/session-proof?deal_id=${encodeURIComponent(dealId)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ session_id: sessionId, provider }),
        })
        if (!proofRes.ok) {
          const text = await proofRes.text().catch(() => '')
          throw new Error(decodeHttpError(text) || `submit session proof failed (${proofRes.status})`)
        }
      }

      for (let i = 0; i < parts.length; i++) {
        if (!(parts[i] instanceof Uint8Array)) {
          throw new Error(`download failed (missing chunk ${i})`)
        }
      }

      const blob = new Blob(parts as BlobPart[], { type: 'application/octet-stream' })
      const url = URL.createObjectURL(blob)
      setDownloadUrl(url)

      setReceiptStatus('submitted')
      setProgress((p) => ({
        ...p,
        phase: 'done',
        receiptsSubmitted: receiptsSubmitted,
      }))

      return { url, blob }
    } catch (e) {
      console.error(e)
      setProgress((p) => ({ ...p, phase: 'error', message: (e as Error).message }))
      if (input.withReceipt) {
        setReceiptStatus('failed')
        setReceiptError((e as Error).message)
      } else {
        setReceiptStatus('idle')
        setReceiptError(null)
      }
      return null
    } finally {
      setLoading(false)
    }
  }

  return { fetchFile, loading, downloadUrl, receiptStatus, receiptError, progress }
}
