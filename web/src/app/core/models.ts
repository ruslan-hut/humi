export interface Reading {
  received_at: number;
  rh: number;
  temp?: number;
  vbat?: number;
  rssi?: number;
}

export interface NodeState {
  slug: string;
  name: string;
  location?: string;
  interval_s: number;
  enabled: boolean;
  created_at: number;
  last_seen?: number;
  online: boolean;
  last?: Reading;
}

export interface Bucket {
  t: number;
  n: number;
  rh_min: number;
  rh_avg: number;
  rh_max: number;
  t_min?: number;
  t_avg?: number;
  t_max?: number;
  vbat_min?: number;
}

export interface Series {
  slug: string;
  from: number;
  to: number;
  bucket_s: number;
  points: Bucket[];
}

export interface Envelope<T> {
  data?: T;
  success: boolean;
  status_message: string;
  timestamp: string;
}

/** Comfort bands for relative humidity, in percent.
 *  TODO(phase5): read these from the rules API once the engine lands, so the
 *  backend stays the single source of truth for thresholds. */
export type Band = 'dry' | 'normal' | 'humid' | 'damp';

export function rhBand(rh: number): Band {
  if (rh < 30) return 'dry';
  if (rh <= 60) return 'normal';
  if (rh <= 70) return 'humid';
  return 'damp';
}

export const BAND_LABEL: Record<Band, string> = {
  dry: 'Dry',
  normal: 'Normal',
  humid: 'Humid',
  damp: 'Damp',
};
