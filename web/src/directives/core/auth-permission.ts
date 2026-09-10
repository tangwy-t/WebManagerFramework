/**
 * v-auth / v-perm 共用的权限判定逻辑。
 *
 * 独立成模块(不 import 任何 store)**是刻意的**:本仓库 vitest 运行在
 * node 环境(无 DOM、无 localStorage),而 `@/store/modules/user` 的导入
 * 链会经由 pinia persist 触达 localStorage 而使测试在导入期就崩溃。
 * 判定逻辑与副作用分离后,权限语义可以被完整覆盖,且不引入 jsdom 依赖。
 */

/**
 * 判断权限集合是否满足所需权限。
 *
 * 契约:
 * - `required` 为字符串或字符串数组(数组 = 满足其一即可,OR)
 * - 空值(空串/空数组/全空白项)视为"无需权限" → true
 * - 其余情况要求 required 中至少一项命中 perms
 */
export function hasAuthPermission(perms: readonly string[], required: string | string[]): boolean {
  const list = Array.isArray(required) ? required : [required]
  const wanted = list.filter((p) => typeof p === 'string' && p.length > 0)
  if (wanted.length === 0) return true
  return wanted.some((p) => perms.includes(p))
}
