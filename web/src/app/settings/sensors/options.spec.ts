import { describe, expect, it } from 'vitest';

import { Rule } from '../../core/models';
import { SLOTS, slugify, toRules, toThresholds } from './options';

describe('slugify', () => {
  it.each([
    ['Bedroom', 'bedroom'],
    ['Living room', 'living-room'],
    ['  Kid’s room #2 ', 'kid-s-room-2'],
    ['Спальня', 'spalnya'],
    ['Детская комната', 'detskaya-komnata'],
    ['Café', 'cafe'],
  ])('%s → %s', (name, slug) => {
    expect(slugify(name)).toBe(slug);
  });

  it('caps the length at 32', () => {
    expect(slugify('a'.repeat(50))).toHaveLength(32);
  });
});

describe('thresholds', () => {
  it('fills missing rules with the defaults', () => {
    const t = toThresholds([]);
    expect(t.rh_gt).toEqual({ enabled: true, threshold: 65, forMin: '60' });
    expect(t.temp_lt.enabled).toBe(false);
  });

  it('round-trips rules', () => {
    const rules: Rule[] = [
      { metric: 'rh', op: 'gt', threshold: 70, for_min: 120, enabled: true },
      { metric: 'rh', op: 'lt', threshold: 25, for_min: 0, enabled: false },
      { metric: 'temp', op: 'gt', threshold: 28.5, for_min: 30, enabled: true },
      { metric: 'temp', op: 'lt', threshold: 12, for_min: 30, enabled: false },
      { metric: 'vbat', op: 'lt', threshold: 3.3, for_min: 180, enabled: true },
      { metric: 'offline', op: 'gt', threshold: 0, for_min: 0, enabled: false },
    ];
    expect(toRules(toThresholds(rules))).toEqual(rules);
  });

  it('sends one rule per slot', () => {
    expect(toRules(toThresholds([]))).toHaveLength(SLOTS.length);
  });
});
