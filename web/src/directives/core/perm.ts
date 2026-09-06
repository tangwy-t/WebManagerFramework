/**
 * v-perm 权限指令
 * 无权限时从 DOM 中移除元素。
 */
import type { Directive } from 'vue'
import { useUserStore } from '@/store/modules/user'

export const perm: Directive<HTMLElement, string | string[]> = {
  mounted(el, binding) {
    const perms = useUserStore().info?.permissions ?? []
    const list = Array.isArray(binding.value) ? binding.value : [binding.value]
    const ok = list.length === 0 || list.some((p) => perms.includes(p))
    if (!ok) el.parentNode?.removeChild(el)
  }
}
