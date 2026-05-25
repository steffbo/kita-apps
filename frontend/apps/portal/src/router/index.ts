import { createRouter, createWebHistory } from 'vue-router';
import type { PortalRole } from '@/lib/modules';
import { useAuthStore } from '@/stores/auth';

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean;
    roles?: PortalRole[];
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/pages/LoginPage.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/passwort-vergessen',
      name: 'password-reset',
      component: () => import('@/pages/PasswordResetPage.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/pages/DashboardPage.vue'),
        },
        {
          path: 'elternstunden',
          name: 'parent-work',
          component: () => import('@/pages/ParentWorkPage.vue'),
          meta: { roles: ['ADMIN', 'LEITUNG', 'TEAM', 'PARENT', 'VORSTAND'] },
        },
        {
          path: 'abnahme',
          name: 'review-queue',
          component: () => import('@/pages/ReviewQueuePage.vue'),
          meta: { roles: ['ADMIN', 'LEITUNG', 'TEAM'] },
        },
        {
          path: 'admin',
          name: 'admin',
          component: () => import('@/pages/AdminPage.vue'),
          meta: { roles: ['ADMIN'] },
        },
      ],
    },
  ],
});

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore();

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  if (to.meta.roles && !authStore.hasAnyRole(to.meta.roles)) {
    next({ name: 'dashboard' });
    return;
  }

  if (to.name === 'login' && authStore.isAuthenticated) {
    next({ name: 'dashboard' });
    return;
  }

  next();
});

export default router;
