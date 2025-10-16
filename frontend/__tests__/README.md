# Frontend Tests

Test structure and documentation for Nimbus frontend.

## Status: Pending React 19 Compatibility

**Jest dependencies temporarily removed** due to React 19 compatibility.

- Testing Library requires React 18
- Project uses React 19
- Tests will be enabled when Testing Library adds React 19 support

## Test Structure

- `middleware.test.ts` - Documents required test cases for JWT middleware
- Helper functions ready for when dependencies are available

## When Dependencies Are Ready

```bash
npm install --save-dev @testing-library/react@next @testing-library/jest-dom jest jest-environment-jsdom
npm test
```
