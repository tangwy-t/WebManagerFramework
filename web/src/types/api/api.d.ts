/**
 * API 接口类型定义（对接 Go 后端契约）
 * 全局命名空间 `Api`，无需导入即可使用。
 */
declare namespace Api {
  /** 通用分页请求参数 */
  interface PageParams {
    page?: number
    pageSize?: number
  }

  /** 认证类型 */
  namespace Auth {
    /** 登录参数 */
    interface LoginParams {
      username: string
      password: string
      captchaKey?: string
      captchaCode?: string
    }

    /** 登录/刷新令牌响应 */
    interface TokenResp {
      accessToken: string
      refreshToken: string
      expiresIn: number
    }

    /** 验证码响应 */
    interface CaptchaResp {
      captchaKey: string
      captchaImage: string
    }

    /** 修改密码参数(个人中心) */
    interface ChangePasswordParams {
      oldPassword: string
      newPassword: string
    }

    /** 锁屏解锁:验证当前登录密码(后端 POST /user/password/verify) */
    interface VerifyPasswordParams {
      password: string
    }

    interface VerifyPasswordResp {
      valid: boolean
    }

    /** 更新个人资料参数(个人中心,仅姓名/邮箱/手机号可自助维护) */
    interface UpdateProfileParams {
      realName: string
      email: string
      phone: string
    }

    /** 头像上传响应 */
    interface AvatarResp {
      avatar: string
    }

    /** 用户信息 */
    interface UserInfo {
      id: string
      username: string
      realName: string
      avatar: string
      email: string
      phone: string
      dataScope: number
      deptId: string
      roles: string[]
      permissions: string[]
    }

    /** 个人中心总览(GET /user/overview,仅本人数据) */
    interface RoleBrief {
      code: string
      name: string
    }

    /** 登录统计:历史成功总数 + 近 30 天成功/失败 */
    interface LoginStats {
      totalLogins: number
      logins30d: number
      failed30d: number
    }

    /** 近 30 天逐日登录活动(date 为 YYYY-MM-DD,服务器本地时区,零填充连续 30 天) */
    interface DailyActivity {
      date: string
      success: number
      failed: number
    }

    /** 最近一次登录尝试(code 0=成功;非 0 为业务错误码,msg 为原因) */
    interface RecentLogin {
      time: string
      ip: string
      browser: string
      os: string
      code: number
      msg: string
    }

    /** 个人中心聚合载荷:身份 + 登录统计 + 活动序列 + 最近登录 */
    interface UserOverview {
      id: string
      username: string
      realName: string
      avatar: string
      email: string
      phone: string
      deptName: string
      roles: RoleBrief[]
      createdAt: string
      lastLoginTime: string | null
      lastLoginIp: string
      stats: LoginStats
      dailyActivity: DailyActivity[]
      recentLogins: RecentLogin[]
    }
  }

  /** 字典类型 */
  namespace Dict {
    /** 字典类型 */
    interface DictType {
      id: string
      code: string
      name: string
      status: number
      remark: string
      createdAt: string
      updatedAt: string
    }

    /** 字典数据 */
    interface DictData {
      id: string
      typeId: string
      listClass: '' | 'primary' | 'success' | 'info' | 'warning' | 'danger'
      label: string
      value: string
      isDefault: number
      sort: number
      status: number
      remark: string
      createdAt: string
      updatedAt: string
    }

    /** 消费端字典项（/dict/codes/:code） */
    interface DictItem {
      label: string
      value: string
      list_class: string
      is_default: boolean
      sort: number
    }

    /** 字典类型表单 */
    type DictTypeForm = Pick<Api.Dict.DictType, 'code' | 'name' | 'status' | 'remark'>
    /** 字典数据表单 */
    type DictDataForm = Pick<
      Api.Dict.DictData,
      'label' | 'value' | 'listClass' | 'isDefault' | 'sort' | 'status' | 'remark'
    >
  }

  /** 系统管理：用户 / 角色 / 菜单 / 部门 */
  namespace System {
    interface User {
      id: string
      username: string
      realName: string
      email: string
      phone: string
      avatar: string
      deptId: string
      deptName: string
      status: number
      remark: string
      lastLoginTime: string | null
      createdAt: string
      roleNames: string[]
      roleIds: string[]
    }
    interface UserQuery extends Api.PageParams {
      username?: string
      realName?: string
      phone?: string
      email?: string
      status?: number
      deptId?: string
      createdAtStart?: string
      createdAtEnd?: string
    }
    interface UserForm {
      id?: string
      username?: string
      password?: string
      realName?: string
      email?: string
      phone?: string
      deptId?: string
      roleIds?: string[]
      status?: number
      remark?: string
    }

