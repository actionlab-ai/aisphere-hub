# AIHub Console — Frontend v3.6 (Polish & Detail Pass)

A Next.js 16 frontend for the AIHub agent skill registry. This version focuses on UI polish, complete bilingual (中文/English) coverage, and removing the legacy `scope` mechanism in favor of the IAM ResourceGrant-derived access mode.

## What's New in v3.6

### 🎨 UI Polish & Detail Fixes

#### 1. Resource ID Validation
- New `ResourceIdInput` component (`src/components/shared/resource-id-input.tsx`) with:
  - Live validation (lowercase letters, digits, hyphens, underscores only)
  - Helpful hint text below the input
  - Green check icon when valid, red error message when invalid
  - `autoCapitalize="none"`, `autoCorrect="off"`, `spellCheck={false}`
- Used in: Skill Create dialog, SkillSet Create dialog
- Version field also validated (must be `x.y.z` semantic version)

#### 2. Display Name vs ID Separation
- **ID** fields are clearly labeled, use monospace font, and show validation hints
- **Display Name** fields accept any text (including CJK) with appropriate placeholders
- Placeholders are localized: "友好的显示名称(支持中文)" / "Friendly display name (supports CJK)"

#### 3. Scope Removed → Access Mode
Per the v3.5 sharing spec, the legacy `scope: PUBLIC | PRIVATE` mechanism is removed from the UI:
- **Skill Create dialog**: no more Scope dropdown
- **SkillSet Create dialog**: no more Scope dropdown
- **Skill Editor Settings tab**: the old "Visibility Scope" switch is replaced with a read-only **Access Mode** card showing the current state (Private / Shared / Public) with an icon, description, and a "Managed by sharing settings" hint
- **SkillSet Detail Settings tab**: same access mode display
- **Skill Card**: the scope badge now shows localized access mode labels (公开/私有) instead of raw "public"/"private"
- **Skills filter**: the "Scope" filter is now labeled "Access Mode" with localized options

The access mode is derived from the resource's IAM ResourceGrant list via `deriveAccessMode()`.

#### 4. Complete i18n Coverage
Added 120+ new i18n strings (zh + en) covering:
- Resource ID validation messages
- Access mode labels and descriptions
- SkillSet create/edit dialog
- SkillSet detail sheet (all 5 tabs)
- Groups page
- Group member list
- IAM page
- Namespaces page
- Ops page (tokens, audit logs, metrics)
- Proposals page (approve/reject with reason form)
- Access page
- Common UI labels (Name, Status, Type, Actions, Created, etc.)

#### 5. Proposals Page — Reject Reason Form
- Replaced the native `prompt()` dialog with a proper textarea form
- Click "Reject" once to reveal the reason input
- Click "Reject" again to confirm with the entered reason

#### 6. Empty States
- IAM page: shows "暂无账号 / No accounts" when the list is empty
- Ops page: shows "暂无 Token / No tokens" and "暂无审计日志 / No audit logs"
- Proposals page: shows "暂无提案 / No proposals"

## File Changes (v3.5 → v3.6)

### New Files
- `src/components/shared/resource-id-input.tsx` — validated ID input component

### Modified Files
- `src/lib/utils.ts` — added `isValidResourceId()`, `isValidVersion()`, `sanitizeResourceId()`, `getAccessModeColor()`, `getAccessModeIcon()`
- `src/lib/i18n.tsx` — +120 strings (ID validation, access mode, groups, IAM, namespaces, ops, proposals, access, common UI)
- `src/components/shared/index.ts` — exports `ResourceIdInput`
- `src/components/skills/skill-create-dialog.tsx` — uses `ResourceIdInput`, removed Scope field, added version validation
- `src/components/skills/skill-card.tsx` — scope badge shows localized access mode labels
- `src/components/skills/skill-filters.tsx` — scope filter relabeled to "Access Mode"
- `src/components/groups/group-create-dialog.tsx` — uses `ResourceIdInput`, removed Scope field, full i18n
- `src/components/groups/group-detail-sheet.tsx` — full i18n (all 5 tabs), removed Scope editor, shows access mode
- `src/components/editor/skill-editor.tsx` — Settings tab now shows access mode card (replaces scope switch), removed `useSkillScope` hook
- `src/components/pages/groups-page.tsx` — full i18n
- `src/components/pages/iam-page.tsx` — full i18n, empty state
- `src/components/pages/namespaces-page.tsx` — full i18n
- `src/components/pages/ops-page.tsx` — full i18n, empty states
- `src/components/pages/proposals-page.tsx` — full i18n, reject reason form, empty state

## How to Use

```bash
cd your-aisphere-hub/
rm -rf front/
unzip aisphere-hub-front-clean.zip -d front/
cd front/
npm install
cp .env.example .env.local  # point at your backend
npm run dev
```

## Verification

All changes verified with agent-browser (mocked API):
- ✅ Create Skill dialog: ID validation works (invalid → red error, valid → green check)
- ✅ Create Skill dialog: no Scope field, Display Name accepts CJK
- ✅ Skill Editor Settings: shows "访问状态: 已共享 / Access Mode: Shared" with "Managed by sharing settings" hint
- ✅ Skill Card: scope badge shows "公开" / "Public" (localized)
- ✅ Skills filter: "Access Mode" label with "全部 / All" option
- ✅ All pages render correctly in both Chinese and English
- ✅ ESLint passes with zero errors

## License

Same as the parent aisphere-hub project.
