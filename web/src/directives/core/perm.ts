/**
 * v-perm 权限指令
 * 无权限时从 DOM 中移除元素。
 *
 * 判定逻辑与 v-auth 共用 `auth-permission.hasAuthPermission`:
 * 两个指令此前各写一份同形分支(且 v-auth 那份还读错了数据源),
 * 收敛到单一实现后,权限语义只在一处定义、也只需一处测试。
 *
 * 行为变化说明:旧实现的 `list.length === 0 || ...` 只把**空数组**
 * 视为无需权限,空字符串会被当作权限标识去比对(必然不命中而被移除)。
 * 收敛后空字符串同样视为"无需权限"。实践中不存在 `v-perm=""` 的写法,
 * 因此该差异不影响现有模板(仓库内 21 处 v-perm 均为非空字面量)。
 */
import type { Directive } from 'vue'
import { useUserStore } from '@/store/modules/user'
import { hasAuthPermission } from './auth-permission'

export const perm: Directive<HTMLElement, string | string[]> = {
  mounted(el, binding) {
    const perms = useUserStore().info?.permissions ?? []
    if (!hasAuthPermission(perms, binding.value)) el.parentNode?.removeChild(el)
  }
}
