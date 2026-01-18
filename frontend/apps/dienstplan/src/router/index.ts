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
      path: '/password-reset',
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
          name: 'schedule',
          component: () => import('@/pages/SchedulePage.vue'),
        },
        {
          path: 'employees',
          name: 'employees',
          component: () => import('@/pages/EmployeesPage.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'groups',
          name: 'groups',
          component: () => import('@/pages/GroupsPage.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'special-days',
          name: 'special-days',
          component: () => import('@/pages/SpecialDaysPage.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'statistics',
          name: 'statistics',
          component: () => import('@/pages/StatisticsPage.vue'),
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
    next({ name: 'schedule' });
    return;
  }

  next();
});

export default router;
