<script setup lang="ts">
import { computed, toRefs } from 'vue';
import type { CareHoursHistoryEntry, Child, LegalHoursHistoryEntry } from '@/api/types';
import ChildHoursHistory from '@/components/child/ChildHoursHistory.vue';
import { Edit, Trash2, User, Calendar, MapPin, Clock } from 'lucide-vue-next';
import { formatDate } from '@/utils/format';
import { calculateAge, formatCareHours, isUnderThree } from '@/utils/child';
import { useAuthStore } from '@/stores/auth';

// Deleting a child also deletes its fee data, so Leitung does not get the button.
const auth = useAuthStore();

const props = defineProps<{
  child: Child;
  careHoursHistory: CareHoursHistoryEntry[];
  legalHoursHistory: LegalHoursHistoryEntry[];
  upcomingCareHours: CareHoursHistoryEntry | null;
  upcomingLegalHours: LegalHoursHistoryEntry | null;
  showEditDialog: boolean;
  showDeleteDialog: boolean;
  formatHistoryRange: (entry: { effectiveFrom: string; effectiveUntil?: string | null }) => string;
}>();

const {
  child,
  careHoursHistory,
  legalHoursHistory,
  upcomingCareHours,
  upcomingLegalHours,
  formatHistoryRange,
} = toRefs(props);

const emit = defineEmits<{
  'update:showEditDialog': [value: boolean];
  'update:showDeleteDialog': [value: boolean];
}>();
const showEditDialog = computed({
  get: () => props.showEditDialog,
  set: (value) => emit('update:showEditDialog', value),
});
const showDeleteDialog = computed({
  get: () => props.showDeleteDialog,
  set: (value) => emit('update:showDeleteDialog', value),
});
</script>

<template>
  <!-- Header -->
  <div class="card p-6 mb-6">
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
          <User class="h-8 w-8 text-primary" />
        </div>
        <div>
          <p class="text-base font-semibold text-foreground">
            {{ child.firstName }} {{ child.lastName }}
          </p>
          <p class="text-muted-foreground font-mono">Mitglieds-Nr. {{ child.memberNumber }}</p>
          <div class="flex items-center gap-2 mt-2">
            <span
              :class="[
                'badge',
                child.isActive ? 'badge-success' : '',
              ]"
            >
              {{ child.isActive ? 'Aktiv' : 'Inaktiv' }}
            </span>
            <span
              v-if="isUnderThree(child.birthDate)"
              class="badge badge-warning"
            >
              U3
            </span>
          </div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <button
          @click="showEditDialog = true"
          class="btn btn-secondary btn-icon btn-sm"
          aria-label="Bearbeiten"
          title="Bearbeiten"
        >
          <Edit class="h-5 w-5" />
        </button>
        <button
          v-if="auth.canAccessFees"
          @click="showDeleteDialog = true"
          class="btn btn-danger-outline btn-icon btn-sm"
          aria-label="Löschen"
          title="Löschen"
        >
          <Trash2 class="h-5 w-5" />
        </button>
      </div>
    </div>

    <!-- Info grid -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mt-6 pt-6 border-t">
      <div class="flex items-start gap-3">
        <Calendar class="h-5 w-5 text-muted-foreground mt-0.5" />
        <div>
          <p class="text-sm text-muted-foreground">Geburtsdatum</p>
          <p class="font-medium">{{ formatDate(child.birthDate) }}</p>
          <p class="text-sm text-muted-foreground">{{ calculateAge(child.birthDate) }} Jahre alt</p>
        </div>
      </div>
      <div class="flex items-start gap-3">
        <Calendar class="h-5 w-5 text-muted-foreground mt-0.5" />
        <div>
          <p class="text-sm text-muted-foreground">Eintrittsdatum</p>
          <p class="font-medium">{{ formatDate(child.entryDate) }}</p>
        </div>
      </div>
      <div v-if="child.exitDate" class="flex items-start gap-3">
        <Calendar class="h-5 w-5 text-muted-foreground mt-0.5" />
        <div>
          <p class="text-sm text-muted-foreground">Austrittsdatum</p>
          <p class="font-medium">{{ formatDate(child.exitDate) }}</p>
        </div>
      </div>
      <div v-if="child.street" class="flex items-start gap-3">
        <MapPin class="h-5 w-5 text-muted-foreground mt-0.5" />
        <div>
          <p class="text-sm text-muted-foreground">Adresse</p>
          <p class="font-medium">{{ child.street }} {{ child.streetNo }}</p>
          <p class="text-sm text-muted-foreground">{{ child.postalCode }} {{ child.city }}</p>
        </div>
      </div>
      <div v-if="child.legalHours || child.careHours || legalHoursHistory.length > 0 || careHoursHistory.length > 0" class="flex items-start gap-3">
        <Clock class="h-5 w-5 text-muted-foreground mt-0.5" />
        <div>
          <p class="text-sm text-muted-foreground">Betreuungszeiten</p>
          <p class="font-medium">
            Rechtsanspruch:
            <template v-if="upcomingLegalHours">
              ab {{ formatDate(upcomingLegalHours.effectiveFrom) }}: {{ formatCareHours(upcomingLegalHours.legalHours) }}
            </template>
            <template v-else>
              {{ formatCareHours(child.legalHours) }}
              <span v-if="child.legalHoursUntil" class="text-sm text-muted-foreground">
                (bis {{ formatDate(child.legalHoursUntil) }})
              </span>
            </template>
          </p>
          <p class="font-medium">
            Betreuungszeit:
            <template v-if="upcomingCareHours">
              ab {{ formatDate(upcomingCareHours.effectiveFrom) }}: {{ formatCareHours(upcomingCareHours.careHours) }}
            </template>
            <template v-else>{{ formatCareHours(child.careHours) }}</template>
          </p>
          <ChildHoursHistory
            :careHoursHistory="careHoursHistory"
            :legalHoursHistory="legalHoursHistory"
            :formatHistoryRange="formatHistoryRange"
          />

        </div>
      </div>
    </div>

  </div>
</template>
