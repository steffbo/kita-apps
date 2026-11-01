<script setup lang="ts">
import AppSheet from '@/components/ui/AppSheet.vue';
import { ref, onMounted } from 'vue';
import { api } from '@/api';
import type { KnownIBAN } from '@/api/types';
import { Loader2, Trash2, ShieldOff } from 'lucide-vue-next';
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
  <AppSheet title="Blacklist" size="lg" @close="$emit('close')">
    <p class="mb-3 text-sm text-muted-foreground">
      {{ blacklistTotal }} ignorierte IBANs – Transaktionen von diesen IBANs werden beim Import automatisch ignoriert
    </p>
    <div class="mb-4 flex justify-end">
      <button @click="loadBlacklist" class="btn btn-ghost btn-sm">Aktualisieren</button>
    </div>
    <div>
        <div v-if="blacklistError" class="note note-danger mb-4">
          {{ blacklistError }}
        </div>
        <div v-if="isLoadingBlacklist" class="flex items-center justify-center py-12">
          <Loader2 class="h-8 w-8 animate-spin text-primary" />
        </div>

        <div v-else-if="blacklistedIBANs.length === 0" class="text-center py-12">
          <ShieldOff class="h-12 w-12 text-muted-foreground/60 mx-auto mb-4" />
          <p class="text-muted-foreground">Keine IBANs auf der Blacklist</p>
          <p class="text-sm text-muted-foreground mt-1">
            Klicken Sie bei nicht zugeordneten Transaktionen auf "Ignorieren", um IBANs zur Blacklist hinzuzufugen
          </p>
        </div>

        <div v-else class="card overflow-hidden">
          <div class="overflow-x-auto">
            <table class="table">
              <thead>
                <tr>
                  <th>IBAN</th>
                  <th>Zahler</th>
                  <th>Letzte Transaktion</th>
                  <th>Hinzugefugt am</th>
                  <th class="text-right">Aktionen</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in blacklistedIBANs"
                  :key="item.iban"
                  class="hover:bg-accent"
                >
                  <td>
                    <span class="font-mono text-sm">{{ item.iban }}</span>
                  </td>
                  <td>
                    <span class="font-medium">{{ item.payerName || 'Unbekannt' }}</span>
                  </td>
                  <td class="text-muted-foreground">
                    <div v-if="item.originalDescription" class="truncate max-w-xs text-sm">
                      {{ item.originalDescription }}
                    </div>
                    <div v-if="item.originalAmount" class="text-xs text-muted-foreground">
                      {{ formatCurrency(item.originalAmount) }}
                    </div>
                  </td>
                  <td class="text-muted-foreground text-sm">
                    {{ formatDate(item.createdAt) }}
                  </td>
                  <td class="text-right">
                    <button data-write
                      @click="removeFromBlacklist(item.iban)"
                      class="btn btn-secondary btn-sm"
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
</AppSheet>
</template>
