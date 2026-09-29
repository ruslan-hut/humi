import { Routes } from '@angular/router';

import { adminGuard, authGuard, guestGuard } from './core/guards';

export const routes: Routes = [
  {
    path: 'login',
    canActivate: [guestGuard],
    loadComponent: () => import('./auth/login').then(m => m.Login),
    title: 'Sign in · Humi',
  },
  {
    path: 'join/:token',
    loadComponent: () => import('./auth/join').then(m => m.Join),
    title: 'Join · Humi',
  },
  {
    path: '',
    canActivate: [authGuard],
    children: [
      {
        path: '',
        loadComponent: () => import('./dashboard/dashboard').then(m => m.Dashboard),
        title: 'Humi',
      },
      {
        path: 'n/:slug',
        loadComponent: () => import('./node/node-detail').then(m => m.NodeDetail),
        title: 'Humi',
      },
      {
        path: 'settings',
        loadComponent: () => import('./settings/settings').then(m => m.Settings),
        title: 'Settings · Humi',
      },
      {
        path: 'settings/account',
        loadComponent: () => import('./settings/account/account').then(m => m.Account),
        title: 'Account · Humi',
      },
      {
        path: 'settings/sensors',
        canActivate: [adminGuard],
        loadComponent: () => import('./settings/sensors/sensor-list').then(m => m.SensorList),
        title: 'Sensors · Humi',
      },
      {
        path: 'settings/sensors/new',
        canActivate: [adminGuard],
        loadComponent: () => import('./settings/sensors/sensor-new').then(m => m.SensorNew),
        title: 'Add sensor · Humi',
      },
      {
        path: 'settings/sensors/:slug',
        canActivate: [adminGuard],
        loadComponent: () => import('./settings/sensors/sensor-edit').then(m => m.SensorEdit),
        title: 'Sensor · Humi',
      },
      {
        path: 'settings/users',
        canActivate: [adminGuard],
        loadComponent: () => import('./settings/users/users').then(m => m.Users),
        title: 'Access · Humi',
      },
    ],
  },
  { path: '**', redirectTo: '' },
];
