import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { DecimalPipe } from '@angular/common';
import { RouterLink } from '@angular/router';

import { Api } from '../core/api';
import { BAND_LABEL, Bucket, NodeState, Series, rhBand } from '../core/models';
import { ChartPoint, SeriesChart } from './series-chart/series-chart';

interface Range {
  key: string;
  label: string;
  spanS: number;
  bucketS: number;
}

const RANGES: Range[] = [
  { key: '24h', label: '24 h', spanS: 24 * 3600, bucketS: 900 },
  { key: '7d', label: '7 d', spanS: 7 * 24 * 3600, bucketS: 3600 },
  { key: '30d', label: '30 d', spanS: 30 * 24 * 3600, bucketS: 6 * 3600 },
];

@Component({
  selector: 'app-node-detail',
  imports: [RouterLink, DecimalPipe, SeriesChart],
  templateUrl: './node-detail.html',
  styleUrl: './node-detail.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NodeDetail {
  /** Bound from the route via withComponentInputBinding(). */
  readonly slug = input.required<string>();

  private readonly api = inject(Api);

  readonly ranges = RANGES;
  readonly range = signal<Range>(RANGES[0]);
  readonly node = signal<NodeState | null>(null);
  readonly series = signal<Series | null>(null);
  readonly error = signal<string | null>(null);
  readonly showTable = signal(false);

  readonly humidity = computed<ChartPoint[]>(() =>
    (this.series()?.points ?? []).map(toRH),
  );

  readonly temperature = computed<ChartPoint[]>(() =>
    (this.series()?.points ?? []).filter(hasTemp).map(toTemp),
  );

  readonly rows = computed<Bucket[]>(() => [...(this.series()?.points ?? [])].reverse());

  readonly band = computed(() => {
    const rh = this.node()?.last?.rh;
    return rh === undefined ? null : BAND_LABEL[rhBand(rh)];
  });

  constructor() {
    queueMicrotask(() => {
      this.load();
      this.api.nodes().subscribe({
        next: (nodes) => this.node.set(nodes.find(bySlug(this.slug())) ?? null),
        error: () => this.node.set(null),
      });
    });
  }

  select(range: Range): void {
    this.range.set(range);
    this.load();
  }

  toggleTable(): void {
    this.showTable.update(shown => !shown);
  }

  stamp(t: number): string {
    return new Date(t * 1000).toLocaleString(undefined, {
      day: '2-digit',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  private load(): void {
    const to = Math.floor(Date.now() / 1000);
    const r = this.range();
    this.api.series(this.slug(), to - r.spanS, to, r.bucketS).subscribe({
      next: (s) => {
        this.series.set(s);
        this.error.set(null);
      },
      error: (err: Error) => this.error.set(err.message || 'cannot load history'),
    });
  }
}

function bySlug(slug: string): (n: NodeState) => boolean {
  return (n: NodeState) => n.slug === slug;
}

function toRH(b: Bucket): ChartPoint {
  return { t: b.t, min: b.rh_min, avg: b.rh_avg, max: b.rh_max };
}

function hasTemp(b: Bucket): boolean {
  return b.t_avg !== undefined && b.t_avg !== null;
}

function toTemp(b: Bucket): ChartPoint {
  return { t: b.t, min: b.t_min as number, avg: b.t_avg as number, max: b.t_max as number };
}
