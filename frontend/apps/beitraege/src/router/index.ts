import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

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
          name: 'dashboard',
          component: () => import('@/pages/DashboardPage.vue'),
        },
        {
          path: 'kinder',
          name: 'children',
          component: () => import('@/pages/ChildrenPage.vue'),
        },
        {
          path: 'kinder/:id',
          name: 'child-detail',
          component: () => import('@/pages/ChildDetailPage.vue'),
        },
        {
          path: 'eltern',
          name: 'parents',
          component: () => import('@/pages/ParentsPage.vue'),
        },
        {
          path: 'beitraege',
          name: 'fees',
          component: () => import('@/pages/FeesPage.vue'),
        },
        {
          path: 'import',
          name: 'import',
          component: () => import('@/pages/ImportPage.vue'),
        },
      ],
    },
  ],
});

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore();

  // Wait for auth initialization on first load
  if (!authStore.user && authStore.isAuthenticated) {
    await authStore.initialize();
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  if (to.name === 'login' && authStore.isAuthenticated) {
    next({ name: 'dashboard' });
    return;
  }

  next();
});

export default router;
