'use client';

import { Folder, FileText, FileCode } from 'lucide-react';
import { fmtSize } from '@/lib/utils';
import type { TreeNode } from '@/lib/utils';

interface FileTreeNodesProps {
  nodes: TreeNode[];
  openFile: (path: string) => void;
  depth: number;
  selectedPath?: string;
}

export function SkillFileBrowser({ nodes, openFile, depth = 0, selectedPath }: FileTreeNodesProps) {
  return (
    <>
      {nodes.map((n) => (
        <div key={n.path}>
          {n.type === 'dir' ? (
            <div className="flex items-center gap-1.5 py-0.5 text-xs text-muted-foreground" style={{ paddingLeft: depth * 16 }}>
              <Folder className="h-3.5 w-3.5 shrink-0" /> {n.name}
            </div>
          ) : (
            <button
              className={`flex items-center gap-1.5 py-0.5 text-xs hover:text-foreground transition-colors w-full text-left rounded px-1
                ${selectedPath === n.path ? 'bg-violet-500/10 text-violet-600 dark:text-violet-400' : ''}`}
              style={{ paddingLeft: depth * 16 }}
              onClick={() => openFile(n.path)}
            >
              {n.name.endsWith('.md') ? <FileText className="h-3.5 w-3.5 shrink-0 text-blue-500" /> :
               n.name.endsWith('.json') || n.name.endsWith('.yaml') || n.name.endsWith('.yml') ? <FileCode className="h-3.5 w-3.5 shrink-0 text-amber-500" /> :
               <FileCode className="h-3.5 w-3.5 shrink-0 text-emerald-500" />}
              <span className="truncate">{n.name}</span>
              {n.size !== undefined && <span className="text-muted-foreground ml-auto text-[10px]">{fmtSize(n.size)}</span>}
            </button>
          )}
          {n.children?.length > 0 && (
            <SkillFileBrowser nodes={n.children} openFile={openFile} depth={depth + 1} selectedPath={selectedPath} />
          )}
        </div>
      ))}
    </>
  );
}
