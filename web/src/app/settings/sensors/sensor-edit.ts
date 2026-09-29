import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { FormField, FormRoot, form, max, maxLength, min, required } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';

import { Api } from '../../core/api';
import { NodeState } from '../../core/models';
import { ago, duration } from '../../core/time';
import { Confirm } from '../../ui/confirm';
import { Secret } from '../../ui/secret';
import { Toast } from '../../ui/toast';
import { HOLD_FOR, INTERVALS, SLOTS, Thresholds, status, toRules, toThresholds, withCurrent } from './options';

@Component({
  selector: 'app-sensor-edit',
  imports: [FormField, FormRoot, RouterLink, Secret],
  templateUrl: './sensor-edit.html',
  styleUrl: './sensor-edit.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SensorEdit {
  /** Bound from the route via withComponentInputBinding(). */
  readonly slug = input.required<string>();

  private readonly api = inject(Api);
  private readonly confirm = inject(Confirm);
  private readonly toast = inject(Toast);
  private readonly router = inject(Router);

  protected readonly node = signal<NodeState | null>(null);
  protected readonly state = signal<'loading' | 'missing' | 'ready'>('loading');
  protected readonly token = signal<string | null>(null);
  protected readonly rotating = signal(false);

  protected readonly slots = SLOTS;
  protected readonly ago = ago;
  protected readonly status = computed(() => {
    const n = this.node();
    return n ? status(n) : null;
  });

  // --- general ---

  private readonly general = signal({ name: '', location: '', interval: '900', enabled: true });

  protected readonly intervals = computed(() =>
    withCurrent(INTERVALS, this.general().interval, v => `Every ${duration(Math.round(Number(v) / 60))}`),
  );

  protected readonly g = form(
    this.general,
    (p) => {
      required(p.name, { message: 'Give the sensor a name' });
      maxLength(p.name, 64, { message: 'At most 64 characters' });
    },
    {
      submission: {
        action: async (f) => {
          const v = f().value();
          try {
            const saved = await firstValueFrom(
              this.api.updateNode(this.slug(), {
                name: v.name.trim(),
                location: v.location.trim(),
                interval_s: Number(v.interval),
                enabled: v.enabled,
              }),
            );
            this.node.update(n => (n ? { ...n, ...saved } : n));
            f().reset(generalOf(saved));
          } catch (err) {
            return { kind: 'server', message: (err as Error).message };
          }
          this.toast.show('Saved');
          return undefined;
        },
      },
    },
  );

  // --- thresholds ---

  private readonly thresholds = signal<Thresholds>(toThresholds([]));

  protected readonly holdFor = computed(() => {
    const t = this.thresholds();
    const out: Record<string, { value: string; label: string }[]> = {};
    for (const s of SLOTS) {
      out[s.key] = withCurrent(HOLD_FOR, t[s.key].forMin, v => `For ${duration(Number(v))}`);
    }
    return out;
  });

  protected readonly t = form(
    this.thresholds,
    (p) => {
      for (const s of SLOTS) {
        if (s.min === undefined || s.max === undefined) {
          continue;
        }
        const unit = s.unit ?? '';
        required(p[s.key].threshold, { message: 'Enter a value' });
        min(p[s.key].threshold, s.min, { message: `At least ${s.min} ${unit}` });
        max(p[s.key].threshold, s.max, { message: `At most ${s.max} ${unit}` });
      }
    },
    {
      submission: {
        action: async (f) => {
          try {
            const saved = await firstValueFrom(this.api.saveRules(this.slug(), toRules(f().value())));
            f().reset(toThresholds(saved));
          } catch (err) {
            return { kind: 'server', message: (err as Error).message };
          }
          this.toast.show('Thresholds saved');
          return undefined;
        },
      },
    },
  );

  constructor() {
    queueMicrotask(() => this.load());
  }

  protected async rotate(): Promise<void> {
    const ok = await this.confirm.ask({
      title: 'Issue a new token?',
      message: `The current token stops working now. ${this.node()?.name} will not report until it is flashed with the new one.`,
      confirm: 'Issue token',
      danger: true,
    });
    if (!ok) {
      return;
    }
    this.rotating.set(true);
    this.api.rotateToken(this.slug()).subscribe({
      next: (t) => {
        this.token.set(t);
        this.rotating.set(false);
      },
      error: (err: Error) => {
        this.toast.error(err.message);
        this.rotating.set(false);
      },
    });
  }

  protected async remove(): Promise<void> {
    const name = this.node()?.name ?? this.slug();
    const ok = await this.confirm.ask({
      title: `Delete ${name}?`,
      message: 'Its whole history is deleted with it. This cannot be undone. To stop it for a while, turn it off instead.',
      confirm: 'Delete',
      danger: true,
    });
    if (!ok) {
      return;
    }
    this.api.deleteNode(this.slug()).subscribe({
      next: () => {
        this.toast.show(`${name} deleted`);
        void this.router.navigateByUrl('/settings/sensors');
      },
      error: (err: Error) => this.toast.error(err.message),
    });
  }

  private load(): void {
    this.api.nodes().subscribe({
      next: (list) => {
        const n = list.find(x => x.slug === this.slug());
        if (!n) {
          this.state.set('missing');
          return;
        }
        this.node.set(n);
        this.g().reset(generalOf(n));
        this.state.set('ready');
      },
      error: (err: Error) => {
        this.toast.error(err.message);
        this.state.set('missing');
      },
    });
    this.api.rules(this.slug()).subscribe({
      next: (rules) => this.t().reset(toThresholds(rules)),
      error: (err: Error) => this.toast.error(err.message),
    });
  }
}

function generalOf(n: { name: string; location?: string; interval_s: number; enabled: boolean }) {
  return { name: n.name, location: n.location ?? '', interval: String(n.interval_s), enabled: n.enabled };
}
