/** "5 min ago", "3 h ago", "2 d ago" for a unix time in seconds. */
export function ago(at: number | undefined, never = 'never'): string {
  if (!at) {
    return never;
  }
  const mins = Math.round((Date.now() / 1000 - at) / 60);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours} h ago`;
  return `${Math.round(hours / 24)} d ago`;
}

/** "in 6 d", "in 23 h" for a unix time in seconds. */
export function until(at: number): string {
  const mins = Math.max(0, Math.round((at - Date.now() / 1000) / 60));
  if (mins < 60) return `in ${mins} min`;
  const hours = Math.round(mins / 60);
  if (hours < 48) return `in ${hours} h`;
  return `in ${Math.round(hours / 24)} d`;
}

/** Minutes as the shortest readable duration: "30 min", "2 h", "1 d". */
export function duration(min: number): string {
  if (min === 0) return 'immediately';
  if (min < 60) return `${min} min`;
  if (min % 1440 === 0) return `${min / 1440} d`;
  if (min % 60 === 0) return `${min / 60} h`;
  return `${min} min`;
}
