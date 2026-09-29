import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { Api } from '../../core/api';
import { Auth } from '../../core/auth';
import { Invite, IssuedInvite, Role, UserRow } from '../../core/models';
import { ago, until } from '../../core/time';
import { Confirm } from '../../ui/confirm';
import { Secret } from '../../ui/secret';
import { Toast } from '../../ui/toast';

const ROLES: { value: Role; label: string; hint: string }[] = [
  { value: 'viewer', label: 'Viewer', hint: 'Sees rooms and charts' },
  { value: 'admin', label: 'Admin', hint: 'Also changes sensors and people' },
];

@Component({
  selector: 'app-users',
  imports: [RouterLink, Secret],
  templateUrl: './users.html',
  styleUrl: './users.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Users {
  private readonly api = inject(Api);
  private readonly auth = inject(Auth);
  private readonly confirm = inject(Confirm);
  private readonly toast = inject(Toast);

  protected readonly roles = ROLES;
  protected readonly ago = ago;
  protected readonly until = until;

  protected readonly users = signal<UserRow[] | null>(null);
  protected readonly invites = signal<Invite[]>([]);
  protected readonly error = signal<string | null>(null);

  /** The row whose actions are open. */
  protected readonly open = signal<number | null>(null);
  /** A reset link just issued for the open row. */
  protected readonly resetLink = signal<{ userId: number; url: string } | null>(null);

  protected readonly inviteRole = signal<Role>('viewer');
  protected readonly issued = signal<IssuedInvite | null>(null);
  protected readonly busy = signal(false);

  protected readonly me = computed(() => this.auth.user()?.id);
  private readonly admins = computed(() => (this.users() ?? []).filter(u => u.role === 'admin').length);

  constructor() {
    this.load();
  }

  protected toggle(u: UserRow): void {
    this.open.update(id => (id === u.id ? null : u.id));
    this.resetLink.set(null);
  }

  /** The last admin cannot be demoted; the server enforces it too. */
  protected isLastAdmin(u: UserRow): boolean {
    return u.role === 'admin' && this.admins() === 1;
  }

  protected setRole(u: UserRow, role: Role): void {
    if (u.role === role) {
      return;
    }
    this.api.setRole(u.id, role).subscribe({
      next: (saved) => {
        this.users.update(list => list?.map(x => (x.id === u.id ? { ...x, role: saved.role } : x)) ?? null);
        this.toast.show(`${u.username} is now ${role === 'admin' ? 'an admin' : 'a viewer'}`);
      },
      error: (err: Error) => this.toast.error(err.message),
    });
  }

  protected async reset(u: UserRow): Promise<void> {
    const ok = await this.confirm.ask({
      title: `Reset ${u.username}'s password?`,
      message: 'You get a link to send them, valid for 24 hours. Their devices stay signed in until they use it.',
      confirm: 'Create link',
    });
    if (!ok) {
      return;
    }
    this.api.resetLink(u.id).subscribe({
      next: (inv) => {
        this.resetLink.set({ userId: u.id, url: joinUrl(inv.token) });
        this.loadInvites();
      },
      error: (err: Error) => this.toast.error(err.message),
    });
  }

  protected async remove(u: UserRow): Promise<void> {
    const ok = await this.confirm.ask({
      title: `Remove ${u.username}?`,
      message: 'They are signed out everywhere and can only come back with a new invite.',
      confirm: 'Remove',
      danger: true,
    });
    if (!ok) {
      return;
    }
    this.api.deleteUser(u.id).subscribe({
      next: () => {
        this.users.update(list => list?.filter(x => x.id !== u.id) ?? null);
        this.open.set(null);
        this.toast.show(`${u.username} removed`);
      },
      error: (err: Error) => this.toast.error(err.message),
    });
  }

  protected createInvite(): void {
    this.busy.set(true);
    this.api.createInvite(this.inviteRole()).subscribe({
      next: (inv) => {
        this.issued.set(inv);
        this.busy.set(false);
        this.loadInvites();
      },
      error: (err: Error) => {
        this.toast.error(err.message);
        this.busy.set(false);
      },
    });
  }

  protected inviteUrl(inv: IssuedInvite): string {
    return joinUrl(inv.token);
  }

  protected async revoke(inv: Invite): Promise<void> {
    const what = inv.kind === 'reset' ? `the reset link for ${inv.username}` : `this ${inv.role} invite`;
    const ok = await this.confirm.ask({
      title: 'Revoke link?',
      message: `Anyone holding ${what} will no longer be able to use it.`,
      confirm: 'Revoke',
      danger: true,
    });
    if (!ok) {
      return;
    }
    this.api.revokeInvite(inv.id).subscribe({
      next: () => {
        this.invites.update(list => list.filter(x => x.id !== inv.id));
        if (this.issued()?.id === inv.id) {
          this.issued.set(null);
        }
        this.toast.show('Link revoked');
      },
      error: (err: Error) => this.toast.error(err.message),
    });
  }

  private load(): void {
    this.api.users().subscribe({
      next: (list) => this.users.set(list),
      error: (err: Error) => this.error.set(err.message),
    });
    this.loadInvites();
  }

  private loadInvites(): void {
    this.api.invites().subscribe({
      next: (list) => this.invites.set(list),
      error: (err: Error) => this.toast.error(err.message),
    });
  }
}

function joinUrl(token: string): string {
  return `${location.origin}/join/${token}`;
}
