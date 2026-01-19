<script setup lang="ts">
import { ref, watch, computed } from 'vue';
import { Dialog, Button, Input, Label } from '@/components/ui';
import type { Group, CreateGroupRequest } from '@kita/shared';

const props = defineProps<{
  open: boolean;
  group?: Group | null;
}>();

const emit = defineEmits<{
  'update:open': [value: boolean];
  'save': [data: CreateGroupRequest];
}>();

const isEditing = computed(() => !!props.group?.id);

// Predefined color options
const colorOptions = [
  { value: '#F59E0B', name: 'Orange' },
  { value: '#6366F1', name: 'Indigo' },
  { value: '#10B981', name: 'Grün' },
  { value: '#EC4899', name: 'Pink' },
  { value: '#8B5CF6', name: 'Violet' },
  { value: '#14B8A6', name: 'Türkis' },
  { value: '#F97316', name: 'Dunkelorange' },
  { value: '#3B82F6', name: 'Blau' },
];

// Form state
const form = ref({
  name: '',
  description: '',
  color: '#10B981',
});

// Reset form when dialog opens/group changes
watch(
  () => [props.open, props.group],
  () => {
    if (props.open) {
      if (props.group) {
        form.value = {
          name: props.group.name || '',
          description: props.group.description || '',
          color: props.group.color || '#10B981',
        };
      } else {
        form.value = {
          name: '',
          description: '',
          color: '#10B981',
        };
      }
    }
  },
  { immediate: true }
);

function handleSubmit() {
  emit('save', { ...form.value });
}
</script>

<template>
  <Dialog
    :open="open"
    @update:open="emit('update:open', $event)"
    :title="isEditing ? 'Gruppe bearbeiten' : 'Neue Gruppe'"
    :description="isEditing ? 'Bearbeiten Sie die Gruppeninformationen.' : 'Erstellen Sie eine neue Gruppe.'"
  >
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div class="space-y-2">
        <Label for="name">Gruppenname</Label>
        <Input
          id="name"
          v-model="form.name"
          placeholder="z.B. Sonnenkinder"
          required
        />
      </div>

      <div class="space-y-2">
        <Label for="description">Beschreibung</Label>
        <Input
          id="description"
          v-model="form.description"
          placeholder="Optionale Beschreibung"
        />
      </div>

      <div class="space-y-2">
        <Label>Farbe</Label>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="color in colorOptions"
            :key="color.value"
            type="button"
            @click="form.color = color.value"
            class="w-8 h-8 rounded-full border-2 transition-transform hover:scale-110"
            :class="form.color === color.value ? 'border-stone-900 ring-2 ring-offset-2 ring-stone-400' : 'border-transparent'"
            :style="{ backgroundColor: color.value }"
            :title="color.name"
          />
        </div>
      </div>

      <div class="flex justify-end gap-3 pt-4">
        <Button type="button" variant="outline" @click="emit('update:open', false)">
          Abbrechen
        </Button>
        <Button type="submit">
          {{ isEditing ? 'Speichern' : 'Erstellen' }}
        </Button>
      </div>
    </form>
  </Dialog>
</template>
