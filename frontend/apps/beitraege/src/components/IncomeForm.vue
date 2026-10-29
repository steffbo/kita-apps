<script setup lang="ts">
import type { IncomeDetails } from '@/api/types';

const model = defineModel<IncomeDetails>({ required: true });

interface FieldDef {
  key: keyof IncomeDetails;
  label: string;
  sign: '+' | '-';
  hint?: string;
}

const employeeFields: FieldDef[] = [
  { key: 'grossIncome', label: 'Bruttoeinkommen', sign: '+' },
  { key: 'socialSecurityShare', label: 'AN-Anteile Sozialversicherung', sign: '-' },
  { key: 'privateInsurance', label: 'Private KV/PV', sign: '-' },
  { key: 'tax', label: 'Lohnsteuer / KiSt / SolZu', sign: '-' },
  { key: 'advertisingCosts', label: 'Werbungskosten-Pauschale', sign: '-' },
];

const otherIncomeFields: FieldDef[] = [
  { key: 'minijobIncome', label: 'Minijob', sign: '+' },
  { key: 'unemploymentBenefit', label: 'Arbeitslosengeld', sign: '+' },
  { key: 'capitalIncome', label: 'Kapitalerträge', sign: '+' },
  { key: 'rentalIncome', label: 'Vermietung / Verpachtung', sign: '+' },
  { key: 'otherIncome', label: 'Sonstiges', sign: '+' },
];

const selfEmployedFields: FieldDef[] = [
  { key: 'profit', label: 'Gewinn (Gewerbe / selbst. Arbeit)', sign: '+' },
  { key: 'welfareExpense', label: 'Abgabe pers. Daseinsfürsorge', sign: '-' },
  { key: 'selfEmployedTax', label: 'Steuern (ESt, KiSt, SolZu)', sign: '-' },
];

const benefitFields: FieldDef[] = [
  { key: 'parentalBenefit', label: 'Basiselterngeld', sign: '+', hint: 'beitragsrelevant' },
  { key: 'parentalBenefitPlus', label: 'Elterngeld Plus', sign: '+', hint: 'beitragsrelevant' },
  { key: 'maternityBenefit', label: 'Mutterschaftsgeld', sign: '+', hint: 'beitragsrelevant' },
  { key: 'insurances', label: 'Versicherungen', sign: '-' },
];

const maintenanceFields: FieldDef[] = [
  { key: 'maintenanceToPay', label: 'Unterhalt (zu zahlen)', sign: '-' },
  { key: 'maintenanceReceived', label: 'Unterhalt (erhalten)', sign: '+' },
];

function updateField(key: keyof IncomeDetails, event: Event) {
  const target = event.target as HTMLInputElement;
  const val = parseFloat(target.value) || 0;
  model.value = { ...model.value, [key]: val };
}

</script>

<template>
  <div class="space-y-4">
    <!-- Employee income -->
    <div>
      <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
        Einkommen aus nichtselbständiger Arbeit
      </h4>
      <div class="space-y-2">
        <div v-for="field in employeeFields" :key="field.key" class="flex items-center gap-2">
          <span
            class="w-5 text-center text-xs font-semibold"
            :class="field.sign === '+' ? 'text-success' : 'text-destructive'">
            {{ field.sign }}
          </span>
          <label class="label mb-0 min-w-0 flex-1 font-normal text-muted-foreground">{{ field.label }}</label>
          <input
            type="number"
            step="0.01"
            min="0"
            :value="model[field.key]"
            :aria-label="field.label"
            @input="updateField(field.key, $event)"
            class="input w-28 shrink-0 sm:w-32 text-right tabular-nums" />
        </div>
      </div>
    </div>
    <!-- Other income -->
    <div>
      <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">Sonstige Einnahmen</h4>
      <div class="space-y-2">
        <div v-for="field in otherIncomeFields" :key="field.key" class="flex items-center gap-2">
          <span
            class="w-5 text-center text-xs font-semibold"
            :class="field.sign === '+' ? 'text-success' : 'text-destructive'">
            {{ field.sign }}
          </span>
          <label class="label mb-0 min-w-0 flex-1 font-normal text-muted-foreground">{{ field.label }}</label>
          <input
            type="number"
            step="0.01"
            min="0"
            :value="model[field.key]"
            :aria-label="field.label"
            @input="updateField(field.key, $event)"
            class="input w-28 shrink-0 sm:w-32 text-right tabular-nums" />
        </div>
      </div>
    </div>
    <!-- Self-employed -->
    <div>
      <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
        Einkommen aus selbständiger Arbeit
      </h4>
      <div class="space-y-2">
        <div v-for="field in selfEmployedFields" :key="field.key" class="flex items-center gap-2">
          <span
            class="w-5 text-center text-xs font-semibold"
            :class="field.sign === '+' ? 'text-success' : 'text-destructive'">
            {{ field.sign }}
          </span>
          <label class="label mb-0 min-w-0 flex-1 font-normal text-muted-foreground">{{ field.label }}</label>
          <input
            type="number"
            step="0.01"
            min="0"
            :value="model[field.key]"
            :aria-label="field.label"
            @input="updateField(field.key, $event)"
            class="input w-28 shrink-0 sm:w-32 text-right tabular-nums" />
        </div>
      </div>
    </div>
    <!-- Benefits -->
    <div>
      <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">Leistungen</h4>
      <div class="space-y-2">
        <div v-for="field in benefitFields" :key="field.key" class="flex items-center gap-2">
          <span
            class="w-5 text-center text-xs font-semibold"
            :class="field.sign === '+' ? 'text-success' : 'text-destructive'">
            {{ field.sign }}
          </span>
          <div class="min-w-0 flex-1">
            <span class="text-sm text-muted-foreground">{{ field.label }}</span>
            <span v-if="field.hint" class="ml-1 text-xs text-warning">({{ field.hint }})</span>
          </div>
          <input
            type="number"
            step="0.01"
            min="0"
            :value="model[field.key]"
            :aria-label="field.label"
            @input="updateField(field.key, $event)"
            class="input w-28 shrink-0 sm:w-32 text-right tabular-nums" />
        </div>
      </div>
    </div>
    <!-- Maintenance -->
    <div>
      <h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">Unterhalt</h4>
      <div class="space-y-2">
        <div v-for="field in maintenanceFields" :key="field.key" class="flex items-center gap-2">
          <span
            class="w-5 text-center text-xs font-semibold"
            :class="field.sign === '+' ? 'text-success' : 'text-destructive'">
            {{ field.sign }}
          </span>
          <label class="label mb-0 min-w-0 flex-1 font-normal text-muted-foreground">{{ field.label }}</label>
          <input
            type="number"
            step="0.01"
            min="0"
            :value="model[field.key]"
            :aria-label="field.label"
            @input="updateField(field.key, $event)"
            class="input w-28 shrink-0 sm:w-32 text-right tabular-nums" />
        </div>
      </div>
    </div>
  </div>
</template>
