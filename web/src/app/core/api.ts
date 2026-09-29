import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, map } from 'rxjs';

import {
  Envelope,
  Invite,
  InviteInfo,
  IssuedInvite,
  Node,
  NodeCreated,
  NodeState,
  Role,
  Rule,
  Series,
  Session,
  User,
  UserRow,
} from './models';

export interface NodeInput {
  slug: string;
  name: string;
  location: string;
  interval_s: number;
}

export type NodePatch = Partial<Pick<Node, 'name' | 'location' | 'interval_s' | 'enabled'>>;

@Injectable({ providedIn: 'root' })
export class Api {
  private readonly http = inject(HttpClient);
  private readonly base = '/api/v1';

  // --- nodes ---

  nodes(): Observable<NodeState[]> {
    return this.get<NodeState[]>('/nodes', []);
  }

  series(slug: string, from: number, to: number, bucketS: number): Observable<Series> {
    const params = { from, to, bucket: bucketS };
    return this.http
      .get<Envelope<Series>>(`${this.base}/nodes/${slug}/series`, { params })
      .pipe(map(unwrap<Series>({ slug, from, to, bucket_s: bucketS, points: [] })));
  }

  createNode(input: NodeInput): Observable<NodeCreated> {
    return this.send<NodeCreated>('POST', '/nodes', input);
  }

  updateNode(slug: string, patch: NodePatch): Observable<Node> {
    return this.send<Node>('PATCH', `/nodes/${slug}`, patch);
  }

  deleteNode(slug: string): Observable<void> {
    return this.send<void>('DELETE', `/nodes/${slug}`);
  }

  rotateToken(slug: string): Observable<string> {
    return this.send<{ token: string }>('POST', `/nodes/${slug}/token`).pipe(map(r => r.token));
  }

  rules(slug: string): Observable<Rule[]> {
    return this.get<Rule[]>(`/nodes/${slug}/rules`, []);
  }

  saveRules(slug: string, rules: Rule[]): Observable<Rule[]> {
    return this.send<Rule[]>('PUT', `/nodes/${slug}/rules`, rules);
  }

  // --- session ---

  me(): Observable<User> {
    return this.get<User>('/auth/me');
  }

  login(username: string, password: string): Observable<User> {
    return this.send<User>('POST', '/auth/login', { username, password });
  }

  logout(): Observable<void> {
    return this.send<void>('POST', '/auth/logout');
  }

  invite(token: string): Observable<InviteInfo> {
    return this.get<InviteInfo>(`/auth/invites/${token}`);
  }

  acceptInvite(token: string, body: { username?: string; password: string }): Observable<User> {
    return this.send<User>('POST', `/auth/invites/${token}`, body);
  }

  // --- own account ---

  changePassword(current: string, next: string): Observable<void> {
    return this.send<void>('PUT', '/account/password', { current, new: next });
  }

  sessions(): Observable<Session[]> {
    return this.get<Session[]>('/account/sessions', []);
  }

  revokeSession(id: number): Observable<void> {
    return this.send<void>('DELETE', `/account/sessions/${id}`);
  }

  // --- users and invites (admin) ---

  users(): Observable<UserRow[]> {
    return this.get<UserRow[]>('/users', []);
  }

  setRole(id: number, role: Role): Observable<User> {
    return this.send<User>('PATCH', `/users/${id}`, { role });
  }

  deleteUser(id: number): Observable<void> {
    return this.send<void>('DELETE', `/users/${id}`);
  }

  resetLink(id: number): Observable<IssuedInvite> {
    return this.send<IssuedInvite>('POST', `/users/${id}/reset`);
  }

  invites(): Observable<Invite[]> {
    return this.get<Invite[]>('/invites', []);
  }

  createInvite(role: Role): Observable<IssuedInvite> {
    return this.send<IssuedInvite>('POST', '/invites', { role });
  }

  revokeInvite(id: number): Observable<void> {
    return this.send<void>('DELETE', `/invites/${id}`);
  }

  private get<T>(path: string, fallback?: T): Observable<T> {
    return this.http.get<Envelope<T>>(this.base + path).pipe(map(unwrap<T>(fallback as T)));
  }

  private send<T>(method: string, path: string, body?: unknown): Observable<T> {
    return this.http
      .request<Envelope<T>>(method, this.base + path, { body })
      .pipe(map(unwrap<T>(undefined as T)));
  }
}

function unwrap<T>(fallback: T): (e: Envelope<T>) => T {
  return (e: Envelope<T>) => {
    if (!e.success) {
      throw new Error(e.status_message);
    }
    return e.data ?? fallback;
  };
}
