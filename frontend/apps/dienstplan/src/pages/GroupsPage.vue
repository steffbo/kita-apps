<script setup lang="ts">
import { Plus } from 'lucide-vue-next';

const groups = [
  { id: 1, name: 'Sonnenkinder', description: 'Gruppe 1', color: '#F59E0B', memberCount: 2 },
  { id: 2, name: 'Mondkinder', description: 'Gruppe 2', color: '#6366F1', memberCount: 2 },
  { id: 3, name: 'Sternenkinder', description: 'Gruppe 3', color: '#10B981', memberCount: 2 },
];
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Gruppen</h1>
        <p class="text-stone-600">Verwalten Sie die Kita-Gruppen und deren Zuordnungen</p>
      </div>
      <button class="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md hover:bg-green-700">
        <Plus class="w-4 h-4" />
        Neue Gruppe
      </button>
    </div>

    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="group in groups"
        :key="group.id"
        class="bg-white rounded-lg border border-stone-200 p-6 hover:shadow-md transition-shadow"
      >
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-lg flex items-center justify-center"
              :style="{ backgroundColor: group.color + '20' }"
            >
              <div
                class="w-4 h-4 rounded-full"
                :style="{ backgroundColor: group.color }"
              />
            </div>
            <div>
              <h3 class="font-semibold text-stone-900">{{ group.name }}</h3>
              <p class="text-sm text-stone-500">{{ group.description }}</p>
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between pt-4 border-t border-stone-200">
          <span class="text-sm text-stone-600">
            {{ group.memberCount }} Mitarbeiter zugeordnet
          </span>
          <button class="text-sm text-green-600 hover:text-green-700 font-medium">
            Bearbeiten
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
