import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export const needsStore = defineStore('counter', () => {
  const items = ref([])
  function clearNeeds() {
    fetch("/api/needs", {
      method: 'DELETE',
    })
        .then(response => {
          if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
          }
          items.value = [];
        })
        .then(data => console.log(data))
        .catch(error => console.error('There was an error with the fetch operation: ', error));
  }
  function allNeeds() {
    try {
      // Fetch the data from the URL
      fetch("/api/needs")
          .then(response => response.json())
          .then(data => {items.value = data})
    } catch (error) {
      console.error('There was an error with the fetch operation: ', error);
    }
  }
  function delNeed(id) {
    fetch('/api/needs/' + id, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json', // Тип содержимого, которое мы отправляем
      },
    })
        .then(response => {
          if (response.status === 200) {
            items.value = items.value.filter(item => item.id !== id)
          }
        })
        .catch((error) => {
          console.error('Error:', error); // Логируем возможную ошибку
        });
  }

  return { items, clearNeeds, allNeeds, delNeed }
})
