import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { Auth } from '../core/auth';
import { Confirm } from '../ui/confirm';

@Component({
  selector: 'app-settings',
  imports: [RouterLink],
  template: `
    <header class="page-head">
      <a class="icon-btn" routerLink="/" aria-label="Back to rooms">&larr;</a>
      <h1>Settings</h1>
    </header>

    @if (auth.isAdmin()) {
      <h2 class="section-label">Home</h2>
      <ul class="list">
        <li>
          <a class="row" routerLink="/settings/sensors">
            <span class="main">
              <span class="title">Sensors</span>
              <span class="sub">Names, reporting interval, thresholds, tokens</span>
            </span>
            <span class="chevron" aria-hidden="true">›</span>
          </a>
        </li>
        <li>
          <a class="row" routerLink="/settings/users">
            <span class="main">
              <span class="title">People &amp; access</span>
              <span class="sub">Who can see and change things, invites</span>
            </span>
            <span class="chevron" aria-hidden="true">›</span>
          </a>
        </li>
      </ul>
    }

    <h2 class="section-label">You</h2>
    <ul class="list">
      <li>
        <a class="row" routerLink="/settings/account">
          <span class="main">
            <span class="title">{{ auth.user()?.username }}</span>
            <span class="sub">{{ auth.isAdmin() ? 'Admin' : 'Viewer' }} · password, signed-in devices</span>
          </span>
          <span class="chevron" aria-hidden="true">›</span>
        </a>
      </li>
      <li>
        <button type="button" class="row signout" (click)="signOut()">
          <span class="main"><span class="title">Sign out</span></span>
        </button>
      </li>
    </ul>
  `,
  styles: `
    :host { display: block; max-width: 560px; margin: 0 auto; }
    .signout .title { color: var(--critical); }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Settings {
  protected readonly auth = inject(Auth);
  private readonly confirm = inject(Confirm);
  private readonly router = inject(Router);

  protected async signOut(): Promise<void> {
    const ok = await this.confirm.ask({
      title: 'Sign out?',
      message: 'You will need your username and password to get back in on this device.',
      confirm: 'Sign out',
    });
    if (!ok) {
      return;
    }
    await this.auth.logout().catch(() => undefined);
    await this.router.navigateByUrl('/login');
  }
}
