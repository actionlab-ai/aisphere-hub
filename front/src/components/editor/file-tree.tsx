'use client';

import { useState, useMemo, useCallback } from 'react';
import {
  ChevronRight, ChevronDown, Folder, FolderOpen,
  FileText, FileCode, FileJson, FileCog, FileImage, File, FileType2,
} from 'lucide-react';
import { cn, fmtSize, type TreeNode } from '@/lib/utils';

// File icon mapping by extension
function getFileIcon(name: string) {
  const lower = name.toLowerCase();
  if (lower.endsWith('.md') || lower.endsWith('.markdown')) return <FileText className="h-3.5 w-3.5 text-sky-500" />;
  if (lower.endsWith('.json')) return <FileJson className="h-3.5 w-3.5 text-amber-500" />;
  if (lower.endsWith('.yaml') || lower.endsWith('.yml')) return <FileJson className="h-3.5 w-3.5 text-orange-500" />;
  if (lower.endsWith('.py')) return <FileCode className="h-3.5 w-3.5 text-blue-500" />;
  if (lower.endsWith('.js') || lower.endsWith('.jsx') || lower.endsWith('.mjs') || lower.endsWith('.cjs')) return <FileCode className="h-3.5 w-3.5 text-yellow-500" />;
  if (lower.endsWith('.ts') || lower.endsWith('.tsx')) return <FileCode className="h-3.5 w-3.5 text-blue-600" />;
  if (lower.endsWith('.go')) return <FileCode className="h-3.5 w-3.5 text-cyan-500" />;
  if (lower.endsWith('.sh') || lower.endsWith('.bash')) return <FileCog className="h-3.5 w-3.5 text-emerald-500" />;
  if (lower.endsWith('.env') || lower.endsWith('.conf') || lower.endsWith('.toml') || lower.endsWith('.ini')) return <FileCog className="h-3.5 w-3.5 text-violet-500" />;
  if (lower.endsWith('.png') || lower.endsWith('.jpg') || lower.endsWith('.jpeg') || lower.endsWith('.gif') || lower.endsWith('.svg') || lower.endsWith('.webp')) return <FileImage className="h-3.5 w-3.5 text-fuchsia-500" />;
  if (lower.endsWith('.txt') || lower.endsWith('.log')) return <FileType2 className="h-3.5 w-3.5 text-muted-foreground" />;
  if (lower === 'skill.md' || lower === 'readme.md') return <FileText className="h-3.5 w-3.5 text-violet-500" />;
  return <File className="h-3.5 w-3.5 text-muted-foreground" />;
}

interface FileTreeProps {
  nodes: TreeNode[];
  selectedPath?: string;
  onOpenFile: (path: string) => void;
  /** Initially-expanded top-level folders */
  defaultExpanded?: string[];
  /** Compact mode (smaller padding) */
  compact?: boolean;
}

interface TreeRowProps {
  node: TreeNode;
  depth: number;
  selectedPath?: string;
  expanded: Set<string>;
  toggle: (path: string) => void;
  onOpenFile: (path: string) => void;
  compact?: boolean;
}

