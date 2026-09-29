import { describe, expect, it } from 'vitest';

import { safeNext } from './login';

describe('safeNext', () => {
  it.each([
    [undefined, '/'],
    ['/n/bedroom', '/n/bedroom'],
    ['//evil.example', '/'],
    ['https://evil.example', '/'],
    ['settings', '/'],
  ])('%s → %s', (next, want) => {
    expect(safeNext(next)).toBe(want);
  });
});
