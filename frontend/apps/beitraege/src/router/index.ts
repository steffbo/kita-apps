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
        { path: 'familie', name: 'family', component: () => import('@/pages/FamilyOverviewPage.vue'),
          meta: { requiresParent: true } },
        { path: 'familie/beitraege', name: 'family-fees',
          component: () => import('@/pages/FamilyFeesPage.vue'), meta: { requiresParent: true } },
        { path: 'familie/elternstunden', name: 'family-work',
          component: () => import('@/pages/FamilyWorkPage.vue'), meta: { requiresParent: true } },
        { path: 'familie/daten', name: 'family-data',
          component: () => import('@/pages/FamilyDataPage.vue'), meta: { requiresParent: true } },
        { path: 'familie/:pathMatch(.*)*', redirect: { name: 'family' },
          meta: { requiresParent: true } },
        { path: 'meldungen', name: 'reports', component: () => import('@/pages/ReportsPage.vue'),
          meta: { requiresAdmin: true } },
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/pages/DashboardPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'kinder',
          name: 'children',
          component: () => import('@/pages/ChildrenPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'kinder/import',
          name: 'children-import',
          component: () => import('@/pages/ChildImportPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'kinder/:id',
          name: 'child-detail',
          component: () => import('@/pages/ChildDetailPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'notizen',
          name: 'notes',
          component: () => import('@/pages/NotesPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'eltern',
          name: 'parents',
          component: () => import('@/pages/ParentsPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'eltern/:id',
          name: 'parent-detail',
          component: () => import('@/pages/ParentDetailPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'mitglieder',
          name: 'members',
          component: () => import('@/pages/MembersPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'mitglieder/:id',
          name: 'member-detail',
          component: () => import('@/pages/MemberDetailPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'beitraege',
          name: 'fees',
          component: () => import('@/pages/FeesPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'bankabgleich',
          name: 'bankabgleich',
          alias: 'import',
          component: () => import('@/pages/ImportPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'automatisierung',
          name: 'automation',
          component: () => import('@/pages/AutomationPage.vue'),
          meta: { requiresFees: true, requiresAdmin: true },
        },
        {
          path: 'beitragsordnung',
          name: 'fee-schedules',
          component: () => import('@/pages/FeeSchedulesPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'benutzer',
          name: 'users',
          component: () => import('@/pages/UsersPage.vue'),
          meta: { requiresFees: true, requiresAdmin: true },
        },
        {
          path: 'einstufungen',
          name: 'einstufungen',
          component: () => import('@/pages/EinstufungenPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'einstufungen/:id',
          name: 'einstufung-detail',
          component: () => import('@/pages/EinstufungDetailPage.vue'),
          meta: { requiresFees: true },
        },
        {
          path: 'elternstunden',
          name: 'parent-work',
          component: () => import('@/pages/ParentWorkOverviewPage.vue'),
          meta: { requiresParentWork: true },
        },
        { path: 'elternstunden/familien/:id', name: 'parent-work-detail', component: () => import('@/pages/ParentWorkDetailPage.vue'), meta: { requiresParentWork: true } },
        { path: 'elternstunden/vorstand', name: 'parent-work-board', component: () => import('@/pages/ParentWorkBoardPage.vue'), meta: { requiresParentWork: true } },
        { path: 'elternstunden/regeln', name: 'parent-work-rules', component: () => import('@/pages/ParentWorkRulesPage.vue'), meta: { requiresParentWork: true } },
        { path: 'elternstunden/import', name: 'parent-work-import', component: () => import('@/pages/ParentWorkImportPage.vue'), meta: { requiresParentWork: true } },
      ],
    },
  ],
});

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore();

  // Restore the session from the refresh cookie on first navigation
  await authStore.initialize();

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } });
    return;
  }

  if (authStore.isAuthenticated && authStore.isParent && !to.meta.requiresParent && to.name !== 'login') {
    next({ name: 'family' });
    return;
  }
  if (to.meta.requiresParent && !authStore.isParent) {
    next({ name: authStore.isParentWork ? 'parent-work' : 'dashboard' });
    return;
  }

  if (to.meta.requiresFees && !authStore.canAccessFees) {
    next({ name: 'parent-work' });
    return;
  }

  if (to.meta.requiresAdmin && !authStore.isAdmin) {
    next({ name: 'dashboard' });
    return;
  }

  if (to.meta.requiresParentWork && !authStore.canAccessParentWork) {
    next({ name: 'dashboard' });
    return;
  }

  if (to.name === 'login' && authStore.isAuthenticated) {
    next({ name: authStore.isParent ? 'family' : authStore.isParentWork ? 'parent-work' : 'dashboard' });
    return;
  }

  next();
});

export default router;
