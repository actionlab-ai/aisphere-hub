'use client';

import { useState } from 'react';
import { Loader2, ShieldCheck, Globe, Sparkles, ArrowRight } from 'lucide-react';
import { motion } from 'framer-motion';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { authApi } from '@/lib/api';
import { setTokens } from '@/lib/api/client';
import { useT } from '@/lib/i18n';
import { LanguageToggle } from '@/components/layout/language-toggle';

interface LoginPageProps {
  onDone: () => void;
}

export function LoginPage({ onDone }: LoginPageProps) {
  const t = useT();
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(false);

  const loginWithCasdoor = () => {
    const callbackPath = process.env.NEXT_PUBLIC_AUTH_CALLBACK_PATH || '/auth/callback';
    const callbackUrl = callbackPath.startsWith('http')
      ? callbackPath
      : `${window.location.origin}${callbackPath}`;
    window.location.href = `/v3/auth/oidc/login?redirect=${encodeURIComponent(callbackUrl)}`;
  };

  const loginWithAISphere = () => {
    const here = window.location.origin + window.location.pathname;
    window.location.href = `/v3/auth/aisphere/login?redirect=${encodeURIComponent(here)}`;
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setLoading(true);
    try {
      const result = await authApi.login(username, password);
      setTokens(result.accessToken, result.refreshToken);
      onDone();
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : t('login.loginFailed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="relative flex items-center justify-center min-h-screen bg-gradient-to-br from-background via-background to-violet-50 dark:to-violet-950/30 p-4 overflow-hidden">
      {/* Decorative background blobs */}
      <div className="absolute top-1/4 -left-32 w-96 h-96 rounded-full bg-violet-500/10 dark:bg-violet-500/5 blur-3xl pointer-events-none" />
      <div className="absolute bottom-1/4 -right-32 w-96 h-96 rounded-full bg-fuchsia-500/10 dark:bg-fuchsia-500/5 blur-3xl pointer-events-none" />

      {/* Top-right language toggle */}
      <div className="absolute top-4 right-4 z-10">
        <LanguageToggle />
      </div>

      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="relative w-full max-w-md"
      >
        <Card className="shadow-2xl shadow-violet-500/10 border-violet-500/10 backdrop-blur-sm">
          <CardContent className="pt-8 pb-8 px-8">
            {/* Brand */}
            <div className="flex flex-col items-center text-center mb-6">
              <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-violet-600 to-fuchsia-500 text-white flex items-center justify-center shadow-lg shadow-violet-500/30 mb-3">
                <Sparkles className="h-7 w-7" />
              </div>
              <h1 className="text-2xl font-semibold tracking-tight">{t('login.title')}</h1>
              <p className="text-sm text-muted-foreground mt-1">{t('login.subtitle')}</p>
            </div>

            {/* SSO options */}
            <div className="space-y-2.5">
              <Button
                type="button"
                className="w-full h-11 bg-gradient-to-r from-violet-600 to-fuchsia-500 hover:from-violet-700 hover:to-fuchsia-600 shadow-md shadow-violet-500/20"
                onClick={loginWithAISphere}
              >
                <Globe className="h-4 w-4 mr-2" />
                {t('login.aiSphere')}
                <ArrowRight className="h-3.5 w-3.5 ml-2 opacity-80" />
              </Button>
              <Button
                type="button"
                variant="outline"
                className="w-full h-11"
                onClick={loginWithCasdoor}
              >
                <ShieldCheck className="h-4 w-4 mr-2 text-violet-500" />
                {t('login.casdoor')}
              </Button>
            </div>

            <div className="relative my-5">
              <Separator />
              <div className="absolute inset-0 flex items-center justify-center">
                <span className="bg-card px-3 text-[10px] uppercase tracking-wider text-muted-foreground">
                  {t('login.local')}
                </span>
              </div>
            </div>

            <form onSubmit={submit} className="space-y-3">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">{t('login.username')}</label>
                <Input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="admin"
                  className="h-10"
                  autoComplete="username"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">{t('login.password')}</label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t('login.passwordPlaceholder')}
                  className="h-10"
                  autoComplete="current-password"
                />
              </div>
              {err && (
                <div className="text-xs text-destructive bg-destructive/10 px-3 py-2 rounded-md border border-destructive/20">
                  {err}
                </div>
              )}
              <Button
                type="submit"
                className="w-full h-10"
                disabled={loading}
              >
                {loading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                {t('login.signIn')}
              </Button>
            </form>

            <p className="text-[11px] text-center text-muted-foreground mt-5 leading-relaxed whitespace-pre-line">
              {t('login.note')}
            </p>
          </CardContent>
        </Card>

        <p className="text-center text-[10px] text-muted-foreground/70 mt-4">
          {t('login.footer')}
        </p>
      </motion.div>
    </div>
  );
}
