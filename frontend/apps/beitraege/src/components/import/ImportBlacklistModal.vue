<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { api } from '@/api';
import type { KnownIBAN } from '@/api/types';
import { Loader2, XCircle, Trash2, ShieldOff } from 'lucide-vue-next';
import { formatCurrency, formatDate } from '@/utils/format';

defineEmits<{ close: [] }>();

const blacklistedIBANs = ref<KnownIBAN[]>([]);
const blacklistTotal = ref(0);
const isLoadingBlacklist = ref(false);
const blacklistError = ref<string | null>(null);

async function loadBlacklist(): Promise<void> {
  isLoadingBlacklist.value = true;
  blacklistError.value = null;
  try {
    const response = await api.getBlacklist(1, 100);
    blacklistedIBANs.value = response.data;
    blacklistTotal.value = response.total;
  } catch (error) {
    blacklistError.value = error instanceof Error ? error.message : 'Blacklist konnte nicht geladen werden';
  } finally {
    isLoadingBlacklist.value = false;
  }
}

async function removeFromBlacklist(iban: string): Promise<void> {
  blacklistError.value = null;
  try {
    await api.removeFromBlacklist(iban);
    blacklistedIBANs.value = blacklistedIBANs.value.filter(item => item.iban !== iban);
    blacklistTotal.value = Math.max(0, blacklistTotal.value - 1);
  } catch (error) {
    blacklistError.value = error instanceof Error ? error.message : 'Entfernen fehlgeschlagen';
  }
}

onMounted(loadBlacklist);
</script>

<template>
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl shadow-xl max-w-4xl w-full mx-4 max-h-[85vh] overflow-hidden flex flex-col">
      <div class="p-4 border-b flex items-center justify-between">
        <div>
          <h2 class="text-lg font-semibold">Blacklist</h2>
          <p class="text-sm text-gray-600">
            {{ blacklistTotal }} ignorierte IBANs – Transaktionen von diesen IBANs werden beim Import automatisch ignoriert
          </p>
        </div>
        <div class="flex items-center gap-3">
          <button @click="loadBlacklist" class="text-sm text-gray-600 hover:text-gray-900 underline">
            Aktualisieren
          </button>
          <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600">
            <XCircle class="h-5 w-5" />
          </button>
        </div>
      </div>

      <div class="overflow-y-auto p-4">
        <div v-if="blacklistError" class="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {{ blacklistError }}
        </div>
        <div v-if="isLoadingBlacklist" class="flex items-center justify-center py-12">
          <Loader2 class="h-8 w-8 animate-spin text-primary" />
        </div>

        <div v-else-if="blacklistedIBANs.length === 0" class="text-center py-12">
          <ShieldOff class="h-12 w-12 text-gray-300 mx-auto mb-4" />
          <p class="text-gray-600">Keine IBANs auf der Blacklist</p>
          <p class="text-sm text-gray-500 mt-1">
            Klicken Sie bei nicht zugeordneten Transaktionen auf "Ignorieren", um IBANs zur Blacklist hinzuzufugen
          </p>
        </div>

        <div v-else class="rounded-xl border overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead class="bg-gray-50">
                <tr class="text-left text-sm text-gray-500">
                  <th class="px-4 py-3 font-medium">IBAN</th>
                  <th class="px-4 py-3 font-medium">Zahler</th>
                  <th class="px-4 py-3 font-medium">Letzte Transaktion</th>
                  <th class="px-4 py-3 font-medium">Hinzugefugt am</th>
                  <th class="px-4 py-3 font-medium text-right">Aktionen</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in blacklistedIBANs"
                  :key="item.iban"
                  class="border-t hover:bg-gray-50"
                >
                  <td class="px-4 py-3">
                    <span class="font-mono text-sm">{{ item.iban }}</span>
                  </td>
                  <td class="px-4 py-3">
                    <span class="font-medium">{{ item.payerName || 'Unbekannt' }}</span>
                  </td>
                  <td class="px-4 py-3 text-gray-600">
                    <div v-if="item.originalDescription" class="truncate max-w-xs text-sm">
                      {{ item.originalDescription }}
                    </div>
                    <div v-if="item.originalAmount" class="text-xs text-gray-500">
                      {{ formatCurrency(item.originalAmount) }}
                    </div>
                  </td>
                  <td class="px-4 py-3 text-gray-600 text-sm">
                    {{ formatDate(item.createdAt) }}
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button
                      @click="removeFromBlacklist(item.iban)"
                      class="inline-flex items-center gap-1 px-2 py-1 text-xs text-gray-600 hover:text-green-600 hover:bg-green-50 rounded transition-colors"
                      title="Von Blacklist entfernen"
                    >
                      <Trash2 class="h-3 w-3" />
                      Entfernen
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div></template>