function TreeRow({ node, depth, selectedPath, expanded, toggle, onOpenFile, compact }: TreeRowProps) {
  const isDir = node.type === 'dir';
  const isExpanded = expanded.has(node.path);
  const isSelected = selectedPath === node.path;
  const pad = compact ? 8 : 10;

  if (isDir) {
    return (
      <div>
        <button
          onClick={() => toggle(node.path)}
          className={cn(
            'flex items-center gap-1 w-full text-left rounded-sm transition-colors group',
            compact ? 'py-0.5' : 'py-[3px]',
            'hover:bg-accent/70 text-foreground/80 hover:text-foreground',
          )}
          style={{ paddingLeft: depth * pad + 4, paddingRight: 6 }}
        >
          <span className="flex items-center justify-center w-3 h-3 shrink-0 text-muted-foreground">
            {isExpanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          </span>
          <span className="shrink-0 text-muted-foreground group-hover:text-foreground">
            {isExpanded ? <FolderOpen className="h-3.5 w-3.5 text-violet-500" /> : <Folder className="h-3.5 w-3.5 text-violet-500/80" />}
          </span>
          <span className="truncate text-xs font-medium">{node.name}</span>
          {node.children.length > 0 && (
            <span className="ml-auto text-[10px] text-muted-foreground/70 tabular-nums">{node.children.length}</span>
          )}
        </button>
        {isExpanded && node.children.length > 0 && (
          <div>
            {node.children.map((child) => (
              <TreeRow
                key={child.path}
                node={child}
                depth={depth + 1}
                selectedPath={selectedPath}
                expanded={expanded}
                toggle={toggle}
                onOpenFile={onOpenFile}
                compact={compact}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <button
      onClick={() => onOpenFile(node.path)}
      className={cn(
        'flex items-center gap-1 w-full text-left rounded-sm transition-colors relative',
        compact ? 'py-0.5' : 'py-[3px]',
        isSelected
          ? 'bg-violet-600/10 text-violet-700 dark:text-violet-300'
          : 'text-foreground/75 hover:bg-accent/70 hover:text-foreground',
      )}
      style={{ paddingLeft: depth * pad + 4 + 16, paddingRight: 6 }}
      title={node.path}
    >
      {isSelected && (
        <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[2px] h-4 rounded-full bg-violet-500" />
      )}
      <span className="shrink-0">{getFileIcon(node.name)}</span>
      <span className="truncate text-xs">{node.name}</span>
      {node.size !== undefined && node.size > 0 && (
        <span className="ml-auto text-[10px] text-muted-foreground/70 tabular-nums">{fmtSize(node.size)}</span>
      )}
    </button>
  );
}

export function FileTree({
  nodes,
  selectedPath,
  onOpenFile,
  defaultExpanded = [],
  compact = false,
}: FileTreeProps) {
  // Compute initial expanded set: defaultExpanded + all folder paths with selected descendant
  const initialExpanded = useMemo(() => {
    const set = new Set<string>(defaultExpanded);
    if (selectedPath) {
      // Walk up the path, expanding every ancestor folder
      const parts = selectedPath.split('/').filter(Boolean);
      let acc = '';
      for (let i = 0; i < parts.length - 1; i++) {
        acc = acc ? `${acc}/${parts[i]}` : parts[i];
        set.add(acc);
      }
    }
    return set;
  }, [defaultExpanded, selectedPath]);

  const [expanded, setExpanded] = useState<Set<string>>(initialExpanded);

  // Re-sync when selectedPath changes (so opening a file from outside expands its parents)
  const lastSyncedRef = useMemo(() => selectedPath, [selectedPath]);
  if (selectedPath && selectedPath === lastSyncedRef) {
    const parts = selectedPath.split('/').filter(Boolean);
    let acc = '';
    for (let i = 0; i < parts.length - 1; i++) {
      acc = acc ? `${acc}/${parts[i]}` : parts[i];
      if (!expanded.has(acc)) {
        const next = new Set(expanded);
        next.add(acc);
        setExpanded(next);
      }
    }
  }

  const toggle = useCallback((path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  }, []);

  if (nodes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-10 text-center px-4">
        <Folder className="h-8 w-8 text-muted-foreground/50 mb-2" />
        <p className="text-xs text-muted-foreground">No files in this version</p>
      </div>
    );
  }

  return (
    <div className="py-1">
      {nodes.map((n) => (
        <TreeRow
          key={n.path}
          node={n}
          depth={0}
          selectedPath={selectedPath}
          expanded={expanded}
          toggle={toggle}
          onOpenFile={onOpenFile}
          compact={compact}
        />
      ))}
    </div>
  );
}

// ─── Breadcrumbs for the currently-open file ─────────────────────────
interface FileBreadcrumbsProps {
  path: string;
  onNavigate?: (path: string) => void;
}

export function FileBreadcrumbs({ path, onNavigate }: FileBreadcrumbsProps) {
  const parts = path.split('/').filter(Boolean);
  return (
    <nav className="flex items-center gap-0.5 text-xs text-muted-foreground min-w-0">
      {parts.map((part, i) => {
        const isLast = i === parts.length - 1;
        const sub = parts.slice(0, i + 1).join('/');
        return (
          <span key={sub} className="flex items-center gap-0.5 min-w-0">
            {i > 0 && <span className="text-muted-foreground/50">/</span>}
            <button
              onClick={() => !isLast && onNavigate?.(sub)}
              disabled={isLast}
              className={cn(
                'truncate max-w-[160px] px-1 py-0.5 rounded transition-colors',
                isLast
                  ? 'text-foreground font-medium'
                  : 'hover:bg-accent hover:text-foreground',
              )}
              title={sub}
            >
              {part}
            </button>
          </span>
        );
      })}
    </nav>
  );
}
