'use client';

import { useMemo } from 'react';
import dynamic from 'next/dynamic';
import { useTheme } from 'next-themes';
import { javascript } from '@codemirror/lang-javascript';
import { json } from '@codemirror/lang-json';
import { markdown } from '@codemirror/lang-markdown';
import { python } from '@codemirror/lang-python';
import { yaml } from '@codemirror/lang-yaml';
import { EditorView } from '@codemirror/view';
import { syntaxHighlighting, HighlightStyle } from '@codemirror/language';
import { tags as t } from '@lezer/highlight';
import { cn } from '@/lib/utils';

// CodeMirror needs the DOM — load it client-only
const CodeMirror = dynamic(() => import('@uiw/react-codemirror'), { ssr: false });

interface CodeEditorProps {
  value: string;
  onChange?: (value: string) => void;
  path?: string;
  readOnly?: boolean;
  className?: string;
  /** Show line numbers (default true) */
  lineNumbers?: boolean;
}

function getLanguageExtension(path?: string) {
  if (!path) return [];
  const lower = path.toLowerCase();
  if (lower.endsWith('.json')) return [json()];
  if (lower.endsWith('.md') || lower.endsWith('.markdown')) return [markdown()];
  if (lower.endsWith('.py')) return [python()];
  if (lower.endsWith('.yaml') || lower.endsWith('.yml')) return [yaml()];
  if (lower.endsWith('.js') || lower.endsWith('.jsx') || lower.endsWith('.mjs') || lower.endsWith('.cjs') || lower.endsWith('.ts') || lower.endsWith('.tsx')) {
    return [javascript({ jsx: lower.endsWith('x') })];
  }
  // Default to no language extension for unknown types (plain text)
  return [];
}

// ─── Custom light theme ─────────────────────────────────────────────
// Minimal & matches our app palette. Uses CSS variables so it adapts
// to whatever the user's light-mode palette is.

const lightTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: 'transparent',
      color: 'var(--foreground)',
    },
    '.cm-content': {
      caretColor: 'var(--primary)',
    },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--primary)' },
    '.cm-selectionBackground, ::selection': { backgroundColor: 'oklch(0.55 0.22 290 / 0.18)' },
    '.cm-gutters': {
      backgroundColor: 'transparent',
      color: 'var(--muted-foreground)',
      border: 'none',
    },
    '.cm-activeLine': { backgroundColor: 'oklch(0.5 0 0 / 0.04)' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--foreground)' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection':
      { backgroundColor: 'oklch(0.55 0.22 290 / 0.18)' },
  },
  { dark: false },
);

// ─── Custom dark theme ──────────────────────────────────────────────
// Replaces @codemirror/theme-one-dark so we don't need that extra
// dependency. Colors are tuned to match our violet/fuchsia app palette.

const darkTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: 'transparent',
      color: '#e4e4e7',
    },
    '.cm-content': {
      caretColor: '#a78bfa',
      color: '#e4e4e7',
    },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: '#a78bfa' },
    '.cm-selectionBackground, ::selection': { backgroundColor: 'rgba(167, 139, 250, 0.25)' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
      backgroundColor: 'rgba(167, 139, 250, 0.25)',
    },
    '.cm-gutters': {
      backgroundColor: 'transparent',
      color: '#52525b',
      border: 'none',
    },
    '.cm-activeLine': { backgroundColor: 'rgba(255, 255, 255, 0.03)' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent', color: '#a1a1aa' },
    '.cm-foldPlaceholder': {
      backgroundColor: '#27272a',
      border: 'none',
      color: '#71717a',
    },
    '.cm-panels': {
      backgroundColor: '#18181b',
      color: '#e4e4e7',
    },
    '.cm-panels.cm-panels-top': { borderBottom: '1px solid #27272a' },
    '.cm-searchMatch': { backgroundColor: 'rgba(167, 139, 250, 0.2)' },
    '.cm-searchMatch-selected': { backgroundColor: 'rgba(167, 139, 250, 0.4)' },
  },
  { dark: true },
);

// ─── Syntax highlighting styles ─────────────────────────────────────
// Token colors for both light and dark themes. These are applied via
// the `syntaxHighlighting` extension (from @codemirror/language, which
// is a transitive dependency of all @codemirror/lang-* packages).

