<script setup lang="ts">
import { KeyRound, LogOut, Monitor, Moon, Sun, Undo2 } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';
import { useTheme } from '@/composables/useTheme';
import { userRoleLabel } from '@/utils/userRole';

const emit = defineEmits<{ changePassword: []; stopImpersonation: []; logout: [] }>();
const auth = useAuthStore();
const { mode, setTheme } = useTheme();

const themes = [
  { value: 'system', label: 'System', icon: Monitor },
  { value: 'light', label: 'Hell', icon: Sun },
  { value: 'dark', label: 'Dunkel', icon: Moon },
] as const;

const item = 'flex h-11 w-full items-center gap-3 rounded-lg px-3 text-left text-sm hover:bg-accent';
</script>

<template>
  <div class="space-y-3">
    <div class="px-1">
      <p class="truncate font-medium">
        {{ [auth.user?.firstName, auth.user?.lastName].filter(Boolean).join(' ') || auth.user?.email }}
      </p>
      <p class="truncate text-sm text-muted-foreground">
        {{ auth.user?.email }} · {{ auth.user ? userRoleLabel(auth.user.role) : '' }}
      </p>
    </div>
    <div role="radiogroup" aria-label="Design" class="grid grid-cols-3 gap-1 rounded-lg bg-muted p-1">
      <button v-for="t in themes" :key="t.value" type="button" role="radio" :aria-checked="mode === t.value"
        class="flex h-9 items-center justify-center gap-1.5 rounded-md text-sm font-medium text-muted-foreground"
        :class="mode === t.value && 'bg-card text-foreground shadow-sm'"
        @click="setTheme(t.value)"><component :is="t.icon" class="h-4 w-4" />{{ t.label }}</button>
    </div>
    <div class="border-t pt-2">
      <button v-if="auth.isImpersonating" type="button" data-testid="stop-impersonation" :class="item"
        @click="emit('stopImpersonation')"><Undo2 class="h-4 w-4 text-muted-foreground" />Impersonation beenden</button>
      <button v-else type="button" :class="item" @click="emit('changePassword')">
        <KeyRound class="h-4 w-4 text-muted-foreground" />Passwort ändern</button>
      <button type="button" :class="item" @click="emit('logout')">
        <LogOut class="h-4 w-4 text-muted-foreground" />Abmelden</button>
    </div>
  </div>
</template>
