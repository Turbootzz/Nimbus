'use client'

import { useEffect, useState } from 'react'
import { api } from '@/lib/api'
import type { User } from '@/types'

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [stats, setStats] = useState<{ total: number; admins: number; users: number } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  const loadUsers = async () => {
    setLoading(true)
    setError(null)

    try {
      const [usersResponse, statsResponse] = await Promise.all([
        api.getAllUsers(),
        api.getUserStats(),
      ])

      if (usersResponse.error) {
        setError(usersResponse.error.message)
      } else if (usersResponse.data) {
        setUsers(usersResponse.data)
      }

      if (statsResponse.data) {
        setStats(statsResponse.data)
      }
    } catch (err) {
      setError('Failed to load users')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadUsers()
  }, [])

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
        // Refresh user list
        await loadUsers()
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
        // Refresh user list
        await loadUsers()
      }
    } catch (err) {
      alert('Failed to delete user')
      console.error(err)
    } finally {
      setActionLoading(null)
    }
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-text-secondary">Loading users...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-error/10 border-error/20 text-error rounded-lg border p-4">{error}</div>
      </div>
    )
  }

  return (
    <div className="p-6">
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
          <div className="text-text-secondary p-6 text-center">No users found</div>
        )}
      </div>
    </div>
  )
}
