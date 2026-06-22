'use client';

interface InfoItemProps {
  label: string;
  value: string;
  mono?: boolean;
}

export function InfoItem({ label, value, mono }: InfoItemProps) {
  return (
    <div className="rounded-lg bg-muted/50 p-2.5">
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className={`text-sm font-medium mt-0.5 ${mono ? 'font-mono text-xs' : ''}`}>{value}</div>
    </div>
  );
}
