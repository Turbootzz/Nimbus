'use client'

import { useState, useEffect, useCallback } from 'react'
import { api } from '@/lib/api'
import { ThemedInput } from '@/components/ui/ThemedInput'
import {
  DEFAULT_SCOPES,
  EXPIRY_OPTIONS,
  SCOPE_OPTIONS,
  expiryFromDays,
  isExpired,
  toggleScope,
} from '@/lib/api-tokens'
import type { ApiToken, ApiTokenScope } from '@/types'

export default function ApiTokensPage() {
  const [tokens, setTokens] = useState<ApiToken[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [scopes, setScopes] = useState<ApiTokenScope[]>(DEFAULT_SCOPES)
  const [expiryDays, setExpiryDays] = useState(0)
  const [creating, setCreating] = useState(false)
  const [createdToken, setCreatedToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [revokingId, setRevokingId] = useState<string | null>(null)

  const fetchTokens = useCallback(async () => {
    try {
      const response = await api.getApiTokens()
      if (response.data) {
        setTokens(response.data)
        setError(null)
      } else if (response.error) {
        setError(response.error.message)
      }
    } catch {
      setError('Failed to load API tokens')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchTokens()
  }, [fetchTokens])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || scopes.length === 0 || creating) return

    setCreating(true)
    setError(null)
    try {
      const response = await api.createApiToken({
        name: name.trim(),
        scopes,
        expires_at: expiryFromDays(expiryDays),
      })
      if (response.data) {
        setCreatedToken(response.data.token)
        setCopied(false)
        setTokens([response.data.api_token, ...tokens])
        setName('')
        setScopes(DEFAULT_SCOPES)
        setExpiryDays(0)
      } else if (response.error) {
        setError(response.error.message)
      }
    } catch {
      setError('Failed to create token')
    } finally {
      setCreating(false)
    }
  }

  const handleCopy = async () => {
    if (!createdToken) return
    try {
      await navigator.clipboard.writeText(createdToken)
      setCopied(true)
    } catch {
      // Clipboard unavailable (e.g. non-secure context); token stays visible to copy manually
    }
  }

  const handleRevoke = async (token: ApiToken) => {
    if (revokingId) return
    if (!confirm(`Revoke "${token.name}"? Clients using it will stop working immediately.`)) return
    setRevokingId(token.id)
    try {
      const response = await api.deleteApiToken(token.id)
      if (!response.error) {
        setTokens(tokens.filter((t) => t.id !== token.id))
      } else {
        setError(response.error.message)
      }
    } finally {
      setRevokingId(null)
    }
  }

  const formatDate = (date: string | null) => {
    if (!date) return 'Never'
    return new Date(date).toLocaleDateString()
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="max-w-4xl p-4 sm:p-6">
      <div className="wallpaper-panel mb-8">
        <h1 className="text-text-primary mb-2 text-2xl font-bold sm:text-3xl">API Tokens</h1>
        <p className="text-text-secondary text-sm sm:text-base">
          Create personal access tokens for programmatic access. Send them as{' '}
          <code className="bg-card border-card-border rounded border px-1 py-0.5 text-xs">
            Authorization: Bearer &lt;token&gt;
          </code>
        </p>
      </div>

      {error && (
        <div
          role="alert"
          className="mb-6 rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-500"
        >
          {error}
        </div>
      )}

      {createdToken && (
        <div
          role="status"
          className="border-primary/40 bg-primary/5 wallpaper-surface mb-6 rounded-lg border p-4"
        >
          <p className="text-text-primary mb-2 text-sm font-semibold">
            Token created. You won&apos;t be able to see this token again — copy it now.
          </p>
          <div className="flex items-center gap-2">
            <code className="bg-card border-card-border text-text-primary flex-1 overflow-x-auto rounded border px-3 py-2 font-mono text-sm break-all">
              {createdToken}
            </code>
            <button
              onClick={handleCopy}
              className="bg-primary hover:bg-primary/90 shrink-0 rounded-lg px-3 py-2 text-sm font-medium text-white transition-colors"
            >
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
        </div>
      )}

      <form
        onSubmit={handleCreate}
        className="glass-card border-card-border mb-8 rounded-lg border p-6"
      >
        <h2 className="text-text-primary mb-4 text-lg font-semibold">Create new token</h2>
        <div className="space-y-4">
          <ThemedInput
            type="text"
            placeholder="Token name (e.g. Hearth dashboard)"
            aria-label="Token name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={100}
            required
          />
          <fieldset>
            <legend className="text-text-primary mb-2 text-sm font-medium">Scopes</legend>
            <div className="grid gap-2 sm:grid-cols-2">
              {SCOPE_OPTIONS.map((option) => (
                <label
                  key={option.scope}
                  className="border-card-border flex cursor-pointer items-start gap-3 rounded-lg border p-3"
                >
                  <input
                    type="checkbox"
                    checked={scopes.includes(option.scope)}
                    onChange={(e) => setScopes(toggleScope(scopes, option.scope, e.target.checked))}
                    aria-describedby={`scope-${option.scope}-description`}
                    className="accent-primary mt-0.5 h-4 w-4 shrink-0"
                  />
                  <span className="min-w-0">
                    <span className="text-text-primary block text-sm font-medium">
                      {option.label}{' '}
                      <code className="text-text-muted text-xs font-normal">{option.scope}</code>
                    </span>
                    <span
                      id={`scope-${option.scope}-description`}
                      className="text-text-muted block text-xs"
                    >
                      {option.description}
                    </span>
                  </span>
                </label>
              ))}
            </div>
            {scopes.length === 0 && (
              <p className="mt-2 text-xs text-red-500">Select at least one scope.</p>
            )}
          </fieldset>
          <div>
            <label
              htmlFor="token-expiry"
              className="text-text-primary mb-2 block text-sm font-medium"
            >
              Expires
            </label>
            <select
              id="token-expiry"
              value={expiryDays}
              onChange={(e) => setExpiryDays(Number(e.target.value))}
              className="border-card-border focus:border-primary w-full rounded-lg border px-4 py-2 transition focus:ring-2 focus:outline-none sm:w-48"
              style={{
                backgroundColor: 'var(--color-background)',
                color: 'var(--color-text-primary)',
              }}
            >
              {EXPIRY_OPTIONS.map((option) => (
                <option key={option.days} value={option.days}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <button
            type="submit"
            disabled={creating || !name.trim() || scopes.length === 0}
            className="bg-primary hover:bg-primary/90 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            {creating ? 'Creating…' : 'Create Token'}
          </button>
        </div>
      </form>

      <div className="space-y-4">
        {tokens.length === 0 ? (
          <div className="glass-card border-card-border rounded-lg border p-8 text-center">
            <p className="text-text-secondary">No API tokens yet. Create one above.</p>
          </div>
        ) : (
          tokens.map((token) => (
            <div
              key={token.id}
              className="glass-card border-card-border flex items-center justify-between gap-4 rounded-lg border p-4"
            >
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-text-primary font-medium">{token.name}</span>
                  {isExpired(token) && (
                    <span className="rounded-full bg-red-500/10 px-2 py-0.5 text-xs font-medium text-red-500">
                      expired
                    </span>
                  )}
                  {token.scopes.map((scope) => (
                    <span
                      key={scope}
                      className="bg-primary/10 text-primary rounded-full px-2 py-0.5 font-mono text-xs"
                    >
                      {scope}
                    </span>
                  ))}
                </div>
                <p className="text-text-muted mt-1 font-mono text-xs">{token.token_prefix}…</p>
                <p className="text-text-secondary mt-1 text-xs">
                  Created {formatDate(token.created_at)} · Last used{' '}
                  {formatDate(token.last_used_at)} · Expires {formatDate(token.expires_at)}
                </p>
              </div>
              <button
                onClick={() => handleRevoke(token)}
                disabled={revokingId !== null}
                aria-label={`Revoke ${token.name}`}
                className="shrink-0 rounded-lg border border-red-500/30 px-3 py-1.5 text-sm font-medium text-red-500 transition-colors hover:bg-red-500/10 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {revokingId === token.id ? 'Revoking…' : 'Revoke'}
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
