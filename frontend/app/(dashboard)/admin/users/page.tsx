'use client'

import { useEffect, useState, useCallback, useRef } from 'react'
import { api } from '@/lib/api'
import type { User, UserFilterParams } from '@/types'

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [stats, setStats] = useState<{ total: number; admins: number; users: number } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  // Pagination & Filter state
  const [searchTerm, setSearchTerm] = useState('')
  const [roleFilter, setRoleFilter] = useState<'' | 'admin' | 'user'>('')
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [totalUsers, setTotalUsers] = useState(0)
  const [limit] = useState(20)

  // Track latest request to prevent race conditions
  const requestIdRef = useRef(0)

  const loadUsers = async (params?: UserFilterParams) => {
    // Increment request ID and capture it for this request
    requestIdRef.current += 1
    const currentRequestId = requestIdRef.current

    setLoading(true)
    setError(null)

    try {
      const [usersResponse, statsResponse] = await Promise.all([
        api.getAllUsers(params),
        api.getUserStats(),
      ])

      // Only update state if this is still the latest request
      if (currentRequestId !== requestIdRef.current) {
        return // Ignore outdated response
      }

      if (usersResponse.error) {
        setError(usersResponse.error.message)
      } else if (usersResponse.data) {
        setUsers(usersResponse.data.users)
        setTotalUsers(usersResponse.data.total)
        setTotalPages(usersResponse.data.total_pages)
        setCurrentPage(usersResponse.data.page)
      }

      if (statsResponse.data) {
        setStats(statsResponse.data)
      }
    } catch (err) {
      // Only update error if this is still the latest request
      if (currentRequestId === requestIdRef.current) {
        setError('Failed to load users')
        console.error(err)
      }
    } finally {
      // Only update loading if this is still the latest request
      if (currentRequestId === requestIdRef.current) {
        setLoading(false)
      }
    }
  }

  // Build current filter params (DRY helper)
  const getCurrentFilterParams = useCallback(
    (): UserFilterParams => ({
      page: currentPage,
      limit,
      search: searchTerm.trim() || undefined,
      role: roleFilter || undefined,
    }),
    [currentPage, limit, searchTerm, roleFilter]
  )

  // Debounced search - only trigger after user stops typing for 500ms
  useEffect(() => {
    const timer = setTimeout(() => {
      loadUsers(getCurrentFilterParams())
    }, 500) // 500ms delay

    return () => clearTimeout(timer)
  }, [searchTerm, getCurrentFilterParams])

  // Immediate load for page/role changes (no debounce needed)
  useEffect(() => {
    loadUsers(getCurrentFilterParams())
  }, [currentPage, roleFilter, getCurrentFilterParams])

  const handleRoleChange = async (userId: string, currentRole: string) => {
    const newRole = currentRole === 'admin' ? 'user' : 'admin'
    const confirmMessage = `Are you sure you want to change this user's role to ${newRole}?`

    if (!confirm(confirmMessage)) {
      return
    }

    setActionLoading(userId)
    try {
      const response = await api.updateUserRole(userId, newRole)

      if (response.error) {
        alert(`Error: ${response.error.message}`)
      } else {
        loadUsers(getCurrentFilterParams())
      }
    } catch (err) {
      alert('Failed to update user role')
      console.error(err)
    } finally {
      setActionLoading(null)
    }
  }

  const handleDeleteUser = async (userId: string, userName: string) => {
    const confirmMessage = `Are you sure you want to delete user "${userName}"? This action cannot be undone.`

    if (!confirm(confirmMessage)) {
      return
    }

    setActionLoading(userId)
    try {
      const response = await api.deleteUser(userId)

      if (response.error) {
        alert(`Error: ${response.error.message}`)
      } else {
        loadUsers(getCurrentFilterParams())
      }
    } catch (err) {
      alert('Failed to delete user')
      console.error(err)
    } finally {
      setActionLoading(null)
    }
  }

  const handleSearchChange = (value: string) => {
    setSearchTerm(value)
    setCurrentPage(1) // Reset to first page on search
  }

  const handleRoleFilterChange = (value: '' | 'admin' | 'user') => {
    setRoleFilter(value)
    setCurrentPage(1) // Reset to first page on filter change
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  const formatDateTime = (dateString: string | undefined) => {
    if (!dateString) return 'Never'

    const date = new Date(dateString)
    const now = new Date()
    const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000)

    // Less than 10 seconds
    if (diffInSeconds < 10) return 'Just now'

    // 10-59 seconds
    if (diffInSeconds < 60) return `${diffInSeconds}s ago`

    // 1-59 minutes
    const minutes = Math.floor(diffInSeconds / 60)
    if (minutes < 60) return `${minutes}m ago`

    // 1-23 hours
    const hours = Math.floor(diffInSeconds / 3600)
    if (hours < 24) return `${hours}h ago`

    // 1-6 days
    const days = Math.floor(diffInSeconds / 86400)
    if (days < 7) return `${days}d ago`

    // 1-4 weeks
    const weeks = Math.floor(days / 7)
    if (weeks < 4) return `${weeks}w ago`

    // Older than 4 weeks - show date
    return formatDate(dateString)
  }

  return (
    <div className="p-6">
      {/* Error Message */}
      {error && (
        <div className="mb-4">
          <div className="bg-error/10 border-error/20 text-error rounded-lg border p-4">
            {error}
          </div>
        </div>
      )}
      <div className="mb-6">
        <h1 className="text-text-primary text-3xl font-bold">User Management</h1>
        <p className="text-text-secondary mt-1">Manage user accounts and roles</p>
      </div>

      {/* Statistics Cards */}
      {stats && (
        <div className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
          <div className="bg-card border-card-border rounded-lg border p-4">
            <div className="text-text-secondary text-sm">Total Users</div>
            <div className="text-text-primary mt-1 text-3xl font-bold">{stats.total}</div>
          </div>
          <div className="bg-card border-card-border rounded-lg border p-4">
            <div className="text-text-secondary text-sm">Administrators</div>
            <div className="text-primary mt-1 text-3xl font-bold">{stats.admins}</div>
          </div>
          <div className="bg-card border-card-border rounded-lg border p-4">
            <div className="text-text-secondary text-sm">Regular Users</div>
            <div className="text-info mt-1 text-3xl font-bold">{stats.users}</div>
          </div>
        </div>
      )}

      {/* Search and Filter Bar */}
      <div className="bg-card border-card-border mb-4 rounded-lg border p-4">
        <div className="flex flex-col gap-4 md:flex-row md:items-center">
          {/* Search Input */}
          <div className="relative flex-1">
            <input
              type="text"
              placeholder="Search by name or email..."
              value={searchTerm}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="bg-background border-card-border text-text-primary placeholder:text-text-secondary focus:border-primary w-full rounded-lg border px-4 py-2 transition-colors focus:outline-none"
            />
            {loading && (
              <div className="absolute top-1/2 right-3 -translate-y-1/2">
                <div className="border-primary h-4 w-4 animate-spin rounded-full border-2 border-t-transparent"></div>
              </div>
            )}
          </div>

          {/* Role Filter */}
          <div className="w-full md:w-48">
            <select
              value={roleFilter}
              onChange={(e) => handleRoleFilterChange(e.target.value as '' | 'admin' | 'user')}
              className="bg-background border-card-border text-text-primary focus:border-primary w-full rounded-lg border px-4 py-2 transition-colors focus:outline-none"
            >
              <option value="">All Roles</option>
              <option value="admin">Admins Only</option>
              <option value="user">Users Only</option>
            </select>
          </div>

          {/* Results Count */}
          <div className="text-text-secondary text-sm whitespace-nowrap">
            {totalUsers} {totalUsers === 1 ? 'user' : 'users'} found
          </div>
        </div>
      </div>

      {/* Users Table */}
      <div className="bg-card border-card-border overflow-hidden rounded-lg border">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-background border-card-border border-b">
              <tr>
                <th className="text-text-secondary px-6 py-3 text-left text-xs font-medium tracking-wider uppercase">
                  User
                </th>
                <th className="text-text-secondary px-6 py-3 text-left text-xs font-medium tracking-wider uppercase">
                  Email
                </th>
                <th className="text-text-secondary px-6 py-3 text-left text-xs font-medium tracking-wider uppercase">
                  Role
                </th>
                <th className="text-text-secondary px-6 py-3 text-left text-xs font-medium tracking-wider uppercase">
                  Last Activity
                </th>
                <th className="text-text-secondary px-6 py-3 text-left text-xs font-medium tracking-wider uppercase">
                  Joined
                </th>
                <th className="text-text-secondary px-6 py-3 text-right text-xs font-medium tracking-wider uppercase">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-card-border divide-y">
              {users.map((user) => (
                <tr key={user.id} className="hover:bg-background transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-text-primary text-sm font-medium">{user.name}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-text-secondary text-sm">{user.email}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex rounded-full px-2 py-1 text-xs font-semibold ${
                        user.role === 'admin'
                          ? 'bg-primary/10 text-primary'
                          : 'bg-info/10 text-info'
                      }`}
                    >
                      {user.role}
                    </span>
                  </td>
                  <td className="text-text-secondary px-6 py-4 text-sm whitespace-nowrap">
                    {formatDateTime(user.last_activity_at)}
                  </td>
                  <td className="text-text-secondary px-6 py-4 text-sm whitespace-nowrap">
                    {formatDate(user.created_at)}
                  </td>
                  <td className="px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
                    <div className="flex justify-end gap-2">
                      <button
                        onClick={() => handleRoleChange(user.id, user.role)}
                        disabled={actionLoading === user.id}
                        className="bg-primary/10 text-primary hover:bg-primary/20 rounded px-3 py-1 transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {actionLoading === user.id
                          ? 'Loading...'
                          : user.role === 'admin'
                            ? 'Demote'
                            : 'Promote'}
                      </button>
                      <button
                        onClick={() => handleDeleteUser(user.id, user.name)}
                        disabled={actionLoading === user.id}
                        className="bg-error/10 text-error hover:bg-error/20 rounded px-3 py-1 transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {users.length === 0 && (
          <div className="text-text-secondary p-6 text-center">
            {searchTerm || roleFilter ? 'No users match your search criteria' : 'No users found'}
          </div>
        )}
      </div>

      {/* Pagination Controls */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="text-text-secondary text-sm">
            Page {currentPage} of {totalPages}
          </div>

          <div className="flex gap-2">
            <button
              onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
              disabled={currentPage === 1 || loading}
              className="bg-card border-card-border text-text-primary hover:bg-background rounded-lg border px-4 py-2 transition-colors disabled:cursor-not-allowed disabled:opacity-50"
            >
              Previous
            </button>

            {/* Page numbers */}
            <div className="flex gap-1">
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                let pageNum: number
                if (totalPages <= 5) {
                  pageNum = i + 1
                } else if (currentPage <= 3) {
                  pageNum = i + 1
                } else if (currentPage >= totalPages - 2) {
                  pageNum = totalPages - 4 + i
                } else {
                  pageNum = currentPage - 2 + i
                }

                return (
                  <button
                    key={pageNum}
                    onClick={() => setCurrentPage(pageNum)}
                    disabled={loading}
                    className={`rounded-lg border px-3 py-2 transition-colors ${
                      currentPage === pageNum
                        ? 'bg-primary border-primary text-white'
                        : 'bg-card border-card-border text-text-primary hover:bg-background'
                    } disabled:cursor-not-allowed disabled:opacity-50`}
                  >
                    {pageNum}
                  </button>
                )
              })}
            </div>

            <button
              onClick={() => setCurrentPage((prev) => Math.min(totalPages, prev + 1))}
              disabled={currentPage === totalPages || loading}
              className="bg-card border-card-border text-text-primary hover:bg-background rounded-lg border px-4 py-2 transition-colors disabled:cursor-not-allowed disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