    interface Role {
      id: string
      name: string
      code: string
      dataScope: number
      sort: number
      status: number
      remark: string
      menuIds: string[]
      deptIds: string[]
      createdAt: string
    }
    interface RoleQuery extends Api.PageParams {
      name?: string
      code?: string
      status?: number
    }
    interface RoleForm {
      id?: string
      name: string
      code: string
      dataScope?: number
      sort?: number
      status?: number
      remark?: string
      menuIds?: string[]
      deptIds?: string[]
    }

    interface Menu {
      id: string
      parentId: string
      name: string
      type: 'dir' | 'menu' | 'btn'
      perms: string
      path: string
      component: string
      icon: string
      sort: number
      visible: number
      status: number
      children: Menu[]
      createdAt: string
    }
    interface MenuForm {
      id?: string
      // 字符串为已有菜单 ID;数字 0 表示顶级目录(服务端 parent_id=0)。
      parentId?: string | number
      name: string
      type: 'dir' | 'menu' | 'btn'
      perms?: string
      path?: string
      component?: string
      icon?: string
      sort?: number
      visible?: number
      status?: number
    }

    interface Dept {
      id: string
      parentId: string
      ancestors: string
      name: string
      sort: number
      leader: string
      phone: string
      email: string
      status: number
      children: Dept[]
      createdAt: string
    }
    interface DeptForm {
      id?: string
      // 字符串为已有部门 ID;数字 0 表示顶级部门(服务端 parent_id=0)。
      parentId?: string | number
      name: string
      sort?: number
      leader?: string
      phone?: string
      email?: string
      status?: number
    }
  }

  /** 通知公告 */
  namespace Notice {
    interface Notice {
      id: string
      title: string
      content: string
      noticeType: number
      status: number
      priority: number
      publishType: number
      /** 接收范围类型 0=全体成员 1=指定角色 2=指定部门 3=指定个人 */
      targetType: number
      /** 接收对象 ID 逗号分隔 */
      targetIds: string
      /** 接收范围展示文案,如 全体成员 / 指定角色（2个） */
      targetDesc: string
      /** 创建者登录名 */
      createBy: string
      publishTime: string | null
      /** 当前用户阅读状态(仅已发布行有值):0 未读 / 1 已读;草稿/已撤回为 null */
      readStatus?: number | null
      createdAt: string
      updatedAt: string
    }
    interface Query extends Api.PageParams {
      title?: string
      noticeType?: number
      status?: number
    }
    interface Form {
      id?: string
      title: string
      content: string
      noticeType?: number
      priority?: number
      publishType?: number
      /** 当前状态(回显用,提交时后端主导) */
      status?: number
      /** 接收范围类型 0=全体成员 1=指定角色 2=指定部门 3=指定个人 */
      targetType?: number
      /** 接收对象 ID 逗号分隔 */
      targetIds?: string
    }
    /** 收件箱条目(白名单接口 /notices/my) */
    interface MyItem {
      id: string
      title: string
      content: string
      noticeType: number
      priority: number
      isRead: boolean
      publishTime: string | null
    }
    /** 收件箱响应:可见公告列表 + 未读数 */
    interface MyList {
      list: MyItem[]
      unreadCount: number
    }
    /** 已读用户行(阅读用户弹窗) */
    interface ReadUser {
      userId: string
      username: string
      realName: string
      deptName: string
      phone: string
      readTime: string | null
    }
    interface ReadUsersQuery extends Api.PageParams {
      searchValue?: string
    }
    /** 指定个人反查行(接收范围回显) */
    interface TargetUser {
      id: string
      username: string
      realName: string
    }
  }

  /** 参数配置 */
  namespace Config {
    interface Config {
      id: string
      name: string
      configKey: string
      configValue: string
      configType: 'S' | 'N' | 'B' | 'J'
      remark: string
      status: number
      createdAt: string
      updatedAt: string
    }
    interface Query extends Api.PageParams {
      configKey?: string
      configType?: string
      status?: number
    }
    interface Form {
      id?: string
      name: string
      configKey?: string
      configValue: string
      configType: 'S' | 'N' | 'B' | 'J'
      remark?: string
      status?: number
    }
  }

