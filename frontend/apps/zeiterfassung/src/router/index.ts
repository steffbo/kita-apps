import { createRouter, createWebHistory } from 'vue-router';
import { useAuth } from '@kita/shared';

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
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'clock',
          component: () => import('@/pages/ClockPage.vue'),
        },
        {
          path: 'history',
          name: 'history',
          component: () => import('@/pages/HistoryPage.vue'),
        },
        {
          path: 'admin',
          name: 'admin',
          component: () => import('@/pages/AdminPage.vue'),
          meta: { requiresAdmin: true },
        },
      ],
    },
  ],
});

router.beforeEach((to, _from, next) => {
  const { isAuthenticated, isAdmin } = useAuth();

  if (to.meta.requiresAuth && !isAuthenticated.value) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  if (to.meta.requiresAdmin && !isAdmin.value) {
    next({ name: 'clock' });
    return;
  }

  next();
});

export default router;
