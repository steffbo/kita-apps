<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import type { Child, FeeExpectation } from '@/api/types';
import {
  ArrowLeft,
  Edit,
  Loader2,
  User,
  Calendar,
  MapPin,
  Receipt,
  CheckCircle,
  Clock,
  AlertTriangle,
} from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();

const child = ref<Child | null>(null);
const fees = ref<FeeExpectation[]>([]);
const isLoading = ref(true);
const error = ref<string | null>(null);

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
          <button
            class="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <Edit class="h-5 w-5" />
          </button>
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
          <h3 class="text-sm font-medium text-gray-500 mb-3">Eltern</h3>
          <div class="flex flex-wrap gap-2">
            <div
              v-for="parent in child.parents"
              :key="parent.id"
              class="inline-flex items-center gap-2 px-3 py-1.5 bg-gray-100 rounded-lg"
            >
              <User class="h-4 w-4 text-gray-500" />
              <span>{{ parent.firstName }} {{ parent.lastName }}</span>
            </div>
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
  </div>
</template>