const lightHighlightStyle = HighlightStyle.define([
  { tag: t.keyword, color: '#7c3aed' },
  { tag: [t.name, t.deleted, t.character, t.macroName], color: '#18181b' },
  { tag: [t.function(t.variableName), t.labelName], color: '#4f46e5' },
  { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: '#0369a1' },
  { tag: [t.definition(t.name), t.separator], color: '#18181b' },
  { tag: [t.typeName, t.className, t.number, t.changed, t.annotation, t.modifier, t.self, t.namespace], color: '#9333ea' },
  { tag: [t.operator, t.operatorKeyword], color: '#475569' },
  { tag: [t.url, t.escape, t.regexp, t.link], color: '#0d9488' },
  { tag: [t.meta, t.comment], color: '#71717a', fontStyle: 'italic' },
  { tag: t.strong, fontWeight: 'bold' },
  { tag: t.emphasis, fontStyle: 'italic' },
  { tag: t.strikethrough, textDecoration: 'line-through' },
  { tag: t.link, color: '#4f46e5', textDecoration: 'underline' },
  { tag: t.heading, color: '#7c3aed', fontWeight: 'bold' },
  { tag: [t.atom, t.bool, t.special(t.variableName)], color: '#9333ea' },
  { tag: [t.processingInstruction, t.string, t.inserted], color: '#16a34a' },
  { tag: t.invalid, color: '#dc2626' },
]);

const darkHighlightStyle = HighlightStyle.define([
  { tag: t.keyword, color: '#c084fc' },
  { tag: [t.name, t.deleted, t.character, t.macroName], color: '#e4e4e7' },
  { tag: [t.function(t.variableName), t.labelName], color: '#a78bfa' },
  { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: '#67e8f9' },
  { tag: [t.definition(t.name), t.separator], color: '#e4e4e7' },
  { tag: [t.typeName, t.className, t.number, t.changed, t.annotation, t.modifier, t.self, t.namespace], color: '#f0abfc' },
  { tag: [t.operator, t.operatorKeyword], color: '#94a3b8' },
  { tag: [t.url, t.escape, t.regexp, t.link], color: '#2dd4bf' },
  { tag: [t.meta, t.comment], color: '#71717a', fontStyle: 'italic' },
  { tag: t.strong, fontWeight: 'bold' },
  { tag: t.emphasis, fontStyle: 'italic' },
  { tag: t.strikethrough, textDecoration: 'line-through' },
  { tag: t.link, color: '#a78bfa', textDecoration: 'underline' },
  { tag: t.heading, color: '#c084fc', fontWeight: 'bold' },
  { tag: [t.atom, t.bool, t.special(t.variableName)], color: '#f0abfc' },
  { tag: [t.processingInstruction, t.string, t.inserted], color: '#86efac' },
  { tag: t.invalid, color: '#fca5a5' },
]);

export function CodeEditor({
  value,
  onChange,
  path,
  readOnly = false,
  className,
  lineNumbers = true,
}: CodeEditorProps) {
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';

  const extensions = useMemo(() => {
    const lang = getLanguageExtension(path);
    const highlightStyle = isDark ? darkHighlightStyle : lightHighlightStyle;
    return [
      ...lang,
      EditorView.lineWrapping,
      EditorView.editable.of(!readOnly),
      syntaxHighlighting(highlightStyle),
    ];
  }, [path, readOnly, isDark]);

  return (
    <div className={cn('h-full w-full overflow-hidden', className)}>
      <CodeMirror
        value={value}
        onChange={onChange}
        theme={isDark ? darkTheme : lightTheme}
        extensions={extensions}
        readOnly={readOnly}
        basicSetup={{
          lineNumbers,
          foldGutter: true,
          highlightActiveLine: true,
          highlightActiveLineGutter: true,
          autocompletion: false,
          bracketMatching: true,
          closeBrackets: true,
          indentOnInput: true,
          syntaxHighlighting: true,
          highlightSelectionMatches: true,
          tabSize: 2,
          searchKeymap: true,
        }}
        height="100%"
        className="h-full text-[13px]"
      />
    </div>
  );
}
