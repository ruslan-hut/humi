import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { Api } from './api';
import { User } from './models';

/** The signed-in user. The session itself is an HttpOnly cookie the page never
 *  sees; `/auth/me` is how we learn whether it is still valid. */
@Injectable({ providedIn: 'root' })
export class Auth {
  private readonly api = inject(Api);

  /** undefined until the first check has answered. */
  private readonly current = signal<User | null | undefined>(undefined);
  private pending: Promise<User | null> | null = null;

  readonly user = this.current.asReadonly();
  readonly isAdmin = computed(() => this.current()?.role === 'admin');

  /** Resolves the session once; later calls reuse the answer. */
  ready(): Promise<User | null> {
    const known = this.current();
    if (known !== undefined) {
      return Promise.resolve(known);
    }
    this.pending ??= firstValueFrom(this.api.me())
      .then(
        (u) => u,
        () => null,
      )
      .then((u) => {
        this.current.set(u);
        this.pending = null;
        return u;
      });
    return this.pending;
  }

  async login(username: string, password: string): Promise<User> {
    const u = await firstValueFrom(this.api.login(username, password));
    this.current.set(u);
    return u;
  }

  /** For flows that sign in on the server's side, like accepting an invite. */
  signedIn(u: User): void {
    this.current.set(u);
  }

  async logout(): Promise<void> {
    try {
      await firstValueFrom(this.api.logout());
    } finally {
      this.clear();
    }
  }

  clear(): void {
    this.current.set(null);
  }
}
