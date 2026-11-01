<script setup lang="ts">
import { computed, ref } from 'vue';
import { Loader2, UserPlus, UserCog } from 'lucide-vue-next';
import { api } from '@/api';
import AppSheet from '@/components/ui/AppSheet.vue';
import type { UserAccount, UserRole } from '@/api/types';

const MIN_LENGTH = 8;

const props = defineProps<{
  /** Account to edit; absent when creating a new one. */
  user?: UserAccount | null;
  /** True when editing the signed-in account (role and active flag are locked). */
  isSelf?: boolean;
}>();
const emit = defineEmits<{ close: []; saved: [] }>();

const isNew = computed(() => !props.user);
const email = ref(props.user?.email ?? '');
const firstName = ref(props.user?.firstName ?? '');
const lastName = ref(props.user?.lastName ?? '');
const role = ref<UserRole>(props.user?.role ?? 'PARENT_WORK');
const isActive = ref(props.user?.isActive ?? true);
const password = ref('');
const isSaving = ref(false);
const error = ref<string | null>(null);

async function submit() {
  error.value = null;
  if (isNew.value && password.value.length < MIN_LENGTH) {
    error.value = `Das Passwort muss mindestens ${MIN_LENGTH} Zeichen haben.`;
    return;
  }
  const data = {
    email: email.value.trim(),
    firstName: firstName.value.trim() || undefined,
    lastName: lastName.value.trim() || undefined,
    role: role.value,
    isActive: isActive.value,
  };
  isSaving.value = true;
  try {
    if (props.user) {
      await api.updateUser(props.user.id, data);
    } else {
      await api.createUser({ ...data, password: password.value });
    }
    emit('saved');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Speichern fehlgeschlagen';
  } finally {
    isSaving.value = false;
  }
}
</script>

<template>
  <AppSheet :title="isNew ? 'Benutzer anlegen' : 'Benutzer bearbeiten'" size="md" @close="emit('close')">
    <form id="user-form" class="space-y-4" @submit.prevent="submit">
      <div class="flex items-center gap-3 mb-6">
        <div class="p-2 bg-primary/10 rounded-lg">
          <UserPlus v-if="isNew" class="h-6 w-6 text-primary" />
          <UserCog v-else class="h-6 w-6 text-primary" />
        </div>
        <h2 class="text-xl font-semibold">{{ isNew ? 'Benutzer anlegen' : 'Benutzer bearbeiten' }}</h2>
      </div>

      <div class="space-y-4">
        <label class="block">
          <span class="text-sm font-medium text-foreground">E-Mail (Anmeldename) *</span>
          <input v-model="email" type="email" required autocomplete="off" class="mt-1 w-full input" />
        </label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="text-sm font-medium text-foreground">Vorname</span>
            <input v-model="firstName" type="text" class="mt-1 w-full input" />
          </label>
          <label class="block">
            <span class="text-sm font-medium text-foreground">Nachname</span>
            <input v-model="lastName" type="text" class="mt-1 w-full input" />
          </label>
        </div>
        <label class="block">
          <span class="text-sm font-medium text-foreground">Rolle</span>
          <select v-model="role" :disabled="isSelf" class="mt-1 w-full input disabled:bg-muted">
            <option value="ADMIN">Administrator</option>
            <option value="BOARD">Vorstand</option>
            <option value="PARENT_WORK">Elternstunden</option>
            <option value="LEITUNG">Leitung</option>
            <option value="PARENT">Eltern</option>
          </select>
          <span v-if="role === 'BOARD'" class="mt-1 block text-xs text-muted-foreground">
            Sieht alles, auch Beiträge, Einkommen, Einstufungen und Bankdaten, kann aber nichts ändern.
          </span>
          <span v-if="role === 'LEITUNG'" class="mt-1 block text-xs text-muted-foreground">
            Sieht und bearbeitet Kinder, Eltern, Haushalte, Mitglieder und Elternstunden, aber keine Beiträge, Einkommen, Einstufungen, Notizen oder Bankdaten.
          </span>
          <span v-if="role === 'PARENT'" class="mt-1 block text-xs text-muted-foreground">
            Das Konto wird über die E-Mail-Adresse mit dem Elternteil verknüpft.
          </span>
          <span class="mt-1 block text-xs text-muted-foreground">
            Administratoren dürfen zusätzlich Benutzer, Beitragsordnung, Erinnerungen und Bankabruf verwalten.
          </span>
        </label>
        <label class="flex items-center gap-2">
          <input v-model="isActive" type="checkbox" :disabled="isSelf" class="rounded" />
          <span class="text-sm text-foreground">Aktiv (darf sich anmelden)</span>
        </label>
        <p v-if="isSelf" class="text-xs text-muted-foreground">
          Das eigene Konto kann nicht deaktiviert oder herabgestuft werden.
        </p>
        <label v-if="isNew" class="block">
          <span class="text-sm font-medium text-foreground">Startpasswort (mind. {{ MIN_LENGTH }} Zeichen) *</span>
          <input v-model="password" type="text" required autocomplete="off" class="mt-1 w-full input font-mono" />
          <span class="mt-1 block text-xs text-muted-foreground">
            Bitte der Person mitteilen; sie kann es nach der Anmeldung selbst ändern.
          </span>
        </label>

        <div v-if="error" class="note note-danger" role="alert">
          {{ error }}
        </div>
      </div>

    </form>
    <template #footer>
        <button type="button" class="btn btn-secondary" @click="emit('close')">Abbrechen</button>
        <button type="submit" form="user-form" :disabled="isSaving" class="btn btn-primary">
          <Loader2 v-if="isSaving" class="h-4 w-4 animate-spin" />
          {{ isNew ? 'Anlegen' : 'Speichern' }}
        </button>
    </template>
  </AppSheet>
</template>
