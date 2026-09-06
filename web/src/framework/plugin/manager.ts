/**
 * 插件管理器
 * 通过 import.meta.glob 自动发现 src/modules/{name}/index.ts，
 * 按 register → install → mount 生命周期装配插件。
 */
import type { App, Component, Directive } from 'vue'
import type { RouteRecordRaw, Router } from 'vue-router'
import { store } from '@/store'
import type { AppRouteRecord } from '@/types/router'
import type { PluginContext, PluginManifest } from '@/types/plugin'

type ModuleIndex = { default?: PluginManifest }

const pluginModules = import.meta.glob<ModuleIndex>('../../modules/*/index.ts', { eager: true })

class PluginManager {
  private plugins: PluginManifest[] = []
  private installed = new Set<string>()
  private mounted = new Set<string>()
  private ctx: PluginContext | null = null

  constructor() {
    this.plugins = Object.values(pluginModules)
      .map((m) => m.default)
      .filter((p): p is PluginManifest => !!p)
      .filter((p) => p.enabled !== false)
  }

  getPluginNames(): string[] {
    return this.plugins.map((p) => p.name)
  }

  collectRoutes(): AppRouteRecord[] {
    return this.plugins.flatMap((p) => p.routes ?? [])
  }

  install(app: App, router: Router): void {
    this.ctx = this.buildContext(app, router)
    for (const p of this.plugins) {
      if (this.installed.has(p.name)) continue
      p.installStore?.(store)
      p.install?.(this.ctx)
      this.installed.add(p.name)
    }
  }

  mount(): void {
    for (const p of this.plugins) {
      if (this.mounted.has(p.name)) continue
      p.mount?.(this.ctx!)
      this.mounted.add(p.name)
    }
  }

  private buildContext(app: App, router: Router): PluginContext {
    return {
      app,
      router,
      pinia: store,
      registerComponent: (name, comp) => app.component(name, comp as Component),
      registerDirective: (name, dir) => app.directive(name, dir as Directive),
      registerCommand: (name, fn) =>
        ((app.config.globalProperties as Record<string, unknown>)[`$${name}`] = fn),
      addRoutes: (routes) => {
        routes.forEach((r) => {
          if (r.name && !router.hasRoute(r.name)) {
            router.addRoute(r as unknown as RouteRecordRaw)
          }
        })
      }
    }
  }
}

export const pluginManager = new PluginManager()
