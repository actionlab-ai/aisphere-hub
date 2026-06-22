'use client';

import { useState, useMemo } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Check, AlertCircle } from 'lucide-react';
import { cn, isValidResourceId, isValidVersion } from '@/lib/utils';
import { useT } from '@/lib/i18n';

interface ResourceIdInputProps {
  value: string;
  onChange: (value: string) => void;
  label?: string;
  placeholder?: string;
  disabled?: boolean;
  /** Validate as a version string (x.y.z) instead of a resource ID */
  validateVersion?: boolean;
  /** Whether to show the hint text below the input */
  showHint?: boolean;
  /** Whether the field is required */
  required?: boolean;
  className?: string;
}

/**
 * Input with built-in validation for resource IDs (skill name, skillset name, etc.).
 * IDs must be lowercase letters, digits, hyphens and underscores only.
 * Display names can be anything (use a plain Input for those).
 */
export function ResourceIdInput({
  value,
  onChange,
  label,
  placeholder,
  disabled,
  validateVersion = false,
  showHint = true,
  required = false,
  className,
}: ResourceIdInputProps) {
  const t = useT();
  const [touched, setTouched] = useState(false);

  const isValid = useMemo(() => {
    if (!value) return !required;
    return validateVersion ? isValidVersion(value) : isValidResourceId(value);
  }, [value, required, validateVersion]);

  const showError = touched && !isValid && Boolean(value);
  const labelKey = validateVersion ? 'id.versionLabel' : 'id.label';
  const hintKey = validateVersion ? 'id.versionHint' : 'id.hint';
  const invalidKey = validateVersion ? 'id.versionInvalid' : 'id.invalid';

  return (
    <div className={cn('space-y-1.5', className)}>
      {label && (
        <Label className="flex items-center gap-1">
          {label}
          {required && <span className="text-destructive">*</span>}
        </Label>
      )}
      <Input
        value={value}
        onChange={(e) => {
          onChange(e.target.value);
          if (!touched) setTouched(true);
        }}
        onBlur={() => setTouched(true)}
        placeholder={placeholder}
        disabled={disabled}
        className={cn(
          'font-mono text-xs',
          showError && 'border-destructive focus-visible:ring-destructive',
          touched && isValid && value && 'border-emerald-500/50',
        )}
        autoCapitalize="none"
        autoCorrect="off"
        spellCheck={false}
      />
      {showError && (
        <div className="flex items-start gap-1.5 text-[11px] text-destructive">
          <AlertCircle className="h-3 w-3 mt-0.5 shrink-0" />
          <span>{t(invalidKey)}</span>
        </div>
      )}
      {!showError && showHint && (
        <div className="flex items-start gap-1.5 text-[10px] text-muted-foreground">
          {touched && isValid && value ? (
            <Check className="h-3 w-3 mt-0.5 shrink-0 text-emerald-500" />
          ) : null}
          <span>{t(hintKey)}</span>
        </div>
      )}
    </div>
  );
}
