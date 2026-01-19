<script setup lang="ts">
import { ref } from 'vue';
import { Plus, Loader2, Users } from 'lucide-vue-next';
import { 
  useGroups, 
  useCreateGroup, 
  useUpdateGroup, 
  useDeleteGroup,
  useAuth,
  type Group,
  type CreateGroupRequest
} from '@kita/shared';
import { Button } from '@/components/ui';
import GroupFormDialog from '@/components/GroupFormDialog.vue';

const { isAdmin } = useAuth();

// Queries
const { data: groups, isLoading, error, refetch } = useGroups();

// Mutations
const createGroup = useCreateGroup();
const updateGroup = useUpdateGroup();
const deleteGroup = useDeleteGroup();

// Dialog state
const dialogOpen = ref(false);
const selectedGroup = ref<Group | null>(null);
const showDeleteConfirm = ref(false);
const groupToDelete = ref<Group | null>(null);

// Expanded group for showing members
const expandedGroupId = ref<number | null>(null);

function openCreateDialog() {
  selectedGroup.value = null;
  dialogOpen.value = true;
}

function openEditDialog(group: Group) {
  selectedGroup.value = group;
  dialogOpen.value = true;
}

function toggleExpanded(groupId: number) {
  expandedGroupId.value = expandedGroupId.value === groupId ? null : groupId;
}

async function handleSave(data: CreateGroupRequest) {
  try {
    if (selectedGroup.value?.id) {
      await updateGroup.mutateAsync({
        id: selectedGroup.value.id,
        data,
      });
    } else {
      await createGroup.mutateAsync(data);
    }
    dialogOpen.value = false;
  } catch (err) {
    console.error('Failed to save group:', err);
  }
}

function confirmDelete(group: Group) {
  groupToDelete.value = group;
  showDeleteConfirm.value = true;
}

async function handleDelete() {
  if (!groupToDelete.value?.id) return;
  
  try {
    await deleteGroup.mutateAsync(groupToDelete.value.id);
    showDeleteConfirm.value = false;
    groupToDelete.value = null;
  } catch (err) {
    console.error('Failed to delete group:', err);
  }
}

// Member count would need individual group queries with members
// For simplicity, showing group info without member count for now
function getMemberCount(_groupId: number): string {
  return '-'; // Would need useGroupAssignments per group
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-stone-900">Gruppen</h1>
        <p class="text-stone-600">Verwalten Sie die Kita-Gruppen und deren Zuordnungen</p>
      </div>
      <Button v-if="isAdmin" @click="openCreateDialog">
        <Plus class="w-4 h-4 mr-2" />
        Neue Gruppe
      </Button>
    </div>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="w-8 h-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="bg-destructive/10 text-destructive rounded-lg p-4">
      <p>Fehler beim Laden der Gruppen: {{ (error as Error).message }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">
        Erneut versuchen
      </Button>
    </div>

    <!-- Empty state -->
    <div v-else-if="!groups?.length" class="text-center py-12 bg-white rounded-lg border border-stone-200">
      <p class="text-stone-600">Noch keine Gruppen vorhanden.</p>
      <Button v-if="isAdmin" class="mt-4" @click="openCreateDialog">
        <Plus class="w-4 h-4 mr-2" />
        Erste Gruppe erstellen
      </Button>
    </div>

    <!-- Groups grid -->
    <div v-else class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="group in groups"
        :key="group.id"
        class="bg-white rounded-lg border border-stone-200 p-6 hover:shadow-md transition-shadow"
      >
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <div
              class="w-10 h-10 rounded-lg flex items-center justify-center"
              :style="{ backgroundColor: (group.color || '#10B981') + '20' }"
            >
              <div
                class="w-4 h-4 rounded-full"
                :style="{ backgroundColor: group.color || '#10B981' }"
              />
            </div>
            <div>
              <h3 class="font-semibold text-stone-900">{{ group.name }}</h3>
              <p v-if="group.description" class="text-sm text-stone-500">{{ group.description }}</p>
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between pt-4 border-t border-stone-200">
          <button
            @click="toggleExpanded(group.id!)"
            class="flex items-center gap-1 text-sm text-stone-600 hover:text-stone-900"
          >
            <Users class="w-4 h-4" />
            <span>{{ getMemberCount(group.id!) }} Mitarbeiter</span>
          </button>
          <div v-if="isAdmin" class="flex items-center gap-2">
            <Button variant="ghost" size="sm" @click="openEditDialog(group)">
              Bearbeiten
            </Button>
            <Button 
              variant="ghost" 
              size="sm"
              class="text-destructive hover:text-destructive"
              @click="confirmDelete(group)"
            >
              Löschen
            </Button>
          </div>
        </div>

        <!-- Expanded member list -->
        <div
          v-if="expandedGroupId === group.id"
          class="mt-4 pt-4 border-t border-stone-200"
        >
          <h4 class="text-sm font-medium text-stone-700 mb-2">Zugeordnete Mitarbeiter</h4>
          <div class="text-sm text-stone-500">
            <!-- This would show the actual members from API -->
            <p>Mitarbeiter werden aus der API geladen...</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Group Form Dialog -->
    <GroupFormDialog
      v-model:open="dialogOpen"
      :group="selectedGroup"
      @save="handleSave"
    />

    <!-- Delete Confirmation Dialog -->
    <Teleport to="body">
      <div
        v-if="showDeleteConfirm"
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div 
          class="fixed inset-0 bg-black/50" 
          @click="showDeleteConfirm = false"
        />
        <div class="relative bg-white rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
          <h3 class="text-lg font-semibold mb-2">Gruppe löschen?</h3>
          <p class="text-stone-600 mb-4">
            Möchten Sie die Gruppe <strong>{{ groupToDelete?.name }}</strong> wirklich löschen?
            Alle Zuordnungen werden entfernt.
          </p>
          <div class="flex justify-end gap-3">
            <Button variant="outline" @click="showDeleteConfirm = false">
              Abbrechen
            </Button>
            <Button 
              variant="destructive" 
              @click="handleDelete"
              :disabled="deleteGroup.isPending.value"
            >
              <Loader2 v-if="deleteGroup.isPending.value" class="w-4 h-4 mr-2 animate-spin" />
              Löschen
            </Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
