import { defineStore } from 'pinia'
import { ref } from 'vue'

/** 离开搜索页时先播“收合”动画，再跳转。 */
export const useSearchTransitionStore = defineStore('searchTransition', () => {
  const leaving = ref(false)
  const delayedPush = ref(false)
  const pendingTo = ref(null)
  return { leaving, delayedPush, pendingTo }
})
