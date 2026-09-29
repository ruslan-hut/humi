import { DOCUMENT } from '@angular/common';
import { Injectable, inject, signal } from '@angular/core';
import { SwUpdate } from '@angular/service-worker';

/** The event Chrome fires when the page may be installed. Not in lib.dom yet. */
interface InstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

export type InstallMode = 'installed' | 'prompt' | 'ios' | 'none';

/** Service-worker updates and home-screen install. Instantiated by the app
 *  root at start-up, so the one-shot browser events are not missed. */
@Injectable({ providedIn: 'root' })
export class Pwa {
  private readonly sw = inject(SwUpdate);
  private readonly doc = inject(DOCUMENT);
  private readonly win = this.doc.defaultView!;

  /** A new version is downloaded and waits for a reload. */
  readonly updateReady = signal(false);
  readonly install = signal<InstallMode>(this.initialInstallMode());

  private deferred: InstallPromptEvent | null = null;

  constructor() {
    if (this.sw.isEnabled) {
      this.sw.versionUpdates.subscribe((e) => {
        if (e.type === 'VERSION_READY') {
          this.updateReady.set(true);
        }
      });
      // A broken cache (e.g. files evicted by the OS) cannot heal without a reload.
      this.sw.unrecoverable.subscribe(() => this.win.location.reload());
      // Home-screen apps stay alive for days; look for a deploy on every return.
      this.doc.addEventListener('visibilitychange', () => {
        if (this.doc.visibilityState === 'visible') {
          void this.sw.checkForUpdate().catch(() => undefined);
        }
      });
    }

    this.win.addEventListener('beforeinstallprompt', (e) => {
      e.preventDefault();
      this.deferred = e as InstallPromptEvent;
      this.install.set('prompt');
    });
    this.win.addEventListener('appinstalled', () => {
      this.deferred = null;
      this.install.set('installed');
    });
  }

  async reload(): Promise<void> {
    await this.sw.activateUpdate().catch(() => undefined);
    this.win.location.reload();
  }

  async promptInstall(): Promise<void> {
    const e = this.deferred;
    if (!e) {
      return;
    }
    await e.prompt();
    const { outcome } = await e.userChoice;
    this.deferred = null;
    this.install.set(outcome === 'accepted' ? 'installed' : 'none');
  }

  private initialInstallMode(): InstallMode {
    const nav = this.win.navigator as Navigator & { standalone?: boolean };
    if (this.win.matchMedia('(display-mode: standalone)').matches || nav.standalone) {
      return 'installed';
    }
    // iOS never fires beforeinstallprompt; Add to Home Screen is manual.
    const ios = /iPhone|iPad|iPod/.test(nav.userAgent) || (nav.platform === 'MacIntel' && nav.maxTouchPoints > 1);
    return ios ? 'ios' : 'none';
  }
}
