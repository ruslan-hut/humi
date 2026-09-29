import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

import { Auth } from './auth';

/** An API failure with the server's status message and HTTP status. */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
  }
}

/** Endpoints whose 401 means "wrong input", not "session gone". */
const NO_REDIRECT = ['/api/v1/auth/login', '/api/v1/auth/me', '/api/v1/auth/invites/', '/api/v1/account/password'];

/** Marks every request as ours (the server rejects state changes without it),
 *  turns error envelopes into ApiError and sends an expired session to sign-in. */
export const apiInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(Auth);
  const router = inject(Router);

  return next(req.clone({ setHeaders: { 'X-Requested-With': 'humi' } })).pipe(
    catchError((err: unknown) => {
      if (!(err instanceof HttpErrorResponse)) {
        return throwError(() => err);
      }
      if (err.status === 401 && !NO_REDIRECT.some(p => req.url.startsWith(p))) {
        auth.clear();
        void router.navigate(['/login'], { queryParams: { next: router.url } });
      }
      return throwError(() => new ApiError(messageOf(err), err.status));
    }),
  );
};

function messageOf(err: HttpErrorResponse): string {
  const body = err.error as { status_message?: string } | null;
  if (body && typeof body === 'object' && body.status_message) {
    return body.status_message;
  }
  if (err.status === 0) {
    return 'cannot reach the server';
  }
  return `server error ${err.status}`;
}
