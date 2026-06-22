import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function fmtTime(v?: number | string): string {
  if (!v) return '-';
  const d = typeof v === 'number' ? new Date(v) : new Date(v);
  return Number.isNaN(d.getTime()) ? String(v) : d.toLocaleString();
}

export function fmtSize(bytes = 0): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function fmtRelativeTime(v?: number | string): string {
  if (!v) return '-';
  const d = typeof v === 'number' ? new Date(v) : new Date(v);
  if (Number.isNaN(d.getTime())) return String(v);
  const now = Date.now();
  const diff = now - d.getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  if (diff < 604_800_000) return `${Math.floor(diff / 86_400_000)}d ago`;
  return d.toLocaleDateString();
}

export function versionOf(s: { labels?: Record<string, string>; stableVersion?: string; latestVersion?: string; versions?: { version: string }[] }): string {
  return s.labels?.stable || s.labels?.latest || s.stableVersion || s.latestVersion || s.versions?.[0]?.version || '';
}

export type TreeNode = {
  name: string;
  path: string;
  type: 'dir' | 'file';
  size?: number;
  binary?: boolean;
  children: TreeNode[];
};

export function buildTree(files: { path: string; name: string; size?: number; binary?: boolean }[]): TreeNode[] {
  const root: TreeNode[] = [];
  const ensure = (list: TreeNode[], name: string, path: string, type: 'dir' | 'file') => {
    let node = list.find((x) => x.name === name && x.type === type);
    if (!node) {
      node = { name, path, type, children: [] };
      list.push(node);
    }
    return node;
  };
  for (const f of files) {
    const parts = f.path.split('/').filter(Boolean);
    let list = root;
    let acc = '';
    parts.forEach((p, i) => {
      acc = acc ? `${acc}/${p}` : p;
      const isFile = i === parts.length - 1;
      const n = ensure(list, p, acc, isFile ? 'file' : 'dir');
      if (isFile) {
        n.size = f.size;
        n.binary = f.binary;
      }
      list = n.children;
    });
  }
  const sort = (nodes: TreeNode[]) =>
    nodes
      .sort((a, b) => (a.type === b.type ? a.name.localeCompare(b.name) : a.type === 'dir' ? -1 : 1))
      .forEach((n) => sort(n.children));
  sort(root);
  return root;
}

export function getStatusColor(status?: string): string {
  switch (status) {
    case 'online':
    case 'published':
    case 'reviewed':
    case 'approved':
    case 'promoted':
      return 'text-emerald-600 bg-emerald-50 dark:text-emerald-400 dark:bg-emerald-950/50';
    case 'offline':
    case 'rejected':
    case 'disable':
      return 'text-red-600 bg-red-50 dark:text-red-400 dark:bg-red-950/50';
    case 'draft':
    case 'submitted':
    case 'reviewing':
    case 'validating':
      return 'text-amber-600 bg-amber-50 dark:text-amber-400 dark:bg-amber-950/50';
    case 'validated':
      return 'text-sky-600 bg-sky-50 dark:text-sky-400 dark:bg-sky-950/50';
    case 'enable':
    case 'active':
      return 'text-emerald-600 bg-emerald-50 dark:text-emerald-400 dark:bg-emerald-950/50';
    default:
      return 'text-zinc-600 bg-zinc-50 dark:text-zinc-400 dark:bg-zinc-800/50';
  }
}

export function getScopeColor(scope?: string): string {
  switch (scope?.toUpperCase()) {
    case 'PUBLIC':
      return 'text-emerald-600 bg-emerald-50 dark:text-emerald-400 dark:bg-emerald-950/50';
    case 'PRIVATE':
      return 'text-violet-600 bg-violet-50 dark:text-violet-400 dark:bg-violet-950/50';
    default:
      return 'text-zinc-600 bg-zinc-50 dark:text-zinc-400 dark:bg-zinc-800/50';
  }
}

// ─── Resource ID validation ─────────────────────────────────────────
// Resource IDs (skill name, skillset name, etc.) must be valid
// identifiers: lowercase letters, digits, hyphens, underscores.
// Display names can be anything (including CJK).

const RESOURCE_ID_REGEX = /^[a-z0-9][a-z0-9_-]{0,62}$/;

export function isValidResourceId(id: string): boolean {
  return RESOURCE_ID_REGEX.test(id);
}

export function sanitizeResourceId(id: string): string {
  // Lowercase, replace invalid chars with hyphens, trim leading non-alnum
  let s = id.toLowerCase().replace(/[^a-z0-9_-]/g, '-');
  s = s.replace(/^-+/, '');
  return s.slice(0, 63);
}

// Validate semver-like version strings (e.g. "1.0.0", "2.1.3-beta")
const VERSION_REGEX = /^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$/;

export function isValidVersion(v: string): boolean {
  return VERSION_REGEX.test(v);
}

// ─── Access mode helpers (replaces scope) ───────────────────────────

export type AccessMode = 'private' | 'shared' | 'public';

export function getAccessModeColor(mode?: string): string {
  switch (mode?.toLowerCase()) {
    case 'public':
      return 'text-emerald-600 bg-emerald-50 dark:text-emerald-400 dark:bg-emerald-950/50';
    case 'shared':
      return 'text-amber-600 bg-amber-50 dark:text-amber-400 dark:bg-amber-950/50';
    case 'private':
      return 'text-violet-600 bg-violet-50 dark:text-violet-400 dark:bg-violet-950/50';
    default:
      return 'text-zinc-600 bg-zinc-50 dark:text-zinc-400 dark:bg-zinc-800/50';
  }
}

export function getAccessModeIcon(mode?: string): string {
  // Returns a lucide icon name (caller maps to component)
  switch (mode?.toLowerCase()) {
    case 'public': return 'globe';
    case 'shared': return 'users';
    case 'private': return 'lock';
    default: return 'circle';
  }
}
