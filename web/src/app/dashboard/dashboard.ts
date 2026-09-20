import { ChangeDetectionStrategy, Component, OnDestroy, inject, signal } from '@angular/core';

import { Api } from '../core/api';
import { NodeState } from '../core/models';
import { NodeCard } from './node-card/node-card';

const REFRESH_MS = 60_000;

@Component({
  selector: 'app-dashboard',
  imports: [NodeCard],
  templateUrl: './dashboard.html',
  styleUrl: './dashboard.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Dashboard implements OnDestroy {
  private readonly api = inject(Api);

  readonly nodes = signal<NodeState[]>([]);
  readonly error = signal<string | null>(null);
  readonly loading = signal(true);

  private readonly timer = setInterval(() => this.load(), REFRESH_MS);

  constructor() {
    this.load();
  }

  ngOnDestroy(): void {
    clearInterval(this.timer);
  }

  load(): void {
    this.api.nodes().subscribe({
      next: (nodes) => {
        this.nodes.set(nodes);
        this.error.set(null);
        this.loading.set(false);
      },
      error: (err: Error) => {
        this.error.set(err.message || 'cannot reach the server');
        this.loading.set(false);
      },
    });
  }
}
