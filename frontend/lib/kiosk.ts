// Kiosk mode (?kiosk=1 on a board page) hides everything around the tiles,
// for a wall screen. Pure, so ThemeScript can inline it before hydration.
export function isKioskSearch(search: string): boolean {
  return new URLSearchParams(search).get('kiosk') === '1'
}
