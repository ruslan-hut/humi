import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { FormField, FormRoot, form, maxLength, minLength, required, validate } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';

import { Api } from '../../core/api';
import { Auth } from '../../core/auth';
import { Session } from '../../core/models';
import { ago } from '../../core/time';
import { Confirm } from '../../ui/confirm';
import { Toast } from '../../ui/toast';

@Component({
  selector: 'app-account',
  imports: [FormField, FormRoot, RouterLink],
  template: `
    <header class="page-head">
      <a class="icon-btn" routerLink="/settings" aria-label="Back to settings">&larr;</a>
      <h1>{{ auth.user()?.username }}</h1>
      <span class="chip" [class.admin]="auth.isAdmin()">{{ auth.isAdmin() ? 'Admin' : 'Viewer' }}</span>
    </header>

    <form class="panel" [formRoot]="pw">
      <h2>Change password</h2>
      <p class="lead">Your other devices will be signed out.</p>

      <input type="text" class="visually-hidden" autocomplete="username" [value]="auth.user()?.username ?? ''" readonly tabindex="-1" aria-hidden="true" />

      <div class="field">
        <label for="current">Current password</label>
        <input id="current" class="input" type="password" [formField]="pw.current" autocomplete="current-password" />
      </div>
      <div class="field">
        <label for="next">New password</label>
        <input id="next" class="input" type="password" [formField]="pw.next" autocomplete="new-password" />
        <span class="hint">At least 8 characters.</span>
        @if (pw.next().touched()) {
          @for (e of pw.next().errors(); track e.kind) {
            <span class="error">{{ e.message }}</span>
          }
        }
      </div>

      @for (e of pw().errors(); track e.kind) {
        <p class="form-error" role="alert">{{ e.message }}</p>
      }

      <div class="form-actions">
        <button type="submit" class="btn primary" [disabled]="pw().submitting()">Change password</button>
      </div>
    </form>

    <h2 class="section-label">Signed-in devices</h2>
    <ul class="list">
      @for (s of sessions(); track s.id) {
        <li class="row">
          <span class="main">
            <span class="title">{{ device(s.user_agent) }}</span>
            <span class="sub">
              @if (s.current) { This device · } @else { Active {{ ago(s.last_used) }} · }
              signed in {{ ago(s.created_at) }}
            </span>
          </span>
          @if (!s.current) {
            <button type="button" class="btn quiet-danger" (click)="revoke(s)">Sign out</button>
          }
        </li>
      } @empty {
        <li class="row"><span class="sub">Loading…</span></li>
      }
    </ul>
  `,
  styles: `
    :host { display: block; max-width: 560px; margin: 0 auto; }
    .page-head .chip { margin-left: auto; }
    .visually-hidden { position: absolute; width: 1px; height: 1px; opacity: 0; pointer-events: none; }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Account {
  protected readonly auth = inject(Auth);
  private readonly api = inject(Api);
  private readonly confirm = inject(Confirm);
  private readonly toast = inject(Toast);

  protected readonly ago = ago;
  protected readonly sessions = signal<Session[]>([]);

  private readonly model = signal({ current: '', next: '' });

  protected readonly pw = form(
    this.model,
    (p) => {
      required(p.current, { message: 'Enter your current password' });
      required(p.next, { message: 'Pick a new password' });
      minLength(p.next, 8, { message: 'At least 8 characters' });
      maxLength(p.next, 72, { message: 'At most 72 characters' });
      validate(p.next, ({ value, valueOf }) =>
        value() && value() === valueOf(p.current) ? { kind: 'same', message: 'Pick a different password' } : undefined,
      );
    },
    {
      submission: {
        action: async (f) => {
          const { current, next } = f().value();
          try {
            await firstValueFrom(this.api.changePassword(current, next));
          } catch (err) {
            return { kind: 'server', message: (err as Error).message };
          }
          f().reset({ current: '', next: '' });
          this.toast.show('Password changed');
          this.load();
          return undefined;
        },
      },
    },
  );

  constructor() {
    this.load();
  }

  protected async revoke(s: Session): Promise<void> {
    const ok = await this.confirm.ask({
      title: 'Sign out this device?',
      message: `${this.device(s.user_agent)} will have to sign in again.`,
      confirm: 'Sign out',
      danger: true,
    });
    if (!ok) {
      return;
    }
    this.api.revokeSession(s.id).subscribe({
      next: () => {
        this.sessions.update(list => list.filter(x => x.id !== s.id));
        this.toast.show('Device signed out');
      },
      error: (err: Error) => this.toast.error(err.message),
    });
  }

  /** A readable device name from a user agent, good enough to tell phones apart. */
  protected device(ua: string): string {
    const os =
      /iPhone/.test(ua) ? 'iPhone' :
      /iPad/.test(ua) ? 'iPad' :
      /Android/.test(ua) ? 'Android' :
      /Mac OS X/.test(ua) ? 'Mac' :
      /Windows/.test(ua) ? 'Windows' :
      /Linux/.test(ua) ? 'Linux' : '';
    const browser =
      /Edg\//.test(ua) ? 'Edge' :
      /Firefox\//.test(ua) ? 'Firefox' :
      /Chrome\//.test(ua) ? 'Chrome' :
      /Safari\//.test(ua) ? 'Safari' : '';
    return [browser, os].filter(Boolean).join(' on ') || 'Unknown device';
  }

  private load(): void {
    this.api.sessions().subscribe({
      next: (list) => this.sessions.set(list),
      error: (err: Error) => this.toast.error(err.message),
    });
  }
}
