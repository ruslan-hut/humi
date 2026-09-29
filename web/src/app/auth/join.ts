import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { FormField, FormRoot, form, maxLength, minLength, pattern, required, validate } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';

import { Api } from '../core/api';
import { Auth } from '../core/auth';
import { InviteInfo } from '../core/models';
import { until } from '../core/time';

/** Accepts an invite link: a new account (join) or a new password (reset). */
@Component({
  selector: 'app-join',
  imports: [FormField, FormRoot, RouterLink],
  template: `
    @if (state() === 'loading') {
      <p class="notice">Checking the link…</p>
    } @else if (state() === 'invalid') {
      <h1>Link not valid</h1>
      <p class="lead">This invite has expired, was already used, or was revoked. Ask an admin for a new one.</p>
      <a class="btn block" routerLink="/login">Go to sign in</a>
    } @else if (info(); as i) {
      @if (i.kind === 'join') {
        <h1>Join Humi</h1>
        <p class="lead">
          You are invited as <strong>{{ i.role === 'admin' ? 'an admin' : 'a viewer' }}</strong>.
          Pick a username and password. The link expires {{ expires() }}.
        </p>
      } @else {
        <h1>New password</h1>
        <p class="lead">
          Set a new password for <strong>{{ i.username }}</strong>. Every device signed in
          to this account will be signed out.
        </p>
      }

      <form class="panel" [formRoot]="f">
        @if (i.kind === 'join') {
          <div class="field">
            <label for="username">Username</label>
            <input id="username" class="input" [formField]="f.username"
                   autocomplete="username" autocapitalize="none" autocorrect="off" spellcheck="false" />
            <span class="hint">3–32 characters: letters, digits, dot, dash, underscore.</span>
            @if (f.username().touched()) {
              @for (e of f.username().errors(); track e.kind) {
                <span class="error">{{ e.message }}</span>
              }
            }
          </div>
        }
        <div class="field">
          <label for="password">Password</label>
          <input id="password" class="input" type="password" [formField]="f.password" autocomplete="new-password" />
          <span class="hint">At least 8 characters.</span>
          @if (f.password().touched()) {
            @for (e of f.password().errors(); track e.kind) {
              <span class="error">{{ e.message }}</span>
            }
          }
        </div>

        @for (e of f().errors(); track e.kind) {
          <p class="form-error" role="alert">{{ e.message }}</p>
        }

        <button type="submit" class="btn primary block" [disabled]="f().submitting()">
          {{ i.kind === 'join' ? 'Create account' : 'Set password' }}
        </button>
      </form>
    }
  `,
  styleUrl: './auth.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Join {
  readonly token = input.required<string>();

  private readonly api = inject(Api);
  private readonly auth = inject(Auth);
  private readonly router = inject(Router);

  protected readonly info = signal<InviteInfo | null>(null);
  protected readonly state = signal<'loading' | 'invalid' | 'ready'>('loading');
  protected readonly expires = computed(() => until(this.info()?.expires_at ?? 0));

  protected readonly f = form(
    signal({ username: '', password: '' }),
    (p) => {
      validate(p.username, ({ value }) =>
        this.info()?.kind === 'join' && !value() ? { kind: 'required', message: 'Pick a username' } : undefined,
      );
      minLength(p.username, 3, { message: 'At least 3 characters' });
      maxLength(p.username, 32, { message: 'At most 32 characters' });
      pattern(p.username, /^[A-Za-z0-9._-]*$/, { message: 'Letters, digits, dot, dash and underscore only' });
      required(p.password, { message: 'Pick a password' });
      minLength(p.password, 8, { message: 'At least 8 characters' });
      maxLength(p.password, 72, { message: 'At most 72 characters' });
    },
    {
      submission: {
        action: async (f) => {
          const { username, password } = f().value();
          const body = this.info()?.kind === 'join' ? { username: username.trim(), password } : { password };
          try {
            const user = await firstValueFrom(this.api.acceptInvite(this.token(), body));
            this.auth.signedIn(user);
          } catch (err) {
            return { kind: 'server', message: (err as Error).message };
          }
          await this.router.navigateByUrl('/');
          return undefined;
        },
      },
    },
  );

  constructor() {
    queueMicrotask(() =>
      this.api.invite(this.token()).subscribe({
        next: (i) => {
          this.info.set(i);
          this.state.set('ready');
        },
        error: () => this.state.set('invalid'),
      }),
    );
  }
}
