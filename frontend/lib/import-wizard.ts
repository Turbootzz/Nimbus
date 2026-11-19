import type { ConvertedDocument, DocumentGroup, Group } from '@/types'

// For each group of a converted document (by ref): the ID of the existing
// group its tiles go into, or '' to create the group
export type GroupTargets = Record<string, string>

// A group that already exists under the same name is the obvious target
export function defaultTargets(doc: ConvertedDocument, existing: Group[]): GroupTargets {
  const targets: GroupTargets = {}
  for (const group of doc.groups) {
    if (!group.ref) continue
    targets[group.ref] = existing.find((g) => g.name === group.name)?.id ?? ''
  }
  return targets
}

// Whether a document group can be created next to the existing ones. The
// import matches groups by name, so a new group can't share one.
export function canCreate(group: DocumentGroup, existing: Group[]) {
  return !existing.some((g) => g.name === group.name)
}

// Returns the document with its groups mapped onto the chosen existing
// groups. The import (with match_groups) recognises an existing group by its
// name, and the default group by its default flag; document groups that go
// into the same group become one.
export function mapGroups(
  doc: ConvertedDocument,
  targets: GroupTargets,
  existing: Group[]
): ConvertedDocument {
  const groups: DocumentGroup[] = []
  // ref of a dropped group -> ref of the group that took its place
  const moved: Record<string, string> = {}
  const byTarget: Record<string, string> = {}

  for (const group of doc.groups) {
    const target = existing.find((g) => g.id === (group.ref ? targets[group.ref] : ''))
    if (!target || !group.ref) {
      groups.push(group)
      continue
    }
    if (byTarget[target.id]) {
      moved[group.ref] = byTarget[target.id]
      continue
    }
    byTarget[target.id] = group.ref
    // An undefined flag is left out of the JSON that is sent
    groups.push({ ...group, name: target.name, default: target.is_default || undefined })
  }

  const remap = <T extends { group_ref?: string }>(tile: T): T =>
    tile.group_ref && moved[tile.group_ref] ? { ...tile, group_ref: moved[tile.group_ref] } : tile
  return { ...doc, groups, services: doc.services.map(remap), widgets: doc.widgets.map(remap) }
}
