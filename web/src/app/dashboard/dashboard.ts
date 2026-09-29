import { ChangeDetectionStrategy, Component, OnDestroy, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Api } from '../core/api';
import { Auth } from '../core/auth';
import { NodeState } from '../core/models';
import { NodeCard } from './node-card/node-card';

const REFRESH_MS = 60_000;

@Component({
  selector: 'app-dashboard',
  imports: [NodeCard, RouterLink],
  templateUrl: './dashboard.html',
  styleUrl: './dashboard.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Dashboard implements OnDestroy {
  private readonly api = inject(Api);
  protected readonly auth = inject(Auth);

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
