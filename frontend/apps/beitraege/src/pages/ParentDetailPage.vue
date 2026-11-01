<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { api } from '@/api';
import ChangeHistory from '@/components/family/ChangeHistory.vue';
import { useAuthStore } from '@/stores/auth';
import type { Parent, UpdateParentRequest } from '@/api/types';
import {
  AlertTriangle,
  Edit,
  Trash2,
  Loader2,
  User,
  Calendar,
  MapPin,
  X,
  Check,
  Mail,
  Phone,
  Users,
  UserPlus,
  ExternalLink,
} from 'lucide-vue-next';
import AppSheet from '@/components/ui/AppSheet.vue';
import EmptyState from '@/components/ui/EmptyState.vue';
import PageHeader from '@/components/ui/PageHeader.vue';
import { formatDate, formatDateForInput, todayISO } from '@/utils/format';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const parent = ref<Parent | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);

// Edit dialog state
const showEditDialog = ref(false);
const editForm = ref<UpdateParentRequest>({});
const isEditing = ref(false);
const editError = ref<string | null>(null);

// Delete dialog state
const showDeleteDialog = ref(false);
const isDeleting = ref(false);

// Member dialog state
const showMemberDialog = ref(false);
const membershipStart = ref(todayISO());
const isCreatingMember = ref(false);
const memberError = ref<string | null>(null);

const parentId = computed(() => route.params.id as string);

async function loadParent() {
  isLoading.value = true;
  error.value = null;
  try {
    parent.value = await api.getParent(parentId.value);
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Laden';
  } finally {
    isLoading.value = false;
  }
}

onMounted(loadParent);

function openEditDialog() {
  if (!parent.value) return;
  editForm.value = {
    firstName: parent.value.firstName,
    lastName: parent.value.lastName,
    birthDate: parent.value.birthDate ? formatDateForInput(parent.value.birthDate) : undefined,
    email: parent.value.email,
    phone: parent.value.phone,
    street: parent.value.street,
    streetNo: parent.value.streetNo,
    postalCode: parent.value.postalCode,
    city: parent.value.city,
  };
  editError.value = null;
  showEditDialog.value = true;
}

async function handleEdit() {
  if (!parent.value) return;
  isEditing.value = true;
  editError.value = null;
  try {
    const updated = await api.updateParent(parentId.value, editForm.value);
    parent.value = updated;
    showEditDialog.value = false;
  } catch (e) {
    editError.value = e instanceof Error ? e.message : 'Fehler beim Speichern';
  } finally {
    isEditing.value = false;
  }
}

async function handleDelete() {
  isDeleting.value = true;
  try {
    await api.deleteParent(parentId.value);
    router.push('/eltern');
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Löschen';
    showDeleteDialog.value = false;
  } finally {
    isDeleting.value = false;
  }
}

function goToChild(childId: string) {
  router.push(`/kinder/${childId}`);
}

function goToMember(memberId: string) {
  router.push(`/mitglieder/${memberId}`);
}

// Get the oldest child's entry date for suggested membership start
function getOldestChildEntryDate(): string {
  if (!parent.value?.children || parent.value.children.length === 0) {
    return todayISO();
  }

  let oldest = parent.value.children[0].entryDate;
  for (const child of parent.value.children) {
    if (child.entryDate < oldest) {
      oldest = child.entryDate;
    }
  }
  return oldest.split('T')[0];
}

function openMemberDialog() {
  // Default to oldest child's entry date
  membershipStart.value = getOldestChildEntryDate();
  memberError.value = null;
  showMemberDialog.value = true;
}

async function handleCreateMember() {
  isCreatingMember.value = true;
  memberError.value = null;
  try {
    const updated = await api.createMemberFromParent(parentId.value, membershipStart.value);
    parent.value = updated;
    showMemberDialog.value = false;
  } catch (e) {
    memberError.value = e instanceof Error ? e.message : 'Fehler beim Erstellen';
  } finally {
    isCreatingMember.value = false;
  }
}

async function handleUnlinkMember() {
  if (!parent.value?.memberId) return;
  if (!confirm('Möchten Sie die Verknüpfung zum Vereinsmitglied wirklich entfernen? Das Mitglied wird nicht gelöscht.')) return;
  try {
    const updated = await api.unlinkMemberFromParent(parentId.value);
    parent.value = updated;
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Fehler beim Entfernen der Verknüpfung';
  }
}
</script>

