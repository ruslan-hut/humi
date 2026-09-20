import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  OnDestroy,
  computed,
  inject,
  input,
  signal,
} from '@angular/core';

export interface ChartPoint {
  t: number;
  min: number;
  avg: number;
  max: number;
}

interface Tick {
  v: number;
  pos: number;
  label: string;
  anchor: string;
}

interface Marker {
  x: number;
  y: number;
}

interface View {
  w: number;
  h: number;
  band: string;
  line: string;
  end: Marker | null;
  yTicks: Tick[];
  xTicks: Tick[];
  x0: number;
  x1: number;
  y0: number;
  y1: number;
}

interface Hover {
  x: number;
  y: number;
  point: ChartPoint;
  left: number;
  flip: boolean;
}

const PAD_L = 38;
const PAD_R = 14;
const PAD_T = 12;
const PAD_B = 24;

/** A single-series band chart: min/max as a wash, the average as a 2px line.
 *  One measure per chart - a second y-scale is never the answer. */
@Component({
  selector: 'app-series-chart',
  templateUrl: './series-chart.html',
  styleUrl: './series-chart.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '[style.--series]': 'color()',
  },
})
export class SeriesChart implements OnDestroy {
  readonly points = input.required<ChartPoint[]>();
  readonly label = input.required<string>();
  readonly color = input.required<string>();
  readonly unit = input('');
  readonly decimals = input(1);
  readonly height = input(190);

  private readonly host = inject(ElementRef<HTMLElement>);
  private readonly width = signal(320);
  private readonly hoverIndex = signal<number | null>(null);

  private readonly observer = new ResizeObserver((entries) => {
    const w = Math.round(entries[0].contentRect.width);
    if (w > 0) {
      this.width.set(w);
    }
  });

  constructor() {
    this.observer.observe(this.host.nativeElement);
  }

  ngOnDestroy(): void {
    this.observer.disconnect();
  }

  readonly view = computed<View>(() => {
    const pts = this.points();
    const w = this.width();
    const h = this.height();
    const x0 = PAD_L;
    const x1 = w - PAD_R;
    const y0 = h - PAD_B;
    const y1 = PAD_T;

    if (pts.length === 0 || x1 <= x0) {
      return { w, h, band: '', line: '', end: null, yTicks: [], xTicks: [], x0, x1, y0, y1 };
    }

    let lo = Infinity;
    let hi = -Infinity;
    for (const p of pts) {
      lo = Math.min(lo, p.min);
      hi = Math.max(hi, p.max);
    }
    const scale = niceDomain(lo, hi);

    const tFrom = pts[0].t;
    const tTo = pts[pts.length - 1].t;
    const span = Math.max(tTo - tFrom, 1);
    const sx = (t: number) => x0 + ((t - tFrom) / span) * (x1 - x0);
    const sy = (v: number) => y0 - ((v - scale.lo) / (scale.hi - scale.lo)) * (y0 - y1);

    const top: string[] = [];
    const bottom: string[] = [];
    const line: string[] = [];
    for (let i = 0; i < pts.length; i++) {
      const p = pts[i];
      const x = sx(p.t).toFixed(1);
      top.push(`${i === 0 ? 'M' : 'L'}${x},${sy(p.max).toFixed(1)}`);
      bottom.push(`L${x},${sy(p.min).toFixed(1)}`);
      line.push(`${i === 0 ? 'M' : 'L'}${x},${sy(p.avg).toFixed(1)}`);
    }
    bottom.reverse();

    const yTicks: Tick[] = [];
    for (let v = scale.lo; v <= scale.hi + 1e-9; v += scale.step) {
      yTicks.push({ v, pos: sy(v), label: trim(v), anchor: 'end' });
    }

    const last = pts[pts.length - 1];

    return {
      w,
      h,
      band: top.join('') + bottom.join('') + 'Z',
      line: line.join(''),
      end: { x: sx(last.t), y: sy(last.avg) },
      yTicks,
      xTicks: timeTicks(pts, sx, span),
      x0,
      x1,
      y0,
      y1,
    };
  });

  readonly hover = computed<Hover | null>(() => {
    const i = this.hoverIndex();
    const pts = this.points();
    const v = this.view();
    if (i === null || i < 0 || i >= pts.length || pts.length === 0) {
      return null;
    }
    const tFrom = pts[0].t;
    const span = Math.max(pts[pts.length - 1].t - tFrom, 1);
    const p = pts[i];
    const x = v.x0 + ((p.t - tFrom) / span) * (v.x1 - v.x0);
    const lo = Math.min(...pts.map(pointMin));
    const hi = Math.max(...pts.map(pointMax));
    const scale = niceDomain(lo, hi);
    const y = v.y0 - ((p.avg - scale.lo) / (scale.hi - scale.lo)) * (v.y0 - v.y1);
    const flip = x > v.w * 0.6;
    return { x, y, point: p, left: flip ? x - 12 : x + 12, flip };
  });

  onMove(event: PointerEvent): void {
    const pts = this.points();
    if (pts.length === 0) {
      return;
    }
    const v = this.view();
    const rect = (event.currentTarget as SVGElement).getBoundingClientRect();
    const ratio = (event.clientX - rect.left - v.x0) / Math.max(v.x1 - v.x0, 1);
    const i = Math.round(ratio * (pts.length - 1));
    this.hoverIndex.set(Math.max(0, Math.min(pts.length - 1, i)));
  }

  onLeave(): void {
    this.hoverIndex.set(null);
  }

  format(value: number): string {
    return value.toFixed(this.decimals());
  }

  stamp(t: number): string {
    return new Date(t * 1000).toLocaleString(undefined, {
      day: '2-digit',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    });
  }
}

function pointMin(p: ChartPoint): number {
  return p.min;
}

function pointMax(p: ChartPoint): number {
  return p.max;
}

function trim(v: number): string {
  return Number.isInteger(v) ? String(v) : v.toFixed(1);
}

function niceDomain(lo: number, hi: number): { lo: number; hi: number; step: number } {
  if (!isFinite(lo) || !isFinite(hi) || hi - lo < 1e-9) {
    lo = (isFinite(lo) ? lo : 0) - 1;
    hi = lo + 2;
  }
  const raw = (hi - lo) / 4;
  const mag = Math.pow(10, Math.floor(Math.log10(raw)));
  const norm = raw / mag;
  const step = (norm <= 1 ? 1 : norm <= 2 ? 2 : norm <= 5 ? 5 : 10) * mag;
  return { lo: Math.floor(lo / step) * step, hi: Math.ceil(hi / step) * step, step };
}

function timeTicks(pts: ChartPoint[], sx: (t: number) => number, span: number): Tick[] {
  const count = Math.min(4, pts.length);
  const withinDay = span <= 36 * 3600;
  const ticks: Tick[] = [];
  for (let k = 0; k < count; k++) {
    const p = pts[Math.round((k * (pts.length - 1)) / Math.max(count - 1, 1))];
    const d = new Date(p.t * 1000);
    ticks.push({
      v: p.t,
      pos: sx(p.t),
      anchor: k === 0 ? 'start' : k === count - 1 ? 'end' : 'middle',
      label: withinDay
        ? d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
        : d.toLocaleDateString(undefined, { day: '2-digit', month: 'short' }),
    });
  }
  return ticks;
}
