<script setup lang="ts">
import { computed, ref } from 'vue';
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '@/stores/auth';
import {
  LayoutDashboard,
  Users,
  UserCircle,
  UserPlus,
  Receipt,
  Upload,
  Zap,
  LogOut,
  Menu,
  X,
  ChevronDown,
  ClipboardList,
} from 'lucide-vue-next';

const authStore = useAuthStore();
const route = useRoute();
const router = useRouter();
const mobileMenuOpen = ref(false);
const showUserMenu = ref(false);

const currentPath = computed(() => route.path);

const navigation = [
  { name: 'Dashboard', to: '/', icon: LayoutDashboard },
  { name: 'Kinder', to: '/kinder', icon: Users },
  { name: 'Eltern', to: '/eltern', icon: UserCircle },
  { name: 'Mitglieder', to: '/mitglieder', icon: UserPlus },
  { name: 'Beiträge', to: '/beitraege', icon: Receipt },
  { name: 'Einstufungen', to: '/einstufungen', icon: ClipboardList },
  { name: 'Import', to: '/import', icon: Upload },
  { name: 'Automatisierung', to: '/automatisierung', icon: Zap },
];

function isActive(path: string) {
  if (path === '/') {
    return currentPath.value === '/';
  }
  return currentPath.value.startsWith(path);
}

async function handleLogout() {
  await authStore.logout();
  router.push('/login');
}

function toggleUserMenu() {
  showUserMenu.value = !showUserMenu.value;
}
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Mobile menu button -->
    <div class="lg:hidden fixed top-0 left-0 right-0 z-50 bg-white border-b px-4 py-3 flex items-center justify-between">
      <span class="font-semibold text-lg text-primary">Beiträge</span>
      <button @click="mobileMenuOpen = !mobileMenuOpen" class="p-2 rounded-md hover:bg-gray-100">
        <Menu v-if="!mobileMenuOpen" class="h-6 w-6" />
        <X v-else class="h-6 w-6" />
      </button>
    </div>

    <!-- Mobile menu overlay -->
    <div
      v-if="mobileMenuOpen"
      class="lg:hidden fixed inset-0 z-40 bg-black/50"
      @click="mobileMenuOpen = false"
    />

    <!-- Sidebar -->
    <aside
      :class="[
        'fixed inset-y-0 left-0 z-50 w-64 bg-white border-r transform transition-transform duration-200 ease-in-out lg:translate-x-0',
        mobileMenuOpen ? 'translate-x-0' : '-translate-x-full',
      ]"
    >
      <div class="flex flex-col h-full">
        <!-- Logo -->
        <div class="h-16 flex items-center px-6 border-b">
          <span class="font-bold text-xl text-primary">Kita Knirpsenstadt</span>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 px-3 py-4 space-y-1">
          <RouterLink
            v-for="item in navigation"
            :key="item.to"
            :to="item.to"
            :class="[
              'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
              isActive(item.to)
                ? 'bg-primary text-white'
                : 'text-gray-700 hover:bg-gray-100',
            ]"
            @click="mobileMenuOpen = false"
          >
            <component :is="item.icon" class="h-5 w-5" />
            {{ item.name }}
          </RouterLink>
        </nav>

        <!-- User section -->
        <div class="border-t p-4">
          <div class="relative">
            <button
              @click="toggleUserMenu"
              class="w-full flex items-center gap-3 px-2 py-2 rounded-lg hover:bg-gray-100 transition-colors"
            >
              <div class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                <UserCircle class="h-6 w-6 text-primary" />
              </div>
              <div class="flex-1 min-w-0 text-left">
                <p class="text-sm font-medium truncate">
                  {{ authStore.user?.firstName || authStore.user?.email }}
                </p>
                <p class="text-xs text-gray-500 truncate">
                  {{ authStore.user?.role === 'ADMIN' ? 'Administrator' : 'Benutzer' }}
                </p>
              </div>
              <ChevronDown 
                class="w-4 h-4 text-gray-400 transition-transform"
                :class="{ 'rotate-180': showUserMenu }"
              />
            </button>

            <!-- User dropdown menu -->
            <Transition
              enter-active-class="transition ease-out duration-100"
              enter-from-class="transform opacity-0 scale-95"
              enter-to-class="transform opacity-100 scale-100"
              leave-active-class="transition ease-in duration-75"
              leave-from-class="transform opacity-100 scale-100"
              leave-to-class="transform opacity-0 scale-95"
            >
              <div
                v-if="showUserMenu"
                class="absolute bottom-full left-0 right-0 mb-1 bg-white rounded-lg shadow-lg border py-1"
              >
                <button
                  @click="handleLogout"
                  class="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-100"
                >
                  <LogOut class="h-4 w-4" />
                  Abmelden
                </button>
              </div>
            </Transition>
          </div>
        </div>
      </div>
    </aside>

    <!-- Click outside to close user menu -->
    <div
      v-if="showUserMenu"
      class="fixed inset-0 z-40"
      @click="showUserMenu = false"
    />

    <!-- Main content -->
    <main class="lg:pl-64 pt-14 lg:pt-0">
      <div class="p-6">
        <RouterView />
      </div>
    </main>

  </div>
</template>
