import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Api } from '../../core/api';
import { NodeState } from '../../core/models';
import { ago } from '../../core/time';
import { intervalLabel, status } from './options';

@Component({
  selector: 'app-sensor-list',
  imports: [RouterLink],
  template: `
    <header class="page-head">
      <a class="icon-btn" routerLink="/settings" aria-label="Back to settings">&larr;</a>
      <h1>Sensors</h1>
    </header>

    @if (error(); as message) {
      <p class="notice error" role="alert">{{ message }}</p>
    }

    @if (nodes(); as list) {
      @if (list.length) {
        <ul class="list">
          @for (n of list; track n.slug) {
            <li>
              <a class="row" [routerLink]="['/settings/sensors', n.slug]">
                <span class="dot" [attr.data-state]="status(n).key" aria-hidden="true"></span>
                <span class="main">
                  <span class="title">{{ n.name }}</span>
                  <span class="sub">
                    {{ status(n).label }}
                    @if (n.last_seen) { · {{ ago(n.last_seen) }} }
                    · {{ interval(n.interval_s) }}
                  </span>
                </span>
                <span class="chevron" aria-hidden="true">›</span>
              </a>
            </li>
          }
        </ul>
      } @else {
        <p class="notice">No sensors yet. Add one, then put its token into the firmware.</p>
      }
    } @else if (!error()) {
      <p class="notice">Loading…</p>
    }

    <a class="btn primary block" routerLink="/settings/sensors/new">Add sensor</a>
  `,
  styles: `
    :host { display: block; max-width: 560px; margin: 0 auto; }

    .dot {
      flex: none;
      width: 10px;
      height: 10px;
      border-radius: 50%;
      background: var(--ink-muted);

      &[data-state='online'] { background: var(--good); }
      &[data-state='offline'] { background: var(--critical); }
      &[data-state='new'] { background: var(--warning); }
      &[data-state='disabled'] { background: transparent; border: 2px solid var(--ink-muted); }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SensorList {
  private readonly api = inject(Api);

  protected readonly nodes = signal<NodeState[] | null>(null);
  protected readonly error = signal<string | null>(null);
  protected readonly ago = ago;
  protected readonly status = status;
  protected readonly interval = intervalLabel;

  constructor() {
    this.api.nodes().subscribe({
      next: (list) => this.nodes.set(list),
      error: (err: Error) => this.error.set(err.message),
    });
  }
}
