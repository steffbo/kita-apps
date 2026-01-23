<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Member, UpdateMemberRequest } from '@/api/types';
import {
  ArrowLeft,
  Edit,
  Trash2,
  Loader2,
  User,
  Calendar,
  MapPin,
  X,
  Check,
  Mail,
  Phone,
  Hash,
} from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const member = ref<Member | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Edit dialog state
const showEditDialog = ref(false);
const editForm = ref<UpdateMemberRequest>({});
const isEditing = ref(false);
const editError = ref<string | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

const memberId = computed(() => route.params.id as string);

async function loadMember() {
  isLoading.value = true;
  error.value = null;
  try {
    member.value = await api.getMember(memberId.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadMember);

// ESC key handler to close all modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteDialog.value) {
      showDeleteDialog.value = false;
    } else if (showEditDialog.value) {
      showEditDialog.value = false;
    }
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatDateForInput(dateStr: string): string {
  return dateStr.split('T')[0];
}

function openEditDialog() {
  if (!member.value) return;
  editForm.value = {
    firstName: member.value.firstName,
    lastName: member.value.lastName,
    email: member.value.email,
    phone: member.value.phone,
    street: member.value.street,
    streetNo: member.value.streetNo,
    postalCode: member.value.postalCode,
    city: member.value.city,
    membershipStart: formatDateForInput(member.value.membershipStart),
    membershipEnd: member.value.membershipEnd ? formatDateForInput(member.value.membershipEnd) : undefined,
    isActive: member.value.isActive,
  };
  editError.value = null;
  showEditDialog.value = true;
}

async function handleEdit() {
  if (!member.value) return;
  isEditing.value = true;
  editError.value = null;
  try {
    const updated = await api.updateMember(memberId.value, editForm.value);
    member.value = updated;
    showEditDialog.value = false;
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isEditing.value = false;
  }
}

async function handleDelete() {
  isDeleting.value = true;
  try {
    await api.deleteMember(memberId.value);
    router.push('/mitglieder');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showDeleteDialog.value = false;
  } finally {
    isDeleting.value = false;
  }
}

async function handleToggleActive() {
  if (!member.value) return;
  isEditing.value = true;
  try {
    const updated = await api.updateMember(memberId.value, { isActive: !member.value.isActive });
    member.value = updated;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Aktualisieren';
  } finally {
    isEditing.value = false;
  }
}
</script>

<template>
  <div>
    <!-- Back button -->
    <button
      @click="router.push('/mitglieder')"
      class="flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-6"
    >
      <ArrowLeft class="h-4 w-4" />
      Zurück zur Übersicht
    </button>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadMember" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Member details -->
    <div v-else-if="member">
      <!-- Header -->
      <div class="bg-white rounded-xl border p-6 mb-6">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <User class="h-8 w-8 text-primary" />
            </div>
            <div>
              <h1 class="text-2xl font-bold text-gray-900">
                {{ member.firstName }} {{ member.lastName }}
              </h1>
              <div class="flex items-center gap-3 mt-1">
                <span class="text-gray-600 font-mono">{{ member.memberNumber }}</span>
                <span
                  :class="[
                    'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                    member.isActive ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600',
                  ]"
                >
                  {{ member.isActive ? 'Aktiv' : 'Inaktiv' }}
                </span>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button
              @click="handleToggleActive"
              :disabled="isEditing"
              :class="[
                'px-3 py-2 text-sm rounded-lg transition-colors',
                member.isActive
                  ? 'text-amber-700 hover:bg-amber-50'
                  : 'text-green-700 hover:bg-green-50',
              ]"
            >
              {{ member.isActive ? 'Deaktivieren' : 'Aktivieren' }}
            </button>
            <button
              @click="openEditDialog"
              class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
              title="Bearbeiten"
            >
              <Edit class="h-5 w-5" />
            </button>
            <button
              v-if="authStore.isAdmin"
              @click="showDeleteDialog = true"
              class="p-2 text-red-500 hover:text-red-700 hover:bg-red-50 rounded-lg transition-colors"
              title="Löschen"
            >
              <Trash2 class="h-5 w-5" />
            </button>
          </div>
        </div>

        <!-- Info grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mt-6 pt-6 border-t">
          <div class="flex items-start gap-3">
            <Hash class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Mitgliedsnummer</p>
              <p class="font-medium font-mono">{{ member.memberNumber }}</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Mitglied seit</p>
              <p class="font-medium">{{ formatDate(member.membershipStart) }}</p>
            </div>
          </div>
          <div v-if="member.membershipEnd" class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Mitgliedschaft endet</p>
              <p class="font-medium">{{ formatDate(member.membershipEnd) }}</p>
            </div>
          </div>
          <div v-if="member.email" class="flex items-start gap-3">
            <Mail class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">E-Mail</p>
              <a :href="`mailto:${member.email}`" class="font-medium text-primary hover:underline">
                {{ member.email }}
              </a>
            </div>
          </div>
          <div v-if="member.phone" class="flex items-start gap-3">
            <Phone class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Telefon</p>
              <a :href="`tel:${member.phone}`" class="font-medium text-primary hover:underline">
                {{ member.phone }}
              </a>
            </div>
          </div>
          <div v-if="member.street" class="flex items-start gap-3">
            <MapPin class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Adresse</p>
              <p class="font-medium">{{ member.street }} {{ member.streetNo }}</p>
              <p class="text-sm text-gray-500">{{ member.postalCode }} {{ member.city }}</p>
            </div>
          </div>
        </div>

        <!-- Timestamps -->
        <div class="mt-6 pt-6 border-t">
          <div class="flex gap-6 text-sm text-gray-500">
            <span>Erstellt: {{ formatDate(member.createdAt) }}</span>
            <span>Aktualisiert: {{ formatDate(member.updatedAt) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit Dialog -->
    <div
      v-if="showEditDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showEditDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-xl font-semibold">Mitglied bearbeiten</h2>
          <button @click="showEditDialog = false" class="p-1 hover:bg-gray-100 rounded">
            <X class="h-5 w-5" />
          </button>
        </div>

        <form @submit.prevent="handleEdit" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-firstName" class="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
              <input
                id="edit-firstName"
                v-model="editForm.firstName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-lastName" class="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
              <input
                id="edit-lastName"
                v-model="editForm.lastName"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="edit-email" class="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              id="edit-email"
              v-model="editForm.email"
              type="email"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div>
            <label for="edit-phone" class="block text-sm font-medium text-gray-700 mb-1">Telefon</label>
            <input
              id="edit-phone"
              v-model="editForm.phone"
              type="tel"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="edit-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
              <input
                id="edit-street"
                v-model="editForm.street"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-streetNo" class="block text-sm font-medium text-gray-700 mb-1">Hausnr.</label>
              <input
                id="edit-streetNo"
                v-model="editForm.streetNo"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="edit-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                id="edit-postalCode"
                v-model="editForm.postalCode"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div class="col-span-2">
              <label for="edit-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                id="edit-city"
                v-model="editForm.city"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <!-- Membership dates -->
          <div class="pt-4 border-t">
            <h3 class="text-sm font-medium text-gray-700 mb-3">Mitgliedschaft</h3>
            
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="edit-membershipStart" class="block text-sm font-medium text-gray-700 mb-1">Mitglied seit</label>
                <input
                  id="edit-membershipStart"
                  v-model="editForm.membershipStart"
                  type="date"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                />
              </div>
              <div>
                <label for="edit-membershipEnd" class="block text-sm font-medium text-gray-700 mb-1">Endet am</label>
                <input
                  id="edit-membershipEnd"
                  v-model="editForm.membershipEnd"
                  type="date"
                  class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
                />
              </div>
            </div>
          </div>

          <div v-if="editError" class="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p class="text-sm text-red-600">{{ editError }}</p>
          </div>

          <div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showEditDialog = false"
              class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              Abbrechen
            </button>
            <button
              type="submit"
              :disabled="isEditing"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
            >
              <Loader2 v-if="isEditing" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation Dialog -->
    <div
      v-if="showDeleteDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showDeleteDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-red-100 rounded-lg">
            <Trash2 class="h-6 w-6 text-red-600" />
          </div>
          <h2 class="text-xl font-semibold">Mitglied löschen?</h2>
        </div>

        <p class="text-gray-600 mb-6">
          Möchten Sie <strong>{{ member?.firstName }} {{ member?.lastName }}</strong> wirklich löschen?
          Diese Aktion kann nicht rückgängig gemacht werden.
        </p>

        <div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleDelete"
            :disabled="isDeleting"
            class="inline-flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Löschen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
