'use client';

import { cn } from '@/lib/utils';

interface EmptyStateProps {
  icon: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('text-center py-16 text-muted-foreground', className)}>
      <div className="mx-auto mb-3 opacity-30 w-fit">{icon}</div>
      <p className="text-sm font-medium">{title}</p>
      {description && <p className="text-xs mt-1 max-w-xs mx-auto">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
