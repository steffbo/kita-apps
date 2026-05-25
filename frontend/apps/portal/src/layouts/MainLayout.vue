<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink, RouterView, useRouter } from 'vue-router';
import { LogOut, Sprout } from 'lucide-vue-next';
import { portalModules } from '@/lib/modules';
import { useAuthStore } from '@/stores/auth';

const authStore = useAuthStore();
const router = useRouter();

const visibleModules = computed(() =>
  portalModules.filter((module) => authStore.hasAnyRole(module.roles)),
);

async function logout() {
  await authStore.logout();
  router.push({ name: 'login' });
}
</script>

<template>
  <div class="min-h-[100dvh] bg-background">
    <div class="grid min-h-[100dvh] grid-cols-1 lg:grid-cols-[280px_1fr]">
      <aside class="border-b bg-surface lg:border-b-0 lg:border-r">
        <div class="flex h-full flex-col">
          <div class="flex items-center gap-3 border-b px-5 py-5">
            <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Sprout class="h-5 w-5" :stroke-width="1.8" />
            </div>
            <div>
              <p class="text-sm font-semibold leading-5">Kita Knirpsenstadt</p>
              <p class="text-xs text-muted-foreground">Portal</p>
            </div>
          </div>

          <nav class="grid gap-1 px-3 py-4">
            <RouterLink
              v-for="module in visibleModules"
              :key="module.id"
              :to="{ name: module.routeName }"
              class="group flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition hover:bg-muted hover:text-foreground active:translate-y-px"
              active-class="bg-muted text-foreground"
            >
              <component :is="module.icon" class="h-4 w-4" :stroke-width="1.8" />
              <span>{{ module.label }}</span>
              <span
                v-if="module.state === 'planned'"
                class="ml-auto rounded-md bg-amber-soft px-1.5 py-0.5 text-[11px] font-semibold text-amber-foreground"
              >
                später
              </span>
            </RouterLink>
          </nav>

          <div class="mt-auto border-t p-4">
            <div class="mb-3 min-w-0">
              <p class="truncate text-sm font-semibold">{{ authStore.displayName }}</p>
              <p class="truncate text-xs text-muted-foreground">{{ authStore.user?.email }}</p>
            </div>
            <button class="secondary-button w-full justify-between" type="button" @click="logout">
              <span>Abmelden</span>
              <LogOut class="h-4 w-4" :stroke-width="1.8" />
            </button>
          </div>
        </div>
      </aside>

      <main class="min-w-0">
        <RouterView />
      </main>
    </div>
  </div>
</template>
