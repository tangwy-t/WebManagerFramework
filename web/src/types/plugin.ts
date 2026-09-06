/**
 * 插件框架类型定义
 * 模块即插件：src/modules/<name>/ 下的 index.ts 默认导出 PluginManifest。
 */
import type { App, Component, Directive } from 'vue'
import type { Router } from 'vue-router'
import type { Pinia } from 'pinia'
import type { AppRouteRecord } from './router'

/** 插件运行时上下文 */
export interface PluginContext {
  app: App
  router: Router
  pinia: Pinia
  registerComponent(name: string, comp: Component): void
  registerDirective(name: string, dir: Directive): void
  registerCommand(name: string, fn: (...args: unknown[]) => unknown): void
  addRoutes(routes: AppRouteRecord[]): void
}

/** 插件清单 */
export interface PluginManifest {
  name: string
  version: string
  enabled?: boolean
  routes?: AppRouteRecord[]
  installStore?: (pinia: Pinia) => void
  install?: (ctx: PluginContext) => void
  mount?: (ctx: PluginContext) => void
  unmount?: () => void
}
