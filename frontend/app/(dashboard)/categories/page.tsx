'use client'

import { useState, useEffect } from 'react'
import { api } from '@/lib/api'
import { Category, CategoryCreateRequest, CategoryUpdateRequest } from '@/types'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon } from '@heroicons/react/24/outline'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'

// Draggable Category Card Component
function DraggableCategoryCard({
  category,
  onEdit,
  onDelete,
}: {
  category: Category
  onEdit: (category: Category) => void
  onDelete: (categoryId: string) => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: category.id,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="bg-card border-card-border flex items-center justify-between rounded-lg border p-4"
    >
      <div className="flex flex-1 items-center gap-3" {...listeners} {...attributes}>
        <div className="cursor-grab active:cursor-grabbing">
          <svg
            className="h-5 w-5 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 8h16M4 16h16"
            />
          </svg>
        </div>
        <div
          className="h-4 w-4 flex-shrink-0 rounded-full"
          style={{ backgroundColor: category.color }}
        />
        <span className="text-text-primary font-medium">{category.name}</span>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={() => onEdit(category)}
          className="text-text-secondary hover:text-text-primary rounded-md p-2 transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
          aria-label="Edit category"
        >
          <PencilIcon className="h-5 w-5" />
        </button>
        <button
          onClick={() => onDelete(category.id)}
          className="text-text-secondary rounded-md p-2 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          aria-label="Delete category"
        >
          <TrashIcon className="h-5 w-5" />
        </button>
      </div>
    </div>
  )
}

