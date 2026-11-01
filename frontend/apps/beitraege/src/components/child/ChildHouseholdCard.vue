<script setup lang="ts">
import { toRefs } from 'vue';
import { useAuthStore } from '@/stores/auth';
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

// Leitung sees the household and its members, but no income or fee data.
const auth = useAuthStore();
</script>

<template>
  <!-- Household & Income Section -->
  <div class="card p-6 mb-6">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <Home class="h-5 w-5 text-primary" />
        <h2 class="text-base font-semibold">{{ auth.canAccessFees ? 'Haushalt & Einkommen' : 'Haushalt' }}</h2>
      </div>
      <div class="flex items-center gap-2">
        <button data-write
          @click="openLinkParentDialog"
          class="btn btn-secondary btn-sm"
          title="Vorhandenen Elternteil verknüpfen"
        >
          <Link class="h-3 w-3" />
          Verknüpfen
        </button>
        <button data-write
          @click="openCreateParentDialog"
          class="btn btn-secondary btn-sm"
        >
          <Plus class="h-3 w-3" />
          Elternteil
        </button>
        <button data-write
          v-if="child.household && !isEditingHousehold"
          @click="startEditingHousehold"
          class="btn btn-secondary btn-icon btn-sm"
          aria-label="Bearbeiten"
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
        <div v-if="auth.canAccessFees" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          <div>
            <p class="text-sm text-muted-foreground">Einkommensstatus</p>
            <p class="font-medium">{{ getIncomeStatusLabel(child.household.incomeStatus) }}</p>
            <router-link data-write
              v-if="!child.household.incomeStatus || child.household.incomeStatus === 'PENDING'"
              :to="`/einstufungen/neu?childId=${child.id}`"
              class="inline-flex items-center gap-1 mt-1 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
            >
              Einstufung erstellen →
            </router-link>
            <router-link data-write
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

        <div v-if="auth.canAccessFees && trustedIbans.length > 0" class="pt-3 border-t">
          <p class="text-sm text-muted-foreground mb-2">Bekannte IBANs</p>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="iban in trustedIbans"
              :key="iban.iban"
              :title="iban.payerName ? `${iban.payerName} · ${iban.iban}` : iban.iban"
              class="badge"
            >
              <span class="font-mono">{{ maskIban(iban.iban) }}</span>
              <span v-if="iban.transactionCount > 0" class="text-muted-foreground">· {{ iban.transactionCount }} Zahlungen</span>
            </span>
          </div>
        </div>

        <!-- Platzgeld (Childcare Fee) for U3 children -->
        <div v-if="auth.canAccessFees && isUnderThree(child.birthDate)" class="pt-4 border-t">
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
                <p v-if="childcareFee.discountPercent > 0" class="text-sm text-success">
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
                class="inline-flex items-center bg-primary-soft border border-primary/20 rounded-lg text-sm"
              >
                <button
                  @click="openParentDetailModal(parent)"
                  class="inline-flex items-center gap-2 px-3 py-1.5 hover:bg-primary-soft rounded-l-lg transition-colors"
                >
                  <User class="h-4 w-4 text-primary" />
                  <span>{{ parent.firstName }} {{ parent.lastName }}</span>
                </button>
                <button data-write
                  @click="confirmUnlinkParent(parent)"
                  class="p-1.5 text-primary hover:text-destructive hover:bg-destructive-soft rounded-r-lg border-l border-primary/20 transition-colors"
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
                class="inline-flex items-center gap-2 px-3 py-1.5 bg-warning-soft hover:bg-warning-soft border border-warning-line rounded-lg text-sm transition-colors"
              >
                <User class="h-4 w-4 text-warning" />
                <span>{{ sibling.firstName }} {{ sibling.lastName }}</span>
              </router-link>
            </div>
          </div>
        </div>
      </div>

      <!-- Edit Mode -->
      <form v-else @submit.prevent="saveHouseholdEdit" class="space-y-4">
        <div>
          <label for="household-name" class="label">Haushaltsname</label>
          <input
            id="household-name"
            v-model="householdEditForm.name"
            type="text"
            class="input"
          />
        </div>

        <div v-if="auth.canAccessFees">
          <label for="household-incomeStatus" class="label">Einkommensstatus</label>
          <select
            id="household-incomeStatus"
            v-model="householdEditForm.incomeStatus"
            class="input"
          >
            <option v-for="option in incomeStatusOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>

        <div v-if="auth.canAccessFees && (householdEditForm.incomeStatus === 'PROVIDED' || householdEditForm.incomeStatus === 'HISTORIC')">
          <label for="household-income" class="label">Jahreshaushaltseinkommen</label>
          <input
            id="household-income"
            v-model.number="householdEditForm.annualHouseholdIncome"
            type="number"
            min="0"
            step="any"
            class="input"
          />
        </div>

        <div v-if="auth.canAccessFees">
          <label for="household-childrenCount" class="label">Anzahl Kinder (für Beitragsberechnung)</label>
          <input
            id="household-childrenCount"
            v-model.number="householdEditForm.childrenCountForFees"
            type="number"
            min="1"
            max="10"
            placeholder="Automatisch"
            class="input"
          />
          <p class="text-xs text-muted-foreground mt-1">Leer lassen für automatische Zählung der U3-Kinder im Haushalt</p>
        </div>

        <div v-if="householdError" class="note note-danger">
          <p class="text-sm text-destructive">{{ householdError }}</p>
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button
            type="button"
            @click="cancelEditingHousehold"
            class="btn btn-secondary"
          >
            Abbrechen
          </button>
          <button
            type="submit"
            :disabled="isSavingHousehold"
            class="btn btn-primary"
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
        <button data-write
          @click="openLinkParentDialog"
          class="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-primary border border-primary hover:bg-primary/10 rounded-lg transition-colors"
        >
          <Link class="h-4 w-4" />
          Verknüpfen
        </button>
        <button data-write
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
