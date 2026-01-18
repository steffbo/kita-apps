<script setup lang="ts">
import { ref, computed } from 'vue';
import { RouterLink, RouterView, useRoute } from 'vue-router';
import { useAuth } from '@kita/shared';
import {
  CalendarDays,
  Users,
  Layers,
  CalendarClock,
  BarChart3,
  LogOut,
  Menu,
  X,
} from 'lucide-vue-next';

const route = useRoute();
const { user, isAdmin, logout } = useAuth();

const isMobileMenuOpen = ref(false);

const navigation = computed(() => {
  const items = [
    { name: 'Dienstplan', to: '/', icon: CalendarDays },
  ];

  if (isAdmin.value) {
    items.push(
      { name: 'Mitarbeiter', to: '/employees', icon: Users },
      { name: 'Gruppen', to: '/groups', icon: Layers },
      { name: 'Besondere Tage', to: '/special-days', icon: CalendarClock },
      { name: 'Statistiken', to: '/statistics', icon: BarChart3 },
    );
  }

  return items;
});

function isActive(path: string) {
  if (path === '/') {
    return route.path === '/';
  }
  return route.path.startsWith(path);
}

function handleLogout() {
  logout();
}
</script>

<template>
  <div class="min-h-screen bg-stone-50">
    <!-- Mobile menu button -->
    <div class="lg:hidden fixed top-0 left-0 right-0 z-40 bg-white border-b border-stone-200 px-4 py-3 flex items-center justify-between">
      <span class="font-semibold text-stone-900">Dienstplan</span>
      <button
        @click="isMobileMenuOpen = !isMobileMenuOpen"
        class="p-2 rounded-md hover:bg-stone-100"
      >
        <Menu v-if="!isMobileMenuOpen" class="w-5 h-5" />
        <X v-else class="w-5 h-5" />
      </button>
    </div>

    <!-- Sidebar -->
    <aside
      :class="[
        'fixed inset-y-0 left-0 z-30 w-64 bg-white border-r border-stone-200 transform transition-transform duration-200 lg:translate-x-0',
        isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
      ]"
    >
      <div class="flex flex-col h-full">
        <!-- Logo -->
        <div class="px-6 py-5 border-b border-stone-200">
          <h1 class="text-xl font-bold text-stone-900">Knirpsenstadt</h1>
          <p class="text-sm text-stone-500">Dienstplan</p>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 px-3 py-4 space-y-1">
          <RouterLink
            v-for="item in navigation"
            :key="item.to"
            :to="item.to"
            :class="[
              'flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
              isActive(item.to)
                ? 'bg-green-50 text-green-700'
                : 'text-stone-600 hover:bg-stone-100 hover:text-stone-900'
            ]"
            @click="isMobileMenuOpen = false"
          >
            <component :is="item.icon" class="w-5 h-5" />
            {{ item.name }}
          </RouterLink>
        </nav>

        <!-- User info -->
        <div class="px-3 py-4 border-t border-stone-200">
          <div class="flex items-center gap-3 px-3 py-2">
            <div class="w-8 h-8 rounded-full bg-green-100 flex items-center justify-center">
              <span class="text-sm font-medium text-green-700">
                {{ user?.firstName?.[0] }}{{ user?.lastName?.[0] }}
              </span>
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-stone-900 truncate">
                {{ user?.firstName }} {{ user?.lastName }}
              </p>
              <p class="text-xs text-stone-500">
                {{ isAdmin ? 'Leitung' : 'Mitarbeiter' }}
              </p>
            </div>
            <button
              @click="handleLogout"
              class="p-2 rounded-md hover:bg-stone-100 text-stone-500 hover:text-stone-700"
              title="Abmelden"
            >
              <LogOut class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <main class="lg:pl-64 pt-14 lg:pt-0">
      <div class="p-6">
        <RouterView />
      </div>
    </main>

    <!-- Mobile overlay -->
    <div
      v-if="isMobileMenuOpen"
      class="fixed inset-0 bg-black/50 z-20 lg:hidden"
      @click="isMobileMenuOpen = false"
    />
  </div>
</template>
