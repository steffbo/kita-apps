<script setup lang="ts">
import { toRefs } from 'vue';
import type { EffectiveCareHours } from '@/composables/useChildDetailData';
import type { Child, ChildcareFeeResult, KnownIBANSummary, Parent, UpdateHouseholdRequest } from '@/api/types';
import { Edit, Loader2, User, Check, Users, Plus, Link, Unlink, Home, Euro } from 'lucide-vue-next';
import { formatCurrency, formatCurrencyWhole, formatDate } from '@/utils/format';
import { maskIban } from '@/utils/fees';
import { formatCareHours, getIncomeStatusLabel, incomeStatusOptions, isUnderThree } from '@/utils/child';

const props = defineProps<{
  child: Child;
  childcareFee: ChildcareFeeResult | null;
  isLoadingChildcareFee: boolean;
  childcareFeeCareHours: EffectiveCareHours | null;
  trustedIbans: KnownIBANSummary[];
  isEditingHousehold: boolean;
  householdEditForm: UpdateHouseholdRequest;
  isSavingHousehold: boolean;
  householdError: string | null;
  openCreateParentDialog: () => void;
  openLinkParentDialog: () => void;
  confirmUnlinkParent: (parent: Parent) => void;
  openParentDetailModal: (parent: Parent) => void;
  siblings: Child[];
  householdParents: Parent[];
  startEditingHousehold: () => void;
  cancelEditingHousehold: () => void;
  saveHouseholdEdit: () => Promise<void>;
}>();

const {
  child,
  childcareFee,
  isLoadingChildcareFee,
  childcareFeeCareHours,
  trustedIbans,
  isEditingHousehold,
  householdEditForm,
  isSavingHousehold,
  householdError,
  openCreateParentDialog,
  openLinkParentDialog,
  confirmUnlinkParent,
  openParentDetailModal,
  siblings,
  householdParents,
  startEditingHousehold,
  cancelEditingHousehold,
  saveHouseholdEdit,
} = toRefs(props);
</script>

