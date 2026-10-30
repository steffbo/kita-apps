<script setup lang="ts">
import { computed, ref, watch, type Component } from 'vue';
import { useRouter } from 'vue-router';
import { Baby, CornerDownLeft, Search, UserCircle } from 'lucide-vue-next';
import { api } from '@/api';
import { useAuthStore } from '@/stores/auth';
import { useNavigation } from '@/composables/useNavigation';
import AppSheet from '@/components/ui/AppSheet.vue';

const emit = defineEmits<{ close: [] }>();
const router = useRouter();
const auth = useAuthStore();
const { allItems } = useNavigation();

interface Hit { key: string; group: string; label: string; detail?: string; to: string; icon: Component }

const query = ref('');
const people = ref<Hit[]>([]);
const loading = ref(false);
const selected = ref(0);

const pages = computed<Hit[]>(() => {
  const q = query.value.trim().toLowerCase();
  return allItems.value
    .filter(item => !q || item.name.toLowerCase().includes(q))
    .map(item => ({ key: `page:${item.name}`, group: 'Seiten', label: item.name, to: item.to, icon: item.icon }));
});

const hits = computed(() => [...pages.value, ...people.value]);

let timer: ReturnType<typeof setTimeout> | undefined;
let request = 0;
watch(query, (value) => {
  selected.value = 0;
  clearTimeout(timer);
  const q = value.trim();
  if (q.length < 2 || !auth.canAccessMasterData) {
    people.value = [];
    loading.value = false;
    return;
  }
  loading.value = true;
  timer = setTimeout(async () => {
    const current = ++request;
    const [children, parents] = await Promise.allSettled([
      api.getChildren({ search: q, perPage: 6 }),
      api.getParents({ search: q, perPage: 6 }),
    ]);
    if (current !== request) return;
    people.value = [
      ...(children.status === 'fulfilled' ? children.value.data : []).map(child => ({
        key: `child:${child.id}`, group: 'Kinder', label: `${child.firstName} ${child.lastName}`,
        detail: child.memberNumber || undefined, to: `/kinder/${child.id}`, icon: Baby,
      })),
      ...(parents.status === 'fulfilled' ? parents.value.data : []).map(parent => ({
        key: `parent:${parent.id}`, group: 'Eltern', label: `${parent.firstName} ${parent.lastName}`,
        detail: parent.email || undefined, to: `/eltern/${parent.id}`, icon: UserCircle,
      })),
    ];
    loading.value = false;
  }, 200);
});

function open(hit: Hit | undefined) {
  if (!hit) return;
  emit('close');
  router.push(hit.to);
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') selected.value = Math.min(selected.value + 1, hits.value.length - 1);
  else if (event.key === 'ArrowUp') selected.value = Math.max(selected.value - 1, 0);
  else if (event.key === 'Enter') open(hits.value[selected.value]);
  else return;
  event.preventDefault();
  document.getElementById(`hit-${selected.value}`)?.scrollIntoView({ block: 'nearest' });
}
</script>

<template>
  <AppSheet title="Suchen" hide-title @close="emit('close')">
    <div class="flex h-11 items-center gap-2 rounded-lg border bg-background px-3 focus-within:ring-2 focus-within:ring-ring">
      <Search class="h-4 w-4 shrink-0 text-muted-foreground" />
      <input v-model="query" type="search" autofocus aria-label="Suchen" placeholder="Seite, Kind oder Elternteil"
        role="combobox" aria-controls="search-hits" :aria-expanded="hits.length > 0"
        :aria-activedescendant="hits.length ? `hit-${selected}` : undefined"
        class="min-w-0 flex-1 border-0 bg-transparent p-0 outline-none focus:ring-0" @keydown="onKeydown" />
    </div>
    <ul id="search-hits" role="listbox" aria-label="Treffer" class="mt-3 min-h-[12rem]">
      <template v-for="(hit, index) in hits" :key="hit.key">
        <li v-if="index === 0 || hits[index - 1].group !== hit.group" role="presentation"
          class="px-2 pb-1 pt-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground first:pt-0">
          {{ hit.group }}
        </li>
        <li :id="`hit-${index}`" role="option" :aria-selected="index === selected"
          class="flex h-11 cursor-pointer items-center gap-3 rounded-lg px-2 text-sm"
          :class="index === selected ? 'bg-accent' : ''" @mousemove="selected = index" @click="open(hit)">
          <component :is="hit.icon" class="h-4 w-4 shrink-0 text-muted-foreground" />
          <span class="truncate">{{ hit.label }}</span>
          <span v-if="hit.detail" class="truncate text-muted-foreground">{{ hit.detail }}</span>
          <CornerDownLeft v-if="index === selected" class="ml-auto hidden h-4 w-4 text-muted-foreground sm:block" />
        </li>
      </template>
      <li v-if="!hits.length && !loading" class="px-2 py-6 text-center text-sm text-muted-foreground">Keine Treffer</li>
    </ul>
  </AppSheet>
</template>
