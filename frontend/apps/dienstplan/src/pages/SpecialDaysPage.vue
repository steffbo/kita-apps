<script setup lang="ts">
import { ref, computed } from 'vue';
import { Loader2, Calendar, Trash2, CirclePlus } from 'lucide-vue-next';
import { 
  useSpecialDays, 
  useCreateSpecialDay, 
  useUpdateSpecialDay, 
  useDeleteSpecialDay,
  useAuth,
  type SpecialDay,
  type CreateSpecialDayRequest
} from '@kita/shared';
import { formatDate } from '@kita/shared/utils';
import { Button, Badge, Dialog, Input, Label, Select, type SelectOption } from '@/components/ui';

const { isAdmin } = useAuth();

// Current year
const currentYear = ref(new Date().getFullYear());

// Queries
const { data: specialDays, isLoading, error, refetch } = useSpecialDays({ 
  year: currentYear,
  includeHolidays: true 
});
// Holidays are included in specialDays query

// Mutations
const createSpecialDay = useCreateSpecialDay();
const updateSpecialDay = useUpdateSpecialDay();
const deleteSpecialDay = useDeleteSpecialDay();

// Dialog state
const dialogOpen = ref(false);
const selectedDay = ref<SpecialDay | null>(null);

// Form state
const form = ref({
  date: '',
  endDate: '',
  name: '',
  dayType: 'CLOSURE' as 'CLOSURE' | 'TEAM_DAY' | 'EVENT',
  affectsAll: true,
  notes: '',
});

const dayTypeOptions: SelectOption[] = [
  { value: 'CLOSURE', label: 'Schließzeit' },
  { value: 'TEAM_DAY', label: 'Teamtag / Bildungstag' },
  { value: 'EVENT', label: 'Veranstaltung' },
];

// Computed lists
const holidaysList = computed(() => 
  (specialDays.value || []).filter(d => d.dayType === 'HOLIDAY')
);

const closuresList = computed(() => 
  (specialDays.value || []).filter(d => d.dayType === 'CLOSURE')
);

const teamDaysList = computed(() => 
  (specialDays.value || []).filter(d => d.dayType === 'TEAM_DAY')
);

const eventsList = computed(() => 
  (specialDays.value || []).filter(d => d.dayType === 'EVENT')
);

function openCreateDialog(dayType: 'CLOSURE' | 'TEAM_DAY' | 'EVENT' = 'CLOSURE') {
  selectedDay.value = null;
  form.value = {
    date: '',
    endDate: '',
    name: '',
    dayType,
    affectsAll: true,
    notes: '',
  };
  dialogOpen.value = true;
}

function openEditDialog(day: SpecialDay) {
  if (day.dayType === 'HOLIDAY') return; // Can't edit holidays
  
  selectedDay.value = day;
  form.value = {
    date: day.date || '',
    endDate: day.endDate || '',
    name: day.name || '',
    dayType: (day.dayType as 'CLOSURE' | 'TEAM_DAY' | 'EVENT') || 'CLOSURE',
    affectsAll: day.affectsAll ?? true,
    notes: day.notes || '',
  };
  dialogOpen.value = true;
}

async function handleSave() {
  const data: CreateSpecialDayRequest = {
    date: form.value.date,
    endDate: form.value.endDate || undefined,
    name: form.value.name,
    dayType: form.value.dayType,
    affectsAll: form.value.affectsAll,
    notes: form.value.notes || undefined,
  };

  try {
    if (selectedDay.value?.id) {
      await updateSpecialDay.mutateAsync({ id: selectedDay.value.id, data });
    } else {
      await createSpecialDay.mutateAsync(data);
    }
    dialogOpen.value = false;
  } catch (err) {
    console.error('Failed to save special day:', err);
  }
}

async function handleDelete(day: SpecialDay) {
  if (day.dayType === 'HOLIDAY' || !day.id) return;
  
  if (confirm(`"${day.name}" wirklich löschen?`)) {
    try {
      await deleteSpecialDay.mutateAsync(day.id);
    } catch (err) {
      console.error('Failed to delete special day:', err);
    }
  }
}

function previousYear() {
  currentYear.value--;
}

function nextYear() {
  currentYear.value++;
}

