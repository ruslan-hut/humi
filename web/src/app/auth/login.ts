import { ChangeDetectionStrategy, Component, inject, input, signal } from '@angular/core';
import { Router } from '@angular/router';
import { FormField, FormRoot, form, required } from '@angular/forms/signals';

import { Auth } from '../core/auth';

@Component({
  selector: 'app-login',
  imports: [FormField, FormRoot],
  template: `
    <h1>Sign in</h1>
    <p class="lead">Humidity and temperature, room by room.</p>

    <form class="panel" [formRoot]="f">
      <div class="field">
        <label for="username">Username</label>
        <input id="username" class="input" [formField]="f.username"
               autocomplete="username" autocapitalize="none" autocorrect="off" spellcheck="false" />
      </div>
      <div class="field">
        <label for="password">Password</label>
        <input id="password" class="input" type="password" [formField]="f.password"
               autocomplete="current-password" />
      </div>

      @for (e of f().errors(); track e.kind) {
        <p class="form-error" role="alert">{{ e.message }}</p>
      }

      <button type="submit" class="btn primary block" [disabled]="f().submitting()">
        {{ f().submitting() ? 'Signing in…' : 'Sign in' }}
      </button>
    </form>
  `,
  styleUrl: './auth.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Login {
  /** Where to go after signing in, from ?next=. */
  readonly next = input<string>();

  private readonly auth = inject(Auth);
  private readonly router = inject(Router);

  protected readonly f = form(
    signal({ username: '', password: '' }),
    (p) => {
      required(p.username, { message: 'Enter your username' });
      required(p.password, { message: 'Enter your password' });
    },
    {
      submission: {
        action: async (f) => {
          const { username, password } = f().value();
          try {
            await this.auth.login(username.trim(), password);
          } catch (err) {
            return { kind: 'server', message: (err as Error).message };
          }
          await this.router.navigateByUrl(safeNext(this.next()));
          return undefined;
        },
      },
    },
  );
}

/** Only same-app paths; never an absolute URL from the query string. */
export function safeNext(next: string | undefined): string {
  return next && next.startsWith('/') && !next.startsWith('//') ? next : '/';
}
