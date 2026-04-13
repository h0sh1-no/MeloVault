import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSearchQueryStore = defineStore('searchQuery', () => {
  const query = ref('')
  return { query }
})
