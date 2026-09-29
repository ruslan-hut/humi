import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { FormField, FormRoot, form, maxLength, pattern, required } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';

import { Api } from '../../core/api';
import { NodeCreated } from '../../core/models';
import { Secret } from '../../ui/secret';
import { INTERVALS, slugify } from './options';

@Component({
  selector: 'app-sensor-new',
  imports: [FormField, FormRoot, RouterLink, Secret],
  template: `
    <header class="page-head">
      <a class="icon-btn" routerLink="/settings/sensors" aria-label="Back to sensors">&larr;</a>
      <h1>{{ created() ? created()!.node.name : 'Add sensor' }}</h1>
    </header>

    @if (created(); as c) {
      <section class="panel">
        <h2>Sensor token</h2>
        <p class="lead">
          Put it into <code>firmware/node/secrets.h</code> as <code>HUMI_TOKEN</code> and flash the node.
          The sensor appears on the dashboard after its first report.
        </p>
        <app-secret [value]="c.token" [title]="c.node.name + ' token'" />
      </section>
      <a class="btn primary block" [routerLink]="['/settings/sensors', c.node.slug]">Done</a>
    } @else {
      <form class="panel" [formRoot]="f">
        <div class="field">
          <label for="name">Name</label>
          <input id="name" class="input" [formField]="f.name" placeholder="Bedroom" (input)="nameChanged()" autocomplete="off" />
          @if (f.name().touched()) {
            @for (e of f.name().errors(); track e.kind) { <span class="error">{{ e.message }}</span> }
          }
        </div>

        <div class="field">
          <label for="slug">Short id</label>
          <input id="slug" class="input" [formField]="f.slug" (input)="slugTouched = true"
                 autocapitalize="none" autocorrect="off" spellcheck="false" autocomplete="off" />
          <span class="hint">Used in links, e.g. /n/{{ f.slug().value() || 'bedroom' }}. Cannot be changed later.</span>
          @if (f.slug().touched()) {
            @for (e of f.slug().errors(); track e.kind) { <span class="error">{{ e.message }}</span> }
          }
        </div>

        <div class="field">
          <label for="location">Location <span class="optional">optional</span></label>
          <input id="location" class="input" [formField]="f.location" placeholder="2nd floor, by the window" autocomplete="off" />
        </div>

        <div class="field">
          <label for="interval">Reports</label>
          <select id="interval" class="input" [formField]="f.interval">
            @for (c of intervals; track c.value) { <option [value]="c.value">{{ c.label }}</option> }
          </select>
          <span class="hint">More often means a shorter battery life.</span>
        </div>

        @for (e of f().errors(); track e.kind) {
          <p class="form-error" role="alert">{{ e.message }}</p>
        }

        <div class="form-actions">
          <button type="submit" class="btn primary" [disabled]="f().submitting()">Add sensor</button>
        </div>
      </form>
    }
  `,
  styles: `
    :host { display: block; max-width: 560px; margin: 0 auto; }
    .optional { font-weight: 400; color: var(--ink-muted); }
    code { font-size: 0.8125rem; color: var(--ink); }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SensorNew {
  private readonly api = inject(Api);

  protected readonly intervals = INTERVALS;
  protected readonly created = signal<NodeCreated | null>(null);
  /** Once the id is typed by hand it stops following the name. */
  protected slugTouched = false;

  private readonly model = signal({ name: '', slug: '', location: '', interval: '900' });

  protected readonly f = form(
    this.model,
    (p) => {
      required(p.name, { message: 'Give the sensor a name' });
      maxLength(p.name, 64, { message: 'At most 64 characters' });
      required(p.slug, { message: 'Needed for links' });
      pattern(p.slug, /^[a-z0-9][a-z0-9-]{0,31}$/, {
        message: 'Lowercase letters, digits and dashes, up to 32',
      });
    },
    {
      submission: {
        action: async (f) => {
          const v = f().value();
          try {
            const c = await firstValueFrom(
              this.api.createNode({
                slug: v.slug,
                name: v.name.trim(),
                location: v.location.trim(),
                interval_s: Number(v.interval),
              }),
            );
            this.created.set(c);
          } catch (err) {
            return { kind: 'server', message: (err as Error).message };
          }
          return undefined;
        },
      },
    },
  );

  protected nameChanged(): void {
    if (!this.slugTouched) {
      this.model.update(m => ({ ...m, slug: slugify(m.name) }));
    }
  }
}