<template>
  <!-- Household & Income Section -->
  <div class="bg-card rounded-xl border p-6 mb-6">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <Home class="h-5 w-5 text-primary" />
        <h2 class="text-lg font-semibold">Haushalt & Einkommen</h2>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="openLinkParentDialog"
          class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-primary hover:bg-primary/10 rounded-md transition-colors"
          title="Vorhandenen Elternteil verknüpfen"
        >
          <Link class="h-3 w-3" />
          Verknüpfen
        </button>
        <button
          @click="openCreateParentDialog"
          class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 rounded-md transition-colors"
        >
          <Plus class="h-3 w-3" />
          Elternteil
        </button>
        <button
          v-if="child.household && !isEditingHousehold"
          @click="startEditingHousehold"
          class="p-2 text-muted-foreground hover:text-foreground hover:bg-accent rounded-lg transition-colors"
          title="Bearbeiten"
        >
          <Edit class="h-4 w-4" />
        </button>
      </div>
    </div>

    <!-- Has Household -->
    <div v-if="child.household">
      <!-- View Mode -->
      <div v-if="!isEditingHousehold" class="space-y-4">
        <!-- Household Name -->
        <div>
          <p class="text-sm text-muted-foreground">Haushaltsname</p>
          <p class="font-medium">{{ child.household.name }}</p>
        </div>

        <!-- Income Status -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <div>
            <p class="text-sm text-muted-foreground">Einkommensstatus</p>
            <p class="font-medium">{{ getIncomeStatusLabel(child.household.incomeStatus) }}</p>
            <router-link
              v-if="!child.household.incomeStatus || child.household.incomeStatus === 'PENDING'"
              :to="`/einstufungen/neu?childId=${child.id}`"
              class="inline-flex items-center gap-1 mt-1 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
            >
              Einstufung erstellen →
            </router-link>
            <router-link
              v-else
              :to="`/einstufungen/neu?childId=${child.id}&followUp=1`"
              class="inline-flex items-center gap-1 mt-1 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
            >
              Folgeeinstufung erstellen →
            </router-link>
          </div>
          <div v-if="child.household.incomeStatus === 'PROVIDED' || child.household.incomeStatus === 'HISTORIC'">
            <p class="text-sm text-muted-foreground">Jahreshaushaltseinkommen</p>
            <p class="font-medium">{{ formatCurrencyWhole(child.household.annualHouseholdIncome) }}</p>
          </div>
          <div v-if="child.household.childrenCountForFees">
            <p class="text-sm text-muted-foreground">Kinder (Beitragsberechnung)</p>
            <p class="font-medium">{{ child.household.childrenCountForFees }}</p>
          </div>
        </div>

        <div v-if="trustedIbans.length > 0" class="pt-3 border-t">
          <p class="text-sm text-muted-foreground mb-2">Bekannte IBANs</p>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="iban in trustedIbans"
              :key="iban.iban"
              :title="iban.payerName ? `${iban.payerName} · ${iban.iban}` : iban.iban"
              class="inline-flex items-center gap-1 px-2 py-1 bg-muted border border-border rounded-full text-xs text-foreground"
            >
              <span class="font-mono">{{ maskIban(iban.iban) }}</span>
              <span v-if="iban.transactionCount > 0" class="text-muted-foreground">· {{ iban.transactionCount }} Zahlungen</span>
            </span>
          </div>
        </div>

        <!-- Platzgeld (Childcare Fee) for U3 children -->
        <div v-if="isUnderThree(child.birthDate)" class="pt-4 border-t">
          <div class="flex items-start gap-3">
            <Euro class="h-5 w-5 text-primary mt-0.5" />
            <div class="flex-1">
              <p class="text-sm text-muted-foreground">Monatliches Platzgeld</p>
              <div v-if="isLoadingChildcareFee" class="flex items-center gap-2">
                <Loader2 class="h-4 w-4 animate-spin text-muted-foreground" />
                <span class="text-muted-foreground text-sm">Berechne...</span>
              </div>
              <div v-else-if="childcareFee">
                <p class="font-semibold text-lg text-primary">{{ formatCurrency(childcareFee.fee) }}</p>
                <p class="text-sm text-muted-foreground">{{ childcareFee.rule }}</p>
                <p v-if="childcareFeeCareHours" class="text-sm text-muted-foreground">
                  Basis: {{ formatCareHours(childcareFeeCareHours.hours) }}<span v-if="childcareFeeCareHours.effectiveFrom"> (ab {{ formatDate(childcareFeeCareHours.effectiveFrom) }})</span>
                </p>
                <p v-if="childcareFee.discountPercent > 0" class="text-sm text-green-600 dark:text-green-300">
                  Geschwisterrabatt: {{ childcareFee.discountPercent }}%
                </p>
                <p v-if="childcareFee.notes && childcareFee.notes.length > 0" class="text-xs text-muted-foreground mt-1">
                  {{ childcareFee.notes.join(' · ') }}
                </p>
              </div>
              <div v-else>
                <p class="text-muted-foreground text-sm italic">
                  <span v-if="!child.household.incomeStatus || child.household.incomeStatus === 'PENDING'">
                    Einkommen noch nicht angegeben
                  </span>
                  <span v-else-if="child.household.incomeStatus === 'NOT_REQUIRED' || child.household.incomeStatus === 'HISTORIC'">
                    Nicht zutreffend
                  </span>
                  <span v-else-if="!childcareFeeCareHours">
                    Betreuungszeit nicht hinterlegt
                  </span>
                  <span v-else>
                    Kann nicht berechnet werden
                  </span>
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Family Members -->
        <div v-if="householdParents.length > 0 || siblings.length > 0" class="pt-4 border-t">
          <p class="text-sm text-muted-foreground mb-3">Familienmitglieder</p>

          <!-- Parents in Household -->
          <div v-if="householdParents.length > 0" class="mb-3">
            <p class="text-xs text-muted-foreground uppercase tracking-wide mb-2">Eltern</p>
            <div class="flex flex-wrap gap-2">
              <div
                v-for="parent in householdParents"
                :key="parent.id"
                class="inline-flex items-center bg-blue-50 dark:bg-blue-950/40 border border-blue-200 rounded-lg text-sm"
              >
                <button
                  @click="openParentDetailModal(parent)"
                  class="inline-flex items-center gap-2 px-3 py-1.5 hover:bg-blue-100 dark:hover:bg-blue-900/40 rounded-l-lg transition-colors"
                >
                  <User class="h-4 w-4 text-blue-500 dark:text-blue-400" />
                  <span>{{ parent.firstName }} {{ parent.lastName }}</span>
                </button>
                <button
                  @click="confirmUnlinkParent(parent)"
                  class="p-1.5 text-blue-400 hover:text-red-500 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/40 rounded-r-lg border-l border-blue-200 transition-colors"
                  title="Verknüpfung aufheben"
                  aria-label="Verknüpfung aufheben"
                >
                  <Unlink class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          </div>

          <!-- Siblings in Household -->
          <div v-if="siblings.length > 0">
            <p class="text-xs text-muted-foreground uppercase tracking-wide mb-2">Geschwister</p>
            <div class="flex flex-wrap gap-2">
              <router-link
                v-for="sibling in siblings"
                :key="sibling.id"
                :to="`/kinder/${sibling.id}`"
                class="inline-flex items-center gap-2 px-3 py-1.5 bg-amber-50 dark:bg-amber-950/40 hover:bg-amber-100 dark:hover:bg-amber-900/40 border border-amber-200 rounded-lg text-sm transition-colors"
              >
                <User class="h-4 w-4 text-amber-500 dark:text-amber-400" />
                <span>{{ sibling.firstName }} {{ sibling.lastName }}</span>
              </router-link>
            </div>
          </div>
        </div>
      </div>

      <!-- Edit Mode -->
      <form v-else @submit.prevent="saveHouseholdEdit" class="space-y-4">
        <div>
          <label for="household-name" class="block text-sm font-medium text-foreground mb-1">Haushaltsname</label>
          <input
            id="household-name"
            v-model="householdEditForm.name"
            type="text"
            class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div>
          <label for="household-incomeStatus" class="block text-sm font-medium text-foreground mb-1">Einkommensstatus</label>
          <select
            id="household-incomeStatus"
            v-model="householdEditForm.incomeStatus"
            class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none bg-card"
          >
            <option v-for="option in incomeStatusOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>

        <div v-if="householdEditForm.incomeStatus === 'PROVIDED' || householdEditForm.incomeStatus === 'HISTORIC'">
          <label for="household-income" class="block text-sm font-medium text-foreground mb-1">Jahreshaushaltseinkommen</label>
          <input
            id="household-income"
            v-model.number="householdEditForm.annualHouseholdIncome"
            type="number"
            min="0"
            step="any"
            class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
        </div>

        <div>
          <label for="household-childrenCount" class="block text-sm font-medium text-foreground mb-1">Anzahl Kinder (für Beitragsberechnung)</label>
          <input
            id="household-childrenCount"
            v-model.number="householdEditForm.childrenCountForFees"
            type="number"
            min="1"
            max="10"
            placeholder="Automatisch"
            class="w-full px-3 py-2 border border-border rounded-lg focus:ring-2 focus:ring-primary focus:border-transparent outline-none"
          />
          <p class="text-xs text-muted-foreground mt-1">Leer lassen für automatische Zählung der U3-Kinder im Haushalt</p>
        </div>

        <div v-if="householdError" class="p-3 bg-red-50 dark:bg-red-950/40 border border-red-200 rounded-lg">
          <p class="text-sm text-red-600 dark:text-red-300">{{ householdError }}</p>
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button
            type="button"
            @click="cancelEditingHousehold"
            class="px-4 py-2 text-foreground hover:bg-accent rounded-lg transition-colors"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            :disabled="isSavingHousehold"
            class="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            <Loader2 v-if="isSavingHousehold" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
            Speichern
          </button>
        </div>
      </form>
    </div>

    <!-- No Household -->
    <div v-else class="text-center py-6 bg-muted rounded-lg border border-dashed">
      <Users class="h-8 w-8 text-muted-foreground mx-auto mb-2" />
      <p class="text-muted-foreground text-sm mb-1">Noch keine Eltern zugeordnet</p>
      <p class="text-muted-foreground text-xs mb-4">Ein Haushalt wird automatisch erstellt, wenn der erste Elternteil verknüpft wird.</p>
      <div class="flex items-center justify-center gap-2">
        <button
          @click="openLinkParentDialog"
          class="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-primary border border-primary hover:bg-primary/10 rounded-lg transition-colors"
        >
          <Link class="h-4 w-4" />
          Verknüpfen
        </button>
        <button
          @click="openCreateParentDialog"
          class="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 rounded-lg transition-colors"
        >
          <Plus class="h-4 w-4" />
          Neu anlegen
        </button>
      </div>
    </div>
  </div>
</template>
