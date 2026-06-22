# Frontend Casdoor Update

本次前端整合基于你上传的 Next Console，做了以下调整：

1. 保留你现有的 Skills / Groups / Governance / Ops / IAM 页面。
2. 删除无用 mock / 本地 DB 内容：`src/app/api`、`src/lib/db.ts`、`prisma`、`db`、`upload`、`tool-results`、`mini-services`。
3. `next.config.ts` 增加开发期 rewrites：`/v3/**`、`/registry-global/**`、`/registry/**`、`/metrics`、`/healthz` 转发到 SkillHub 后端。
4. 登录页增加 `Sign in with Casdoor`。
5. 新增 `/auth/callback` 页面，接收 SkillHub 后端 OIDC callback relay 过来的 token。
6. npm scripts 改成标准 Next：`dev`、`build`、`start`，不再依赖 bun。

## 运行

```powershell
cd front
$env:SKILLHUB_BACKEND_URL="http://127.0.0.1:8848"
npm install
npm run dev
```

生产 Next server：

```powershell
npm run build
npm run start
```
