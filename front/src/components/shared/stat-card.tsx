'use client';

import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  trend?: string;
  accent?: 'violet' | 'emerald' | 'amber' | 'sky' | 'rose';
}

const accentClasses: Record<NonNullable<StatCardProps['accent']>, string> = {
  violet: 'bg-violet-500/10 text-violet-600 dark:text-violet-400',
  emerald: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  amber: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
  sky: 'bg-sky-500/10 text-sky-600 dark:text-sky-400',
  rose: 'bg-rose-500/10 text-rose-600 dark:text-rose-400',
};

export function StatCard({ icon, label, value, trend, accent = 'violet' }: StatCardProps) {
  return (
    <Card className="border-border/50 hover:border-border transition-colors">
      <CardContent className="p-3.5 flex items-center gap-3">
        <div
          className={cn(
            'flex items-center justify-center w-10 h-10 rounded-lg shrink-0 transition-transform',
            accentClasses[accent],
          )}
        >
          {icon}
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-[10px] text-muted-foreground uppercase tracking-wide truncate">{label}</div>
          <div className="text-xl font-semibold tracking-tight tabular-nums leading-tight">{value}</div>
          {trend && <div className="text-[10px] text-emerald-600 dark:text-emerald-400 mt-0.5">{trend}</div>}
        </div>
      </CardContent>
    </Card>
  );
}
