<script setup>
import { needsStore } from '@/stores/counter'

const store = needsStore()
</script>

<template>
  <ul>
    <li v-for="item in store.items" :key="item.id">
      {{ item.name }}
      <button @click="store.delNeed(item.id)" class="send-button" >Del</button>
    </li>
  </ul>
</template>
