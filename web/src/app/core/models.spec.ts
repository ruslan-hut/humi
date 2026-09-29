import { describe, expect, it } from 'vitest';

import { rhBand } from './models';

describe('rhBand', () => {
  const node = { rh_low: 30, rh_high: 65 };

  it.each([
    [29.9, 'dry'],
    [30, 'normal'],
    [65, 'normal'],
    [65.1, 'humid'],
    [75, 'humid'],
    [75.1, 'damp'],
  ])('%s%% is %s', (rh, band) => {
    expect(rhBand(rh, node)).toBe(band);
  });

  it('has no side without an enabled rule', () => {
    expect(rhBand(5, {})).toBe('normal');
    expect(rhBand(95, { rh_low: 30 })).toBe('normal');
  });
});
