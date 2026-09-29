export interface Reading {
  received_at: number;
  rh: number;
  temp?: number;
  vbat?: number;
  rssi?: number;
}

export interface Node {
  slug: string;
  name: string;
  location?: string;
  interval_s: number;
  enabled: boolean;
  created_at: number;
  last_seen?: number;
}

export interface NodeState extends Node {
  online: boolean;
  last?: Reading;
  /** Thresholds of the node's enabled humidity rules. */
  rh_low?: number;
  rh_high?: number;
}

export interface NodeCreated {
  node: Node;
  token: string;
}

export type Metric = 'rh' | 'temp' | 'vbat' | 'offline';
export type Op = 'gt' | 'lt';

export interface Rule {
  id?: number;
  metric: Metric;
  op: Op;
  threshold: number;
  for_min: number;
  enabled: boolean;
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

export type Role = 'admin' | 'viewer';

export interface User {
  id: number;
  username: string;
  role: Role;
  created_at: number;
}

export interface UserRow extends User {
  last_active?: number;
  sessions: number;
}

export interface Session {
  id: number;
  created_at: number;
  last_used: number;
  expires_at: number;
  user_agent: string;
  current: boolean;
}

export type InviteKind = 'join' | 'reset';

export interface Invite {
  id: number;
  kind: InviteKind;
  role?: Role;
  username?: string;
  created_by?: string;
  created_at: number;
  expires_at: number;
}

/** A freshly issued invite; the token is shown once. */
export interface IssuedInvite {
  id: number;
  token: string;
  kind: InviteKind;
  role?: Role;
  username?: string;
  expires_at: number;
}

/** What an invite link reveals before it is accepted. */
export interface InviteInfo {
  kind: InviteKind;
  role?: Role;
  username?: string;
  expires_at: number;
}

export interface Envelope<T> {
  data?: T;
  success: boolean;
  status_message: string;
  timestamp: string;
}

/** Comfort bands for relative humidity. The thresholds come from each node's
 *  rules; damp is a further step above the high one. */
export type Band = 'dry' | 'normal' | 'humid' | 'damp';

const DAMP_STEP = 10;

export function rhBand(rh: number, node: Pick<NodeState, 'rh_low' | 'rh_high'>): Band {
  if (node.rh_low !== undefined && rh < node.rh_low) return 'dry';
  if (node.rh_high === undefined || rh <= node.rh_high) return 'normal';
  if (rh <= node.rh_high + DAMP_STEP) return 'humid';
  return 'damp';
}

export const BAND_LABEL: Record<Band, string> = {
  dry: 'Dry',
  normal: 'Normal',
  humid: 'Humid',
  damp: 'Damp',
};
