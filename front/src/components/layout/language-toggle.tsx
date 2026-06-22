'use client';

import { Globe, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from '@/components/ui/dropdown-menu';
import { useI18n, type Locale } from '@/lib/i18n';
import { cn } from '@/lib/utils';

const LOCALES: { value: Locale; label: string; native: string }[] = [
  { value: 'zh', label: 'Chinese', native: '中文' },
  { value: 'en', label: 'English', native: 'English' },
];

export function LanguageToggle({ compact = false }: { compact?: boolean }) {
  const { locale, setLocale } = useI18n();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size={compact ? 'icon' : 'sm'}
          className={cn(compact ? 'h-7 w-7' : 'h-7 px-2', 'gap-1.5 text-muted-foreground hover:text-foreground')}
          title="Language / 语言"
        >
          <Globe className="h-3.5 w-3.5" />
          {!compact && (
            <span className="text-xs uppercase">{locale}</span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        <DropdownMenuLabel className="text-[10px] uppercase tracking-wider text-muted-foreground">
          Language / 语言
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {LOCALES.map((l) => (
          <DropdownMenuItem
            key={l.value}
            onClick={() => setLocale(l.value)}
            className="flex items-center justify-between gap-2 cursor-pointer"
          >
            <div className="flex flex-col">
              <span className="text-xs font-medium">{l.native}</span>
              <span className="text-[10px] text-muted-foreground">{l.label}</span>
            </div>
            {locale === l.value && <Check className="h-3.5 w-3.5 text-violet-500" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