// Format date range for display
function formatDateRange(day: SpecialDay): string {
  const start = formatDate(day.date || '');
  if (day.endDate && day.endDate !== day.date) {
    return `${start} - ${formatDate(day.endDate)}`;
  }
  return start;
}


</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Besondere Tage</h1>
        <p class="text-stone-600">Feiertage, Schließzeiten und Veranstaltungen</p>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="icon" @click="previousYear">
          <span class="sr-only">Vorheriges Jahr</span>
          &lt;
        </Button>
        <span class="font-semibold text-lg px-4">{{ currentYear }}</span>
        <Button variant="outline" size="icon" @click="nextYear">
          <span class="sr-only">Nächstes Jahr</span>
          &gt;
        </Button>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="w-8 h-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-destructive/10 text-destructive rounded-lg p-4">
      <p>Fehler beim Laden: {{ (error as Error).message }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">
        Erneut versuchen
      </Button>
    </div>

    <!-- Content -->
    <div v-else class="space-y-6">
      <!-- Holidays Section -->
      <div class="bg-white rounded-lg border border-stone-200 p-6">
        <h2 class="text-lg font-semibold text-stone-900 mb-4 flex items-center gap-2">
          <Calendar class="w-5 h-5 text-red-500" />
          Feiertage Brandenburg
        </h2>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="day in holidaysList"
            :key="day.id"
            class="flex items-center justify-between p-3 bg-red-50 rounded-lg"
          >
            <div>
              <div class="font-medium text-stone-900">{{ day.name }}</div>
              <div class="text-sm text-stone-500">{{ formatDate(day.date || '') }}</div>
            </div>
          </div>
        </div>
        <p v-if="!holidaysList.length" class="text-stone-500">
          Keine Feiertage für {{ currentYear }} gefunden.
        </p>
      </div>

      <!-- Closures Section -->
      <div class="bg-white rounded-lg border border-stone-200 p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-stone-900 flex items-center gap-2">
            <Calendar class="w-5 h-5 text-orange-500" />
            Schließzeiten
          </h2>
          <Button
            v-if="isAdmin"
            variant="ghost"
            size="icon"
            aria-label="Neuer Eintrag"
            @click="openCreateDialog('CLOSURE')"
          >
            <CirclePlus class="w-5 h-5 text-orange-500" />
          </Button>
        </div>
        <div class="space-y-2">
          <div
            v-for="day in closuresList"
            :key="day.id"
            class="flex items-center justify-between p-3 bg-orange-50 rounded-lg hover:bg-orange-100 cursor-pointer"
            @click="openEditDialog(day)"
          >
            <div>
              <div class="font-medium text-stone-900">{{ day.name }}</div>
              <div class="text-sm text-stone-500">{{ formatDateRange(day) }}</div>
            </div>
            <div class="flex items-center gap-2">
              <Badge class="bg-orange-100 text-orange-700" variant="outline">Schließzeit</Badge>
              <Button 
                v-if="isAdmin"
                variant="ghost" 
                size="icon" 
                aria-label="Löschen"
                @click.stop="handleDelete(day)"
              >
                <Trash2 class="w-4 h-4 text-destructive" />
              </Button>
            </div>
          </div>
        </div>
        <p v-if="!closuresList.length" class="text-stone-500">
          Keine Schließzeiten für {{ currentYear }} eingetragen.
        </p>
      </div>

      <!-- Team Days Section -->
      <div class="bg-white rounded-lg border border-stone-200 p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-stone-900 flex items-center gap-2">
            <Calendar class="w-5 h-5 text-purple-500" />
            Teamtage / Bildungstage
          </h2>
          <Button
            v-if="isAdmin"
            variant="ghost"
            size="icon"
            aria-label="Neuer Eintrag"
            @click="openCreateDialog('TEAM_DAY')"
          >
            <CirclePlus class="w-5 h-5 text-purple-500" />
          </Button>
        </div>
        <div class="space-y-2">
          <div
            v-for="day in teamDaysList"
            :key="day.id"
            class="flex items-center justify-between p-3 bg-purple-50 rounded-lg hover:bg-purple-100 cursor-pointer"
            @click="openEditDialog(day)"
          >
            <div>
              <div class="font-medium text-stone-900">{{ day.name }}</div>
              <div class="text-sm text-stone-500">{{ formatDateRange(day) }}</div>
            </div>
            <div class="flex items-center gap-2">
              <Badge class="bg-purple-100 text-purple-700" variant="outline">Teamtag</Badge>
              <Button 
                v-if="isAdmin"
                variant="ghost" 
                size="icon" 
                aria-label="Löschen"
                @click.stop="handleDelete(day)"
              >
                <Trash2 class="w-4 h-4 text-destructive" />
              </Button>
            </div>
          </div>
        </div>
        <p v-if="!teamDaysList.length" class="text-stone-500">
          Keine Teamtage für {{ currentYear }} eingetragen.
        </p>
      </div>

      <!-- Events Section -->
      <div class="bg-white rounded-lg border border-stone-200 p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-stone-900 flex items-center gap-2">
            <Calendar class="w-5 h-5 text-blue-500" />
            Veranstaltungen
          </h2>
          <Button
            v-if="isAdmin"
            variant="ghost"
            size="icon"
            aria-label="Neuer Eintrag"
            @click="openCreateDialog('EVENT')"
          >
            <CirclePlus class="w-5 h-5 text-blue-500" />
          </Button>
        </div>
        <div class="space-y-2">
          <div
            v-for="day in eventsList"
            :key="day.id"
            class="flex items-center justify-between p-3 bg-blue-50 rounded-lg hover:bg-blue-100 cursor-pointer"
            @click="openEditDialog(day)"
          >
            <div>
              <div class="font-medium text-stone-900">{{ day.name }}</div>
              <div class="text-sm text-stone-500">{{ formatDateRange(day) }}</div>
              <div v-if="day.notes" class="text-xs text-stone-400 mt-1">{{ day.notes }}</div>
            </div>
            <div class="flex items-center gap-2">
              <Badge class="bg-blue-100 text-blue-700" variant="outline">Veranstaltung</Badge>
              <Button 
                v-if="isAdmin"
                variant="ghost" 
                size="icon" 
                aria-label="Löschen"
                @click.stop="handleDelete(day)"
              >
                <Trash2 class="w-4 h-4 text-destructive" />
              </Button>
            </div>
          </div>
        </div>
        <p v-if="!eventsList.length" class="text-stone-500">
          Keine Veranstaltungen für {{ currentYear }} eingetragen.
        </p>
      </div>
    </div>

    <!-- Create/Edit Dialog -->
    <Dialog
      v-model:open="dialogOpen"
      :title="selectedDay ? 'Eintrag bearbeiten' : 'Neuer Eintrag'"
    >
      <form @submit.prevent="handleSave" class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="date">Startdatum</Label>
            <Input
              id="date"
              v-model="form.date"
              type="date"
              required
            />
          </div>
          <div class="space-y-2">
            <Label for="endDate">Enddatum (optional)</Label>
            <Input
              id="endDate"
              v-model="form.endDate"
              type="date"
              :min="form.date"
            />
          </div>
        </div>

        <div class="space-y-2">
          <Label for="name">Bezeichnung</Label>
          <Input
            id="name"
            v-model="form.name"
            placeholder="z.B. Sommerschließzeit"
            required
          />
        </div>

        <div class="space-y-2">
          <Label for="dayType">Typ</Label>
          <Select
            v-model="form.dayType"
            :options="dayTypeOptions"
          />
        </div>

        <div class="space-y-2">
          <Label for="notes">Notizen (optional)</Label>
          <Input
            id="notes"
            v-model="form.notes"
            placeholder="Zusätzliche Informationen"
          />
        </div>

        <div class="flex justify-end gap-3 pt-4">
          <Button type="button" variant="outline" @click="dialogOpen = false">
            Abbrechen
          </Button>
          <Button 
            type="submit"
            :disabled="createSpecialDay.isPending.value || updateSpecialDay.isPending.value"
          >
            <Loader2 
              v-if="createSpecialDay.isPending.value || updateSpecialDay.isPending.value" 
              class="w-4 h-4 mr-2 animate-spin" 
            />
            {{ selectedDay ? 'Speichern' : 'Erstellen' }}
          </Button>
        </div>
      </form>
    </Dialog>
  </div>
</template>
