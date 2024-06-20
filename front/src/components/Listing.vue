<script setup>
import { needsStore } from '@/stores/counter'

const store = needsStore()
</script>

<template>
  <div v-if="store.items.length > 0" class="menu-container container">
    <menu class="open-menu">
      <div class="open-menu-title">Список покупок</div>
      <ul class="open-menu-list">
        <li v-for="item in store.items" :key="item.id" class="open-menu-list-li">
          <div class="open-menu-list-li-text">{{ item.name }}</div>
          <button @click="store.delNeed(item.id)" class="open-menu-list-li-close">
            <img
              class="open-menu-list-li-close-img"
              src="/icons/close.svg"
              alt=""
            />
          </button>
        </li>
      </ul>
    </menu>
  </div>

  <div v-else class="basket-container container">
    <div class="basket-block">
      <img
        class="basket-block-image"
        src="/image/basket.svg"
        alt="Корзина"
      />
    </div>
    <div class="descr">
      Привет, Артур
      <br />
      Что нужно купить?
    </div>
  </div>

</template>
