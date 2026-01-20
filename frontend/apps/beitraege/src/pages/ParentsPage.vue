<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { api } from '@/api';
import type { Parent } from '@/api/types';
import {
  Plus,
  Search,
  Loader2,
  User,
  Mail,
  Phone,
} from 'lucide-vue-next';

const parents = ref<Parent[]>([]);
const total = ref(0);
const isLoading = ref(true);
const error = ref<string | null>(null);
const searchQuery = ref('');

async function loadParents() {
  isLoading.value = true;
  error.value = null;
  try {
    const response = await api.getParents({
      search: searchQuery.value || undefined,
      limit: 100,
    });
    parents.value = response.data;
    total.value = response.total;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadParents);

function formatCurrency(amount: number | undefined): string {
  if (!amount) return '-';
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: 'EUR',
    maximumFractionDigits: 0,
  }).format(amount);
}
</script>

<template>
  <div>
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Eltern</h1>
        <p class="text-gray-600 mt-1">{{ total }} Eltern registriert</p>
      </div>
      <button
        class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
      >
        <Plus class="h-4 w-4" />
        Elternteil hinzufügen
      </button>
    </div>

    <!-- Search -->
    <div class="relative mb-6">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
      <input
        v-model="searchQuery"
        @input="loadParents"
        type="text"
        placeholder="Suchen nach Name oder E-Mail..."
        class="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
      />
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
      <p class="text-red-600">{{ error }}</p>
      <button @click="loadParents" class="mt-2 text-sm text-red-700 underline">
        Erneut versuchen
      </button>
    </div>

    <!-- Parents grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="parent in parents"
        :key="parent.id"
        class="bg-white rounded-xl border p-4 hover:shadow-md transition-shadow cursor-pointer"
      >
        <div class="flex items-start gap-3">
          <div class="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
            <User class="h-6 w-6 text-primary" />
          </div>
          <div class="flex-1 min-w-0">
            <h3 class="font-semibold text-gray-900">
              {{ parent.firstName }} {{ parent.lastName }}
            </h3>
            <div v-if="parent.email" class="flex items-center gap-1 text-sm text-gray-600 mt-1">
              <Mail class="h-3.5 w-3.5" />
              <span class="truncate">{{ parent.email }}</span>
            </div>
            <div v-if="parent.phone" class="flex items-center gap-1 text-sm text-gray-600 mt-0.5">
              <Phone class="h-3.5 w-3.5" />
              <span>{{ parent.phone }}</span>
            </div>
            <div v-if="parent.annualHouseholdIncome" class="mt-2">
              <span class="text-xs text-gray-500">Haushaltseinkommen:</span>
              <span class="ml-1 text-sm font-medium">
                {{ formatCurrency(parent.annualHouseholdIncome) }}/Jahr
              </span>
            </div>
            <div v-if="parent.children && parent.children.length > 0" class="mt-2">
              <span class="text-xs text-gray-500">Kinder:</span>
              <div class="flex flex-wrap gap-1 mt-1">
                <span
                  v-for="child in parent.children"
                  :key="child.id"
                  class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-gray-100 text-gray-700"
                >
                  {{ child.firstName }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="parents.length === 0" class="col-span-full text-center py-12 text-gray-500">
        Keine Eltern gefunden
      </div>
    </div>
  </div>
</template>
