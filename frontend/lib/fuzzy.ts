// Fuzzy matching for the command palette: the query's characters must appear
// in order in the text. Consecutive characters, word starts and an early
// first match score higher.

const separator = /[\s\-_./:]/

// alignmentScore scores matching query from text position start on, taking
// each next character at its first occurrence
function alignmentScore(q: string, t: string, start: number): number | null {
  let score = 0
  let previous = -2
  let ti = start
  for (const char of q) {
    const found = t.indexOf(char, ti)
    if (found === -1) return null
    score += 1
    if (found === previous + 1) score += 2 // consecutive
    if (found === 0 || separator.test(t[found - 1])) score += 3 // word start
    previous = found
    ti = found + 1
  }
  return score
}

// fuzzyScore returns how well query matches text (always above 0), or null
// for no match. Every place the first character occurs is tried as a start,
// so "admin" finds the word in "Radarr Admin" rather than the a and d of
// "Radarr".
export function fuzzyScore(query: string, text: string): number | null {
  const q = query.toLowerCase()
  const t = text.toLowerCase()
  if (!q) return 0

  let best: number | null = null
  for (let start = t.indexOf(q[0]); start !== -1; start = t.indexOf(q[0], start + 1)) {
    const score = alignmentScore(q, t, start)
    if (score === null) break // later starts can't match either
    // A match near the start, and in a short text, beats a scattered one
    const total = score - start * 0.1 - t.length * 0.01
    if (best === null || total > best) best = total
  }
  // Kept positive, so a field's weight always raises it
  return best === null ? null : Math.max(best, 0.1)
}

export interface SearchField {
  text: string | undefined
  weight: number // the name counts more than the URL
}

// bestScore is the best weighted score over an item's fields, or null
export function bestScore(query: string, fields: SearchField[]): number | null {
  let best: number | null = null
  for (const field of fields) {
    if (!field.text) continue
    const score = fuzzyScore(query, field.text)
    if (score !== null && (best === null || score * field.weight > best)) {
      best = score * field.weight
    }
  }
  return best
}
