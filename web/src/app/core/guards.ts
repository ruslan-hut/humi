import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { Auth } from './auth';

/** Any signed-in user. */
export const authGuard: CanActivateFn = async (_route, state) => {
  const router = inject(Router);
  const user = await inject(Auth).ready();
  return user ? true : router.createUrlTree(['/login'], { queryParams: { next: state.url } });
};

/** Admins only; a viewer who follows a settings link lands on the dashboard. */
export const adminGuard: CanActivateFn = async (_route, state) => {
  const router = inject(Router);
  const user = await inject(Auth).ready();
  if (!user) {
    return router.createUrlTree(['/login'], { queryParams: { next: state.url } });
  }
  return user.role === 'admin' ? true : router.createUrlTree(['/settings']);
};

/** The sign-in page is pointless with a live session. */
export const guestGuard: CanActivateFn = async () => {
  const router = inject(Router);
  const user = await inject(Auth).ready();
  return user ? router.createUrlTree(['/']) : true;
};
