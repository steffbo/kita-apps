<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import type { Child, FeeExpectation, UpdateChildRequest } from '@/api/types';
import {
  ArrowLeft,
  Edit,
  Trash2,
  Loader2,
  User,
  Calendar,
  MapPin,
  Receipt,
  CheckCircle,
  Clock,
  AlertTriangle,
  X,
  Check,
  Users,
} from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();

const child = ref<Child | null>(null);
const fees = ref<FeeExpectation[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Edit dialog state
const showEditDialog = ref(false);
const editForm = ref<UpdateChildRequest>({});
const isEditing = ref(false);
const editError = ref<string | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

const childId = computed(() => route.params.id as string);

async function loadChild() {
  isLoading.value = true;
  error.value = null;
  try {
    child.value = await api.getChild(childId.value);
    const feesResponse = await api.getFees({ childId: childId.value, limit: 50 });
    fees.value = feesResponse.data;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadChild);

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('de-DE');
}

function formatDateForInput(dateStr: string): string {
  return dateStr.split('T')[0];
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
  }).format(amount);
}

function getFeeTypeName(type: string): string {
  switch (type) {
    case 'MEMBERSHIP':
      return 'Vereinsbeitrag';
    case 'FOOD':
      return 'Essensgeld';
    case 'CHILDCARE':
      return 'Platzgeld';
    default:
      return type;
  }
}

function getMonthName(month: number): string {
  return new Date(2000, month - 1).toLocaleString('de-DE', { month: 'long' });
}

function calculateAge(birthDate: string): number {
  const birth = new Date(birthDate);
  const today = new Date();
  let age = today.getFullYear() - birth.getFullYear();
  const m = today.getMonth() - birth.getMonth();
  if (m < 0 || (m === 0 && today.getDate() < birth.getDate())) {
    age--;
  }
  return age;
}

function isUnderThree(birthDate: string): boolean {
  return calculateAge(birthDate) < 3;
}

function openEditDialog() {
  if (!child.value) return;
  editForm.value = {
    firstName: child.value.firstName,
    lastName: child.value.lastName,
    birthDate: formatDateForInput(child.value.birthDate),
    entryDate: formatDateForInput(child.value.entryDate),
    street: child.value.street,
    houseNumber: child.value.houseNumber,
    postalCode: child.value.postalCode,
    city: child.value.city,
    isActive: child.value.isActive,
  };
  editError.value = null;
  showEditDialog.value = true;
}

async function handleEdit() {
  if (!child.value) return;
  isEditing.value = true;
  editError.value = null;
  try {
    const updated = await api.updateChild(childId.value, editForm.value);
    child.value = updated;
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
    await api.deleteChild(childId.value);
    router.push('/kinder');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showDeleteDialog.value = false;
  } finally {
    isDeleting.value = false;
  }
}

function goToParent(parentId: string) {
  router.push(`/eltern?highlight=${parentId}`);
}

const openFees = computed(() => fees.value.filter(f => !f.isPaid));
const paidFees = computed(() => fees.value.filter(f => f.isPaid));
</script>