  /** 定时任务 */
  namespace Job {
    interface Job {
      id: string
      name: string
      jobGroup: string
      cronExpression: string
      invokeTarget: string
      invokeParams: string
      concurrent: number
      retryCount: number
      retryInterval: number
      status: number
      runAtStartup: number
      remark: string
      nextRunTime: string | null
      createdAt: string
      updatedAt: string
    }
    /** 可注册的调用目标（GET /jobs/targets） */
    interface TargetInfo {
      target: string
      displayName: string
      hasParams: boolean
      paramSchema?: Record<string, any>
    }
    /** 调度器健康状态（GET /jobs/health） */
    interface Health {
      schedulerRunning: boolean
      instanceId: string
      totalJobs: number
      runningJobs: number
      uptime: string
    }
    interface JobLog {
      id: string
      jobId: string
      jobName: string
      jobGroup: string
      invokeTarget: string
      triggerType: number
      startTime: string
      endTime: string | null
      costTime: number
      status: number
      errorMsg: string
    }
    interface Query extends Api.PageParams {
      name?: string
      jobGroup?: string
      status?: number
    }
    interface LogQuery extends Api.PageParams {
      jobId?: string
      status?: number
      startTime?: string
      endTime?: string
    }
    interface Form {
      id?: string
      name?: string
      jobGroup?: string
      cronExpression?: string
      invokeTarget?: string
      invokeParams?: string
      concurrent?: number
      retryCount?: number
      retryInterval?: number
      status?: number
      runAtStartup?: number
      remark?: string
    }
  }

  /** 日志 */
  namespace Log {
    interface OperationLog {
      id: string
      userId: string
      username: string
      module: string
      operationType: string
      requestMethod: string
      requestUrl: string
      requestParams: string
      responseResult: string
      costTime: number
      ip: string
      code: number
      errorMsg: string
      operTime: string
    }
    interface OperationLogQuery extends Api.PageParams {
      username?: string
      module?: string
      operationType?: string
      code?: number
      startTime?: string
      endTime?: string
    }
    interface LoginLog {
      id: string
      userId: string
      username: string
      ip: string
      location: string
      browser: string
      os: string
      code: number
      msg: string
      loginTime: string
    }
    interface LoginLogQuery extends Api.PageParams {
      username?: string
      ip?: string
      code?: number
      startTime?: string
      endTime?: string
    }
  }

  /** 监控 */
  namespace Monitor {
    interface ServerStats {
      server: {
        version: string
        buildTime: string
        commitHash: string
        startTime: string
        uptime: string
        uptimeSeconds: number
      }
      /** 宿主机信息（后端 v1.1 起返回，旧版本可能只有零值字段） */
      host?: {
        hostname?: string
        os?: string
        platform?: string
        platformVersion?: string
        kernelVersion?: string
        kernelArch?: string
        goVersion?: string
        arch?: string
        pid?: number
        bootTime?: string
        uptimeSeconds?: number
        error?: string
      }
      cpu: {
        numCPU: number
        usagePercent: number | null
        perCore?: number[]
        load?: { load1: number | null; load5: number | null; load15: number | null } | null
      }
      memory: {
        allocMB: number
        totalAllocMB: number
        sysMB: number
        heapAllocMB: number
        heapSysMB: number
        system?: {
          totalMB: number
          usedMB: number
          availableMB: number
          usedPercent: number
        } | null
        swap?: { totalMB: number; usedMB: number; freeMB: number; usedPercent: number } | null
      }
      goroutines: { count: number }
      gc: { numGC: number; pauseTotalMs: number | null; lastPauseMs: number | null }
      disk: { path: string; totalGB: number; usedGB: number; freeGB: number; usagePercent: number }
    }
  }

  /** 文件管理 */
  namespace File {
    /** 文件分类(与后端 entity.FileCategory* 常量一致) */
    type FileCategory = 'image' | 'video' | 'audio' | 'document' | 'archive' | 'code' | 'other'

    /** 文件条目 */
    interface FileItem {
      id: string
      name: string
      originalName: string
      size: number
      mimeType: string
      ext: string
      category: FileCategory
      module?: string
      moduleId?: string
      storageType: string
      createdBy?: string
      createdAt: string
      updatedAt: string
    }

    /** 列表查询参数 */
    interface FileQuery {
      page?: number
      pageSize?: number
      keyword?: string
      category?: FileCategory
      sortBy?: 'createdAt' | 'name' | 'size'
      sortOrder?: 'asc' | 'desc'
    }

    /** 概览统计 */
    interface FileStats {
      total: number
      totalSize: number
      weekUploads: number
      categoryCounts: Record<FileCategory, number>
    }

    /** 上传响应 */
    interface UploadResp {
      items: FileItem[]
    }
  }
}
