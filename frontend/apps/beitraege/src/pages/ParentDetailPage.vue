<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import type { Parent, UpdateParentRequest } from '@/api/types';
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
  Users,
  UserPlus,
  ExternalLink,
} from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const parent = ref<Parent | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Edit dialog state
const showEditDialog = ref(false);
const editForm = ref<UpdateParentRequest>({});
const isEditing = ref(false);
const editError = ref<string | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

// Member dialog state
const showMemberDialog = ref(false);
const membershipStart = ref(new Date().toISOString().split('T')[0]);
const isCreatingMember = ref(false);
const memberError = ref<string | null>(null);

const parentId = computed(() => route.params.id as string);

async function loadParent() {
  isLoading.value = true;
  error.value = null;
  try {
    parent.value = await api.getParent(parentId.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadParent);

// ESC key handler to close all modals
function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (showDeleteDialog.value) {
      showDeleteDialog.value = false;
    } else if (showMemberDialog.value) {
      showMemberDialog.value = false;
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
  if (!parent.value) return;
  editForm.value = {
    firstName: parent.value.firstName,
    lastName: parent.value.lastName,
    birthDate: parent.value.birthDate ? formatDateForInput(parent.value.birthDate) : undefined,
    email: parent.value.email,
    phone: parent.value.phone,
    street: parent.value.street,
    streetNo: parent.value.streetNo,
    postalCode: parent.value.postalCode,
    city: parent.value.city,
  };
  editError.value = null;
  showEditDialog.value = true;
}

async function handleEdit() {
  if (!parent.value) return;
  isEditing.value = true;
  editError.value = null;
  try {
    const updated = await api.updateParent(parentId.value, editForm.value);
    parent.value = updated;
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
    await api.deleteParent(parentId.value);
    router.push('/eltern');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showDeleteDialog.value = false;
  } finally {
    isDeleting.value = false;
  }
}

function goToChild(childId: string) {
  router.push(`/kinder/${childId}`);
}

function goToMember(memberId: string) {
  router.push(`/mitglieder/${memberId}`);
}

function openMemberDialog() {
  membershipStart.value = new Date().toISOString().split('T')[0];
  memberError.value = null;
  showMemberDialog.value = true;
}

async function handleCreateMember() {
  isCreatingMember.value = true;
  memberError.value = null;
  try {
    const updated = await api.createMemberFromParent(parentId.value, membershipStart.value);
    parent.value = updated;
    showMemberDialog.value = false;
  } catch (e) {
    memberError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreatingMember.value = false;
  }
}

async function handleUnlinkMember() {
  if (!parent.value?.memberId) return;
  if (!confirm('Möchten Sie die Verknüpfung zum Vereinsmitglied wirklich entfernen? Das Mitglied wird nicht gelöscht.')) return;
  try {
    const updated = await api.unlinkMemberFromParent(parentId.value);
    parent.value = updated;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Entfernen der Verknüpfung';
  }
}
</script>

<template>
  <div>
    <!-- Back button -->
    <button
      @click="router.push('/eltern')"
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
      <button @click="loadParent" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Parent details -->
    <div v-else-if="parent">
      <!-- Header -->
      <div class="bg-white rounded-xl border p-6 mb-6">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <User class="h-8 w-8 text-primary" />
            </div>
            <div>
              <h1 class="text-2xl font-bold text-gray-900">
                {{ parent.firstName }} {{ parent.lastName }}
              </h1>
              <p class="text-gray-600">Elternteil</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
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
          <div v-if="parent.birthDate" class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Geburtsdatum</p>
              <p class="font-medium">{{ formatDate(parent.birthDate) }}</p>
            </div>
          </div>
          <div v-if="parent.email" class="flex items-start gap-3">
            <Mail class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">E-Mail</p>
              <a :href="`mailto:${parent.email}`" class="font-medium text-primary hover:underline">
                {{ parent.email }}
              </a>
            </div>
          </div>
          <div v-if="parent.phone" class="flex items-start gap-3">
            <Phone class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Telefon</p>
              <a :href="`tel:${parent.phone}`" class="font-medium text-primary hover:underline">
                {{ parent.phone }}
              </a>
            </div>
          </div>
          <div v-if="parent.street" class="flex items-start gap-3">
            <MapPin class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Adresse</p>
              <p class="font-medium">{{ parent.street }} {{ parent.streetNo }}</p>
              <p class="text-sm text-gray-500">{{ parent.postalCode }} {{ parent.city }}</p>
            </div>
          </div>
        </div>

        <!-- Children Section -->
        <div class="mt-6 pt-6 border-t">
          <div class="flex items-center gap-2 mb-4">
            <Users class="h-4 w-4 text-gray-500" />
            <h3 class="text-sm font-medium text-gray-500">Verknüpfte Kinder</h3>
          </div>

          <div v-if="parent.children && parent.children.length > 0" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <button
              v-for="child in parent.children"
              :key="child.id"
              @click="goToChild(child.id)"
              class="bg-gray-50 rounded-lg border p-4 hover:border-primary/50 hover:bg-gray-100 transition-colors text-left"
            >
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                  <User class="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p class="font-medium text-gray-900">{{ child.firstName }} {{ child.lastName }}</p>
                  <p class="text-sm text-gray-500 font-mono">{{ child.memberNumber }}</p>
                </div>
              </div>
            </button>
          </div>

          <div v-else class="text-center py-6 bg-gray-50 rounded-lg border border-dashed">
            <Users class="h-8 w-8 text-gray-400 mx-auto mb-2" />
            <p class="text-gray-500 text-sm">Keine Kinder verknüpft</p>
          </div>
        </div>

        <!-- Membership Section -->
        <div class="mt-6 pt-6 border-t">
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <UserPlus class="h-4 w-4 text-gray-500" />
              <h3 class="text-sm font-medium text-gray-500">Vereinsmitgliedschaft</h3>
            </div>
          </div>

          <!-- Already a member -->
          <div v-if="parent.member" class="bg-green-50 rounded-lg border border-green-200 p-4">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-green-100 flex items-center justify-center">
                  <UserPlus class="h-5 w-5 text-green-600" />
                </div>
                <div>
                  <p class="font-medium text-gray-900">{{ parent.member.memberNumber }}</p>
                  <p class="text-sm text-gray-500">
                    Mitglied seit {{ formatDate(parent.member.membershipStart) }}
                  </p>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <button
                  @click="goToMember(parent.member!.id)"
                  class="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-primary hover:bg-primary/10 rounded-lg transition-colors"
                >
                  <ExternalLink class="h-4 w-4" />
                  Details
                </button>
                <button
                  @click="handleUnlinkMember"
                  class="inline-flex items-center gap-1 px-3 py-1.5 text-sm text-gray-500 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                  title="Verknüpfung entfernen"
                >
                  <X class="h-4 w-4" />
                </button>
              </div>
            </div>
          </div>

          <!-- Not a member yet -->
          <div v-else class="text-center py-6 bg-gray-50 rounded-lg border border-dashed">
            <UserPlus class="h-8 w-8 text-gray-400 mx-auto mb-2" />
            <p class="text-gray-500 text-sm mb-3">Kein Vereinsmitglied</p>
            <button
              @click="openMemberDialog"
              class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
            >
              <UserPlus class="h-4 w-4" />
              Zum Mitglied machen
            </button>
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
          <h2 class="text-xl font-semibold">Elternteil bearbeiten</h2>
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
            <label for="edit-birthDate" class="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
            <input
              id="edit-birthDate"
              v-model="editForm.birthDate"
              type="date"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
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
          <h2 class="text-xl font-semibold">Elternteil löschen?</h2>
        </div>

        <p class="text-gray-600 mb-6">
          Möchten Sie <strong>{{ parent?.firstName }} {{ parent?.lastName }}</strong> wirklich löschen?
          Diese Aktion kann nicht rückgängig gemacht werden. Die Verknüpfungen zu Kindern werden ebenfalls entfernt.
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

    <!-- Create Member Dialog -->
    <div
      v-if="showMemberDialog"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      @click.self="showMemberDialog = false"
    >
      <div class="bg-white rounded-xl shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-primary/10 rounded-lg">
            <UserPlus class="h-6 w-6 text-primary" />
          </div>
          <h2 class="text-xl font-semibold">Vereinsmitglied erstellen</h2>
        </div>

        <p class="text-gray-600 mb-4">
          <strong>{{ parent?.firstName }} {{ parent?.lastName }}</strong> wird als Vereinsmitglied registriert.
          Die Kontaktdaten werden übernommen.
        </p>

        <div class="mb-6">
          <label for="membershipStart" class="block text-sm font-medium text-gray-700 mb-1">
            Mitgliedschaft ab
          </label>
          <input
            id="membershipStart"
            v-model="membershipStart"
            type="date"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div v-if="memberError" class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600">{{ memberError }}</p>
        </div>

        <div class="flex justify-end gap-3">
          <button
            @click="showMemberDialog = false"
            class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="handleCreateMember"
            :disabled="isCreatingMember"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isCreatingMember" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
            Erstellen
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
