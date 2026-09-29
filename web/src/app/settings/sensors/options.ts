import { Metric, NodeState, Op, Rule } from '../../core/models';
import { duration } from '../../core/time';

export interface Choice {
  value: string;
  label: string;
}

/** Reporting intervals. Every wake costs battery, so the list stops at 1 min. */
export const INTERVALS: Choice[] = [60, 300, 600, 900, 1800, 3600].map(s => ({
  value: String(s),
  label: s === 900 ? 'Every 15 min (recommended)' : `Every ${duration(s / 60)}`,
}));

/** How long a threshold must hold before it counts. */
export const HOLD_FOR: Choice[] = [0, 15, 30, 60, 120, 180, 360, 720].map(m => ({
  value: String(m),
  label: m === 0 ? 'Immediately' : `For ${duration(m)}`,
}));

/** The choices, plus the current value when it was set some other way. */
export function withCurrent(choices: Choice[], value: string, label: (v: string) => string): Choice[] {
  return choices.some(c => c.value === value) ? choices : [...choices, { value, label: label(value) }];
}

export function intervalLabel(s: number): string {
  return `every ${duration(Math.round(s / 60))}`;
}

export type SlotKey = 'rh_gt' | 'rh_lt' | 'temp_gt' | 'temp_lt' | 'vbat_lt' | 'offline';

export interface Slot {
  key: SlotKey;
  metric: Metric;
  op: Op;
  title: string;
  /** "Above" / "Below"; absent for offline, which has no threshold. */
  word?: string;
  unit?: string;
  min?: number;
  max?: number;
  step?: number;
  threshold: number;
  forMin: number;
  enabled: boolean;
}

/** Defaults mirror the rules the server seeds for a new node. */
export const SLOTS: Slot[] = [
  { key: 'rh_gt', metric: 'rh', op: 'gt', title: 'Too humid', word: 'Above', unit: '%', min: 0, max: 100, step: 1, threshold: 65, forMin: 60, enabled: true },
  { key: 'rh_lt', metric: 'rh', op: 'lt', title: 'Too dry', word: 'Below', unit: '%', min: 0, max: 100, step: 1, threshold: 30, forMin: 60, enabled: true },
  { key: 'temp_gt', metric: 'temp', op: 'gt', title: 'Too warm', word: 'Above', unit: '°C', min: -50, max: 100, step: 0.5, threshold: 30, forMin: 30, enabled: false },
  { key: 'temp_lt', metric: 'temp', op: 'lt', title: 'Too cold', word: 'Below', unit: '°C', min: -50, max: 100, step: 0.5, threshold: 10, forMin: 30, enabled: false },
  { key: 'vbat_lt', metric: 'vbat', op: 'lt', title: 'Battery low', word: 'Below', unit: 'V', min: 2, max: 5, step: 0.05, threshold: 3.4, forMin: 180, enabled: true },
  { key: 'offline', metric: 'offline', op: 'gt', title: 'Offline', threshold: 0, forMin: 0, enabled: true },
];

export interface SlotValue {
  enabled: boolean;
  threshold: number;
  forMin: string;
}

export type Thresholds = Record<SlotKey, SlotValue>;

export function toThresholds(rules: Rule[]): Thresholds {
  const out = {} as Thresholds;
  for (const s of SLOTS) {
    const r = rules.find(x => x.metric === s.metric && x.op === s.op);
    out[s.key] = r
      ? { enabled: r.enabled, threshold: r.threshold, forMin: String(r.for_min) }
      : { enabled: s.enabled, threshold: s.threshold, forMin: String(s.forMin) };
  }
  return out;
}

export function toRules(t: Thresholds): Rule[] {
  return SLOTS.map(s => ({
    metric: s.metric,
    op: s.op,
    threshold: s.key === 'offline' ? 0 : t[s.key].threshold,
    for_min: s.key === 'offline' ? 0 : Number(t[s.key].forMin),
    enabled: t[s.key].enabled,
  }));
}

export type Status = { key: 'online' | 'offline' | 'disabled' | 'new'; label: string };

export function status(n: NodeState): Status {
  if (!n.enabled) return { key: 'disabled', label: 'Disabled' };
  if (!n.last_seen) return { key: 'new', label: 'Waiting for first reading' };
  return n.online ? { key: 'online', label: 'Online' } : { key: 'offline', label: 'Offline' };
}

const CYRILLIC: Record<string, string> = {
  а: 'a', б: 'b', в: 'v', г: 'g', д: 'd', е: 'e', ё: 'e', ж: 'zh', з: 'z', и: 'i', й: 'y',
  к: 'k', л: 'l', м: 'm', н: 'n', о: 'o', п: 'p', р: 'r', с: 's', т: 't', у: 'u', ф: 'f',
  х: 'h', ц: 'ts', ч: 'ch', ш: 'sh', щ: 'sch', ъ: '', ы: 'y', ь: '', э: 'e', ю: 'yu', я: 'ya',
  є: 'ye', і: 'i', ї: 'yi', ґ: 'g',
};

/** "Living room" → living-room, "Спальня" → spalnya. */
export function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/[а-яёєіїґ]/g, ch => CYRILLIC[ch] ?? '')
    .normalize('NFKD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 32);
}
