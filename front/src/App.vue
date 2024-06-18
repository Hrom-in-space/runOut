<script setup>
import Panel from './components/Panel.vue'
import Listing from './components/Listing.vue'
import { needsStore } from '@/stores/counter'
import { onMounted } from "vue";

const store = needsStore()
onMounted(() => {store.allNeeds()})
</script>

<template>
  <main>
    <Listing />
  </main>
  <footer>
    <Panel />
  </footer>
</template>