<template>
  <div>
    <!-- Back button -->
    <button
      @click="router.push('/kinder')"
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
      <button @click="loadChild" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Child details -->
    <div v-else-if="child">
      <!-- Header -->
      <div class="bg-white rounded-xl border p-6 mb-6">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <User class="h-8 w-8 text-primary" />
            </div>
            <div>
              <h1 class="text-2xl font-bold text-gray-900">
                {{ child.firstName }} {{ child.lastName }}
              </h1>
              <p class="text-gray-600 font-mono">Mitglieds-Nr. {{ child.memberNumber }}</p>
              <div class="flex items-center gap-2 mt-2">
                <span
                  :class="[
                    'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                    child.isActive ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600',
                  ]"
                >
                  {{ child.isActive ? 'Aktiv' : 'Inaktiv' }}
                </span>
                <span
                  v-if="isUnderThree(child.birthDate)"
                  class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700"
                >
                  U3
                </span>
              </div>
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
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Geburtsdatum</p>
              <p class="font-medium">{{ formatDate(child.birthDate) }}</p>
              <p class="text-sm text-gray-500">{{ calculateAge(child.birthDate) }} Jahre alt</p>
            </div>
          </div>
          <div class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Eintrittsdatum</p>
              <p class="font-medium">{{ formatDate(child.entryDate) }}</p>
            </div>
          </div>
          <div v-if="child.street" class="flex items-start gap-3">
            <MapPin class="h-5 w-5 text-gray-400 mt-0.5" />
            <div>
              <p class="text-sm text-gray-500">Adresse</p>
              <p class="font-medium">{{ child.street }} {{ child.houseNumber }}</p>
              <p class="text-sm text-gray-500">{{ child.postalCode }} {{ child.city }}</p>
            </div>
          </div>
        </div>

        <!-- Parents -->
        <div v-if="child.parents && child.parents.length > 0" class="mt-6 pt-6 border-t">
          <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
            <Users class="h-4 w-4" />
            Eltern
          </h3>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="parent in child.parents"
              :key="parent.id"
              @click="goToParent(parent.id)"
              class="inline-flex items-center gap-2 px-3 py-1.5 bg-gray-100 hover:bg-gray-200 rounded-lg transition-colors cursor-pointer"
            >
              <User class="h-4 w-4 text-gray-500" />
              <span>{{ parent.firstName }} {{ parent.lastName }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Fees section -->
      <div class="bg-white rounded-xl border p-6">
        <div class="flex items-center gap-2 mb-6">
          <Receipt class="h-5 w-5 text-primary" />
          <h2 class="text-lg font-semibold">Beiträge</h2>
        </div>

        <!-- Open fees -->
        <div v-if="openFees.length > 0" class="mb-6">
          <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
            <Clock class="h-4 w-4" />
            Offene Beiträge ({{ openFees.length }})
          </h3>
          <div class="space-y-2">
            <div
              v-for="fee in openFees"
              :key="fee.id"
              class="flex items-center justify-between p-3 bg-amber-50 border border-amber-200 rounded-lg"
            >
              <div class="flex items-center gap-3">
                <AlertTriangle
                  v-if="new Date(fee.dueDate) < new Date()"
                  class="h-5 w-5 text-red-500"
                />
                <Clock v-else class="h-5 w-5 text-amber-500" />
                <div>
                  <p class="font-medium">{{ getFeeTypeName(fee.feeType) }}</p>
                  <p class="text-sm text-gray-600">
                    {{ fee.month ? getMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
                    · Fällig: {{ formatDate(fee.dueDate) }}
                  </p>
                </div>
              </div>
              <p class="font-semibold">{{ formatCurrency(fee.amount) }}</p>
            </div>
          </div>
        </div>

        <!-- Paid fees -->
        <div v-if="paidFees.length > 0">
          <h3 class="text-sm font-medium text-gray-500 mb-3 flex items-center gap-2">
            <CheckCircle class="h-4 w-4" />
            Bezahlte Beiträge ({{ paidFees.length }})
          </h3>
          <div class="space-y-2">
            <div
              v-for="fee in paidFees"
              :key="fee.id"
              class="flex items-center justify-between p-3 bg-green-50 border border-green-200 rounded-lg"
            >
              <div class="flex items-center gap-3">
                <CheckCircle class="h-5 w-5 text-green-500" />
                <div>
                  <p class="font-medium">{{ getFeeTypeName(fee.feeType) }}</p>
                  <p class="text-sm text-gray-600">
                    {{ fee.month ? getMonthName(fee.month) + ' ' : '' }}{{ fee.year }}
                  </p>
                </div>
              </div>
              <p class="font-semibold text-green-700">{{ formatCurrency(fee.amount) }}</p>
            </div>
          </div>
        </div>

        <div v-if="fees.length === 0" class="text-center py-8 text-gray-500">
          Keine Beiträge vorhanden
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
          <h2 class="text-xl font-semibold">Kind bearbeiten</h2>
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

          <div class="grid grid-cols-2 gap-4">
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
              <label for="edit-entryDate" class="block text-sm font-medium text-gray-700 mb-1">Eintrittsdatum</label>
              <input
                id="edit-entryDate"
                v-model="editForm.entryDate"
                type="date"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label for="edit-street" class="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input
              id="edit-street"
              v-model="editForm.street"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
            />
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="edit-houseNumber" class="block text-sm font-medium text-gray-700 mb-1">Hausnummer</label>
              <input
                id="edit-houseNumber"
                v-model="editForm.houseNumber"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-postalCode" class="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                id="edit-postalCode"
                v-model="editForm.postalCode"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
            <div>
              <label for="edit-city" class="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                id="edit-city"
                v-model="editForm.city"
                type="text"
                class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
              />
            </div>
          </div>

          <div>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                v-model="editForm.isActive"
                type="checkbox"
                class="w-4 h-4 text-primary rounded border-gray-300 focus:ring-primary"
              />
              <span class="text-sm text-gray-700">Kind ist aktiv</span>
            </label>
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
          <h2 class="text-xl font-semibold">Kind löschen?</h2>
        </div>

        <p class="text-gray-600 mb-6">
          Möchten Sie <strong>{{ child?.firstName }} {{ child?.lastName }}</strong> wirklich löschen?
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
