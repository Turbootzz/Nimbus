// Matches MaxCustomCSSBytes in the backend, which also checks the CSS itself
export const MAX_CUSTOM_CSS_BYTES = 20 * 1024

// Safe mode: a page opened with ?safe=1 leaves the user's custom CSS out
// until the next full load, so CSS that hides the page can be fixed. Read
// once, so it also holds while moving between pages.
export const safeMode =
  typeof window !== 'undefined' && new URLSearchParams(window.location.search).get('safe') === '1'
