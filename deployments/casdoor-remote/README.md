# Casdoor Remote Authz Setup

1. In Casdoor, create a Casbin Model named `skillhub_model` and paste `skillhub_model.conf`.
2. Create a Permission named `skillhub_permission` and bind it to `skillhub_model`.
3. Add policies from `skillhub_policy_seed.csv` to the Permission.
4. Bind Casdoor users to roles such as `aihub-admin` or `skill-admin`.
5. Start SkillHub with `authz.provider=casdoor-remote`.

SkillHub will call Casdoor `/api/enforce?permissionId=<owner>/skillhub_permission` with body `[subject, object, action]`.
