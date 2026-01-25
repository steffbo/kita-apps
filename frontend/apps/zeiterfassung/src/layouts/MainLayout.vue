<script setup lang="ts">
import { ref } from 'vue';
import { RouterLink, RouterView, useRoute } from 'vue-router';
import { useAuth, ChangePasswordModal } from '@kita/shared';
import {
  Clock,
  History,
  LogOut,
  Menu,
  X,
  KeyRound,
  ChevronDown,
} from 'lucide-vue-next';

const route = useRoute();
const { user, isAdmin, logout, changePassword } = useAuth();

const isMobileMenuOpen = ref(false);
const showUserMenu = ref(false);
const showChangePassword = ref(false);

const navigation = [
  { name: 'Stempeluhr', to: '/', icon: Clock },
  { name: 'Übersicht', to: '/history', icon: History },
];

function isActive(path: string) {
  if (path === '/') {
    return route.path === '/';
  }
  return route.path.startsWith(path);
}

function handleLogout() {
  logout();
}

function toggleUserMenu() {
  showUserMenu.value = !showUserMenu.value;
}

function openChangePassword() {
  showUserMenu.value = false;
  showChangePassword.value = true;
}
</script>

<template>
  <div class="min-h-screen bg-stone-50">
    <!-- Mobile menu button -->
    <div class="lg:hidden fixed top-0 left-0 right-0 z-40 bg-white border-b border-stone-200 px-4 py-3 flex items-center justify-between">
      <span class="font-semibold text-stone-900">Zeiterfassung</span>
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
          <p class="text-sm text-stone-500">Zeiterfassung</p>
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
          <div class="relative">
            <button
              @click="toggleUserMenu"
              class="w-full flex items-center gap-3 px-3 py-2 rounded-md hover:bg-stone-100 transition-colors"
            >
              <div class="w-8 h-8 rounded-full bg-green-100 flex items-center justify-center">
                <span class="text-sm font-medium text-green-700">
                  {{ user?.firstName?.[0] }}{{ user?.lastName?.[0] }}
                </span>
              </div>
              <div class="flex-1 min-w-0 text-left">
                <p class="text-sm font-medium text-stone-900 truncate">
                  {{ user?.firstName }} {{ user?.lastName }}
                </p>
                <p class="text-xs text-stone-500">
                  {{ isAdmin ? 'Leitung' : 'Mitarbeiter' }}
                </p>
              </div>
              <ChevronDown 
                class="w-4 h-4 text-stone-400 transition-transform"
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
                class="absolute bottom-full left-0 right-0 mb-1 bg-white rounded-md shadow-lg border border-stone-200 py-1"
              >
                <button
                  @click="openChangePassword"
                  class="w-full flex items-center gap-2 px-3 py-2 text-sm text-stone-700 hover:bg-stone-100"
                >
                  <KeyRound class="w-4 h-4" />
                  Passwort ändern
                </button>
                <button
                  @click="handleLogout"
                  class="w-full flex items-center gap-2 px-3 py-2 text-sm text-stone-700 hover:bg-stone-100"
                >
                  <LogOut class="w-4 h-4" />
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
      class="fixed inset-0 z-20"
      @click="showUserMenu = false"
    />

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

    <!-- Change Password Modal -->
    <ChangePasswordModal
      v-model:visible="showChangePassword"
      :change-password-fn="changePassword"
    />
  </div>
</template>
