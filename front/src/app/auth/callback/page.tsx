'use client';

import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { setTokens } from '@/lib/api/client';
import { useT } from '@/lib/i18n';

function readTokenParam(params: URLSearchParams, key: string): string {
  return params.get(key) || params.get(key.replace('_', '')) || '';
}

export default function AuthCallbackPage() {
  const t = useT();
  // Compute the message ONCE during initial render based on the URL params.
  const [message] = useState(() => {
    if (typeof window === 'undefined') return '';
    const hashParams = new URLSearchParams(window.location.hash.replace(/^#/, ''));
    const queryParams = new URLSearchParams(window.location.search.replace(/^\?/, ''));
    const accessToken = readTokenParam(hashParams, 'access_token') || readTokenParam(queryParams, 'access_token');
    if (!accessToken) {
      return t('login.noToken');
    }
    return t('login.storing');
  });

  useEffect(() => {
    const hashParams = new URLSearchParams(window.location.hash.replace(/^#/, ''));
    const queryParams = new URLSearchParams(window.location.search.replace(/^\?/, ''));
    const accessToken = readTokenParam(hashParams, 'access_token') || readTokenParam(queryParams, 'access_token');
    const refreshToken = readTokenParam(hashParams, 'refresh_token') || readTokenParam(queryParams, 'refresh_token');
    const next = queryParams.get('next') || process.env.NEXT_PUBLIC_AUTH_REDIRECT_AFTER_LOGIN || '/';

    if (!accessToken) {
      return;
    }
    setTokens(accessToken, refreshToken || undefined);
    window.history.replaceState({}, document.title, window.location.pathname);
    window.location.replace(next.startsWith('/') ? next : '/');
  }, []);

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="flex items-center gap-3 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" />
        <span>{message || t('login.completing')}</span>
      </div>
    </div>
  );
}
