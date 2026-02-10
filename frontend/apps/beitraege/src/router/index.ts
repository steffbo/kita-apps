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
          path: 'kinder/import',
          name: 'children-import',
          component: () => import('@/pages/ChildImportPage.vue'),
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
          path: 'eltern/:id',
          name: 'parent-detail',
          component: () => import('@/pages/ParentDetailPage.vue'),
        },
        {
          path: 'mitglieder',
          name: 'members',
          component: () => import('@/pages/MembersPage.vue'),
        },
        {
          path: 'mitglieder/:id',
          name: 'member-detail',
          component: () => import('@/pages/MemberDetailPage.vue'),
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
        {
          path: 'automatisierung',
          name: 'automation',
          component: () => import('@/pages/AutomationPage.vue'),
        },
        {
          path: 'einstufungen',
          name: 'einstufungen',
          component: () => import('@/pages/EinstufungenPage.vue'),
        },
        {
          path: 'einstufungen/:id',
          name: 'einstufung-detail',
          component: () => import('@/pages/EinstufungDetailPage.vue'),
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
