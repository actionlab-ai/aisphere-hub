# 架构图

```text
Human User / Agent / Service
          |
          | login / token
          v
      Casdoor Service
          |
          | OIDC/JWT access_token
          v
      SkillHub Backend
          |
          | Principal(subject, roles, org)
          v
      Embedded Casbin Enforcer
          |
          | allow / deny
          v
Skill / Group / Version / Proposal / Runtime
```

- Casdoor 是外部 IAM 服务。
- Casbin 是 SkillHub 后端依赖库。
- SkillHub 不保存用户密码，不做完整组织用户中心。
- SkillHub 只保存自己的 Skill 资源和 Casbin resource policy。
