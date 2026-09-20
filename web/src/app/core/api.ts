import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, map } from 'rxjs';

import { Envelope, NodeState, Series } from './models';

@Injectable({ providedIn: 'root' })
export class Api {
  private readonly http = inject(HttpClient);
  private readonly base = '/api/v1';

  nodes(): Observable<NodeState[]> {
    return this.http
      .get<Envelope<NodeState[]>>(`${this.base}/nodes`)
      .pipe(map(unwrap<NodeState[]>([])));
  }

  series(slug: string, from: number, to: number, bucketS: number): Observable<Series> {
    const params = { from, to, bucket: bucketS };
    return this.http
      .get<Envelope<Series>>(`${this.base}/nodes/${slug}/series`, { params })
      .pipe(map(unwrap<Series>({ slug, from, to, bucket_s: bucketS, points: [] })));
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