<template>
  <div>
    <PageHeader :title="parent ? `${parent.firstName} ${parent.lastName}` : 'Elternteil'" back="/eltern">
      <template #actions v-if="parent">
        <button data-write type="button" class="btn btn-secondary btn-icon" title="Bearbeiten" aria-label="Bearbeiten" @click="openEditDialog"><Edit /></button>
        <button data-write v-if="authStore.isAdmin" type="button" class="btn btn-danger-outline btn-icon" title="Löschen" aria-label="Löschen" @click="showDeleteDialog = true"><Trash2 /></button>
      </template>
    </PageHeader>

    <!-- Loading state -->
    <div v-if="isLoading" class="flex items-center justify-center py-12">
      <Loader2 class="h-8 w-8 animate-spin text-primary" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="note note-danger">
      <AlertTriangle />
      <div>
        <p class="text-destructive">{{ error }}</p>
      <button @click="loadParent" class="btn btn-secondary btn-sm">
        Erneut versuchen
      </button>
      </div>
    </div>

    <!-- Parent details -->
    <div v-else-if="parent">
      <!-- Header -->
      <div class="card p-6 mb-6">

        <!-- Info grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          <div v-if="parent.birthDate" class="flex items-start gap-3">
            <Calendar class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Geburtsdatum</p>
              <p class="font-medium">{{ formatDate(parent.birthDate) }}</p>
            </div>
          </div>
          <div v-if="parent.email" class="flex items-start gap-3">
            <Mail class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">E-Mail</p>
              <a :href="`mailto:${parent.email}`" class="font-medium text-primary hover:underline">
                {{ parent.email }}
              </a>
            </div>
          </div>
          <div v-if="parent.phone" class="flex items-start gap-3">
            <Phone class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Telefon</p>
              <a :href="`tel:${parent.phone}`" class="font-medium text-primary hover:underline">
                {{ parent.phone }}
              </a>
            </div>
          </div>
          <div v-if="parent.street" class="flex items-start gap-3">
            <MapPin class="h-5 w-5 text-muted-foreground mt-0.5" />
            <div>
              <p class="text-sm text-muted-foreground">Adresse</p>
              <p class="font-medium">{{ parent.street }} {{ parent.streetNo }}</p>
              <p class="text-sm text-muted-foreground">{{ parent.postalCode }} {{ parent.city }}</p>
            </div>
          </div>
        </div>

        <!-- Children Section -->
        <div class="mt-6 pt-6 border-t">
          <div class="flex items-center gap-2 mb-4">
            <Users class="h-4 w-4 text-muted-foreground" />
            <h3 class="text-sm font-medium text-muted-foreground">Verknüpfte Kinder</h3>
          </div>

          <div v-if="parent.children && parent.children.length > 0" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <button
              v-for="child in parent.children"
              :key="child.id"
              @click="goToChild(child.id)"
              class="bg-muted rounded-lg border p-4 hover:border-primary/50 hover:bg-accent transition-colors text-left"
            >
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                  <User class="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p class="font-medium text-foreground">{{ child.firstName }} {{ child.lastName }}</p>
                  <p class="text-sm text-muted-foreground font-mono">{{ child.memberNumber }}</p>
                </div>
              </div>
            </button>
          </div>

          <EmptyState v-else :icon="Users" text="Keine Kinder verknüpft" />
        </div>

        <!-- Membership Section -->
        <div class="mt-6 pt-6 border-t">
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <UserPlus class="h-4 w-4 text-muted-foreground" />
              <h3 class="text-sm font-medium text-muted-foreground">Vereinsmitgliedschaft</h3>
            </div>
          </div>

          <!-- Already a member -->
          <div v-if="parent.member" class="note note-success">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-full bg-success-soft flex items-center justify-center">
                  <UserPlus class="h-5 w-5 text-success" />
                </div>
                <div>
                  <p class="font-medium text-foreground">{{ parent.member.memberNumber }}</p>
                  <p class="text-sm text-muted-foreground">
                    Mitglied seit {{ formatDate(parent.member.membershipStart) }}
                  </p>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <button
                  @click="goToMember(parent.member!.id)"
                  class="btn btn-secondary btn-sm"
                >
                  <ExternalLink class="h-4 w-4" />
                  Details
                </button>
                <button data-write
                  @click="handleUnlinkMember"
                  class="btn btn-ghost btn-icon btn-sm"
                  title="Verknüpfung entfernen" aria-label="Verknüpfung entfernen"
                >
                  <X class="h-4 w-4" />
                </button>
              </div>
            </div>
          </div>

          <!-- Not a member yet -->
          <div v-else class="text-center py-6 bg-muted rounded-lg border border-dashed">
            <UserPlus class="h-8 w-8 text-muted-foreground mx-auto mb-2" />
            <p class="text-muted-foreground text-sm mb-3">Kein Vereinsmitglied</p>
            <button data-write
              @click="openMemberDialog"
              class="btn btn-primary"
            >
              <UserPlus class="h-4 w-4" />
              Zum Mitglied machen
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit Dialog -->
    <AppSheet v-if="showEditDialog" title="Elternteil bearbeiten" @close="showEditDialog = false">

        <form id="edit-parent-form" @submit.prevent="handleEdit" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-firstName" class="label">Vorname</label>
              <input
                id="edit-firstName"
                v-model="editForm.firstName"
                type="text"
                class="input"
              />
            </div>
            <div>
              <label for="edit-lastName" class="label">Nachname</label>
              <input
                id="edit-lastName"
                v-model="editForm.lastName"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div>
            <label for="edit-birthDate" class="label">Geburtsdatum</label>
            <input
              id="edit-birthDate"
              v-model="editForm.birthDate"
              type="date"
              class="input"
            />
          </div>

          <div>
            <label for="edit-email" class="label">E-Mail</label>
            <input
              id="edit-email"
              v-model="editForm.email"
              type="email"
              class="input"
            />
          </div>

          <div>
            <label for="edit-phone" class="label">Telefon</label>
            <input
              id="edit-phone"
              v-model="editForm.phone"
              type="tel"
              class="input"
            />
          </div>

          <div class="grid grid-cols-4 gap-4">
            <div class="col-span-3">
              <label for="edit-street" class="label">Straße</label>
              <input
                id="edit-street"
                v-model="editForm.street"
                type="text"
                class="input"
              />
            </div>
            <div>
              <label for="edit-streetNo" class="label">Hausnr.</label>
              <input
                id="edit-streetNo"
                v-model="editForm.streetNo"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div class="grid grid-cols-3 gap-4">
            <div>
              <label for="edit-postalCode" class="label">PLZ</label>
              <input
                id="edit-postalCode"
                v-model="editForm.postalCode"
                type="text"
                class="input"
              />
            </div>
            <div class="col-span-2">
              <label for="edit-city" class="label">Ort</label>
              <input
                id="edit-city"
                v-model="editForm.city"
                type="text"
                class="input"
              />
            </div>
          </div>

          <div v-if="editError" class="note note-danger">
      <AlertTriangle />
      <p class="text-sm text-destructive">{{ editError }}</p>
          </div>


        </form>

    <template #footer><div class="flex justify-end gap-3 pt-4">
            <button
              type="button"
              @click="showEditDialog = false"
              class="btn btn-secondary"
            >
              Abbrechen
            </button>
            <button form="edit-parent-form" type="submit"
              :disabled="isEditing"
              class="btn btn-primary"
            >
              <Loader2 v-if="isEditing" class="h-4 w-4 animate-spin" />
              <Check v-else class="h-4 w-4" />
              Speichern
            </button>
          </div></template></AppSheet>

    <!-- Delete Confirmation Dialog -->
    <AppSheet v-if="showDeleteDialog" title="Elternteil löschen?" @close="showDeleteDialog = false">

        <p class="text-muted-foreground mb-6">
          Möchten Sie <strong>{{ parent?.firstName }} {{ parent?.lastName }}</strong> wirklich löschen?
          Diese Aktion kann nicht rückgängig gemacht werden. Die Verknüpfungen zu Kindern werden ebenfalls entfernt.
        </p>



    <template #footer><div class="flex justify-end gap-3">
          <button
            @click="showDeleteDialog = false"
            class="btn btn-secondary"
          >
            Abbrechen
          </button>
          <button
            @click="handleDelete"
            :disabled="isDeleting"
            class="btn btn-danger"
          >
            <Loader2 v-if="isDeleting" class="h-4 w-4 animate-spin" />
            <Trash2 v-else class="h-4 w-4" />
            Löschen
          </button>
        </div></template></AppSheet>

    <!-- Create Member Dialog -->
    <AppSheet v-if="showMemberDialog" title="Vereinsmitglied erstellen" @close="showMemberDialog = false">

        <p class="text-muted-foreground mb-4">
          <strong>{{ parent?.firstName }} {{ parent?.lastName }}</strong> wird als Vereinsmitglied registriert.
          Die Kontaktdaten werden übernommen.
        </p>

        <div class="mb-6">
          <label for="membershipStart" class="label">
            Mitgliedschaft ab
          </label>
          <input
            id="membershipStart"
            v-model="membershipStart"
            type="date"
            class="input"
          />
          <p v-if="parent?.children && parent.children.length > 0" class="mt-1 text-xs text-muted-foreground">
            Vorausgefüllt mit dem Eintrittsdatum des ältesten Kindes
          </p>
        </div>

        <div v-if="memberError" class="mb-4 note note-danger">
      <AlertTriangle />
      <p class="text-sm text-destructive">{{ memberError }}</p>
        </div>



    <template #footer><div class="flex justify-end gap-3">
          <button
            @click="showMemberDialog = false"
            class="btn btn-secondary"
          >
            Abbrechen
          </button>
          <button
            @click="handleCreateMember"
            :disabled="isCreatingMember"
            class="btn btn-primary"
          >
            <Loader2 v-if="isCreatingMember" class="h-4 w-4 animate-spin" />
            <Check v-else class="h-4 w-4" />
            Erstellen
          </button>
        </div></template></AppSheet>
    <ChangeHistory v-if="authStore.canAccessFees" kind="parent" :id="String(route.params.id)" class="mt-6" />
  </div>
</template>
