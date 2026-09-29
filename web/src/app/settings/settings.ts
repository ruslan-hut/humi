import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';

import { Auth } from '../core/auth';
import { Pwa } from '../core/pwa';
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

    @switch (pwa.install()) {
      @case ('prompt') {
        <ul class="list">
          <li>
            <button type="button" class="row" (click)="pwa.promptInstall()">
              <img class="app-icon" src="icons/icon-192.png" alt="" />
              <span class="main">
                <span class="title">Install Humi</span>
                <span class="sub">Opens full screen from the home screen</span>
              </span>
              <span class="chevron" aria-hidden="true">›</span>
            </button>
          </li>
        </ul>
      }
      @case ('ios') {
        <div class="list ios-hint">
          <img class="app-icon" src="icons/icon-192.png" alt="" />
          <p>
            <strong>Install Humi:</strong> in Safari tap Share
            <svg class="share" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-label="Share icon"><path d="M12 3v12M8 7l4-4 4 4M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-7"/></svg>
            then <em>Add to Home Screen</em>.
          </p>
        </div>
      }
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

    .app-icon { flex: none; width: 36px; height: 36px; border-radius: 9px; }

    .ios-hint {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 16px;

      p { margin: 0; font-size: 0.875rem; line-height: 1.45; color: var(--ink-secondary); }
      strong { color: var(--ink); font-weight: 600; }
      .share { width: 16px; height: 16px; vertical-align: -3px; color: var(--rh); }
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Settings {
  protected readonly auth = inject(Auth);
  protected readonly pwa = inject(Pwa);
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
