import { Routes } from '@angular/router';

export const routes: Routes = [
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
  { path: '**', redirectTo: '' },
];