// Category Form Modal
function CategoryFormModal({
  isOpen,
  onClose,
  category,
  onSave,
}: {
  isOpen: boolean
  onClose: () => void
  category: Category | null
  onSave: (data: CategoryCreateRequest | CategoryUpdateRequest) => Promise<void>
}) {
  const [name, setName] = useState('')
  const [color, setColor] = useState('#6366f1')
  const [error, setError] = useState('')

  useEffect(() => {
    if (category) {
      setName(category.name)
      setColor(category.color)
    } else {
      setName('')
      setColor('#6366f1')
    }
    setError('')
  }, [category, isOpen])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!name.trim()) {
      setError('Category name is required')
      return
    }

    if (!/^#[0-9A-Fa-f]{6}$/.test(color)) {
      setError('Invalid color format. Use hex format like #6366f1')
      return
    }

    onSave({ name: name.trim(), color })
    onClose()
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="bg-card border-card-border w-full max-w-md rounded-lg border p-6">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-text-primary text-xl font-bold">
            {category ? 'Edit Category' : 'Create Category'}
          </h2>
          <button
            onClick={onClose}
            className="text-text-secondary hover:text-text-primary transition-colors"
          >
            <XMarkIcon className="h-6 w-6" />
          </button>
        </div>

        {error && (
          <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="name" className="text-text-secondary mb-2 block text-sm font-medium">
              Category Name
            </label>
            <input
              type="text"
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="border-card-border bg-background text-text-primary w-full rounded-lg border px-4 py-2 focus:border-transparent focus:ring-2 focus:ring-blue-500"
              placeholder="e.g., Work, Personal"
              required
            />
          </div>

          <div>
            <label htmlFor="color" className="text-text-secondary mb-2 block text-sm font-medium">
              Color
            </label>
            <div className="flex items-center gap-3">
              <input
                type="color"
                id="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                className="h-10 w-16 cursor-pointer rounded"
              />
              <input
                type="text"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                className="border-card-border bg-background text-text-primary flex-1 rounded-lg border px-4 py-2 font-mono focus:border-transparent focus:ring-2 focus:ring-blue-500"
                pattern="^#[0-9A-Fa-f]{6}$"
                placeholder="#6366f1"
              />
            </div>
          </div>

          <div className="border-card-border flex justify-end gap-3 border-t pt-4">
            <button
              type="button"
              onClick={onClose}
              className="text-text-secondary hover:text-text-primary rounded-lg px-4 py-2 text-sm font-medium transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              {category ? 'Save Changes' : 'Create Category'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// Main Categories Page
export default function CategoriesPage() {
  const [categories, setCategories] = useState<Category[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingCategory, setEditingCategory] = useState<Category | null>(null)

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  useEffect(() => {
    fetchCategories()
  }, [])

  const fetchCategories = async () => {
    setIsLoading(true)
    setError('')

    try {
      const response = await api.getCategories()
      if (response.error) {
        setError(response.error.message)
      } else if (response.data) {
        setCategories(response.data)
      }
    } catch (error) {
      console.error('Failed to fetch categories:', error)
      setError('Failed to load categories. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event

    if (!over || active.id === over.id) {
      return
    }

    const oldIndex = categories.findIndex((cat) => cat.id === active.id)
    const newIndex = categories.findIndex((cat) => cat.id === over.id)

    const newCategories = arrayMove(categories, oldIndex, newIndex)
    setCategories(newCategories)

    // Update positions on server
    try {
      const positions = newCategories.map((cat, index) => ({
        id: cat.id,
        position: index,
      }))

      await api.reorderCategories({ categories: positions })
    } catch (error) {
      console.error('Failed to reorder categories:', error)
      // Revert on error
      fetchCategories()
    }
  }

  const handleCreate = async (data: CategoryCreateRequest | CategoryUpdateRequest) => {
    try {
      const response = await api.createCategory(data as CategoryCreateRequest)
      if (response.error) {
        setError(response.error.message)
      } else {
        fetchCategories()
      }
    } catch (error) {
      console.error('Failed to create category:', error)
      setError('Failed to create category. Please try again.')
    }
  }

  const handleUpdate = async (data: CategoryCreateRequest | CategoryUpdateRequest) => {
    if (!editingCategory) return

    try {
      const response = await api.updateCategory(editingCategory.id, data as CategoryUpdateRequest)
      if (response.error) {
        setError(response.error.message)
      } else {
        fetchCategories()
      }
    } catch (error) {
      console.error('Failed to update category:', error)
      setError('Failed to update category. Please try again.')
    }
  }

  const handleDelete = async (categoryId: string) => {
    if (
      !confirm(
        'Are you sure you want to delete this category? Services in this category will become uncategorized.'
      )
    ) {
      return
    }

    try {
      const response = await api.deleteCategory(categoryId)
      if (response.error) {
        setError(response.error.message)
      } else {
        fetchCategories()
      }
    } catch (error) {
      console.error('Failed to delete category:', error)
      setError('Failed to delete category. Please try again.')
    }
  }

  const handleEdit = (category: Category) => {
    setEditingCategory(category)
    setIsModalOpen(true)
  }

  const handleCloseModal = () => {
    setIsModalOpen(false)
    setEditingCategory(null)
  }

  if (isLoading) {
    return (
      <div className="flex min-h-96 items-center justify-center">
        <div className="text-center">
          <div className="border-primary mx-auto mb-4 h-12 w-12 animate-spin rounded-full border-t-2 border-b-2"></div>
          <p className="text-text-secondary">Loading categories...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-3xl">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-text-primary text-3xl font-bold">Categories</h1>
          <p className="text-text-secondary mt-1">Organize your services into categories</p>
        </div>
        <button
          onClick={() => {
            setEditingCategory(null)
            setIsModalOpen(true)
          }}
          className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
        >
          <PlusIcon className="h-5 w-5" />
          New Category
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {error}
        </div>
      )}

      {categories.length === 0 ? (
        <div className="bg-card border-card-border rounded-lg border p-12 text-center">
          <div className="text-text-muted mb-4">
            <svg
              className="mx-auto h-16 w-16"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
              />
            </svg>
          </div>
          <h3 className="text-text-primary mb-2 text-lg font-medium">No categories yet</h3>
          <p className="text-text-secondary mx-auto mb-6 max-w-sm">
            Create your first category to start organizing your services
          </p>
          <button
            onClick={() => {
              setEditingCategory(null)
              setIsModalOpen(true)
            }}
            className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
          >
            <PlusIcon className="h-5 w-5" />
            Create Category
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={categories.map((cat) => cat.id)}
              strategy={verticalListSortingStrategy}
            >
              {categories.map((category) => (
                <DraggableCategoryCard
                  key={category.id}
                  category={category}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                />
              ))}
            </SortableContext>
          </DndContext>
        </div>
      )}

      <CategoryFormModal
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        category={editingCategory}
        onSave={editingCategory ? handleUpdate : handleCreate}
      />
    </div>
  )
}
